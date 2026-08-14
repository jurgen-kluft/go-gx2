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

	fluid := &FluidEffect{
		CurrentVelocity:  currentVelocity,
		PreviousVelocity: previousVelocity,
		CurrentDensity:   currentDensity,
		PreviousDensity:  previousDensity,
		DiffusionRate:    0,
		Viscosity:        0,
		FadeRate:         0,
	}

	return fluid
}

func (f *VelocityData) reset() {
	for i := range f.XVel {
		f.XVel[i] = 0
		f.YVel[i] = 0
	}
}

func (f *DensityData) reset() {
	for i := range f.Density {
		f.Density[i] = 0
	}
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

func (f *FluidEffect) fade(dt float32, grid []float32, fadeRate float32) {
	for i := range grid {
		grid[i] -= dt * fadeRate
		if grid[i] < 0 {
			grid[i] = 0
		}
	}
}

func (f *FluidEffect) advect(b boundaryAction, dt float32, grid []float32, gridPrev []float32, xVelocities, yVelocities []float32) {
	for y := 1; y <= f.Size; y++ {
		yIdx := y * f.PaddedSize

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

			if px < 0.5 {
				px = 0.5
			} else if px > float32(f.Size)+0.5 {
				px = float32(f.Size) + 0.5
			}

			if py < 0.5 {
				py = 0.5
			} else if py > float32(f.Size)+0.5 {
				py = float32(f.Size) + 0.5
			}

			val := f.bilinearInterpolate(px, py, gridPrev)

			grid[xIdx] = val
			xIdx++
		}
	}
	f.setBoundaries(b, grid)
}

func (f *FluidEffect) diffuse(b boundaryAction, dt float32, grid []float32, gridPrev []float32, diffusionRate float32) {
	// diffuse the density field
	// high density cells diffuse to low density cells
	var relaxationSteps int = 20

	// diffusion delta
	diffusionFactor := dt * diffusionRate * float32(f.Size) * float32(f.Size)

	// Gauss-Seidel Relaxation
	for range relaxationSteps {
		for x := 1; x <= f.Size; x++ {
			for y := 1; y <= f.Size; y++ {
				self := gridPrev[x+(y*f.PaddedSize)]

				right := grid[x+1+(y*f.PaddedSize)]
				left := grid[x-1+(y*f.PaddedSize)]
				bottom := grid[x+(y+1)*f.PaddedSize]
				top := grid[x+(y-1)*f.PaddedSize]

				sumOfNeighborValues := right + left + bottom + top
				var numNeighbors float32 = 4.0

				diffusedValue := (self + sumOfNeighborValues*diffusionFactor) / (1 + numNeighbors*diffusionFactor)
				grid[x+(y*f.PaddedSize)] = diffusedValue
			}
		}
		f.setBoundaries(b, grid)
	}
}

func (f *FluidEffect) project(xVelocities, yVelocities, xVelocitiesPrev, yVelocitiesPrev []float32) {
	for x := 1; x <= f.Size; x++ {
		for y := 1; y <= f.Size; y++ {
			a := xVelocities[x+1+(y*f.PaddedSize)]
			b := xVelocities[x-1+(y*f.PaddedSize)]
			c := yVelocities[x+(y+1)*f.PaddedSize]
			d := yVelocities[x+(y-1)*f.PaddedSize]

			divergence := -0.5 * (a - b + c - d) / float32(f.Size)

			yVelocitiesPrev[x+(y*f.PaddedSize)] = divergence
			xVelocitiesPrev[x+(y*f.PaddedSize)] = 0.0
		}
	}
	f.setBoundaries(copyBoundary, yVelocitiesPrev)
	f.setBoundaries(copyBoundary, xVelocitiesPrev)

	var relaxationSteps int = 20
	for range relaxationSteps {
		for x := 1; x <= f.Size; x++ {
			for y := 1; y <= f.Size; y++ {
				va := yVelocitiesPrev[x+(y*f.PaddedSize)]
				vb := xVelocitiesPrev[x-1+(y*f.PaddedSize)]
				vc := xVelocitiesPrev[x+1+(y*f.PaddedSize)]
				vd := xVelocitiesPrev[x+(y-1)*f.PaddedSize]
				ve := xVelocitiesPrev[x+(y+1)*f.PaddedSize]
				vf := (va + vb + vc + vd + ve) / 4.0
				xVelocitiesPrev[x+(y*f.PaddedSize)] = vf
			}
		}
		f.setBoundaries(copyBoundary, xVelocities)
	}

	for x := 1; x <= f.Size; x++ {
		for y := 1; y <= f.Size; y++ {
			a := xVelocitiesPrev[x+1+(y*f.PaddedSize)]
			b := xVelocitiesPrev[x-1+(y*f.PaddedSize)]
			c := xVelocitiesPrev[x+(y+1)*f.PaddedSize]
			d := xVelocitiesPrev[x+(y-1)*f.PaddedSize]

			xVelocities[x+(y*f.PaddedSize)] -= 0.5 * float32(f.Size) * (a - b)
			yVelocities[x+(y*f.PaddedSize)] -= 0.5 * float32(f.Size) * (c - d)
		}
	}
	f.setBoundaries(reflectBoundary, yVelocities)
	f.setBoundaries(reflectBoundary, xVelocities)
}

