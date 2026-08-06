package spritepack

import (
	"image/color"
	_ "image/png"
	"math"
)

type ColorRGBA uint32

const (
	ColorWhite ColorRGBA = 0xFFFFFFFF
	ColorBlack ColorRGBA = 0xFF000000
)

func NewColorRGBA8888(rgba8888 uint32) ColorRGBA {
	return ColorRGBA(rgba8888)
}

func NewColorFromR8G8B8A8(a, r, g, b uint8) ColorRGBA {
	return ColorRGBA(uint32(a)<<24 | uint32(r)<<16 | uint32(g)<<8 | uint32(b))
}

func NewColorFromRGBA8888(rgba8888 uint32) ColorRGBA {
	return ColorRGBA(rgba8888)
}

func NewColorFromRGB888(rgb888 uint32) ColorRGBA {
	return ColorRGBA(0xFF000000 | (rgb888 & 0xFFFFFF))
}

func NewColorFromRGB565(rgb565 uint16) ColorRGBA {
	r := uint8((rgb565 >> 11) & 0x1F)
	g := uint8((rgb565 >> 5) & 0x3F)
	b := uint8(rgb565 & 0x1F)
	return ColorRGBA(0xFF000000 | (uint32(r)*255/31)<<16 | (uint32(g)*255/63)<<8 | (uint32(b) * 255 / 31))
}

func (c ColorRGBA) ToRGB565() uint16 {
	r := uint8((uint16(c>>24) * 31) / 255)
	g := uint8((uint16(c>>16) * 63) / 255)
	b := uint8((uint16(c>>8) * 31) / 255)

	return uint16(r)<<11 | uint16(g)<<5 | uint16(b)
}

func (c ColorRGBA) ToRGB888() uint32 {
	return uint32(c>>24)<<16 | uint32(c>>16)<<8 | uint32(c>>8)
}

func (c ColorRGBA) ToRGBA8888() uint32 {
	return uint32(c>>24)<<24 | uint32(c>>16)<<16 | uint32(c>>8)<<8 | uint32(c>>0)
}

func (c ColorRGBA) ToR8G8B8A8() (r, g, b, a uint8) {
	a = uint8(c >> 24)
	r = uint8(c >> 16)
	g = uint8(c >> 8)
	b = uint8(c)
	return
}

//  .d8888b.           888                       8888888b.          888          888    888
// d88P  Y88b          888                       888   Y88b         888          888    888
// 888    888          888                       888    888         888          888    888
// 888         .d88b.  888  .d88b.  888d888      888   d88P 8888b.  888  .d88b.  888888 888888 .d88b.  .d8888b
// 888        d88""88b 888 d88""88b 888P"        8888888P"     "88b 888 d8P  Y8b 888    888   d8P  Y8b 88K
// 888    888 888  888 888 888  888 888          888       .d888888 888 88888888 888    888   88888888 "Y8888b.
// Y88b  d88P Y88..88P 888 Y88..88P 888          888       888  888 888 Y8b.     Y88b.  Y88b. Y8b.          X88
//  "Y8888P"   "Y88P"  888  "Y88P"  888          888       "Y888888 888  "Y8888   "Y888  "Y888 "Y8888   88888P'

type PaletteRGBA []ColorRGBA
type PaletteRGB565 []uint16

func NewPaletteRGBA(colors []ColorRGBA) PaletteRGBA {
	return PaletteRGBA(colors)
}

func ComparePalettes(p1, p2 PaletteRGBA) bool {
	if len(p1) != len(p2) {
		return false
	}
	for i := range p1 {
		if p1[i] != p2[i] {
			return false
		}
	}
	return true
}

func (p PaletteRGBA) ToPaletteRGB565() PaletteRGB565 {
	palette := make([]uint16, len(p))
	for i, c := range p {
		palette[i] = c.ToRGB565()
	}
	return palette
}

