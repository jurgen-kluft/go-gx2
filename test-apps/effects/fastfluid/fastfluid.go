package fx_fastfluid

import fx_common "github.com/jurgen-kluft/go-gx2/test-apps/effects/common"

// NewEffect creates a new FluidEffect instance with the specified width and height.
// The width and height represent the number of fluid cells in the X and Y dimensions.
func NewEffect(width, height int32) *FluidEffect {
	// Add 2 to both dimensions to create the outer boundary padding layer
	paddedX := int32(width + 2)
	paddedY := int32(height + 2)

	bounds := FluidBounds{
		CellCountX:       int32(width),
		CellCountY:       int32(height),
		PaddedCellCountX: paddedX,
		PaddedCellCountY: paddedY,
	}

	totalCells := paddedX * paddedY

	return &FluidEffect{
		Bounds: bounds,

		// Default screen-saver balance values (0-255 scale)
		Viscosity:     15, // How thick the fluid feels
		DiffusionRate: 10, // How quickly colors blend together
		FadeRate:      8,  // How fast the smoke dissolves over time

		// Allocate double buffers for velocity and density fields
		VelocityX: Velocity{
			Current:  make([]int16, totalCells),
			Previous: make([]int16, totalCells),
		},
		VelocityY: Velocity{
			Current:  make([]int16, totalCells),
			Previous: make([]int16, totalCells),
		},
		Density: Density{
			Current:  make([]uint8, totalCells),
			Previous: make([]uint8, totalCells),
		},

		// Set the number of Jacobi relaxation iterations for diffusion and projection
		RelaxationIterations: 4,

		// Set the fixed time step for simulation updates (in seconds)
		TimeStep: 1.0 / 60.0,

		// Allocate workspace scratch spaces for the projection linear solver
		PScratch:   make([]int16, totalCells),
		DivScratch: make([]int16, totalCells),
	}
}

type Velocity struct {
	Current  []int16
	Previous []int16
}

func (f *Velocity) swap() {
	f.Current, f.Previous = f.Previous, f.Current
}

type Density struct {
	Current  []uint8
	Previous []uint8
}

func (d *Density) swap() {
	d.Current, d.Previous = d.Previous, d.Current
}

type FluidBounds struct {
	CellCountX       int32
	CellCountY       int32
	PaddedCellCountX int32
	PaddedCellCountY int32
}

type FluidEffect struct {
	AccumulatedTime float32
	TimeStep        float32

	Bounds FluidBounds

	RelaxationIterations int32 // Number of Jacobi relaxation iterations for diffusion and projection

	// Configurable parameters
	Viscosity     uint8 // Scaled integer viscosity rate
	DiffusionRate uint8 // Scaled integer diffusion rate
	FadeRate      uint8 // 0-255 rate used in our fast fade function

	// Double buffers for physics swapping
	VelocityX Velocity
	VelocityY Velocity
	Density   Density

	// Workspace scratch buffers for the projection linear solver
	// Allocating these once prevents heavy runtime memory garbage collection!
	PScratch   []int16
	DivScratch []int16
}

type boundaryType uint8

const (
	boundaryCopy     boundaryType = iota // 0: Normal scalar copy (Density, Pressure)
	boundaryReflectX                     // 1: Reflect X-velocities at vertical walls
	boundaryReflectY                     // 2: Reflect Y-velocities at horizontal walls
)

