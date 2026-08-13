package main

import "math"

// Constants matching classic retro video specifications
const (
	LutRadius       = 128
	LutSize         = LutRadius * 2
	contourBinCount = 512

	maxVelocity           = 4.0
	restRadius            = 35.0
	verletDamping         = 0.985
	constraintIterations  = 4
	spokeStiffness        = 0.2
	neighborStiffness     = 0.55
	areaStiffness         = 0.35
	contactRestitution    = 0.8
	contactCorrection     = 0.5
	contactSlop           = 0.1
	contactReleaseBand    = 8.0
	contactDentResponse   = 0.45
	contactVelocityDent   = 0.8
	maxContactDent        = 8.0
	energyRestoreDelay    = 60
	energyRestoreDeadband = 0.03
	energyRestoreRate     = 0.01
	maxRestoredSpeedStep  = 0.05

	fieldStrength    = 180000
	surfaceThreshold = 140
)

func newHeatPalette() [256]uint16 {
	var palette [256]uint16
	for i := 0; i < 256; i++ {
		var r, g, b uint8
		if i < 128 {
			r = uint8(i * 2)
		} else if i < 200 {
			r = 255
			g = uint8((i - 128) * 3)
		} else {
			r = 255
			g = 255
			b = uint8((i - 200) * 4)
		}
		palette[i] = convertToRGB565(r, g, b)
	}
	return palette
}

type VerletPoint struct {
	position         Vec2
	previousPosition Vec2
}

// Ball structure tracking individual 2D velocity vectors
type Ball struct {
	x, y   float32
	dx, dy float32

	points       []VerletPoint
	contourRadii []float32
	targetArea   float32
}

func iterationStiffness(stiffness float32) float32 {
	return 1 - float32(math.Pow(float64(1-stiffness), 1.0/constraintIterations))
}

func (b *Ball) area() float32 {
	area := float32(0)
	for pointIdx := range b.points {
		area += cross(b.points[pointIdx].position, b.points[(pointIdx+1)%len(b.points)].position)
	}
	return area * 0.5
}

func (b *Ball) updateVerlet() {
	for pointIdx := range b.points {
		point := &b.points[pointIdx]
		velocity := point.position.sub(point.previousPosition).scale(verletDamping)
		point.previousPosition = point.position
		point.position = point.position.add(velocity)
	}
	b.solveConstraints()
}

func (b *Ball) solveConstraints() {
	spokeStep := iterationStiffness(spokeStiffness)
	neighborStep := iterationStiffness(neighborStiffness)
	areaStep := iterationStiffness(areaStiffness)
	neighborRestLength := float32(2 * restRadius * math.Sin(math.Pi/float64(len(b.points))))
	for iteration := 0; iteration < constraintIterations; iteration++ {
		for pointIdx := range b.points {
			point := &b.points[pointIdx]
			length := point.position.length()
			if length > 0 {
				correction := point.position.scale((length - restRadius) / length * spokeStep)
				point.position = point.position.sub(correction)
			}
		}

		for pointIdx := range b.points {
			nextIdx := (pointIdx + 1) % len(b.points)
			delta := b.points[nextIdx].position.sub(b.points[pointIdx].position)
			length := delta.length()
			if length == 0 {
				continue
			}
			correction := delta.scale((length - neighborRestLength) / length * 0.5 * neighborStep)
			b.points[pointIdx].position = b.points[pointIdx].position.add(correction)
			b.points[nextIdx].position = b.points[nextIdx].position.sub(correction)
		}

		currentArea := b.area()
		if currentArea > 0 && b.targetArea > 0 {
			targetScale := float32(math.Sqrt(float64(b.targetArea / currentArea)))
			scale := 1 + (targetScale-1)*areaStep
			for pointIdx := range b.points {
				b.points[pointIdx].position = b.points[pointIdx].position.scale(scale)
			}
		}
	}
}

func (b *Ball) resolveWallContact(direction Vec2, penetration float32) {
	if penetration <= 0 {
		return
	}

	if direction.x < 0 {
		b.x += penetration
	} else if direction.x > 0 {
		b.x -= penetration
	} else if direction.y < 0 {
		b.y += penetration
	} else {
		b.y -= penetration
	}

	normalVelocity := b.dx*direction.x + b.dy*direction.y
	closingSpeed := float32(0)
	if normalVelocity > 0 {
		closingSpeed = normalVelocity
		impulse := (1 + contactRestitution) * normalVelocity
		b.dx -= direction.x * impulse
		b.dy -= direction.y * impulse
	}
	b.applyContactDent(direction, penetration, closingSpeed)
}