func (p PaletteRGBA) ToPaletteFormat(fmt PaletteFormat) []byte {
	switch fmt {
	case FMT_PALETTE_RGB888:
		data := make([]byte, 4*len(p))
		for i, c := range p {
			rgba := c.ToRGBA8888()
			data[4*i+0] = uint8(rgba >> 24)
			data[4*i+1] = uint8(rgba >> 16)
			data[4*i+2] = uint8(rgba >> 8)
			data[4*i+3] = uint8(rgba)
		}
		return data
	case FMT_PALETTE_RGB565:
		data := make([]byte, 2*len(p))
		for i, c := range p {
			rgb565 := c.ToRGB565()
			data[2*i+0] = uint8(rgb565 >> 8)
			data[2*i+1] = uint8(rgb565)
		}
		return data
	default:
		return nil
	}
}

// ThemeAnchor defines how a specific "role" changes from the original style to a theme style
type ThemeAnchor struct {
	OrigH, OrigS, OrigV       float64 // The original reference color (HSV)
	TargetH, TargetS, TargetV float64 // The new themed color (HSV)
}

// Theme ID enum
type ThemeID int

const (
	ThemeDark ThemeID = iota
	ThemeBirthday
	ThemeChristmas
	ThemeSummer
	ThemeWinter
	ThemeAutumn
	ThemeSpring
	ThemeRaining

	// Total count tracking
	ThemeCount
)

// String helper for logging or debugging
func (t ThemeID) String() string {
	return [...]string{
		"Dark", "Birthday", "Christmas", "Summer",
		"Winter", "Autumn", "Spring", "Raining",
	}[t]
}

// ThemeConfig holds the mathematical parameters for a given seasonal/style preset
type ThemeConfig struct {
	Name          string
	SatMultiplier float64
	ValMultiplier float64
	Anchors       []ThemeAnchor
}

