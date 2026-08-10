package palpak

import (
	"fmt"
	"io"

	"github.com/jurgen-kluft/go-gx2/common"
)

type PalPack struct {
	PaletteColorRGBA   []common.PaletteRGBA
	PaletteColorRGB565 []common.PaletteRGB565
}

func Build(cfgs *PalPackCfg) (*PalPack, error) {
	palPack := &PalPack{
		PaletteColorRGBA:   make([]common.PaletteRGBA, 0, 8),
		PaletteColorRGB565: make([]common.PaletteRGB565, 0, 8),
	}

	var ok bool
	for _, cfg := range cfgs.Palettes {
		palette := common.PaletteRGBA{}
		if cfg.ImageFile != "" {
			// Load palette from image file
			palette, _, ok = common.BuildPaletteFromImageFile(cfg.ImageFile)
			if !ok {
				return nil, fmt.Errorf("failed to build palette from image file: %s", cfg.ImageFile)
			}
		} else if cfg.PaletteFile != "" {
			// Load palette from palette file
			paletteCfg, err := LoadPaletteFile(cfg.PaletteFile)
			if err != nil {
				return nil, err
			}
			for _, color := range paletteCfg.Colors {
				r := uint8((color >> 24) & 0xFF)
				g := uint8((color >> 16) & 0xFF)
				b := uint8((color >> 8) & 0xFF)
				a := uint8(color & 0xFF)
				palette = append(palette, common.NewColorFromR8G8B8A8(a, r, g, b))
			}
		} else {
			return nil, fmt.Errorf("palette entry must have either image_file or palette_file")
		}

		if cfg.PaletteDepth == 16 {
			// Convert RGBA palette to RGB565
			rgb565Palette := make(common.PaletteRGB565, 0, len(palette))
			for _, color := range palette {
				rgb565Color := common.NewColorRGB565FromColorRGBA(color)
				rgb565Palette = append(rgb565Palette, rgb565Color)
			}

			// Use mapping to obtain palette index
			if index, ok := cfgs.Mapping[cfg.Name]; ok {
				// Extend the slice to accommodate the index
				for len(palPack.PaletteColorRGB565) <= index {
					palPack.PaletteColorRGB565 = append(palPack.PaletteColorRGB565, nil)
				}
				palPack.PaletteColorRGB565[index] = rgb565Palette
			}
		} else if cfg.PaletteDepth == 32 || cfg.PaletteDepth == 0 {
			// Store RGBA palette
			if index, ok := cfgs.Mapping[cfg.Name]; ok {
				// Extend the slice to accommodate the index
				for len(palPack.PaletteColorRGBA) <= index {
					palPack.PaletteColorRGBA = append(palPack.PaletteColorRGBA, nil)
				}
				palPack.PaletteColorRGBA[index] = palette
			}
		}
	}

	return palPack, nil
}

func (palPack *PalPack) MemSize() int {
	size := 0

	for _, palette := range palPack.PaletteColorRGBA {
		// Each color is 4 bytes (RGBA)
		size += len(palette) * 4
	}

	for _, palette := range palPack.PaletteColorRGB565 {
		// Each color is 2 bytes (RGB565)
		size += len(palette) * 2
	}
	return size
}

// 8888888b.          888      8888888b.                   888           888888b.   d8b
// 888   Y88b         888      888   Y88b                  888           888  "88b  Y8P
// 888    888         888      888    888                  888           888  .88P
// 888   d88P 8888b.  888      888   d88P 8888b.   .d8888b 888  888      8888888K.  888 88888b.   8888b.  888d888 888  888
// 8888888P"     "88b 888      8888888P"     "88b d88P"    888 .88P      888  "Y88b 888 888 "88b     "88b 888P"   888  888
// 888       .d888888 888      888       .d888888 888      888888K       888    888 888 888  888 .d888888 888     888  888
// 888       888  888 888      888       888  888 Y88b.    888 "88b      888   d88P 888 888  888 888  888 888     Y88b 888
// 888       "Y888888 888      888       "Y888888  "Y8888P 888  888      8888888P"  888 888  888 "Y888888 888      "Y88888

type BinaryPalette struct {
	Type uint8
	Data []byte
}

type BinaryPalPack struct {
	Palettes []BinaryPalette
}

func (pp *PalPack) Convert() (*BinaryPalPack, error) {
	// Implement
	return nil, nil
}

func (bpp *BinaryPalPack) Convert() (*PalPack, error) {
	// Implement
	return nil, nil
}

func (pp *BinaryPalPack) WritePack(w io.Writer) error {
	// Implement
	return nil
}

func ReadPack(r io.Reader) (*BinaryPalPack, error) {
	// Implement
	return nil, nil
}