func (b *Ball) keepInside(width, height float32) {
	if penetration := b.supportRadius(Vec2{x: -1}) - b.x; penetration > 0 {
		b.x += penetration
	}
	if penetration := b.x + b.supportRadius(Vec2{x: 1}) - width; penetration > 0 {
		b.x -= penetration
	}
	if penetration := b.supportRadius(Vec2{y: -1}) - b.y; penetration > 0 {
		b.y += penetration
	}
	if penetration := b.y + b.supportRadius(Vec2{y: 1}) - height; penetration > 0 {
		b.y -= penetration
	}
}

func (b *Ball) supportRadius(direction Vec2) float32 {
	support := float32(0)
	for _, point := range b.points {
		projection := point.position.dot(direction)
		if projection > support {
			support = projection
		}
	}
	return support
}

func (b *Ball) applyContactDent(direction Vec2, penetration, closingSpeed float32) {
	dent := penetration*contactDentResponse + closingSpeed*contactVelocityDent
	if dent > maxContactDent {
		dent = maxContactDent
	}
	for pointIdx := range b.points {
		point := &b.points[pointIdx]
		length := point.position.length()
		if length == 0 {
			continue
		}
		alignment := point.position.dot(direction) / length
		const contactLobeStart = 0.25
		if alignment <= contactLobeStart {
			continue
		}
		weight := (alignment - contactLobeStart) / (1 - contactLobeStart)
		weight *= weight
		displacement := direction.scale(dent * weight)
		point.position = point.position.sub(displacement)
		point.previousPosition = point.previousPosition.sub(displacement.scale(0.7))
	}
}

func (b *Ball) rebuildContour() {
	for bin := range b.contourRadii {
		angle := 2 * math.Pi * float64(bin) / float64(len(b.contourRadii))
		direction := Vec2{x: float32(math.Cos(angle)), y: float32(math.Sin(angle))}
		radius := float32(LutRadius)
		found := false
		for pointIdx := range b.points {
			start := b.points[pointIdx].position
			end := b.points[(pointIdx+1)%len(b.points)].position
			edge := end.sub(start)
			denominator := cross(direction, edge)
			if absFloat32(denominator) < 0.000001 {
				continue
			}
			distance := cross(start, edge) / denominator
			edgeFraction := cross(start, direction) / denominator
			if distance >= 0 && edgeFraction >= 0 && edgeFraction <= 1 && distance < radius {
				radius = distance
				found = true
			}
		}
		if !found || radius < 1 {
			radius = restRadius
		}
		b.contourRadii[bin] = radius
	}
}

// MetaBallEffect encapsulates all buffers, configurations, and states for the effect.
type MetaBallEffect struct {
	// Static Lookup Tables and Render Buffers
	intensityTable  [LutSize * LutSize]uint8
	angleBinTable   [LutSize * LutSize]uint16
	masterAccumGrid []uint8
	liquidPalette   [256]uint16

	numBalls           int32
	pointCount         int
	neighborRestLength float32
	gridSize           int32
	width              int32
	height             int32

	// Physics tuning constants
	attractionForce           float32 // Gentle pull to keep balls from drifting too far apart
	targetTranslationalEnergy float32

	// Dynamic State Data
	balls      []Ball
	randState  uint32 // Serves as the mutable state variable for pseudoRand
	frameCount uint32
}

// NewMetaBallEffect instantiates and safely initializes a new effect state.
func NewMetaBallEffect(seed uint32, numBalls int32, pointCount int, width, height int32) *MetaBallEffect {
	if pointCount < 3 {
		panic("metaball point count must be at least 3")
	}
	effect := &MetaBallEffect{
		numBalls:           numBalls,
		pointCount:         pointCount,
		neighborRestLength: float32(2 * restRadius * math.Sin(math.Pi/float64(pointCount))),
		gridSize:           width * height,
		width:              width,
		height:             height,
		balls:              make([]Ball, numBalls),
		masterAccumGrid:    make([]uint8, width*height),

		attractionForce: 1.0 / 25600, // Gentle pull to keep balls from drifting too far apart

		randState:  seed,
		frameCount: 0,
	}
	effect.init()
	effect.targetTranslationalEnergy = effect.translationalEnergy()
	return effect
}