// Global lookup table indexed directly by the ThemeID enum
var ThemeRegistry = [ThemeCount]ThemeConfig{
	ThemeDark: {
		Name:          "Dark",
		SatMultiplier: 1.15,
		ValMultiplier: 0.45,
		Anchors:       []ThemeAnchor{}, // No specific color swaps; purely a value compression
	},
	ThemeBirthday: {
		Name:          "Birthday",
		SatMultiplier: 1.20,
		ValMultiplier: 1.10,
		Anchors: []ThemeAnchor{
			{OrigH: 220, OrigS: 0.75, OrigV: 0.80, TargetH: 330, TargetS: 0.85, TargetV: 0.90}, // Blue -> Party Pink
			{OrigH: 180, OrigS: 0.65, OrigV: 0.90, TargetH: 50, TargetS: 0.90, TargetV: 0.95},  // Cyan -> Confetti Gold
		},
	},
	ThemeChristmas: {
		Name:          "Christmas",
		SatMultiplier: 1.00,
		ValMultiplier: 0.90,
		Anchors: []ThemeAnchor{
			{OrigH: 220, OrigS: 0.75, OrigV: 0.80, TargetH: 0, TargetS: 0.85, TargetV: 0.75},   // Blue -> Ribbon Red
			{OrigH: 180, OrigS: 0.65, OrigV: 0.90, TargetH: 135, TargetS: 0.80, TargetV: 0.70}, // Cyan -> Pine Green
		},
	},
	ThemeSummer: {
		Name:          "Summer",
		SatMultiplier: 1.10,
		ValMultiplier: 1.15,
		Anchors: []ThemeAnchor{
			{OrigH: 220, OrigS: 0.75, OrigV: 0.80, TargetH: 190, TargetS: 0.75, TargetV: 0.90}, // Blue -> Turquoise
			{OrigH: 180, OrigS: 0.65, OrigV: 0.90, TargetH: 55, TargetS: 0.85, TargetV: 1.00},  // Cyan -> Beach Yellow
		},
	},
	ThemeWinter: {
		Name:          "Winter",
		SatMultiplier: 0.65,
		ValMultiplier: 1.20,
		Anchors: []ThemeAnchor{
			{OrigH: 220, OrigS: 0.75, OrigV: 0.80, TargetH: 210, TargetS: 0.40, TargetV: 0.85}, // Blue -> Glacier Faded Blue
			{OrigH: 180, OrigS: 0.65, OrigV: 0.90, TargetH: 180, TargetS: 0.10, TargetV: 1.00}, // Cyan -> Frost White
		},
	},
	ThemeAutumn: {
		Name:          "Autumn",
		SatMultiplier: 0.90,
		ValMultiplier: 0.85,
		Anchors: []ThemeAnchor{
			{OrigH: 220, OrigS: 0.75, OrigV: 0.80, TargetH: 25, TargetS: 0.85, TargetV: 0.70}, // Blue -> Burnt Orange
			{OrigH: 180, OrigS: 0.65, OrigV: 0.90, TargetH: 45, TargetS: 0.80, TargetV: 0.85}, // Cyan -> Foliage Yellow
		},
	},
	ThemeSpring: {
		Name:          "Spring",
		SatMultiplier: 0.55,
		ValMultiplier: 1.15,
		Anchors: []ThemeAnchor{
			{OrigH: 220, OrigS: 0.75, OrigV: 0.80, TargetH: 270, TargetS: 0.45, TargetV: 0.90}, // Blue -> Pastel Lavender
			{OrigH: 180, OrigS: 0.65, OrigV: 0.90, TargetH: 110, TargetS: 0.50, TargetV: 0.95}, // Cyan -> Mint Green
		},
	},
	ThemeRaining: {
		Name:          "Raining",
		SatMultiplier: 0.40,
		ValMultiplier: 0.60,
		Anchors: []ThemeAnchor{
			{OrigH: 220, OrigS: 0.75, OrigV: 0.80, TargetH: 215, TargetS: 0.45, TargetV: 0.50}, // Blue -> Overcast Storm Blue
			{OrigH: 180, OrigS: 0.65, OrigV: 0.90, TargetH: 195, TargetS: 0.30, TargetV: 0.70}, // Cyan -> Wet Puddle Teal
		},
	},
}

// GenerateThemedPalette transforms a 256-color base palette into a specific seasonal/environmental style
func GenerateThemedPalette(basePalette []color.RGBA, anchors []ThemeAnchor, globalSatMult, globalValMult float64) []color.RGBA {
	themed := make([]color.RGBA, len(basePalette))

	for i, c := range basePalette {
		if c.A == 0 { // Preserve transparency completely
			themed[i] = c
			continue
		}

		h, s, v := RGBToHSV(c.R, c.G, c.B)

		// If anchors are provided, find the closest matching structural color and shift toward it
		if len(anchors) > 0 {
			closestDist := math.MaxFloat64
			var bestAnchor ThemeAnchor

			for _, anchor := range anchors {
				// Calculate a weighted distance in HSV space (Hue difference handles wrapping around 360)
				dh := math.Min(math.Abs(h-anchor.OrigH), 360.0-math.Abs(h-anchor.OrigH)) / 360.0
				ds := s - anchor.OrigS
				dv := v - anchor.OrigV

				// Hue differences are heavily weighted for context, value/sat capture lighting/gradients
				dist := (dh * dh * 3.0) + (ds * ds) + (dv * dv)

				if dist < closestDist {
					closestDist = dist
					bestAnchor = anchor
				}
			}

			// Apply the shift relative to the anchor
			hueDiff := h - bestAnchor.OrigH
			h = bestAnchor.TargetH + hueDiff // Retain local gradient steps

			// Blend saturation and value transitions gently
			s = bestAnchor.TargetS * (s / math.Max(bestAnchor.OrigS, 0.01))
			v = bestAnchor.TargetV * (v / math.Max(bestAnchor.OrigV, 0.01))
		}

		// Apply global environmental modifiers and clamp boundaries
		h = math.Mod(h, 360.0)
		if h < 0 {
			h += 360.0
		}

		s = math.Max(0.0, math.Min(1.0, s*globalSatMult))
		v = math.Max(0.0, math.Min(1.0, v*globalValMult))

		r, g, b := HSVToRGB(h, s, v)
		themed[i] = color.RGBA{R: r, G: g, B: b, A: c.A}
	}

	return themed
}