func advectDensityFixedPoint(bounds FluidBounds, grid []uint8, gridPrev []uint8, xVelocities, yVelocities []int16) {
	paddedX := bounds.PaddedCellCountX
	cellX := int32(bounds.CellCountX)
	cellY := int32(bounds.CellCountY)
	const dt_q12 int32 = 68 // Constant 60 FPS scaling factor

	limit := paddedX * bounds.PaddedCellCountY
	grid = grid[:limit]
	gridPrev = gridPrev[:limit]
	xVel := xVelocities[:limit]
	yVel := yVelocities[:limit]

	yIdx := paddedX
	for y := int32(1); y <= cellY; y++ {
		xIdx := yIdx + 1
		for x := int32(1); x <= cellX; x++ {
			// Pull matching int16 velocities, map to Q4.20, shift to Q20.8
			backX := (int32(xVel[xIdx]) * dt_q12) >> 12
			backY := (int32(yVel[xIdx]) * dt_q12) >> 12

			px_q8 := (x << 8) - backX
			py_q8 := (y << 8) - backY

			if px_q8 < 0x0100 {
				px_q8 = 0x0100
			} else if px_q8 > (cellX << 8) {
				px_q8 = cellX << 8
			}
			if py_q8 < 0x0100 {
				py_q8 = 0x0100
			} else if py_q8 > (cellY << 8) {
				py_q8 = cellY << 8
			}

			x0 := px_q8 >> 8
			y0 := py_q8 >> 8
			fx := px_q8 & 0xFF
			fy := py_q8 & 0xFF

			ifx := 256 - fx
			ify := 256 - fy

			idx00 := y0*paddedX + x0
			idx10 := idx00 + 1
			idx01 := idx00 + paddedX
			idx11 := idx01 + 1

			// Blends values pulled from the uint8 gridPrev array, outputs straight back into uint8 grid
			top := int32(gridPrev[idx00])*ifx + int32(gridPrev[idx10])*fx
			bottom := int32(gridPrev[idx01])*ifx + int32(gridPrev[idx11])*fx

			grid[xIdx] = uint8((top*ify + bottom*fy) >> 16)
			xIdx++
		}
		yIdx += paddedX
	}
	setDensityBoundaries(bounds, grid)
}

func advectVelocity(bounds FluidBounds, b boundaryType, grid []int16, gridPrev []int16, xVelocities, yVelocities []int16) {
	paddedX := bounds.PaddedCellCountX
	cellX := int32(bounds.CellCountX)
	cellY := int32(bounds.CellCountY)
	const dt_q12 int32 = 68 // Constant 60 FPS scaling factor

	// Pre-slice truncate to entirely remove Go's hidden bounds checks
	limit := paddedX * bounds.PaddedCellCountY
	grid = grid[:limit]
	gridPrev = gridPrev[:limit]
	xVel := xVelocities[:limit]
	yVel := yVelocities[:limit]

	yIdx := paddedX
	for y := int32(1); y <= cellY; y++ {
		xIdx := yIdx + 1
		for x := int32(1); x <= cellX; x++ {
			// Read Q8.8 velocity (16-bit) and promote to 32-bit for math
			xv := int32(xVel[xIdx])
			yv := int32(yVel[xIdx])

			// Multiply Q8.8 * Q0.12 = Q4.20 intermediate
			// Shift right by 12 to leave an 8-bit fraction (Q20.8)
			backX := (xv * dt_q12) >> 12
			backY := (yv * dt_q12) >> 12

			// Subtract from current position (shifted to Q8 coordinate space)
			px_q8 := (x << 8) - backX
			py_q8 := (y << 8) - backY

			// Fast Integer Clamping to screen boundaries (in Q8 space)
			if px_q8 < 0x0100 {
				px_q8 = 0x0100
			} else if px_q8 > (cellX << 8) {
				px_q8 = cellX << 8
			}
			if py_q8 < 0x0100 {
				py_q8 = 0x0100
			} else if py_q8 > (cellY << 8) {
				py_q8 = cellY << 8
			}

			// --- Inline 1-Cycle Interpolation Extraction ---
			x0 := px_q8 >> 8
			y0 := py_q8 >> 8
			fx := px_q8 & 0xFF // 8-bit fraction [0-255]
			fy := py_q8 & 0xFF // 8-bit fraction [0-255]

			ifx := 256 - fx
			ify := 256 - fy

			// Compute array indices
			idx00 := y0*paddedX + x0
			idx10 := idx00 + 1
			idx01 := idx00 + paddedX
			idx11 := idx01 + 1

			// 4-Point Weighted Bilinear Blending for 16-bit integer velocities
			// Divide by 65536 (>> 16) at the end to pull back to int16 range
			top := int32(gridPrev[idx00])*ifx + int32(gridPrev[idx10])*fx
			bottom := int32(gridPrev[idx01])*ifx + int32(gridPrev[idx11])*fx

			grid[xIdx] = int16((top*ify + bottom*fy) >> 16)

			xIdx++
		}
		yIdx += paddedX
	}

	setVelocityBoundaries(bounds, b, grid)
}

