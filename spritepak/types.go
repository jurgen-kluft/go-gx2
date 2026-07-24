package spritepack

import (
	"fmt"
	_ "image/png"
)

type Sprite struct {
	Width       uint16
	Height      uint16
	PixelFormat PixelFormat
	AlphaFormat AlphaFormat
	Reserved    uint16
	PixelData   []byte
	AlphaData   []byte
}

type SpritePack struct {
	Sprites []Sprite
}

// ===== Alpha Format =====
type AlphaFormat uint8

const (
	FMT_ALPHA_A0 AlphaFormat = 0
	FMT_ALPHA_A1 AlphaFormat = 1
	FMT_ALPHA_A2 AlphaFormat = 2
	FMT_ALPHA_A4 AlphaFormat = 4
	FMT_ALPHA_A8 AlphaFormat = 8
)

func AlphaFormatFromString(s string) (AlphaFormat, error) {
	switch s {
	case "A0":
		return FMT_ALPHA_A0, nil
	case "A1":
		return FMT_ALPHA_A1, nil
	case "A2":
		return FMT_ALPHA_A2, nil
	case "A4":
		return FMT_ALPHA_A4, nil
	case "A8":
		return FMT_ALPHA_A8, nil
	}
	return FMT_ALPHA_A0, fmt.Errorf("unsupported alpha format: %s", s)
}

// ===== Pixel Format =====
type PixelFormat uint8

const (
	FMT_PIXEL_RGB565   PixelFormat = 0x01 // RGB565 (16-bit) with no alpha
	FMT_PIXEL_RGBA8888 PixelFormat = 0x02 // RGBA8888 (32-bit)
	FMT_PIXEL_I8       PixelFormat = 0x03 // Indexed 8-bit (with RGBA palette)
)

func PixelFormatFromString(s string) (PixelFormat, error) {
	switch s {
	case "RGB565":
		return FMT_PIXEL_RGB565, nil
	case "RGBA8888":
		return FMT_PIXEL_RGBA8888, nil
	case "I8":
		return FMT_PIXEL_I8, nil
	}
	return 0, fmt.Errorf("unsupported format: %s", s)
}

// ===== Palette Format =====
type PaletteFormat uint8

const (
	FMT_PALETTE_RGBA8888 PaletteFormat = 0x01 // RGBA8888 (32-bit)
	FMT_PALETTE_RGB565   PaletteFormat = 0x02 // RGB565 (16-bit)
)
