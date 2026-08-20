package fx_fastripple

// 2D Water Ripples

import (
	"fmt"

	fx_common "github.com/jurgen-kluft/go-gx2/test-apps/effects/common"
)

type RandU32 = fx_common.RandU
type FrameBuffer = fx_common.FrameBuffer

const (
	DampeningScale = 4096
)

type RippleEffect struct {
	AccumulatedTime float32
	StepTime        float32

	Width          int32
	Height         int32
	PixelSize      int32
	RainPercentage uint8
	RainIntensity  uint8
	RandState      RandU32 // Random number generator state for deterministic randomness

	Current         []int16
	Previous        []int16
	DampeningFactor int32
	Palette         [256]uint16
}

func NewEffect(width, height, pixelSize int32) *RippleEffect {
	effect := &RippleEffect{
		AccumulatedTime: 0.0,
		StepTime:        0.016, // 60 FPS
		Width:           width,
		Height:          height,
		PixelSize:       pixelSize,
		RainPercentage:  25,         // 25 / 255 = ~10% chance of rain per frame
		RainIntensity:   255,        // Maximum intensity for rain drops
		RandState:       0xDEADBEEF, // Initialize with a fixed seed for deterministic randomness
		Current:         make([]int16, width*height),
		Previous:        make([]int16, width*height),
		DampeningFactor: 4093, // 0.99 (4.12 fixed point)
	}

	for i := range effect.Current {
		effect.Current[i] = 0
		effect.Previous[i] = 0
	}

	type rgb888 struct {
		r uint8
		g uint8
		b uint8
	}

	interpolateColor := func(start, end rgb888, factor float32) rgb888 {
		rs := float32(start.r)
		gs := float32(start.g)
		bs := float32(start.b)

		re := float32(end.r)
		ge := float32(end.g)
		be := float32(end.b)

		r := uint8(rs*(1.0-factor) + re*factor)
		g := uint8(gs*(1.0-factor) + ge*factor)
		b := uint8(bs*(1.0-factor) + be*factor)

		return rgb888{r, g, b}
	}

	toRgb565 := func(c rgb888) uint16 {
		return ((uint16(c.r) >> 3) << 11) | ((uint16(c.g) >> 2) << 5) | (uint16(c.b) >> 3)
	}

	// Generate a palette, where 127 is close to white (top of the wave), and 0 is
	// normal blue water. -127 is the bottom of the wave, which is very dark blue.
	// darkBlue := uint16(0x0024)  // ultra-dark midnight ocean blue
	// oceanBlue := uint16(0x03B7) // Ocean blue color in RGB565 format
	// waveTop := uint16(0xF7BE)   // a bright, icy ocean white
	bottom := rgb888{0, 0, 4}    // Dark blue for the bottom of the wave
	middle := rgb888{0, 0, 128}  // Ocean blue for the surface
	top := rgb888{255, 255, 255} // White for the top of the wave
	for i := 0; i < 256; i++ {
		if i < 127 {
			// Interpolate between dark blue and ocean blue for values below 127
			factor := float32(i) / 127.0
			effect.Palette[i] = toRgb565(interpolateColor(bottom, middle, factor))
		} else {
			// Interpolate between ocean blue and white for values above 127
			factor := float32(i-127) / 128.0
			effect.Palette[i] = toRgb565(interpolateColor(middle, top, factor))
		}
	}

	// DEBUG, a randomized palette
	// for i := 0; i < 256; i++ {
	// 	effect.Palette[i] = uint16(rand.Intn(65536)) // Random color for testing
	// }

	return effect
}

func clampInt32(value int32) int32 {
	if value > 32767 {
		value = 32767
	} else if value < -32768 {
		value = -32768
	}
	return value
}