func projectVelocity(bounds FluidBounds, xVel, yVel []int16, p, div []int16, relaxationIterations int32) {
	paddedX := bounds.PaddedCellCountX
	cellX := int32(bounds.CellCountX)
	cellY := int32(bounds.CellCountY)

	// Pre-slice truncate to entirely remove Go's hidden bounds checks
	limit := paddedX * bounds.PaddedCellCountY
	xVel = xVel[:limit]
	yVel = yVel[:limit]
	p = p[:limit]
	div = div[:limit]

	// -------------------------------------------------------------------------
	// STEP 1: Compute Divergence (How much fluid is entering/leaving each cell)
	// -------------------------------------------------------------------------
	yIdx := paddedX
	for y := int32(1); y <= cellY; y++ {
		xIdx := yIdx + 1
		for x := int32(1); x <= cellX; x++ {
			// Standard divergence formula: (u[x+1] - u[x-1] + v[y+1] - v[y-1]) * -0.5
			// For speed, we multiply by -128 instead of -0.5, shifting right by 8 later.
			// Pre-scaling prevents truncation errors in integer math.
			du := int32(xVel[xIdx+1]) - int32(xVel[xIdx-1])
			dv := int32(yVel[xIdx+paddedX]) - int32(yVel[xIdx-paddedX])

			div[xIdx] = int16((-du - dv) >> 1) // Scaled divergence
			p[xIdx] = 0                        // Reset pressure accumulation array
			xIdx++
		}
		yIdx += paddedX
	}

	// Apply boundaries to the helper arrays
	setVelocityBoundaries(bounds, boundaryCopy, div)
	setVelocityBoundaries(bounds, boundaryCopy, p)

	// -------------------------------------------------------------------------
	// STEP 2: Jacobi Relaxation to Solve for Pressure
	// -------------------------------------------------------------------------
	// In standard solvers, each cell is calculated as: (L + R + U + D - div) / 4.
	// Since dividing by 4 is a simple bit-shift right (>> 2), this loop executes
	// in pure 1-cycle integer assembly instructions!
	for iter := int32(0); iter < relaxationIterations; iter++ {
		yIdx = paddedX
		for y := int32(1); y <= cellY; y++ {
			xIdx := yIdx + 1
			for x := int32(1); x <= cellX; x++ {
				neighborSum := int32(p[xIdx-1]) + int32(p[xIdx+1]) +
					int32(p[xIdx-paddedX]) + int32(p[xIdx+paddedX])

				// (Neighbors + Divergence) / 4
				p[xIdx] = int16((neighborSum + int32(div[xIdx])) >> 2)
				xIdx++
			}
			yIdx += paddedX
		}
		setVelocityBoundaries(bounds, boundaryCopy, p)
	}

	// -------------------------------------------------------------------------
	// STEP 3: Subtract Pressure Gradient (Subtract compression to enforce conservation)
	// -------------------------------------------------------------------------
	yIdx = paddedX
	for y := int32(1); y <= cellY; y++ {
		xIdx := yIdx + 1
		for x := int32(1); x <= cellX; x++ {
			// Subtract the pressure gradient from the velocities to make the field incompressible
			// We shift left by 1 to scale the pressure gradient to match the velocity range
			xVel[xIdx] -= int16((int32(p[xIdx+1]) - int32(p[xIdx-1])) << 1)
			yVel[xIdx] -= int16((int32(p[xIdx+paddedX]) - int32(p[xIdx-paddedX])) << 1)
			xIdx++
		}
		yIdx += paddedX
	}

	// Restore physical velocity boundaries
	setVelocityBoundaries(bounds, boundaryReflectX, xVel)
	setVelocityBoundaries(bounds, boundaryReflectY, yVel)
}

