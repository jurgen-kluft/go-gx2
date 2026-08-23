package fx_fluid

import (
	"math/rand"

	fx_common "github.com/jurgen-kluft/go-gx2/test-apps/effects/common"
)

type boundaryAction int

const (
	copyBoundary boundaryAction = iota
	reflectBoundary
)

type swapStateAction int

const (
	swapVelocities swapStateAction = iota
	swapDensities
)

type VelocityData struct {
	XVel []float32 // XVelocities, range
	YVel []float32
}

type DensityData struct {
	Density []float32 // Density, range 0.0 - 1.0
}

type FluidEffect struct {
	CellCountX       int32
	CellCountY       int32
	PaddedCellCountX int32
	PaddedCellCountY int32
	CurrentVelocity  *VelocityData
	PreviousVelocity *VelocityData
	CurrentDensity   *DensityData
	PreviousDensity  *DensityData
	RelaxationSteps  int32
	DiffusionRate    float32
	Viscosity        float32
	FadeRate         float32
}

func NewEffect(cellCountX, cellCountY int32) *FluidEffect {
	// allocate additional space for boundary conditions
	paddedCellCountX := cellCountX + 2
	paddedCellCountY := cellCountY + 2
	makeArray2d := func() []float32 { return make([]float32, paddedCellCountX*paddedCellCountY) }

	currentVelocity := &VelocityData{
		XVel: makeArray2d(),
		YVel: makeArray2d(),
	}

	previousVelocity := &VelocityData{
		XVel: makeArray2d(),
		YVel: makeArray2d(),
	}

	currentDensity := &DensityData{
		Density: makeArray2d(),
	}

	previousDensity := &DensityData{
		Density: makeArray2d(),
	}

	diffusionRate := float32(0.0000125)
	viscosity := float32(0.0001)
	fadeRate := float32(0.0025)

	fluid := &FluidEffect{
		CellCountX:       cellCountX,
		CellCountY:       cellCountY,
		PaddedCellCountX: paddedCellCountX,
		PaddedCellCountY: paddedCellCountY,
		CurrentVelocity:  currentVelocity,
		PreviousVelocity: previousVelocity,
		CurrentDensity:   currentDensity,
		PreviousDensity:  previousDensity,
		RelaxationSteps:  20,
		DiffusionRate:    diffusionRate,
		Viscosity:        viscosity,
		FadeRate:         fadeRate,
	}

	return fluid
}

func (f *VelocityData) reset() {
	clear(f.XVel)
	clear(f.YVel)
}

func (f *DensityData) reset() {
	clear(f.Density)
}

func (f *FluidEffect) reset() {
	f.CurrentVelocity.reset()
	f.PreviousVelocity.reset()
	f.CurrentDensity.reset()
	f.PreviousDensity.reset()
}

func (f *FluidEffect) ProcessFrame(dt float32, fb *fx_common.FrameBuffer) {
	// randomly add some density and velocity to the fluid simulation
	if rand.Float32() < 0.5 {
		x := rand.Int31n(f.CellCountX) + 1
		y := rand.Int31n(f.CellCountY) + 1
		f.AddDensity(x, y, 1.0)
		f.AddVelocity(x, y, rand.Float32()*2-1, rand.Float32()*2-1)
	}

	f.simulateVelocity(dt)
	f.simulateDensity(dt)
	f.DrawDensityField(fb, color{r: 255, g: 255, b: 1}, false)
}

func (f *FluidEffect) AddDensity(x, y int32, val float32) {
	f.CurrentDensity.Density[x+(y*f.PaddedCellCountX)] = val
}

func (f *FluidEffect) AddVelocity(x, y int32, xval, yval float32) {
	f.CurrentVelocity.XVel[x+(y*f.PaddedCellCountX)] = xval
	f.CurrentVelocity.YVel[x+(y*f.PaddedCellCountX)] = yval
}

func (f *FluidEffect) swapState(s swapStateAction) {
	switch s {
	case swapVelocities:
		f.CurrentVelocity, f.PreviousVelocity = f.PreviousVelocity, f.CurrentVelocity
	case swapDensities:
		f.CurrentDensity, f.PreviousDensity = f.PreviousDensity, f.CurrentDensity
	}
}

