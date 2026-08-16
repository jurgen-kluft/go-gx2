package fx_ripple

import (
	"math"

	fx_common "github.com/jurgen-kluft/go-gx2/test-apps/effects/common"
)

// https://github.com/jeantimex/ripples/blob/main/src/main.ts

type ParticleBase struct {
	OriginX float32
	OriginY float32
	X       float32
	Y       float32
	VX      float32
	VY      float32
}

type SimulationSettings struct {
	DotSpacing           float32
	DotRadius            float32
	RippleScaleAmplitude float32
	FieldCellSize        float32
	RippleSpeed          float32
	DropRadius           float32
	DropStrength         float32
	DragDropRadius       float32
	DragDropStrength     float32
	DragMinDistance      float32
	RippleForce          float32
	SpringStrength       float32
	MotionDamping        float32

	Idle_Lightness              float32
	Active_Lightness            float32
	Tone_Steps                  float32
	Position_Epsilon            float32
	Scale_Epsilon               float32
	Scale_Rest_Height_Epsilon   float32
	Scale_Rest_Speed_Epsilon    float32
	Ripple_Height_For_Max_Scale float32
	Ripple_Height_Range         float32
}

func NewDefaultSimulationSettings() SimulationSettings {
	return SimulationSettings{
		DotSpacing:           15,
		DotRadius:            3,
		RippleScaleAmplitude: 0.49,
		FieldCellSize:        18,
		RippleSpeed:          0.3,
		DropRadius:           79,
		DropStrength:         5.4,
		DragDropRadius:       25,
		DragDropStrength:     0.4,
		DragMinDistance:      3,
		RippleForce:          36300,
		SpringStrength:       7,
		MotionDamping:        11,

		Idle_Lightness:              222,
		Active_Lightness:            28,
		Tone_Steps:                  12,
		Position_Epsilon:            0.1,
		Scale_Epsilon:               0.003,
		Scale_Rest_Height_Epsilon:   0.18,
		Scale_Rest_Speed_Epsilon:    2.4,
		Ripple_Height_For_Max_Scale: 5,
		Ripple_Height_Range:         5,
	}
}

type RippleField struct {
	Settings       SimulationSettings
	Width          int32
	Height         int32
	CellSize       int32
	Columns        int32
	Rows           int32
	Heights        []float32
	Velocities     []float32
	NextHeights    []float32
	NextVelocities []float32

	AccumulatedTime float32
}

func NewEffect(width, height, cellSize int32) *RippleField {
	effect := &RippleField{
		Settings: NewDefaultSimulationSettings(),
		Width:    width,
		Height:   height,
		CellSize: cellSize,
	}
	effect.resize(width, height)
	return effect
}

func (r *RippleField) resize(width, height int32) {
	r.Width = width
	r.Height = height
	r.Columns = r.Width / r.CellSize
	r.Rows = r.Height / r.CellSize
	makeFloat32Array := func() []float32 { return make([]float32, r.Columns*r.Rows) }
	r.Heights = makeFloat32Array()
	r.Velocities = makeFloat32Array()
	r.NextHeights = makeFloat32Array()
	r.NextVelocities = makeFloat32Array()
}

func (r *RippleField) disturb(x, y, radius int32, strength float32) {
	minColumn := clamp(int32((x-radius)/r.CellSize), 0, r.Columns-1)
	maxColumn := clamp(int32((x+radius)/r.CellSize), 0, r.Columns-1)
	minRow := clamp(int32((y-radius)/r.CellSize), 0, r.Rows-1)
	maxRow := clamp(int32((y+radius)/r.CellSize), 0, r.Rows-1)
	radiusSquared := radius * radius

	for row := int32(minRow); row <= int32(maxRow); row++ {
		rowOffset := row * r.Columns
		for column := int32(minColumn); column <= int32(maxColumn); column++ {
			sampleX := column * r.CellSize
			sampleY := row * r.CellSize
			distanceSquared := (x-sampleX)*(x-sampleX) + (y-sampleY)*(y-sampleY)
			if distanceSquared > radiusSquared {
				continue
			}

			normalizedDistance := 1 - sqrtf(float32(distanceSquared))/float32(radius)
			drop := 0.5 - float32(math.Cos(float64(normalizedDistance*math.Pi)))*0.5
			r.Heights[rowOffset+column] += drop * strength
		}
	}
}

func (r *RippleField) heightAt(column, row int32) float32 {
	return r.Heights[row*r.Columns+column]
}

func (r *RippleField) swapHeights() {
	r.Heights, r.NextHeights = r.NextHeights, r.Heights
}

func (r *RippleField) swapVelocities() {
	r.Velocities, r.NextVelocities = r.NextVelocities, r.Velocities
}