func setDensityBoundaries(bounds FluidBounds, grid []uint8) {
	paddedX := bounds.PaddedCellCountX
	cellX := int32(bounds.CellCountX)
	cellY := int32(bounds.CellCountY)

	// Clean truncation to eliminate hidden Go boundary verification branches
	grid = grid[:paddedX*bounds.PaddedCellCountY]

	// 1. Handle Top and Bottom Horizontal Boundaries (Fast sequential memory loop)
	topRowIdx := int32(0)
	botRowIdx := int32(cellY+1) * paddedX

	for x := int32(1); x <= cellX; x++ {
		tIdx := topRowIdx + x
		bIdx := botRowIdx + x

		// Density simply duplicates the adjacent cell value inward
		grid[tIdx] = grid[tIdx+paddedX]
		grid[bIdx] = grid[bIdx-paddedX]
	}

	// 2. Handle Left and Right Vertical Boundaries
	yIdx := paddedX
	for y := int32(1); y <= cellY; y++ {
		lIdx := yIdx
		rIdx := yIdx + cellX + 1

		// Duplicate adjacent horizontal cell values inward
		grid[lIdx] = grid[lIdx+1]
		grid[rIdx] = grid[rIdx-1]

		yIdx += paddedX
	}

	// 3. Handle the 4 Corners (Average of their immediate neighbors using shifts)
	grid[0] = uint8((uint16(grid[1]) + uint16(grid[paddedX])) >> 1)
	grid[cellX+1] = uint8((uint16(grid[cellX]) + uint16(grid[cellX+1+paddedX])) >> 1)

	botLeftIdx := (cellY + 1) * paddedX
	grid[botLeftIdx] = uint8((uint16(grid[botLeftIdx+1]) + uint16(grid[botLeftIdx-paddedX])) >> 1)

	botRightIdx := botLeftIdx + cellX + 1
	grid[botRightIdx] = uint8((uint16(grid[botRightIdx-1]) + uint16(grid[botRightIdx-paddedX])) >> 1)
}

func setVelocityBoundaries(bounds FluidBounds, b boundaryType, grid []int16) {
	paddedX := bounds.PaddedCellCountX
	cellX := bounds.CellCountX
	cellY := bounds.CellCountY

	// Clean truncation to eliminate hidden Go boundary verification branches
	grid = grid[:paddedX*bounds.PaddedCellCountY]

	// 1. Handle Top and Bottom Horizontal Boundaries (Blazing fast sequential memory)
	// Top row: y = 0, Bottom row: y = cellY + 1
	topRowIdx := int32(0)
	botRowIdx := (cellY + 1) * paddedX

	for x := int32(1); x <= cellX; x++ {
		tIdx := topRowIdx + x
		bIdx := botRowIdx + x

		if b == boundaryReflectY { // Y-Velocity boundary reflection
			grid[tIdx] = -grid[tIdx+paddedX]
			grid[bIdx] = -grid[bIdx-paddedX]
		} else {
			grid[tIdx] = grid[tIdx+paddedX]
			grid[bIdx] = grid[bIdx-paddedX]
		}
	}

	// 2. Handle Left and Right Vertical Boundaries
	// Left column: x = 0, Right column: x = cellX + 1
	yIdx := paddedX
	for y := int32(1); y <= cellY; y++ {
		lIdx := yIdx             // x = 0
		rIdx := yIdx + cellX + 1 // x = cellX + 1

		if b == boundaryReflectX { // X-Velocity boundary reflection
			grid[lIdx] = -grid[lIdx+1]
			grid[rIdx] = -grid[rIdx-1]
		} else {
			grid[lIdx] = grid[lIdx+1]
			grid[rIdx] = grid[rIdx-1]
		}
		yIdx += paddedX
	}

	// 3. Handle the 4 Corners (Average of their immediate neighbors)
	// Top-Left, Top-Right, Bottom-Left, Bottom-Right
	grid[0] = (grid[1] + grid[paddedX]) >> 1
	grid[cellX+1] = (grid[cellX] + grid[cellX+1+paddedX]) >> 1

	botLeftIdx := (cellY + 1) * paddedX
	grid[botLeftIdx] = (grid[botLeftIdx+1] + grid[botLeftIdx-paddedX]) >> 1

	botRightIdx := botLeftIdx + cellX + 1
	grid[botRightIdx] = (grid[botRightIdx-1] + grid[botRightIdx-paddedX]) >> 1
}