func (f *FluidEffect) simulateVelocity(dt float32) {
	var viscosity float32 = f.Viscosity

	f.swapState(swapVelocities)
	f.diffuse(reflectBoundary, dt, f.CurrentVelocity.XVel, f.PreviousVelocity.XVel, viscosity)
	f.diffuse(reflectBoundary, dt, f.CurrentVelocity.YVel, f.PreviousVelocity.YVel, viscosity)
	f.project(f.CurrentVelocity.XVel, f.CurrentVelocity.YVel, f.PreviousVelocity.XVel, f.PreviousVelocity.YVel)

	f.swapState(swapVelocities)
	f.advect(reflectBoundary, dt, f.CurrentVelocity.XVel, f.PreviousVelocity.XVel, f.PreviousVelocity.XVel, f.PreviousVelocity.YVel)
	f.advect(reflectBoundary, dt, f.CurrentVelocity.YVel, f.PreviousVelocity.YVel, f.PreviousVelocity.XVel, f.PreviousVelocity.YVel)
	f.project(f.CurrentVelocity.XVel, f.CurrentVelocity.YVel, f.PreviousVelocity.XVel, f.PreviousVelocity.YVel)
}

func (f *FluidEffect) simulateDensity(dt float32) {
	f.fade(dt, f.CurrentDensity.Density, f.FadeRate)

	f.swapState(swapDensities)
	f.diffuse(copyBoundary, dt, f.CurrentDensity.Density, f.PreviousDensity.Density, f.DiffusionRate)

	f.swapState(swapDensities)
	f.advect(copyBoundary, dt, f.CurrentDensity.Density, f.PreviousDensity.Density, f.CurrentVelocity.XVel, f.CurrentVelocity.YVel)
}

func (f *FluidEffect) fade(dt float32, grid []float32, fadeRate float32) {
	for i := range grid {
		value := grid[i] - dt*fadeRate
		grid[i] = fabs(value)
	}
}

func (f *FluidEffect) advect(b boundaryAction, dt float32, grid []float32, gridPrev []float32, xVelocities, yVelocities []float32) {

	half := float32(0.5)
	cellCountXPlusHalf := float32(f.CellCountX) + half
	cellCountYPlusHalf := float32(f.CellCountY) + half

	yIdx := 1 * f.PaddedCellCountX
	for y := int32(1); y <= f.CellCountY; y++ {
		xIdx := yIdx
		for x := int32(1); x <= f.CellCountX; x++ {
			xv := xVelocities[xIdx]
			yv := yVelocities[xIdx]

			// calculate previous x and y positions of the current grid particle
			// by moving backwards along the velocity field.
			// by calculating new density based on previous particle position,
			// the simulation becomes bounded.
			px := float32(x) - xv*dt
			py := float32(y) - yv*dt

			px = clampf(px, half, cellCountXPlusHalf)
			py = clampf(py, half, cellCountYPlusHalf)

			val := f.bilinearInterpolate(px, py, gridPrev)

			grid[xIdx] = val
			xIdx++
		}
		yIdx += f.PaddedCellCountX
	}
	f.setBoundaries(b, grid)
}

func (f *FluidEffect) diffuse(b boundaryAction, dt float32, grid []float32, gridPrev []float32, diffusionRate float32) {
	// diffuse the density field
	// high density cells diffuse to low density cells

	// diffusion delta
	diffusionFactor := dt * diffusionRate * float32(f.CellCountX) * float32(f.CellCountX)

	const numNeighbors float32 = 4.0

	// Gauss-Seidel Relaxation
	for i := int32(0); i < f.RelaxationSteps; i++ {
		ys := 1 * f.PaddedCellCountX
		ye := ys + f.CellCountY
		for ys < ye {
			xs := ys + 1
			xe := xs + f.CellCountX
			for xs < xe {
				self := gridPrev[xs]

				right := grid[xs+1]
				left := grid[xs-1]
				bottom := grid[xs+f.PaddedCellCountX]
				top := grid[xs-f.PaddedCellCountX]

				sumOfNeighborValues := right + left + bottom + top
				diffusedValue := (self + sumOfNeighborValues*diffusionFactor) / (1 + numNeighbors*diffusionFactor)
				grid[xs] = diffusedValue

				xs++
			}
			ys += f.PaddedCellCountX
		}
		f.setBoundaries(b, grid)
	}
}

