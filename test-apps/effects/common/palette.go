package fx_common

import (
	"math"
)

type color3f struct {
	r, g, b float32
}

func (c color3f) Add(o color3f) color3f {
	return color3f{c.r + o.r, c.g + o.g, c.b + o.b}
}
func (c color3f) Mul(o color3f) color3f {
	return color3f{c.r * o.r, c.g * o.g, c.b * o.b}
}

func (c color3f) MulScalar(s float32) color3f {
	return color3f{c.r * s, c.g * s, c.b * s}
}

func computeNextColor(t float32, a color3f, b color3f, c color3f, d color3f) color3f {
	ct := c.MulScalar(t)
	ctd := ct.Add(d)
	twopictd := ctd.MulScalar(2.0 * math.Pi)
	cos := color3f{float32(math.Cos(float64(twopictd.r))), float32(math.Cos(float64(twopictd.g))), float32(math.Cos(float64(twopictd.b)))}
	return a.Add(b.Mul(cos))
}

type PaletteConfig struct {
	A color3f
	B color3f
	C color3f
	D color3f
}

func ComputePalette(config PaletteConfig) [256]uint16 {
	palette := [256]uint16{}

	for i := 0; i < 256; i++ {
		t := float32(i) / 255.0
		color := computeNextColor(t, config.A, config.B, config.C, config.D)
		r := uint8(Clampf(color.r*255.0, 0, 255))
		g := uint8(Clampf(color.g*255.0, 0, 255))
		b := uint8(Clampf(color.b*255.0, 0, 255))
		palette[i] = ConvertToRGB565(r, g, b)
	}

	return palette
}

func ShiftPalette(palette [256]uint16, shift int) [256]uint16 {
	shiftedPalette := [256]uint16{}
	for i := 0; i < 256; i++ {
		shiftedIndex := (i + shift) % 256
		if shiftedIndex < 0 {
			shiftedIndex += 256
		}
		shiftedPalette[i] = palette[shiftedIndex]
	}
	return shiftedPalette
}

//      a                     b               c                d
// 0.5, 0.5, 0.5		0.5, 0.5, 0.5	1.0, 1.0, 1.0	0.00, 0.33, 0.67
// 0.5, 0.5, 0.5		0.5, 0.5, 0.5	1.0, 1.0, 1.0	0.00, 0.10, 0.20
// 0.5, 0.5, 0.5		0.5, 0.5, 0.5	1.0, 1.0, 1.0	0.30, 0.20, 0.20
// 0.5, 0.5, 0.5		0.5, 0.5, 0.5	1.0, 1.0, 0.5	0.80, 0.90, 0.30
// 0.5, 0.5, 0.5		0.5, 0.5, 0.5	1.0, 0.7, 0.4	0.00, 0.15, 0.20
// 0.5, 0.5, 0.5		0.5, 0.5, 0.5	2.0, 1.0, 0.0	0.50, 0.20, 0.25
// 0.8, 0.5, 0.4		0.2, 0.4, 0.2	2.0, 1.0, 1.0	0.00, 0.25, 0.25

var PaletteConfigurations = []PaletteConfig{
	{color3f{0.5, 0.5, 0.5}, color3f{0.5, 0.5, 0.5}, color3f{1.0, 1.0, 1.0}, color3f{0.00, 0.33, 0.67}},
	{color3f{0.5, 0.5, 0.5}, color3f{0.5, 0.5, 0.5}, color3f{1.0, 1.0, 1.0}, color3f{0.00, 0.10, 0.20}},
	{color3f{0.5, 0.5, 0.5}, color3f{0.5, 0.5, 0.5}, color3f{1.0, 1.0, 1.0}, color3f{0.30, 0.20, 0.20}},
	{color3f{0.5, 0.5, 0.5}, color3f{0.5, 0.5, 0.5}, color3f{1.0, 1.0, 0.5}, color3f{0.80, 0.90, 0.30}},
	{color3f{0.5, 0.5, 0.5}, color3f{0.5, 0.5, 0.5}, color3f{1.0, 0.7, 0.4}, color3f{0.00, 0.15, 0.20}},
	{color3f{0.5, 0.5, 0.5}, color3f{0.5, 0.5, 0.5}, color3f{2.0, 1.0, 0.0}, color3f{0.50, 0.20, 0.25}},
	{color3f{0.8, 0.5, 0.4}, color3f{0.2, 0.4, 0.2}, color3f{2.0, 1.0, 1.0}, color3f{0.00, 0.25, 0.25}},
}

func linearInterpolateColor(t float32, start color3f, end color3f) color3f {
	return color3f{
		r: start.r + (end.r-start.r)*t,
		g: start.g + (end.g-start.g)*t,
		b: start.b + (end.b-start.b)*t,
	}
}

func NewFirePalette() [256]uint16 {

	// The fire palette goes from white hot to red to yellow, but then transitions to black smoke that fades to grey:
	// - hot white -> red
	// - red -> yellow
	// - yellow -> dark black smoke
	// - dark black smoke -> grey smoke
	// - grey smoke -> black

	palette := [256]uint16{}

	// maybe for every range should set a percentage, so that the palette can be adjusted to have
	// more or less of each color range, and then the ranges can be computed based on the percentages.
	// also hotwhite is at the end of 255, and grey smoke / black smoke is at the beginning of 0, so
	// the palette is reversed from what you might expect.
	hotwhite_to_red := float32(5.0)
	red_to_yellow := float32(15.0)
	yellow_to_black := float32(20.0)
	black_to_grey := float32(30.0)
	grey_to_black := float32(30.0)

	hotWhiteColor := color3f{1.0, 1.0, 1.0}
	redColor := color3f{1.0, 0.0, 0.0}
	yellowColor := color3f{1.0, 1.0, 0.0}
	blackColor := color3f{0.0, 0.0, 0.0}
	greyColor := color3f{0.5, 0.5, 0.5}

	// Compute the palette based on the ranges
	for i := 0; i < 256; i++ {
		t := float32(i) / 255.0
		var color color3f
		if t < hotwhite_to_red/100.0 {
			// hot white to red
			color = linearInterpolateColor(t/(hotwhite_to_red/100.0), hotWhiteColor, redColor)
		} else if t < (hotwhite_to_red+red_to_yellow)/100.0 {
			// red to yellow
			color = linearInterpolateColor((t-hotwhite_to_red/100.0)/(red_to_yellow/100.0), redColor, yellowColor)
		} else if t < (hotwhite_to_red+red_to_yellow+yellow_to_black)/100.0 {
			// yellow to black
			color = linearInterpolateColor((t-(hotwhite_to_red+red_to_yellow)/100.0)/(yellow_to_black/100.0), yellowColor, blackColor)
		} else if t < (hotwhite_to_red+red_to_yellow+yellow_to_black+black_to_grey)/100.0 {
			// black to black
			color = linearInterpolateColor((t-(hotwhite_to_red+red_to_yellow+yellow_to_black)/100.0)/(black_to_grey/100.0), blackColor, blackColor)
		} else {
			// grey to black
			color = linearInterpolateColor((t-(hotwhite_to_red+red_to_yellow+yellow_to_black+black_to_grey)/100.0)/(grey_to_black/100.0), greyColor, blackColor)
		}
		r := uint8(Clampf(color.r*255.0, 0, 255))
		g := uint8(Clampf(color.g*255.0, 0, 255))
		b := uint8(Clampf(color.b*255.0, 0, 255))
		palette[i] = ConvertToRGB565(r, g, b)
	}

	return palette
}