func diffuseVelocity(bounds FluidBounds, b boundaryType, vel *Velocity, relaxationIterations int32, diffRate uint8) {
	paddedX := bounds.PaddedCellCountX
	cellX := bounds.CellCountX
	cellY := bounds.CellCountY
	limit := paddedX * bounds.PaddedCellCountY

	// Set up zero-overhead local working pointers for our iterations
	src := vel.Previous[:limit]
	dst := vel.Current[:limit]

	a := int32(diffRate)
	denom := 1 + 4*a
	recipScale := (1 << 16) / denom

	for iter := int32(0); iter < relaxationIterations; iter++ {
		yIdx := paddedX
		for y := int32(1); y <= cellY; y++ {
			xIdx := yIdx + 1
			for x := int32(1); x <= cellX; x++ {
				neighborSum := int32(src[xIdx-1]) + int32(src[xIdx+1]) +
					int32(src[xIdx-paddedX]) + int32(src[xIdx+paddedX])

				num := int32(src[xIdx]) + (a * neighborSum)
				dst[xIdx] = int16((num * recipScale) >> 16)
				xIdx++
			}
			yIdx += paddedX
		}
		setVelocityBoundaries(bounds, b, dst)

		// Ping-pong local array scopes for next iteration step
		src, dst = dst, src
	}

	// Because we alternate pointers every iteration:
	// 'src' ALWAYS holds the final, most recently stabilized physics state.
	// 'dst' holds the leftover un-updated scratch data.
	// Assigning them directly updates the Velocity struct fields with zero copies!
	vel.Current = src
	vel.Previous = dst
}

func fadeDensity(bounds FluidBounds, density *Density, fadeRate uint8) {
	paddedX := bounds.PaddedCellCountX
	cellX := int32(bounds.CellCountX)
	cellY := int32(bounds.CellCountY)

	// Clean truncation to eliminate hidden Go boundary verification branches
	limit := paddedX * bounds.PaddedCellCountY
	grid := density.Current[:limit]

	// Cache fadeRate locally for register speed
	// fadeRate input is 0-255 (e.g., 5 means minor fade, 40 means rapid fade)
	rate := uint32(fadeRate)

	yIdx := paddedX
	for y := int32(1); y <= cellY; y++ {
		xIdx := yIdx + 1
		for x := int32(1); x <= cellX; x++ {
			val := uint32(grid[xIdx])

			// 1. If the cell is already completely clear, skip the math entirely.
			// This branch is highly predictable and speeds up empty areas of your screen.
			if val > 0 {
				// 2. Perform a fractional decay math operation:
				// New Value = Old Value - ((Old Value * Rate) >> 8) - 1
				// The extra "- 1" forces the density to hit true zero quickly,
				// preventing dead visual remnants from lingering on the screen.
				decay := ((val * rate) >> 8) + 1

				if val <= decay {
					grid[xIdx] = 0
				} else {
					grid[xIdx] = uint8(val - decay)
				}
			}

			xIdx++
		}
		yIdx += paddedX
	}
}