// 8888888b.   .d8888b.  888888b.            d88P              Y88b          888    888  .d8888b.  888     888
// 888   Y88b d88P  Y88b 888  "88b         d88P                  Y88b        888    888 d88P  Y88b 888     888
// 888    888 888    888 888  .88P        d88P                    Y88b       888    888 Y88b.      888     888
// 888   d88P 888        8888888K.       o88P   oooooooooooooooo   Y88L      8888888888  "Y888b.   Y88b   d88P
// 8888888P"  888  88888 888  "Y88b      Y88b                      d88P      888    888     "Y88b.  Y88b d88P
// 888 T88b   888    888 888    888       Y88b                    d88P       888    888       "888   Y88o88P
// 888  T88b  Y88b  d88P 888   d88P        Y88b                  d88P        888    888 Y88b  d88P    Y888P
// 888   T88b  "Y8888P88 8888888P"           Y88b              d88P          888    888  "Y8888P"      Y8P

// RGBToHSV converts red, green, and blue values (0-255) into
// Hue (0-360), Saturation (0-1), and Value (0-1).
func RGBToHSV(r, g, b uint8) (h, s, v float64) {
	// Normalize RGB values to the range [0, 1]
	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0

	// Find the maximum and minimum values among R, G, B
	max := math.Max(rf, math.Max(gf, bf))
	min := math.Min(rf, math.Min(gf, bf))
	delta := max - min

	// 1. Calculate Value (Brightness)
	v = max

	// 2. Calculate Saturation
	if max == 0 {
		s = 0
		h = 0 // Hue is undefined when max is 0
		return
	}
	s = delta / max

	// 3. Calculate Hue
	if delta == 0 {
		h = 0 // Color is a shade of gray
		return
	}

	switch max {
	case rf:
		h = (gf - bf) / delta
		if gf < bf {
			h += 6.0
		}
	case gf:
		h = ((bf - rf) / delta) + 2.0
	case bf:
		h = ((rf - gf) / delta) + 4.0
	}

	// Convert hue to degrees
	h *= 60.0

	return h, s, v
}

// HSVToRGB converts HSV values to RGB values.
// h (Hue) is in the range [0, 360]
// s (Saturation) is in the range [0, 1]
// v (Value) is in the range [0, 1]
// Returns r, g, b in the range [0, 255]
func HSVToRGB(h, s, v float64) (r, g, b uint8) {
	// Handle grayscale / zero saturation case
	if s <= 0 {
		val := uint8(math.Round(v * 255))
		return val, val, val
	}

	// Clamp hue and calculate the 60-degree sector
	if h >= 360.0 {
		h = 0.0
	}
	h /= 60.0
	i := int(math.Floor(h))
	f := h - float64(i)

	// Calculate intermediate values
	p := v * (1.0 - s)
	q := v * (1.0 - (s * f))
	t := v * (1.0 - (s * (1.0 - f)))

	var rf, gf, bf float64

	// Map intermediate values to RGB based on the sector
	switch i {
	case 0:
		rf, gf, bf = v, t, p
	case 1:
		rf, gf, bf = q, v, p
	case 2:
		rf, gf, bf = p, v, t
	case 3:
		rf, gf, bf = p, q, v
	case 4:
		rf, gf, bf = t, p, v
	default: // case 5:
		rf, gf, bf = v, p, q
	}

	// Scale to [0, 255] and round to nearest integer
	r = uint8(math.Round(rf * 255))
	g = uint8(math.Round(gf * 255))
	b = uint8(math.Round(bf * 255))
	return
}