func (m *MetaBallEffect) translationalEnergy() float32 {
	energy := float32(0)
	for _, ball := range m.balls {
		energy += ball.dx*ball.dx + ball.dy*ball.dy
	}
	return energy
}

func (m *MetaBallEffect) restoreTranslationalEnergy() {
	if m.frameCount < energyRestoreDelay || m.targetTranslationalEnergy <= 0 || len(m.balls) == 0 {
		return
	}

	currentEnergy := m.translationalEnergy()
	minimumEnergy := m.targetTranslationalEnergy * (1 - energyRestoreDeadband)
	if currentEnergy >= minimumEnergy {
		return
	}

	start := int(m.frameCount % uint32(len(m.balls)))
	slowestIdx := start
	slowestSpeedSq := float32(math.MaxFloat32)
	for offset := 0; offset < len(m.balls); offset++ {
		ballIdx := (start + offset) % len(m.balls)
		ball := &m.balls[ballIdx]
		speedSq := ball.dx*ball.dx + ball.dy*ball.dy
		if speedSq < slowestSpeedSq {
			slowestSpeedSq = speedSq
			slowestIdx = ballIdx
		}
	}

	energyBudget := (m.targetTranslationalEnergy - currentEnergy) * energyRestoreRate
	oldSpeed := float32(math.Sqrt(float64(slowestSpeedSq)))
	maxNewSpeed := oldSpeed + maxRestoredSpeedStep
	if maxNewSpeed > maxVelocity {
		maxNewSpeed = maxVelocity
	}
	maxAddedEnergy := maxNewSpeed*maxNewSpeed - slowestSpeedSq
	if energyBudget > maxAddedEnergy {
		energyBudget = maxAddedEnergy
	}
	if remaining := m.targetTranslationalEnergy - currentEnergy; energyBudget > remaining {
		energyBudget = remaining
	}
	if energyBudget <= 0 {
		return
	}

	ball := &m.balls[slowestIdx]
	direction := Vec2{}
	if oldSpeed > 0.0001 {
		direction = Vec2{x: ball.dx / oldSpeed, y: ball.dy / oldSpeed}
	} else {
		centroid := Vec2{}
		for ballIdx := range m.balls {
			if ballIdx != slowestIdx {
				centroid.x += m.balls[ballIdx].x
				centroid.y += m.balls[ballIdx].y
			}
		}
		if len(m.balls) > 1 {
			centroid = centroid.scale(1 / float32(len(m.balls)-1))
		}
		direction = Vec2{x: ball.x - centroid.x, y: ball.y - centroid.y}
		length := direction.length()
		if length > 0.0001 {
			direction = direction.scale(1 / length)
		} else {
			angle := 2 * math.Pi * float64(slowestIdx+int(m.frameCount)) / float64(len(m.balls)+1)
			direction = Vec2{x: float32(math.Cos(angle)), y: float32(math.Sin(angle))}
		}
	}

	newSpeed := float32(math.Sqrt(float64(slowestSpeedSq + energyBudget)))
	ball.dx = direction.x * newSpeed
	ball.dy = direction.y * newSpeed
}

// pseudoRand implements your exact PCG-XSH-RR hash mixing function
func (m *MetaBallEffect) pseudoRand() uint32 {
	state := m.randState*uint32(747796405) + uint32(2891336453)
	word := ((state >> ((state >> 28) + 4)) ^ state) * uint32(277803737)
	m.randState += 1
	return (word >> 22) ^ word
}

// customRandFloat scales pseudoRand output into a specific [min, max) range.
func (m *MetaBallEffect) customRandFloat(min, max float32) float32 {
	unit := float32(m.pseudoRand()>>8) / float32(1<<24)
	return min + unit*(max-min)
}