func diffuseDensity(bounds FluidBounds, density *Density, relaxationIterations int32, diffRate uint8) {
	paddedX := bounds.PaddedCellCountX
	cellX := bounds.CellCountX
	cellY := bounds.CellCountY
	limit := paddedX * bounds.PaddedCellCountY

	src := density.Previous[:limit]
	dst := density.Current[:limit]

	a := int32(diffRate)
	denom := 1 + 4*a
	recipScale := (1 << 16) / denom

	for iter := int32(0); iter < relaxationIterations; iter++ {
		yIdx := paddedX
		for y := int32(1); y <= cellY; y++ {
			xIdx := yIdx + 1
			for x := int32(1); x <= cellX; x++ {
				neighborSum := int32(src[xIdx-1]) + int32(src[xIdx+1]) +
					int32(src[xIdx-paddedX]) + int32(src[xIdx+paddedX])

				num := int32(src[xIdx]) + (a * neighborSum)
				dst[xIdx] = uint8((num * recipScale) >> 16)
				xIdx++
			}
			yIdx += paddedX
		}
		setDensityBoundaries(bounds, dst)

		src, dst = dst, src
	}

	// Update the parent struct field references with zero copies
	density.Current = src
	density.Previous = dst
}

func (f *FluidEffect) simulateVelocity() {
	// 1. Swap velocity buffers to make current data the "previous" state
	f.VelocityX.swap()
	f.VelocityY.swap()

	// 2. Diffuse the velocities using Jacobi relaxation (4 iterations)
	// Passing the type-safe BoundaryReflect configurations to handle boundary physics
	diffuseVelocity(f.Bounds, boundaryReflectX, &f.VelocityX, f.RelaxationIterations, f.Viscosity)
	diffuseVelocity(f.Bounds, boundaryReflectY, &f.VelocityY, f.RelaxationIterations, f.Viscosity)

	// 3. Project to fix any numerical compression leaks from the diffusion pass
	projectVelocity(f.Bounds, f.VelocityX.Current, f.VelocityY.Current, f.PScratch, f.DivScratch, f.RelaxationIterations)

	// 4. Swap states again to prepare for the advection step
	f.VelocityX.swap()
	f.VelocityY.swap()

	// 5. Advect velocities backward through their own moving velocity field
	advectVelocity(f.Bounds, boundaryReflectX, f.VelocityX.Current, f.VelocityX.Previous, f.VelocityX.Previous, f.VelocityY.Previous)
	advectVelocity(f.Bounds, boundaryReflectY, f.VelocityY.Current, f.VelocityY.Previous, f.VelocityX.Previous, f.VelocityY.Previous)

	// 6. Final projection sweep ensures the output remains completely stable and swirling
	projectVelocity(f.Bounds, f.VelocityX.Current, f.VelocityY.Current, f.PScratch, f.DivScratch, f.RelaxationIterations)
}

func (f *FluidEffect) simulateDensity() {
	// 1. Apply the fast proportional integer decay sweep to dissolve the smoke
	fadeDensity(f.Bounds, &f.Density, f.FadeRate)

	// 2. Swap density buffers to prepare for diffusion
	f.Density.swap()

	// 3. Diffuse density outwards smoothly across surrounding grid neighbors
	diffuseDensity(f.Bounds, &f.Density, f.RelaxationIterations, f.DiffusionRate)

	// 4. Swap density buffers once more to prepare for the advection pass
	f.Density.swap()

	// 5. Advect density backward along the newly updated velocity field vector paths
	advectDensityFixedPoint(f.Bounds, f.Density.Current, f.Density.Previous, f.VelocityX.Current, f.VelocityY.Current)
}

func (f *FluidEffect) ProcessFrame(dt float32, fb *fx_common.FrameBuffer) {
	f.AccumulatedTime += dt

	// Simulate the fluid physics in fixed time steps to ensure stability and consistency
	for f.AccumulatedTime >= f.TimeStep {
		f.simulateVelocity()
		f.simulateDensity()
		f.AccumulatedTime -= f.TimeStep
	}

	// Draw

}
