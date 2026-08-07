package spritepack

import (
	"fmt"
	_ "image/png"
)

type SpritePack struct {
	Sprites []Sprite
}

type Sprite struct {
	Width       uint16
	Height      uint16
	PixelFormat PixelFormat
	AlphaFormat AlphaFormat
	Reserved    uint16
	PixelData   []byte
	AlphaData   []byte
}

type AlphaFormat uint8

const (
	FMT_ALPHA_NONE AlphaFormat = 0
	FMT_ALPHA_MASK AlphaFormat = 1 // 1-bit alpha or mask
	FMT_ALPHA_A2   AlphaFormat = 2
	FMT_ALPHA_A4   AlphaFormat = 4
	FMT_ALPHA_A8   AlphaFormat = 8
)

type PixelFormat uint8

const (
	FMT_PIXEL_RGB565 PixelFormat = 0x01 // RGB565 (16-bit) with no alpha
	FMT_PIXEL_RGB888 PixelFormat = 0x02 // RGB888 (24-bit)
	FMT_PIXEL_I8     PixelFormat = 0x03 // Indexed 8-bit
)

type PaletteFormat uint8

const (
	FMT_PALETTE_RGB888 PaletteFormat = 0x01 // RGBB888 (24-bit)
	FMT_PALETTE_RGB565 PaletteFormat = 0x02 // RGB565 (16-bit)
)

type Rect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

func AlphaFormatFromInt(s int) (AlphaFormat, error) {
	switch s {
	case 0:
		return FMT_ALPHA_NONE, nil
	case 1:
		return FMT_ALPHA_MASK, nil
	case 2:
		return FMT_ALPHA_A2, nil
	case 4:
		return FMT_ALPHA_A4, nil
	case 8:
		return FMT_ALPHA_A8, nil
	}
	return FMT_ALPHA_NONE, fmt.Errorf("unsupported alpha format: %d", s)
}

func AlphaFormatFromString(s string) (AlphaFormat, error) {
	switch s {
	case "A0", "NONE":
		return FMT_ALPHA_NONE, nil
	case "A1", "MASK":
		return FMT_ALPHA_MASK, nil
	case "A2":
		return FMT_ALPHA_A2, nil
	case "A4":
		return FMT_ALPHA_A4, nil
	case "A8":
		return FMT_ALPHA_A8, nil
	}
	return FMT_ALPHA_NONE, fmt.Errorf("unsupported alpha format: %s", s)
}

func PixelFormatFromString(s string) (PixelFormat, error) {
	switch s {
	case "RGB565":
		return FMT_PIXEL_RGB565, nil
	case "RGBA8888":
		return FMT_PIXEL_RGB888, nil
	case "I8":
		return FMT_PIXEL_I8, nil
	}
	return 0, fmt.Errorf("unsupported format: %s", s)
}
