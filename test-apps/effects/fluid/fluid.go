package fx_fluid

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
	XVel []float32
	YVel []float32
}

type DensityData struct {
	Density []float32
}

type FluidEffect struct {
	Size             int
	PaddedSize       int
	CurrentVelocity  *VelocityData
	PreviousVelocity *VelocityData
	CurrentDensity   *DensityData
	PreviousDensity  *DensityData
	RelaxationSteps  int32
	DiffusionRate    float32
	Viscosity        float32
	FadeRate         float32
}

func NewFluidEffect(cellSize int) *FluidEffect {
	// allocate additional space for boundary conditions
	paddedSize := cellSize + 2
	makeArray2d := func() []float32 { return make([]float32, paddedSize*paddedSize) }

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
		Size:             cellSize,
		PaddedSize:       paddedSize,
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

func (f *FluidEffect) Reset() {
	f.CurrentVelocity.reset()
	f.PreviousVelocity.reset()
	f.CurrentDensity.reset()
	f.PreviousDensity.reset()
}

func (f *FluidEffect) Simulate(dt float32) {
	f.simulateVelocity(dt)
	f.simulateDensity(dt)
}

func (f *FluidEffect) AddDensity(x, y int, val float32) {
	f.CurrentDensity.Density[x+(y*f.PaddedSize)] = val
}

func (f *FluidEffect) AddVelocity(x, y int, xval, yval float32) {
	f.CurrentVelocity.XVel[x+(y*f.PaddedSize)] = xval
	f.CurrentVelocity.YVel[x+(y*f.PaddedSize)] = yval
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

func fabs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

func (f *FluidEffect) fade(dt float32, grid []float32, fadeRate float32) {
	for i := range grid {
		value := grid[i] - dt*fadeRate
		grid[i] = fabs(value)
	}
}

func clampf(value, min, max float32) float32 {
	if value < min {
		return min
	} else if value > max {
		return max
	}
	return value
}

func (f *FluidEffect) advect(b boundaryAction, dt float32, grid []float32, gridPrev []float32, xVelocities, yVelocities []float32) {

	half := float32(0.5)
	sizePlusHalf := float32(f.Size) + half

	yIdx := 1 * f.PaddedSize
	for y := 1; y <= f.Size; y++ {
		xIdx := yIdx
		for x := 1; x <= f.Size; x++ {
			xv := xVelocities[xIdx]
			yv := yVelocities[xIdx]

			// calculate previous x and y positions of the current grid particle
			// by moving backwards along the velocity field.
			// by calculating new density based on previous particle position,
			// the simulation becomes bounded.
			px := float32(x) - xv*dt
			py := float32(y) - yv*dt

			px = clampf(px, half, sizePlusHalf)
			py = clampf(py, half, sizePlusHalf)

			val := f.bilinearInterpolate(px, py, gridPrev)

			grid[xIdx] = val
			xIdx++
		}
		yIdx += f.PaddedSize
	}
	f.setBoundaries(b, grid)
}

func (f *FluidEffect) diffuse(b boundaryAction, dt float32, grid []float32, gridPrev []float32, diffusionRate float32) {
	// diffuse the density field
	// high density cells diffuse to low density cells

	// diffusion delta
	diffusionFactor := dt * diffusionRate * float32(f.Size) * float32(f.Size)

	const numNeighbors float32 = 4.0

	// Gauss-Seidel Relaxation
	for range f.RelaxationSteps {
		ys := 1 * f.PaddedSize
		ye := ys + f.Size + 1
		for ys <= ye {
			xs := ys + 1
			xe := xs + f.Size
			for xs < xe {
				self := gridPrev[xs]

				right := grid[xs+1]
				left := grid[xs-1]
				bottom := grid[xs+f.PaddedSize]
				top := grid[xs-f.PaddedSize]

				sumOfNeighborValues := right + left + bottom + top
				diffusedValue := (self + sumOfNeighborValues*diffusionFactor) / (1 + numNeighbors*diffusionFactor)
				grid[xs] = diffusedValue
			}
		}
		f.setBoundaries(b, grid)
	}
}

func (f *FluidEffect) project(xVelocities, yVelocities, xVelocitiesPrev, yVelocitiesPrev []float32) {

	minHalfDivSize := float32(-0.5) / float32(f.Size)

	ys := 1 * f.PaddedSize
	ye := ys + f.Size
	for ys < ye {
		xs := ys + 1
		xe := xs + f.Size
		for xs < xe {
			a := xVelocities[xs+1]
			b := xVelocities[xs-1]
			c := yVelocities[xs+f.PaddedSize]
			d := yVelocities[xs-f.PaddedSize]

			divergence := minHalfDivSize * (a - b + c - d)

			yVelocitiesPrev[xs] = divergence
			xVelocitiesPrev[xs] = 0.0
			xs++
		}
		ys += f.PaddedSize
	}

	f.setBoundaries(copyBoundary, yVelocitiesPrev)
	f.setBoundaries(copyBoundary, xVelocitiesPrev)

	invFour := float32(1.0 / 4.0)

	for range f.RelaxationSteps {
		ys = 1 * f.PaddedSize
		ye = ys + f.Size + 1
		for ys <= ye {
			xs := ys + 1
			xe := xs + f.Size
			for xs < xe {
				va := yVelocitiesPrev[xs]
				vb := xVelocitiesPrev[xs-1]
				vc := xVelocitiesPrev[xs+1]
				vd := xVelocitiesPrev[xs-f.PaddedSize]
				ve := xVelocitiesPrev[xs+f.PaddedSize]
				vf := (va + vb + vc + vd + ve) * invFour
				xVelocitiesPrev[xs] = vf
				xs++
			}
			ys += f.PaddedSize
		}
		f.setBoundaries(copyBoundary, xVelocitiesPrev)
	}

	halfSize := float32(0.5) * float32(f.Size)

	ys = 1 * f.PaddedSize
	ye = ys + f.Size + 1
	for ys <= ye {
		xs := ys + 1
		xe := xs + f.Size
		for xs < xe {
			a := xVelocitiesPrev[xs+1]
			b := xVelocitiesPrev[xs-1]
			c := xVelocitiesPrev[xs+f.PaddedSize]
			d := xVelocitiesPrev[xs-f.PaddedSize]

			xVelocities[xs] -= halfSize * (a - b)
			yVelocities[xs] -= halfSize * (c - d)
			xs++
		}
		ys += f.PaddedSize
	}
	f.setBoundaries(reflectBoundary, yVelocities)
	f.setBoundaries(reflectBoundary, xVelocities)
}

func (f *FluidEffect) setBoundaries(b boundaryAction, grid []float32) {
	// copy or reflect cell values to boundary cells as needed
	// to ensure the simulation is properly contained

	if b == reflectBoundary {
		// Left boundary
		leftColumnIndex := f.PaddedSize
		for i := 1; i <= f.Size; i++ {
			grid[leftColumnIndex] = -grid[leftColumnIndex+1]
			leftColumnIndex += f.PaddedSize
		}

		// Right boundary
		rightColumnIndex := f.PaddedSize + f.Size + 1
		for i := 1; i <= f.Size; i++ {
			grid[rightColumnIndex] = -grid[rightColumnIndex-1]
			rightColumnIndex += f.PaddedSize
		}

		// Top boundary
		topRow := grid[0:f.PaddedSize]
		topRowOneDown := grid[f.PaddedSize : 2*f.PaddedSize]
		for i := 1; i <= f.Size; i++ {
			topRow[i] = -topRowOneDown[i]
		}

		// Bottom boundary
		bottomRow := grid[(f.Size+1)*f.PaddedSize : (f.Size+2)*f.PaddedSize]
		bottomRowOneUp := grid[f.Size*f.PaddedSize : (f.Size+1)*f.PaddedSize]
		for i := 1; i <= f.Size; i++ {
			bottomRow[i] = -bottomRowOneUp[i]
		}
	} else {
		// Left boundary
		leftColumnIndex := f.PaddedSize
		for i := 1; i <= f.Size; i++ {
			grid[leftColumnIndex] = grid[leftColumnIndex+1]
			leftColumnIndex += f.PaddedSize
		}

		// Right boundary
		rightColumnIndex := f.PaddedSize + f.Size + 1
		for i := 1; i <= f.Size; i++ {
			grid[rightColumnIndex] = grid[rightColumnIndex-1]
			rightColumnIndex += f.PaddedSize
		}

		// Top boundary
		topRow := grid[0:f.PaddedSize]
		topRowOneDown := grid[f.PaddedSize : 2*f.PaddedSize]
		for i := 1; i <= f.Size; i++ {
			topRow[i] = topRowOneDown[i]
		}

		// Bottom boundary
		bottomRow := grid[(f.Size+1)*f.PaddedSize : (f.Size+2)*f.PaddedSize]
		bottomRowOneUp := grid[f.Size*f.PaddedSize : (f.Size+1)*f.PaddedSize]
		for i := 1; i <= f.Size; i++ {
			bottomRow[i] = bottomRowOneUp[i]
		}
	}

	// Corners
	grid[0] = 0.5 * (grid[1] + grid[f.PaddedSize])                                                                            // Top-left corner
	grid[f.PaddedSize-1] = 0.5 * (grid[f.PaddedSize-2] + grid[2*f.PaddedSize-1])                                              // Top-right corner
	grid[(f.Size+1)*f.PaddedSize] = 0.5 * (grid[f.Size*f.PaddedSize] + grid[(f.Size+1)*f.PaddedSize+1])                       // Bottom-left corner
	grid[(f.PaddedSize*f.PaddedSize)-1] = 0.5 * (grid[(f.PaddedSize*f.PaddedSize)-2] + grid[(f.PaddedSize-1)*f.PaddedSize-1]) // Bottom-right corner
}

func (f *FluidEffect) bilinearInterpolate(x, y float32, grid []float32) float32 {
	// truncate x and y and get the indexes for the 4 adjacent cells at this position
	x0 := int(x)
	y0 := int(y)

	// calculate the floating point distance between the cell center and interpolation position
	// resulting in a value in the range 0.0 - 1.0, which represents the x and y contributions
	dx := x - float32(x0)
	dy := y - float32(y0)

	// get the values at the 4 adjacent cells that will be interpolated
	xx := x0 + y0*f.PaddedSize
	v00 := grid[xx]
	xx++
	v10 := grid[xx]

	xx += f.PaddedSize // move to the next row
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
