package main

type FireEffect struct {
	rand_state  uint32   // Random number generator state for deterministic randomness
	fire_grid   []uint8  // 8-bit[fire_width * fire_height] heat values for each pixel in the fire effect
	fire_width  int32    // Width of the fire effect grid
	fire_height int32    // Height of the fire effect grid
	palette     []uint16 // color palette mapping heat values to RGB565 colors
}

func NewFireEffect(width, height int32) *FireEffect {
	fe := &FireEffect{
		rand_state:  12345, // Initialize with a default seed
		fire_grid:   make([]uint8, width*(height+1)),
		fire_width:  width,
		fire_height: height,
		palette:     make([]uint16, 256),
	}

	var r, g, b uint8 = 0, 0, 0
	for i := range 256 {

		if i < 85 { // Phase 1: Deep red to bright red
			r = (uint8(i) * 3)
			g = 0
			b = 0
		} else if i < 170 { // Phase 2: Red transitions to Orange and Yellow
			r = 255
			g = uint8((i - 85) * 3)
			b = 0
		} else if i < 230 { // Phase 3: Yellow transitions to White
			r = 255
			g = 255
			b = uint8((i - 170) * 3)
		} else {
			r = 255
			g = 255
			b = 255
		}

		// Pack into a 16-bit pixel value (Format: RGB565)
		// Adjust shifts depending on the window framework's expected pixel format
		fe.palette[i] = (uint16(r>>3) << 11) | (uint16(g>>2) << 5) | uint16(b>>3)
	}

	return fe
}

func (fe *FireEffect) ProcessFrame(frameBuffer *FrameBuffer) {
	fe.Update()
	fe.Render(frameBuffer)
}

func (fe *FireEffect) pseudoRand() uint32 {
	state := fe.rand_state*uint32(747796405) + uint32(2891336453)
	word := ((state >> ((state >> 28) + 4)) ^ state) * uint32(277803737)
	fe.rand_state += 1
	return (word >> 22) ^ word
}

func (fe *FireEffect) seedFineBottom() {
	bottom_row_offset := (fe.fire_height - 1) * fe.fire_width

	x := int32(0)
	for x < fe.fire_width {
		r := fe.pseudoRand() & 255
		if r > 50 { // 80% chance to spark, creating baseline turbulence
			heat := 192 + int32(fe.pseudoRand()&63) // Random heat value between 192 and 255
			fe.fire_grid[bottom_row_offset+x] = uint8(heat)
		} else {
			fe.fire_grid[bottom_row_offset+x] = 0
		}
		x++
	}
}

func (fe *FireEffect) Update() {
	// Inject maximum heat energy into the bottom row
	fe.seedFineBottom()

	// Propagate energy upward through the grid matrix
	y := int32(0)
	for y < (fe.fire_height - 1) {
		x := int32(1)
		for x < (fe.fire_width - 1) {
			current_idx := y*fe.fire_width + x

			// Look strictly at the three pixels directly below
			src_center := current_idx + fe.fire_width
			src_left := src_center - 1
			src_right := src_center + 1

			// Heavily weight the center pixel to force sharp vertical propagation
			total := int(fe.fire_grid[src_left]) + (int(fe.fire_grid[src_center]) * 2) + int(fe.fire_grid[src_right])

			// Step 3: Apply Asymmetric Non-Linear Decay
			current_heat := int32(total >> 2) // Divide by 4
			decay := int32(0)

			// Dynamic cooling: Cold areas die instantly; hot cores fight back
			if current_heat > 180 {
				r := fe.pseudoRand() & 255 // White/Yellow cores stay sharp
				if r < 128 {
					decay = 0
				} else {
					decay = int32(fe.pseudoRand() & 1)
				}
			} else if current_heat > 120 {
				decay = int32(fe.pseudoRand() & 1) // White/Yellow cores stay sharp
			} else if current_heat > 80 {
				decay = 1 + int32(fe.pseudoRand()&1) // Orange transitions naturally
			} else if current_heat > 20 {
				decay = 2 + int32(fe.pseudoRand()&1) // Red embers fade slowly
			} else {
				decay = 3 + int32(fe.pseudoRand()&1) // Red edges drop to black aggressively
			}

			new_val := current_heat - decay
			if new_val < 0 {
				new_val = 0
			}

			// Step 4: Write straight up or slightly offset to simulate licking tips
			// Introduce a rare horizontal drift (wind) to distort straight columns
			wind := int32(0)
			r := fe.pseudoRand() & 255
			if r < 38 { // ~15% of 256
				wind = -1 // Drift left
			} else if r > 217 { // ~85% of 256
				wind = 1
			} // Drift right

			target_idx := current_idx - fe.fire_width + wind

			// Check bounds before writing horizontally shifted pixel
			if target_idx >= 0 && target_idx < int32(len(fe.fire_grid)) {
				fe.fire_grid[target_idx] = uint8(new_val)
			}

			x++
		}
		y++
	}
}

