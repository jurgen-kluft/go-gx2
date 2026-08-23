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
