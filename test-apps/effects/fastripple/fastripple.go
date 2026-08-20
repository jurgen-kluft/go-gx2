package fx_fastripple

// 2D Water Ripples

import (
	"math"

	fx_common "github.com/jurgen-kluft/go-gx2/test-apps/effects/common"
)

type RandU32 = fx_common.RandU
type FrameBuffer = fx_common.FrameBuffer

const (
	DampeningScale         = 16384 // 2.14 fixed point representation of 1.0
	PaletteSize            = 256
	DefaultPaletteExponent = 0.46
	OrthogonalWeight       = 5
	DiagonalWeight         = 2
	StencilDivisor         = 14
)

type rgb888 struct {
	r uint8
	g uint8
	b uint8
}

type paletteStop struct {
	position float64
	color    rgb888
}

func interpolateColor(start, end rgb888, factor float64) rgb888 {
	return rgb888{
		r: uint8(float64(start.r) + float64(int(end.r)-int(start.r))*factor),
		g: uint8(float64(start.g) + float64(int(end.g)-int(start.g))*factor),
		b: uint8(float64(start.b) + float64(int(end.b)-int(start.b))*factor),
	}
}

func toRgb565(color rgb888) uint16 {
	return ((uint16(color.r) >> 3) << 11) | ((uint16(color.g) >> 2) << 5) | (uint16(color.b) >> 3)
}

func interpolatePalette(stops []paletteStop, position float64) rgb888 {
	for index := 1; index < len(stops); index++ {
		if position <= stops[index].position {
			start := stops[index-1]
			end := stops[index]
			factor := (position - start.position) / (end.position - start.position)
			return interpolateColor(start.color, end.color, factor)
		}
	}
	return stops[len(stops)-1].color
}

func generatePalette(exponent float64) [PaletteSize]uint16 {
	const (
		lowerMiddle = PaletteSize/2 - 1
		upperMiddle = PaletteSize / 2
	)

	stops := []paletteStop{
		{position: -1.0, color: rgb888{8, 8, 8}},
		{position: -0.4, color: rgb888{20, 20, 20}},
		{position: 0.0, color: rgb888{20, 20, 110}},
		{position: 0.4, color: rgb888{220, 220, 220}},
		{position: 1.0, color: rgb888{0, 25, 255}},
	}
	var palette [PaletteSize]uint16

	for index := 0; index <= lowerMiddle; index++ {
		distance := float64(lowerMiddle-index) / float64(lowerMiddle)
		palette[index] = toRgb565(interpolatePalette(stops, -math.Pow(distance, exponent)))
	}
	for index := upperMiddle; index < PaletteSize; index++ {
		distance := float64(index-upperMiddle) / float64(PaletteSize-1-upperMiddle)
		palette[index] = toRgb565(interpolatePalette(stops, math.Pow(distance, exponent)))
	}

	return palette
}

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
	PaletteExponent float64
	Palette         [PaletteSize]uint16
}

func NewEffect(width, height, pixelSize int32) *RippleEffect {
	effect := &RippleEffect{
		AccumulatedTime: 0.0,
		StepTime:        0.016, // 60 FPS
		Width:           width,
		Height:          height,
		PixelSize:       pixelSize,
		RainPercentage:  15,         // 25 / 255 = ~10% chance of rain per frame
		RainIntensity:   255,        // Maximum intensity for rain drops
		RandState:       0xDEADBEEF, // Initialize with a fixed seed for deterministic randomness
		Current:         make([]int16, width*height),
		Previous:        make([]int16, width*height),
		DampeningFactor: 16250, // Slightly less than 1.0 in 2.14 fixed point
		PaletteExponent: DefaultPaletteExponent,
	}
	effect.Palette = generatePalette(effect.PaletteExponent)

	for i := range effect.Current {
		effect.Current[i] = 0
		effect.Previous[i] = 0
	}

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

func computeNewValue(orthoSum, diagSum, currentValue int32, dampeningFactor int32) int16 {
	//scaledSum := orthoSum*OrthogonalWeight + diagSum*DiagonalWeight
	scaledSum := diagSum
	//newValue := (scaledSum / StencilDivisor) - currentValue
	newValue := (scaledSum / 2) - currentValue
	newValue = (newValue * dampeningFactor) / DampeningScale
	newValue = clampInt32(newValue)
	return int16(newValue)
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

			r.Current[idxX] = computeNewValue(orthoSum, diagSum, int32(r.Current[idxX]), r.DampeningFactor)
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

		r.Current[idx] = computeNewValue(orthoSum, diagSum, int32(r.Current[idx]), r.DampeningFactor)
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

		r.Current[idx] = computeNewValue(orthoSum, diagSum, int32(r.Current[idx]), r.DampeningFactor)
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

		r.Current[idx] = computeNewValue(orthoSum, diagSum, int32(r.Current[idx]), r.DampeningFactor)
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

		r.Current[idx] = computeNewValue(orthoSum, diagSum, int32(r.Current[idx]), r.DampeningFactor)
	}

	// Helper function to isolate the repetitive math for the 4 single corner indices
	calcCorner := func(idx int32, ortho1, ortho2, diagonal int32) {
		orthoSum := int32(r.Previous[ortho1])
		orthoSum += int32(r.Previous[ortho2])
		orthoSum += int32(r.Previous[ortho1])
		orthoSum += int32(r.Previous[ortho2])

		diagSum := int32(r.Previous[diagonal])
		diagSum += int32(r.Previous[diagonal])
		diagSum += int32(r.Previous[diagonal])
		diagSum += int32(r.Previous[diagonal])

		r.Current[idx] = computeNewValue(orthoSum, diagSum, int32(r.Current[idx]), r.DampeningFactor)
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

func (r *RippleEffect) paletteIndex(value int32) int32 {
	if value > DampeningScale {
		value = DampeningScale
	} else if value < -DampeningScale {
		value = -DampeningScale
	}

	shiftedValue := int64(value + DampeningScale)
	return int32(shiftedValue * (PaletteSize - 1) / (2 * DampeningScale))
}

func (r *RippleEffect) draw(fb *FrameBuffer) {
	// debug, determine the min and max index used for the palette
	minIdx := int32(65536)
	maxIdx := int32(-65536)

	for y := int32(0); y < r.Height; y++ {
		for x := int32(0); x < r.Width; x++ {
			idx := y*r.Width + x

			value := int32(r.Current[idx])
			paletteIdx := r.paletteIndex(value)

			if paletteIdx < minIdx {
				minIdx = paletteIdx
			}
			if paletteIdx > maxIdx {
				maxIdx = paletteIdx
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
	//fmt.Printf("Min palette index: %d, Max palette index: %d\n", minIdx, maxIdx)
}

func (r *RippleEffect) SetRain(percentage uint8, intensity uint8) {
	r.RainPercentage = percentage
	r.RainIntensity = intensity
}

func (r *RippleEffect) SetPaletteExponent(exponent float64) {
	if exponent <= 0 || math.IsNaN(exponent) || math.IsInf(exponent, 0) {
		panic("palette exponent must be finite and greater than zero")
	}
	r.PaletteExponent = exponent
	r.Palette = generatePalette(exponent)
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

		//fmt.Printf("Adding rain drop at (%d, %d) with intensity %d\n", x, y, intensity)
	}

	r.update()
	r.draw(fb)
}