func (r *RippleField) step() {

	// for row := int32(0); row < r.Rows; row++ {
	// 	for column := int32(0); column < r.Columns; column++ {
	// 		index := row*r.Columns + column
	// 		height := r.Heights[index]
	// 		average := (r.heightAt(column-1, row) +
	// 			r.heightAt(column+1, row) +
	// 			r.heightAt(column, row-1) +
	// 			r.heightAt(column, row+1)) * 0.25

	// 		velocity := r.Velocities[index]
	// 		velocity += (average - height) * r.Settings.RippleSpeed
	// 		velocity *= 0.995

	// 		r.NextVelocities[index] = velocity
	// 		r.NextHeights[index] = height + velocity
	// 	}
	// }

	// We want to run the loop without having to check boundaries for row and column, so we
	// will handle introducing special cases.
	row := int32(0)
	column := int32(0)

	// -------------------------------------------------------------
	// Handle left top corner
	column = int32(0)
	index := row*r.Columns + column
	height := r.Heights[index]
	average := (r.heightAt(column+1, row) + r.heightAt(column, row+1)) * 0.5

	velocity := r.Velocities[index]
	velocity += (average - height) * r.Settings.RippleSpeed
	velocity *= 0.995

	r.NextVelocities[index] = velocity
	r.NextHeights[index] = height + velocity
	// -------------------------------------------------------------

	// -------------------------------------------------------------
	// Handle right top corner
	column = r.Columns - 1
	index = row*r.Columns + column
	height = r.Heights[index]
	average = (r.heightAt(column-1, row) + r.heightAt(column, row+1)) * 0.5

	velocity = r.Velocities[index]
	velocity += (average - height) * r.Settings.RippleSpeed
	velocity *= 0.995

	r.NextVelocities[index] = velocity
	r.NextHeights[index] = height + velocity
	// -------------------------------------------------------------

	// -------------------------------------------------------------
	// Handle top row (excluding corners)
	for column := int32(1); column < r.Columns-1; column++ {
		index := row*r.Columns + column
		height := r.Heights[index]
		average := (r.heightAt(column-1, row) +
			r.heightAt(column+1, row) +
			r.heightAt(column, row+1)) * 0.3333

		velocity := r.Velocities[index]
		velocity += (average - height) * r.Settings.RippleSpeed
		velocity *= 0.995

		r.NextVelocities[index] = velocity
		r.NextHeights[index] = height + velocity
	}

	// -------------------------------------------------------------
	// Handle grid, ignoring boundaries (we will handle them separately)
	for row := int32(1); row < r.Rows-1; row++ {
		for column := int32(1); column < r.Columns-1; column++ {
			index := row*r.Columns + column
			height := r.Heights[index]
			average := (r.heightAt(column-1, row) +
				r.heightAt(column+1, row) +
				r.heightAt(column, row-1) +
				r.heightAt(column, row+1)) * 0.25

			velocity := r.Velocities[index]
			velocity += (average - height) * r.Settings.RippleSpeed
			velocity *= 0.995

			r.NextVelocities[index] = velocity
			r.NextHeights[index] = height + velocity
		}
	}

	// -------------------------------------------------------------
	// Handle bottom row (excluding corners)
	row = r.Rows - 1
	for column := int32(1); column < r.Columns-1; column++ {
		index := row*r.Columns + column
		height := r.Heights[index]
		average := (r.heightAt(column-1, row) +
			r.heightAt(column+1, row) +
			r.heightAt(column, row-1)) * 0.3333

		velocity := r.Velocities[index]
		velocity += (average - height) * r.Settings.RippleSpeed
		velocity *= 0.995

		r.NextVelocities[index] = velocity
		r.NextHeights[index] = height + velocity
	}

	// -------------------------------------------------------------
	// Handle left bottom corner
	column = int32(0)
	index = row*r.Columns + column
	height = r.Heights[index]
	average = (r.heightAt(column+1, row) + r.heightAt(column, row-1)) * 0.5

	velocity = r.Velocities[index]
	velocity += (average - height) * r.Settings.RippleSpeed
	velocity *= 0.995

	r.NextVelocities[index] = velocity
	r.NextHeights[index] = height + velocity

	// -------------------------------------------------------------
	// Handle right bottom corner
	column = r.Columns - 1
	index = row*r.Columns + column
	height = r.Heights[index]
	average = (r.heightAt(column-1, row) + r.heightAt(column, row-1)) * 0.5

	velocity = r.Velocities[index]
	velocity += (average - height) * r.Settings.RippleSpeed
	velocity *= 0.995

	r.NextVelocities[index] = velocity
	r.NextHeights[index] = height + velocity

	// -------------------------------------------------------------
	// Done, swap the buffers
	r.swapHeights()
	r.swapVelocities()
}

func (r *RippleField) sampleHeight(x, y float32) float32 {
	clampedX := clamp(int32(x), 0, r.Width)
	clampedY := clamp(int32(y), 0, r.Height)
	gridX := clampedX / r.CellSize
	gridY := clampedY / r.CellSize
	x0 := gridX
	y0 := gridY
	x1 := min(gridX+1, r.Columns-1)
	y1 := min(gridY+1, r.Rows-1)
	tx := gridX - x0
	ty := gridY - y0
	h00 := r.heightAt(x0, y0)
	h10 := r.heightAt(x1, y0)
	h01 := r.heightAt(x0, y1)
	h11 := r.heightAt(x1, y1)
	top := lerpf(h00, h10, float32(tx))
	bottom := lerpf(h01, h11, float32(tx))
	return lerpf(top, bottom, float32(ty))
}