func (fe *FireEffect) Render(frameBuffer *FrameBuffer) {
	xOffset := (int(frameBuffer.Width) - int(fe.fire_width)) / 2

	n := int32(2)
	y := int32(0)
	for y < fe.fire_height-n {
		x := int32(0)
		for x < fe.fire_width {
			idx := y*fe.fire_width + x
			color := fe.palette[fe.fire_grid[idx]]
			frameBuffer.Pixels[int(y)*frameBuffer.Width+int(x)+int(xOffset)] = color
			x++
		}
		y++
	}

	for y < fe.fire_height {
		x := int32(0)
		for x < fe.fire_width {
			idx := y*fe.fire_width + x
			paletteIndex := fe.fire_grid[idx]
			if paletteIndex > 8 {
				color := fe.palette[paletteIndex]
				frameBuffer.Pixels[int(y)*frameBuffer.Width+int(x)+int(xOffset)] = color
			} else {
				r := 255
				g := 255
				b := 32
				color := (uint16(r>>3) << 11) | (uint16(g>>2) << 5) | uint16(b>>3)
				frameBuffer.Pixels[int(y)*frameBuffer.Width+int(x)+int(xOffset)] = color
			}
			x++
		}
		y++
	}

	// Pass 2: Render the bottom 8 lines with a live pixel-blending filter
	// for y < fe.fire_height {
	// 	x := int32(0)
	// 	for x < fe.fire_width {
	// 		current_idx := y*fe.fire_width + x

	// 		// Safe boundary checks for horizontal neighbors
	// 		var left, right, mid, up uint8
	// 		if x > 0 {
	// 			left = fe.fire_grid[current_idx-1]
	// 		} else {
	// 			left = fe.fire_grid[current_idx]
	// 		}

	// 		if x < fe.fire_width-1 {
	// 			right = fe.fire_grid[current_idx+1]
	// 		} else {
	// 			right = fe.fire_grid[current_idx]
	// 		}

	// 		mid = fe.fire_grid[current_idx]

	// 		// Safe boundary check for vertical neighbor above
	// 		if y > 0 {
	// 			up = fe.fire_grid[current_idx-fe.fire_width]
	// 		} else {
	// 			up = mid
	// 		}

	// 		// Compute a 4-way spatial average of raw heat intensities
	// 		blurred_intensity := (left + right + mid + up) >> 2 // Divide by 4

	// 		// Map the smoothed value directly to the 32-bit screen output
	// 		frameBuffer.Pixels[int(y)*frameBuffer.Width+int(x)+int(xOffset)] = fe.palette[blurred_intensity]
	// 		x++
	// 	}
	// 	y++
	// }
	// for y < fe.fire_height {
	// 	x := int32(0)
	// 	for x < fe.fire_width {
	// 		color := uint16(31) << 11
	// 		frameBuffer.Pixels[int(y)*frameBuffer.Width+int(x)+int(xOffset)] = color
	// 		x++
	// 	}
	// 	y++
	// }
}