func (f *FluidEffect) project(xVelocities, yVelocities, xVelocitiesPrev, yVelocitiesPrev []float32) {

	minHalfDivSize := float32(-0.5) / float32(f.CellCountX)

	ys := 1 * f.PaddedCellCountX
	ye := ys + f.CellCountY
	for ys < ye {
		xs := ys + 1
		xe := xs + f.CellCountX
		for xs < xe {
			a := xVelocities[xs+1]
			b := xVelocities[xs-1]
			c := yVelocities[xs+f.PaddedCellCountX]
			d := yVelocities[xs-f.PaddedCellCountX]

			divergence := minHalfDivSize * (a - b + c - d)

			yVelocitiesPrev[xs] = divergence
			xVelocitiesPrev[xs] = 0.0
			xs++
		}
		ys += f.PaddedCellCountX
	}

	f.setBoundaries(copyBoundary, yVelocitiesPrev)
	f.setBoundaries(copyBoundary, xVelocitiesPrev)

	invFour := float32(1.0 / 4.0)

	for range f.RelaxationSteps {
		ys = 1 * f.PaddedCellCountX
		ye = ys + f.CellCountY
		for ys < ye {
			xs := ys + 1
			xe := xs + f.CellCountX
			for xs < xe {
				va := yVelocitiesPrev[xs]
				vb := xVelocitiesPrev[xs-1]
				vc := xVelocitiesPrev[xs+1]
				vd := xVelocitiesPrev[xs-f.PaddedCellCountX]
				ve := xVelocitiesPrev[xs+f.PaddedCellCountX]
				vf := (va + vb + vc + vd + ve) * invFour
				xVelocitiesPrev[xs] = vf
				xs++
			}
			ys += f.PaddedCellCountX
		}
		f.setBoundaries(copyBoundary, xVelocitiesPrev)
	}

	halfSize := float32(0.5) * float32(f.CellCountX)

	ys = 1 * f.PaddedCellCountX
	ye = ys + f.CellCountY
	for ys < ye {
		xs := ys + 1
		xe := xs + f.CellCountX
		for xs < xe {
			a := xVelocitiesPrev[xs+1]
			b := xVelocitiesPrev[xs-1]
			c := xVelocitiesPrev[xs+f.PaddedCellCountX]
			d := xVelocitiesPrev[xs-f.PaddedCellCountX]

			xVelocities[xs] -= halfSize * (a - b)
			yVelocities[xs] -= halfSize * (c - d)
			xs++
		}
		ys += f.PaddedCellCountX
	}
	f.setBoundaries(reflectBoundary, yVelocities)
	f.setBoundaries(reflectBoundary, xVelocities)
}