func (f *FluidEffect) setBoundaries(b boundaryAction, grid []float32) {
	// copy or reflect cell values to boundary cells as needed
	// to ensure the simulation is properly contained

	// for i := 1; i <= f.Size; i++ {
	// 	//gridRow := grid[i*f.PaddedSize : (i+1)*f.PaddedSize]
	// 	if b == reflectBoundary {
	// 		grid[0+(i*f.PaddedSize)] = -grid[1+(i*f.PaddedSize)]
	// 		grid[(f.Size+1)+(i*f.PaddedSize)] = -grid[f.Size+(i*f.PaddedSize)]
	// 	} else {
	// 		grid[0+(i*f.PaddedSize)] = grid[1+(i*f.PaddedSize)]
	// 		grid[(f.Size+1)+(i*f.PaddedSize)] = grid[f.Size+(i*f.PaddedSize)]
	// 	}

	// 	if b == reflectBoundary {
	// 		grid[i+(0*f.PaddedSize)] = -grid[i+(1*f.PaddedSize)]
	// 		grid[i+((f.Size+1)*f.PaddedSize)] = -grid[i+(f.Size*f.PaddedSize)]
	// 	} else {
	// 		grid[i+(0*f.PaddedSize)] = grid[i+(1*f.PaddedSize)]
	// 		grid[i+((f.Size+1)*f.PaddedSize)] = grid[i+(f.Size*f.PaddedSize)]
	// 	}

	// 	grid[0+(0*f.PaddedSize)] = 0.5 * (grid[1+(0*f.PaddedSize)] + grid[0+(1*f.PaddedSize)])
	// 	grid[0+((f.Size+1)*f.PaddedSize)] = 0.5 * (grid[1+((f.Size+1)*f.PaddedSize)] + grid[0+(f.Size*f.PaddedSize)])
	// 	grid[(f.Size+1)+(0*f.PaddedSize)] = 0.5 * (grid[f.Size+(0*f.PaddedSize)] + grid[(f.Size+1)+(1*f.PaddedSize)])
	// 	grid[(f.Size+1)+((f.Size+1)*f.PaddedSize)] = 0.5 * (grid[f.Size+((f.Size+1)*f.PaddedSize)] + grid[(f.Size+1)+(f.Size*f.PaddedSize)])
	// }

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
	x1 := x0 + 1
	y1 := y0 + 1

	// calculate the floating point distance between the cell center and interpolation position
	// resulting in a value in the range 0.0 - 1.0, which represents the x and y contributions
	dx := x - float32(x0)
	dy := y - float32(y0)

	// get the values at the 4 adjacent cells that will be interpolated
	v00 := grid[x0+(y0*f.PaddedSize)]
	v01 := grid[x0+(y1*f.PaddedSize)]
	v10 := grid[x1+(y0*f.PaddedSize)]
	v11 := grid[x1+(y1*f.PaddedSize)]

	// calculate the new density using the unit square method of bilinear interpolation
	// -- on a unit square, the four points are interpolated as:
	// 	 f(x,y) is appromixately f(0,0)(1-x)(1-y)+f(0,1)(1-x)y+f(1,0)x(1-y)+f(1,1)xy
	return v00*(1-dx)*(1-dy) +
		v01*(1-dx)*dy +
		v10*dx*(1-dy) +
		v11*dx*dy
}