// init builds lookup tables and assigns initial coordinates upon creation
func (m *MetaBallEffect) init() {
	// 1. Generate Inverse-Distance Intensity LUT
	for y := 0; y < LutSize; y++ {
		for x := 0; x < LutSize; x++ {
			dx := int32(x - LutRadius)
			dy := int32(y - LutRadius)
			distSq := (dx * dx) + (dy * dy)

			if distSq == 0 {
				distSq = 1
			}

			// Classic 1/r field formula scaled to fit cleanly into a single byte
			intensity := fieldStrength / distSq
			if intensity > 255 {
				intensity = 255
			}
			m.intensityTable[y*LutSize+x] = uint8(intensity)

			angle := math.Atan2(float64(dy), float64(dx))
			if angle < 0 {
				angle += 2 * math.Pi
			}
			m.angleBinTable[y*LutSize+x] = uint16(angle / (2 * math.Pi) * contourBinCount)
		}
	}

	// 2. Generate a metallic/electric liquid palette (Deep Red -> Bright Orange -> Yellow)
	m.liquidPalette = newHeatPalette()

	// 3. Populate physics fields in pixel units.
	for i := 0; i < int(m.numBalls); i++ {
		m.balls[i].x = m.customRandFloat(50, float32(m.width-50))
		m.balls[i].y = m.customRandFloat(50, float32(m.height-50))

		m.balls[i].dx = m.customRandFloat(0.5, 1.5)
		m.balls[i].dy = m.customRandFloat(0.5, 1.5)

		if m.pseudoRand()%2 == 1 {
			m.balls[i].dx = -m.balls[i].dx
		}
		if m.pseudoRand()%2 == 1 {
			m.balls[i].dy = -m.balls[i].dy
		}

		m.balls[i].points = make([]VerletPoint, m.pointCount)
		m.balls[i].contourRadii = make([]float32, contourBinCount)
		for pointIdx := range m.balls[i].points {
			angle := 2 * math.Pi * float64(pointIdx) / float64(m.pointCount)
			position := Vec2{
				x: restRadius * float32(math.Cos(angle)),
				y: restRadius * float32(math.Sin(angle)),
			}
			m.balls[i].points[pointIdx] = VerletPoint{
				position:         position,
				previousPosition: position,
			}
		}
		m.balls[i].targetArea = m.balls[i].area()
		m.balls[i].rebuildContour()
	}
}

// updatePhysics applies ring collisions, Verlet integration, movement, and wall bounces.
func (m *MetaBallEffect) updatePhysics() {
	// 1. Resolve each ball pair once using the deformed ring support radii.
	for i := 0; i < int(m.numBalls); i++ {
		for j := i + 1; j < int(m.numBalls); j++ {
			dx := m.balls[j].x - m.balls[i].x
			dy := m.balls[j].y - m.balls[i].y
			distSq := (dx * dx) + (dy * dy)
			distance := float32(math.Sqrt(float64(distSq)))
			normal := Vec2{x: 1}
			if distance > 0 {
				normal = Vec2{x: dx / distance, y: dy / distance}
			}

			contactDistance := m.balls[i].supportRadius(normal) + m.balls[j].supportRadius(normal.scale(-1))
			if penetration := contactDistance - distance; penetration > 0 {
				correction := (penetration - contactSlop) * contactCorrection * 0.5
				if correction > 0 {
					m.balls[i].x -= normal.x * correction
					m.balls[i].y -= normal.y * correction
					m.balls[j].x += normal.x * correction
					m.balls[j].y += normal.y * correction
				}

				relativeNormalVelocity := (m.balls[j].dx-m.balls[i].dx)*normal.x + (m.balls[j].dy-m.balls[i].dy)*normal.y
				closingSpeed := float32(0)
				if relativeNormalVelocity < 0 {
					closingSpeed = -relativeNormalVelocity
					impulse := -(1 + contactRestitution) * relativeNormalVelocity * 0.5
					m.balls[i].dx -= normal.x * impulse
					m.balls[i].dy -= normal.y * impulse
					m.balls[j].dx += normal.x * impulse
					m.balls[j].dy += normal.y * impulse
				}
				m.balls[i].applyContactDent(normal, penetration, closingSpeed)
				m.balls[j].applyContactDent(normal.scale(-1), penetration, closingSpeed)
			} else if distance > contactDistance+contactReleaseBand && distSq < 25000 {
				m.balls[i].dx += dx * m.attractionForce
				m.balls[i].dy += dy * m.attractionForce
				m.balls[j].dx -= dx * m.attractionForce
				m.balls[j].dy -= dy * m.attractionForce
			}
		}
	}

	// 2. Integrate and constrain each deformable circumference.
	for i := 0; i < int(m.numBalls); i++ {
		if m.frameCount%4 == 0 {
			targetPoint := int(m.pseudoRand() % uint32(len(m.balls[i].points)))
			point := &m.balls[i].points[targetPoint]
			length := point.position.length()
			if length > 0 {
				impulse := m.customRandFloat(-0.05, 0.05)
				normal := point.position.scale(1 / length)
				point.previousPosition = point.previousPosition.sub(normal.scale(impulse))
			}
		}
		m.balls[i].updateVerlet()
	}
	m.frameCount++

	// 3. Apply velocities and handle window boundary collisions
	for i := 0; i < int(m.numBalls); i++ {
		// Velocity drag clamping to prevent kinetic explosions from intense close-up repulsions
		if m.balls[i].dx > maxVelocity {
			m.balls[i].dx = maxVelocity
		}
		if m.balls[i].dx < -maxVelocity {
			m.balls[i].dx = -maxVelocity
		}
		if m.balls[i].dy > maxVelocity {
			m.balls[i].dy = maxVelocity
		}
		if m.balls[i].dy < -maxVelocity {
			m.balls[i].dy = -maxVelocity
		}

		m.balls[i].x += m.balls[i].dx
		m.balls[i].y += m.balls[i].dy

		ball := &m.balls[i]
		collided := false
		if penetration := ball.supportRadius(Vec2{x: -1}) - ball.x; penetration > 0 {
			ball.resolveWallContact(Vec2{x: -1}, penetration)
			collided = true
		}
		if penetration := ball.x + ball.supportRadius(Vec2{x: 1}) - float32(m.width); penetration > 0 {
			ball.resolveWallContact(Vec2{x: 1}, penetration)
			collided = true
		}
		if penetration := ball.supportRadius(Vec2{y: -1}) - ball.y; penetration > 0 {
			ball.resolveWallContact(Vec2{y: -1}, penetration)
			collided = true
		}
		if penetration := ball.y + ball.supportRadius(Vec2{y: 1}) - float32(m.height); penetration > 0 {
			ball.resolveWallContact(Vec2{y: 1}, penetration)
			collided = true
		}
		if collided {
			ball.solveConstraints()
			ball.keepInside(float32(m.width), float32(m.height))
		}
	}
	m.restoreTranslationalEnergy()
}

