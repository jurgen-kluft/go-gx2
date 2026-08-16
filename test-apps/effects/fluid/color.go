package fx_fluid

type color struct {
	r, g, b uint8
}

func rgbToHSV(c color) (hue, saturation, value float32) {
	// Convert RGB to HSV
	hsv := [3]float32{float32(c.r) / 255.0, float32(c.g) / 255.0, float32(c.b) / 255.0}

	max := int8(0)
	min := int8(0)
	for i := int8(1); i < 3; i++ {
		if hsv[i] > hsv[max] {
			max = i
		}
		if hsv[i] < hsv[min] {
			min = i
		}
	}

	delta := hsv[max] - hsv[min]

	if max == 0 {
		saturation = 0
	} else {
		saturation = delta / hsv[max]
	}

	if delta == 0 {
		hue = 0
	} else {
		switch max {
		case 0:
			hue = (hsv[1] - hsv[2]) / delta
			if hsv[1] < hsv[2] {
				hue += 6
			}
		case 1:
			hue = (hsv[2]-hsv[0])/delta + 2
		case 2:
			hue = (hsv[0]-hsv[1])/delta + 4
		}
		hue *= 60
	}

	value = hsv[max]
	return
}

func hsvToRGBFast(hue, sat, val uint8) (outColor color) {
	if sat == 0 {
		outColor.r, outColor.g, outColor.b = val, val, val
		return
	}

	// 1. Shift by 6 divides 256 into 4 perfect sectors (0 to 3)
	// To get 6 sectors, we use a slightly different math scale or standard 0-255 mapping:
	// Let's use the industry standard FastLED style mapping where each sector is 42.5 steps.
	// Since 256 / 6 = 42.66, we can divide by 43 using a fast approximation,
	// OR we can do a 3-sector / 4-sector approach.

	// If you explicitly want a '>> 6' shift, it means you have 4 sectors (64 steps each).
	// To get 6 sectors natively with bitshifts, we use 256 total range:

	// Fast sector extraction for 6 sectors in 0-255 range:
	// (hue * 6) >> 8 gives a perfect 0-5 sector index!
	sector := (uint16(hue) * 6) >> 8

	// Remainder calculation using pure shifts/scales
	// offset within the 42.66 step sector
	rem := uint32((uint16(hue) * 6) & 255)

	v := uint32(val)
	s := uint32(sat)

	p := uint8((v * (255 - s)) >> 8)
	// rem is 0-255 here, so we scale down by 8 bits
	q := uint8((v * (255 - ((s * rem) >> 8))) >> 8)
	t := uint8((v * (255 - ((s * (255 - rem)) >> 8))) >> 8)

	switch sector {
	case 0:
		outColor.r, outColor.g, outColor.b = val, t, p
	case 1:
		outColor.r, outColor.g, outColor.b = q, val, p
	case 2:
		outColor.r, outColor.g, outColor.b = p, val, t
	case 3:
		outColor.r, outColor.g, outColor.b = p, q, val
	case 4:
		outColor.r, outColor.g, outColor.b = t, p, val
	default: // sector 5
		outColor.r, outColor.g, outColor.b = val, p, q
	}

	return
}

func rgbToHSVFast(r, g, b uint8) (h, s, v uint8) {
	min := r
	if g < min {
		min = g
	}
	if b < min {
		min = b
	}

	max := r
	if g > max {
		max = g
	}
	if b > max {
		max = b
	}

	// Value (V) is simply the maximum component
	v = max
	if v == 0 {
		return 0, 0, 0 // Black: Hue and Saturation are 0
	}

	// Delta (chroma)
	delta := max - min
	if delta == 0 {
		return 0, 0, v // Grayscale: Saturation is 0, Hue is undefined (0)
	}

	// Saturation (S) = (delta / max) * 255
	// Scaled up by 255 before dividing to maintain integer precision
	s = uint8((uint32(delta) * 255) / uint32(max))

	// Calculate Hue (H) scaled to a 0-255 wheel
	// Standard formulas use 60-degree sectors (total 360).
	// To fit 6 sectors into 256 steps, each sector is exactly 42.66 steps wide (256 / 6).
	// Instead of float math, we multiply by 43 and apply a fast rounding shift.
	var hSector int32
	r32, g32, b32 := int32(r), int32(g), int32(b)
	d32 := int32(delta)

	if max == r {
		// Sector 0 (Red -> Green) or Sector 5 (Magenta -> Red)
		hSector = ((g32 - b32) * 43) / d32
	} else if max == g {
		// Sector 1 (Green -> Blue) or Sector 2 (Cyan -> Green)
		hSector = 85 + (((b32 - r32) * 43) / d32)
	} else {
		// Sector 3 (Blue -> Cyan) or Sector 4 (Magenta -> Blue)
		hSector = 171 + (((r32 - g32) * 43) / d32)
	}

	// Handle negative wrapping around the 0-255 wheel cleanly
	if hSector < 0 {
		hSector += 256
	}

	h = uint8(hSector)
	return
}

func hsvToRGB(hue, saturation, value float32) (outColor color) {
	c := value * saturation
	m := value - c

	// Scale hue to sector float (0.0 to 6.0)
	hSectorFloat := hue * (1.0 / 60.0)
	sector := int(hSectorFloat)

	// Handle the edge case where hue was exactly 360.0
	if sector == 6 {
		sector = 5
		hSectorFloat = 6.0
	}

	// Calculate fractional remainder without math.Mod
	f := hSectorFloat - float32(sector)

	x1 := c * f
	x2 := c * (1.0 - f)

	var r1, g1, b1 float32
	switch sector {
	case 0:
		r1, g1, b1 = c, x1, 0
	case 1:
		r1, g1, b1 = x2, c, 0
	case 2:
		r1, g1, b1 = 0, c, x1
	case 3:
		r1, g1, b1 = 0, x2, c
	case 4:
		r1, g1, b1 = x1, 0, c
	case 5:
		r1, g1, b1 = c, 0, x2
	}

	outColor.r = uint8((r1 + m) * 255.0)
	outColor.g = uint8((g1 + m) * 255.0)
	outColor.b = uint8((b1 + m) * 255.0)

	return
}