func (r *RippleField) gradientAt(x, y float32) (float32, float32) {
	epsilon := float32(r.CellSize)
	left := r.sampleHeight(x-epsilon, y)
	right := r.sampleHeight(x+epsilon, y)
	top := r.sampleHeight(x, y-epsilon)
	bottom := r.sampleHeight(x, y+epsilon)
	return (right - left) / (epsilon * 2), (bottom - top) / (epsilon * 2)
}

func (r *RippleField) updateParticles(dt float32, particles []ParticleBase) {
	for i := range particles {
		particle := &particles[i]
		gradientX, gradientY := r.gradientAt(particle.X, particle.Y)
		rippleFx := -gradientX * r.Settings.RippleForce
		rippleFy := -gradientY * r.Settings.RippleForce
		springFx := (particle.OriginX - particle.X) * r.Settings.SpringStrength
		springFy := (particle.OriginY - particle.Y) * r.Settings.SpringStrength
		dampingFx := -particle.VX * r.Settings.MotionDamping
		dampingFy := -particle.VY * r.Settings.MotionDamping

		ax := rippleFx + springFx + dampingFx
		ay := rippleFy + springFy + dampingFy

		particle.VX += ax * dt
		particle.VY += ay * dt
		particle.X += particle.VX * dt
		particle.Y += particle.VY * dt
	}
}

func (r *RippleField) UpdateSimulation(dt float32, particles []ParticleBase) {
	r.AccumulatedTime += minf(dt, 0.1) // maxFrameDelta

	steps := 0
	maxSubsteps := 10
	fixedTimeStep := float32(1.0 / 60.0)

	for r.AccumulatedTime >= fixedTimeStep && steps < maxSubsteps {
		r.step()
		r.updateParticles(fixedTimeStep, particles)
		r.AccumulatedTime -= fixedTimeStep
		steps++
	}
}

func (r *RippleField) getParticleSamplePoint(particle ParticleBase) (float32, float32) {
	return particle.X, particle.Y
}

func (r *RippleField) getParticleScale(particle ParticleBase) float32 {
	sampleX, sampleY := r.getParticleSamplePoint(particle)
	height := r.sampleHeight(sampleX, sampleY)
	speed := hypotf(particle.VX, particle.VY)

	if math.Abs(float64(height)) <= float64(r.Settings.Scale_Rest_Height_Epsilon) || speed <= r.Settings.Scale_Rest_Speed_Epsilon {
		return 1.0
	}

	normalizedHeight := clampf(inverseLerpf(-r.Settings.Ripple_Height_For_Max_Scale, r.Settings.Ripple_Height_For_Max_Scale, height)*2-1, -1, 1)
	scale := 1 + normalizedHeight*r.Settings.RippleScaleAmplitude

	if absf(scale-1) <= r.Settings.Scale_Epsilon {
		return 1.0
	}

	return scale
}

// function getParticleLightness(particle: Particle) {
//   const samplePoint = getParticleSamplePoint(particle)
//   const height = rippleField.surfaceHeightAt(samplePoint.x, samplePoint.y)
//   const normalizedHeight = clamp(
//     inverseLerp(0, TONE_RIPPLE_HEIGHT_RANGE, Math.abs(height)),
//     0,
//     1,
//   )
//   return Math.round(lerp(TEXT_IDLE_LIGHTNESS, TEXT_ACTIVE_LIGHTNESS, normalizedHeight))
// }

func (r *RippleField) getParticleLightness(particle ParticleBase) float32 {
	sampleX, sampleY := r.getParticleSamplePoint(particle)
	height := r.sampleHeight(sampleX, sampleY)
	normalizedHeight := clampf(inverseLerpf(0, r.Settings.Ripple_Height_Range, absf(height)), 0, 1)
	return lerpf(r.Settings.Idle_Lightness, r.Settings.Active_Lightness, normalizedHeight)
}

func drawCircle(fb *fx_common.FrameBuffer, x, y, radius, lightness float32) {

	// Scan-line circle drawing algorithm
	r2 := radius * radius
	minX := int32(x - radius)
	maxX := int32(x + radius)
	minY := int32(y - radius)
	maxY := int32(y + radius)

	for py := minY; py <= maxY; py++ {
		for px := minX; px <= maxX; px++ {
			dx := float32(px) - x
			dy := float32(py) - y
			if dx*dx+dy*dy <= r2 {
				r := uint8(lightness)
				g := uint8(lightness)
				b := uint8(lightness)

				fb.Pixels[py*fb.Width+px] = fx_common.ConvertToRGB565(r, g, b)
			}
		}
	}

}

func (r *RippleField) Draw(fb *fx_common.FrameBuffer, particles []ParticleBase) {
	for i := range particles {
		particle := &particles[i]
		lightness := r.getParticleLightness(*particle)
		scale := r.getParticleScale(*particle)
		radius := r.Settings.DotRadius * scale
		drawCircle(fb, particle.X, particle.Y, radius, lightness)
	}
}