func (r *RippleEffect) update() {
	// Main interior engine (completely avoids all borders)
	idxY := r.Width
	for y := int32(1); y < r.Height-1; y++ {
		idxX := idxY + 1
		idxEnd := idxY + r.Width - 1

		for idxX < idxEnd {
			orthoSum := int32(r.Previous[idxX-1])
			orthoSum += int32(r.Previous[idxX+1])
			orthoSum += int32(r.Previous[idxX-r.Width])
			orthoSum += int32(r.Previous[idxX+r.Width])

			diagSum := int32(r.Previous[idxX-r.Width-1])
			diagSum += int32(r.Previous[idxX-r.Width+1])
			diagSum += int32(r.Previous[idxX+r.Width-1])
			diagSum += int32(r.Previous[idxX+r.Width+1])

			scaledSum := (orthoSum << 1) + diagSum
			newValue := (scaledSum / 6) - int32(r.Current[idxX])

			// Apply dampening
			newValue = (newValue * r.DampeningFactor) / DampeningScale

			newValue = clampInt32(newValue)

			r.Current[idxX] = int16(newValue)
			idxX++
		}
		idxY += r.Width
	}

	// TOP BORDER (y = 0, skipping corners x = 0 and x = Width-1)
	for x := int32(1); x < r.Width-1; x++ {
		idx := x

		orthoSum := int32(r.Previous[idx-1])
		orthoSum += int32(r.Previous[idx+1])
		orthoSum += int32(r.Previous[idx])
		orthoSum += int32(r.Previous[idx+r.Width])

		diagSum := int32(r.Previous[idx-1])
		diagSum += int32(r.Previous[idx+1])
		diagSum += int32(r.Previous[idx+r.Width-1])
		diagSum += int32(r.Previous[idx+r.Width+1])

		scaledSum := ((orthoSum << 1) + diagSum) << 2
		newValue := ((scaledSum / 6) >> 2) - int32(r.Current[idx])

		newValue = (newValue * r.DampeningFactor) / DampeningScale
		newValue = clampInt32(newValue)
		r.Current[idx] = int16(newValue)
	}

	// BOTTOM BORDER (y = Height-1, skipping corners)
	botRowOffset := (r.Height - 1) * r.Width
	for x := int32(1); x < r.Width-1; x++ {
		idx := botRowOffset + x

		orthoSum := int32(r.Previous[idx-1])
		orthoSum += int32(r.Previous[idx+1])
		orthoSum += int32(r.Previous[idx-r.Width])
		orthoSum += int32(r.Previous[idx])

		diagSum := int32(r.Previous[idx-r.Width-1])
		diagSum += int32(r.Previous[idx-r.Width+1])
		diagSum += int32(r.Previous[idx-1])
		diagSum += int32(r.Previous[idx+1])

		scaledSum := ((orthoSum << 1) + diagSum) << 2
		newValue := ((scaledSum / 6) >> 2) - int32(r.Current[idx])

		newValue = (newValue * r.DampeningFactor) / DampeningScale
		newValue = clampInt32(newValue)
		r.Current[idx] = int16(newValue)
	}

	// LEFT BORDER (x = 0, skipping corners)
	for y := int32(1); y < r.Height-1; y++ {
		idx := y * r.Width

		orthoSum := int32(r.Previous[idx])
		orthoSum += int32(r.Previous[idx+1])
		orthoSum += int32(r.Previous[idx-r.Width])
		orthoSum += int32(r.Previous[idx+r.Width])

		diagSum := int32(r.Previous[idx-r.Width])
		diagSum += int32(r.Previous[idx-r.Width+1])
		diagSum += int32(r.Previous[idx+r.Width])
		diagSum += int32(r.Previous[idx+r.Width+1])

		scaledSum := ((orthoSum << 1) + diagSum) << 2
		newValue := ((scaledSum / 6) >> 2) - int32(r.Current[idx])

		newValue = (newValue * r.DampeningFactor) / DampeningScale
		newValue = clampInt32(newValue)
		r.Current[idx] = int16(newValue)
	}

	// RIGHT BORDER (x = Width-1, skipping corners)
	for y := int32(1); y < r.Height-1; y++ {
		idx := (y * r.Width) + r.Width - 1

		orthoSum := int32(r.Previous[idx-1])
		orthoSum += int32(r.Previous[idx])
		orthoSum += int32(r.Previous[idx-r.Width])
		orthoSum += int32(r.Previous[idx+r.Width])

		diagSum := int32(r.Previous[idx-r.Width-1])
		diagSum += int32(r.Previous[idx-r.Width])
		diagSum += int32(r.Previous[idx+r.Width-1])
		diagSum += int32(r.Previous[idx+r.Width])

		scaledSum := ((orthoSum << 1) + diagSum) << 2
		newValue := ((scaledSum / 6) >> 2) - int32(r.Current[idx])

		newValue = (newValue * r.DampeningFactor) / DampeningScale
		newValue = clampInt32(newValue)
		r.Current[idx] = int16(newValue)
	}

	// Helper function to isolate the repetitive math for the 4 single corner indices
	calcCorner := func(idx int32, ortho1, ortho2, diagonal int32) {
		orthoSum := int32(r.Previous[ortho1])
		orthoSum += int32(r.Previous[ortho2])
		orthoSum += int32(r.Previous[ortho2])
		orthoSum += int32(r.Previous[ortho2])

		diagSum := int32(r.Previous[diagonal])
		diagSum += int32(r.Previous[diagonal])
		diagSum += int32(r.Previous[diagonal])
		diagSum += int32(r.Previous[diagonal])

		scaledSum := ((orthoSum << 1) + diagSum) << 2
		newValue := ((scaledSum / 6) >> 2) - int32(r.Current[idx])

		newValue = (newValue * r.DampeningFactor) / DampeningScale
		newValue = clampInt32(newValue)
		r.Current[idx] = int16(newValue)
	}

	w := r.Width
	h := r.Height

	// Top-Left Corner (0, 0) -> Right, Down, Down-Right
	calcCorner(0, 1, w, w+1)

	// Top-Right Corner (Width-1, 0) -> Left, Down, Down-Left
	calcCorner(w-1, w-2, w+w-1, w+w-2)

	// Bottom-Left Corner (0, Height-1) -> Up, Right, Up-Right
	calcCorner((h-1)*w, (h-2)*w, (h-1)*w+1, (h-2)*w+1)

	// Bottom-Right Corner (Width-1, Height-1) -> Up, Left, Up-Left
	calcCorner(h*w-1, (h-1)*w-1, h*w-2, (h-1)*w-2)

	r.Current, r.Previous = r.Previous, r.Current
}