// ProcessFrame runs one physics update and rendering step directly into the target slice
func (m *MetaBallEffect) update() {
	// Step 1: Execute movement and collision physics
	m.updatePhysics()

	// Step 2: Clear screen grid
	for i := 0; i < int(m.gridSize); i++ {
		m.masterAccumGrid[i] = 0
	}

	// Step 3: Map field overlays onto screenspace coordinates
	for b := 0; b < int(m.numBalls); b++ {
		m.balls[b].rebuildContour()
		bx := int32(m.balls[b].x)
		by := int32(m.balls[b].y)

		for y := int32(0); y < m.height; y++ {
			dy := y - by
			if dy < -LutRadius || dy >= LutRadius {
				continue
			}
			gridY := y * m.width

			for x := int32(0); x < m.width; x++ {
				dx := x - bx
				if dx < -LutRadius || dx >= LutRadius {
					continue
				}

				angleTableIdx := (dy+LutRadius)*LutSize + dx + LutRadius
				angleBin := m.angleBinTable[angleTableIdx]
				contourRadius := m.balls[b].contourRadii[angleBin]
				warpScale := float32(restRadius) / contourRadius
				distX := float32(dx) * warpScale
				distY := float32(dy) * warpScale

				lutX := int32(distX) + LutRadius
				lutY := int32(distY) + LutRadius

				// Bounds safety checking
				if lutX >= 0 && lutX < LutSize && lutY >= 0 && lutY < LutSize {
					energy := m.intensityTable[lutY*LutSize+lutX]
					gridIdx := gridY + x

					if uint32(m.masterAccumGrid[gridIdx])+uint32(energy) > 255 {
						m.masterAccumGrid[gridIdx] = 255
					} else {
						m.masterAccumGrid[gridIdx] += energy
					}
				}
			}
		}
	}
}

func (m *MetaBallEffect) render(frameBuffer *FrameBuffer) {
	for y := int32(0); y < m.height; y++ {
		for x := int32(0); x < m.width; x++ {
			idx := y*m.width + x
			totalEnergy := m.masterAccumGrid[idx]
			if totalEnergy > surfaceThreshold {
				frameBuffer.Pixels[idx] = m.liquidPalette[totalEnergy]
			} else {
				frameBuffer.Pixels[idx] = 0
			}
		}
	}
}

func (m *MetaBallEffect) ProcessFrame(deltaTime float32, frameBuffer *FrameBuffer) {
	m.update()
	m.render(frameBuffer)
}