func (f *FluidEffect) setBoundaries(b boundaryAction, grid []float32) {
	// copy or reflect cell values to boundary cells as needed
	// to ensure the simulation is properly contained

	if b == reflectBoundary {
		// Left boundary
		leftColumnIndex := f.PaddedCellCountX
		for i := int32(1); i <= f.CellCountY; i++ {
			grid[leftColumnIndex] = -grid[leftColumnIndex+1]
			leftColumnIndex += f.PaddedCellCountX
		}

		// Right boundary
		rightColumnIndex := f.PaddedCellCountX + f.CellCountX + 1
		for i := int32(1); i <= f.CellCountY; i++ {
			grid[rightColumnIndex] = -grid[rightColumnIndex-1]
			rightColumnIndex += f.PaddedCellCountX
		}

		// Top boundary
		topRow := grid[0:f.PaddedCellCountX]
		topRowOneDown := grid[f.PaddedCellCountX : 2*f.PaddedCellCountX]
		for i := int32(1); i <= f.CellCountX; i++ {
			topRow[i] = -topRowOneDown[i]
		}

		// Bottom boundary
		bottomRow := grid[(f.CellCountX+1)*f.PaddedCellCountX : (f.CellCountX+2)*f.PaddedCellCountX]
		bottomRowOneUp := grid[f.CellCountX*f.PaddedCellCountX : (f.CellCountX+1)*f.PaddedCellCountX]
		for i := int32(1); i <= f.CellCountX; i++ {
			bottomRow[i] = -bottomRowOneUp[i]
		}
	} else {
		// Left boundary
		leftColumnIndex := f.PaddedCellCountX
		for i := int32(1); i <= f.CellCountY; i++ {
			grid[leftColumnIndex] = grid[leftColumnIndex+1]
			leftColumnIndex += f.PaddedCellCountX
		}

		// Right boundary
		rightColumnIndex := f.PaddedCellCountX + f.CellCountX + 1
		for i := int32(1); i <= f.CellCountY; i++ {
			grid[rightColumnIndex] = grid[rightColumnIndex-1]
			rightColumnIndex += f.PaddedCellCountX
		}

		// Top boundary
		topRow := grid[0:f.PaddedCellCountX]
		topRowOneDown := grid[f.PaddedCellCountX : 2*f.PaddedCellCountX]
		for i := int32(1); i <= f.CellCountX; i++ {
			topRow[i] = topRowOneDown[i]
		}

		// Bottom boundary
		bottomRow := grid[(f.CellCountX+1)*f.PaddedCellCountX : (f.CellCountX+2)*f.PaddedCellCountX]
		bottomRowOneUp := grid[f.CellCountX*f.PaddedCellCountX : (f.CellCountX+1)*f.PaddedCellCountX]
		for i := int32(1); i <= f.CellCountX; i++ {
			bottomRow[i] = bottomRowOneUp[i]
		}
	}

	// Corners
	grid[0] = 0.5 * (grid[1] + grid[f.PaddedCellCountX])                                                                                                          // Top-left corner
	grid[f.PaddedCellCountX-1] = 0.5 * (grid[f.PaddedCellCountX-2] + grid[2*f.PaddedCellCountX-1])                                                                // Top-right corner
	grid[(f.CellCountY+1)*f.PaddedCellCountX] = 0.5 * (grid[f.CellCountY*f.PaddedCellCountX] + grid[(f.CellCountY+1)*f.PaddedCellCountX+1])                       // Bottom-left corner
	grid[(f.PaddedCellCountX*f.PaddedCellCountY)-1] = 0.5 * (grid[(f.PaddedCellCountX*f.PaddedCellCountY)-2] + grid[(f.PaddedCellCountY-1)*f.PaddedCellCountX-1]) // Bottom-right corner
}

func (f *FluidEffect) bilinearInterpolate(x, y float32, grid []float32) float32 {
	// truncate x and y and get the indexes for the 4 adjacent cells at this position
	x0 := int32(x)
	y0 := int32(y)

	// calculate the floating point distance between the cell center and interpolation position
	// resulting in a value in the range 0.0 - 1.0, which represents the x and y contributions
	dx := x - float32(x0)
	dy := y - float32(y0)

	// get the values at the 4 adjacent cells that will be interpolated
	xx := x0 + y0*f.PaddedCellCountX
	v00 := grid[xx]
	xx++
	v10 := grid[xx]

	xx += f.PaddedCellCountX // move to the next row
	v11 := grid[xx]
	xx--
	v01 := grid[xx]

	// calculate the new density using the unit square method of bilinear interpolation
	// -- on a unit square, the four points are interpolated as:
	// 	 f(x,y) is appromixately f(0,0)(1-x)(1-y)+f(0,1)(1-x)y+f(1,0)x(1-y)+f(1,1)xy

	// this has 8 multiplications, but can be optimized!
	// return v00*(1-dx)*(1-dy) + v01*(1-dx)*dy + v10*dx*(1-dy) + v11*dx*dy

	// to reduce the number of multiplications, we can expand the above function to:
	// v00*(1 - dx - dy + dx*dy) + v01*(dy - dx*dy) + v10*(dx - dx*dy) + v11*(dx*dy)
	// which can then be optimized to 4 multiplications:
	// v00 + (v01 - v00)*dy + (v10 - v00)*dx + (v00 + v11 - v01 - v10)*dx*dy
	// then we can factor out the dx term to end up with 3 multiplications instead of 4:
	// return v00 + (v01-v00)*dy + dx*((v10-v00)+(-(v10-v00)+v11-v01)*dy)
	// also the term (v10 - v00) is used twice, one as is, the other negated
	v10MinusV00 := v10 - v00
	return v00 + (v01-v00)*dy + dx*(v10MinusV00+(v11-v01-v10MinusV00)*dy)
}