func (r *RippleEffect) draw(fb *FrameBuffer) {
	// debug, determine the min and max index used for the palette
	minIdx := int32(65536)
	maxIdx := int32(-65536)

	for y := int32(0); y < r.Height; y++ {
		for x := int32(0); x < r.Width; x++ {
			idx := y*r.Width + x

			// Grid values are normally between 1.0 and -1.0, and we are using fixed point 4.12
			// So here we
			value := int32(r.Current[idx])
			paletteIdx := (value >> 4) + 128 // Convert from 4.12 fixed point to 8-bit index

			if paletteIdx < minIdx {
				minIdx = paletteIdx
			}
			if paletteIdx > maxIdx {
				maxIdx = paletteIdx
			}

			if paletteIdx < 0 {
				paletteIdx = 0
			} else if paletteIdx > 255 {
				paletteIdx = 255
			}

			color := r.Palette[paletteIdx]

			// Draw the pixel as a rectangle of size PixelSize x PixelSize
			posX := x * r.PixelSize
			posY := y * r.PixelSize
			for py := int32(0); py < r.PixelSize; py++ {
				for px := int32(0); px < r.PixelSize; px++ {
					fb.Pixels[fb.Width*(posY+py)+(posX+px)] = color
				}
			}
		}
	}
	fmt.Printf("Min palette index: %d, Max palette index: %d\n", minIdx, maxIdx)
}

func (r *RippleEffect) SetRain(percentage uint8, intensity uint8) {
	r.RainPercentage = percentage
	r.RainIntensity = intensity
}

func (r *RippleEffect) ProcessFrame(dt float32, fb *FrameBuffer) {
	r.AccumulatedTime += dt
	if r.AccumulatedTime < r.StepTime {
		return
	}
	r.AccumulatedTime -= r.StepTime

	// Randomly add rain drops based on the rain percentage
	rnd := &r.RandState
	if (rnd.PseudoRand() & 0xFF) < uint32(r.RainPercentage) {
		x := int32(rnd.PseudoRand()%uint32(r.Width-2)) + 1
		y := int32(rnd.PseudoRand()%uint32(r.Height-2)) + 1
		idx := y*r.Width + x
		// fixed point 4.12
		// 1.0 = 4096 which is DampeningScale, so we scale the intensity accordingly
		intensity := int16((0.6*float32(DampeningScale) + 0.4*DampeningScale*(float32(r.RainIntensity)/255.0)*float32(rnd.PseudoRand()&0xFF)/255.0))
		r.Previous[idx] = intensity

		fmt.Printf("Adding rain drop at (%d, %d) with intensity %d\n", x, y, intensity)
	}

	r.update()
	r.draw(fb)
}
