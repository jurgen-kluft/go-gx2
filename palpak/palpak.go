package palpak

import (
	"fmt"

	"github.com/jurgen-kluft/go-gx2/common"
)

type PalPack struct {
	Palettes []common.PaletteRGBA
}

func Build(cfgs *PalPackCfg) (*PalPack, error) {
	palPack := &PalPack{}

	for _, cfg := range cfgs.Palettes {
		if cfg.ImageFile != "" {
			// Load palette from image file
			palette, err := common.BuildPaletteFromImageFile(cfg.ImageFile)
			if err != nil {
				return nil, err
			}
			palPack.Palettes = append(palPack.Palettes, palette)
		} else if cfg.PaletteFile != "" {
			// Load palette from palette file
			paletteCfg, err := LoadPaletteFile(cfg.PaletteFile)
			if err != nil {
				return nil, err
			}
			palette := common.PaletteRGBA{}
			for _, color := range paletteCfg.Colors {
				r := uint8((color >> 24) & 0xFF)
				g := uint8((color >> 16) & 0xFF)
				b := uint8((color >> 8) & 0xFF)
				a := uint8(color & 0xFF)
				palette = append(palette, common.NewColorFromR8G8B8A8(a, r, g, b))
			}
			palPack.Palettes = append(palPack.Palettes, palette)
		} else {
			return nil, fmt.Errorf("palette entry must have either image_file or palette_file")
		}
	}

	return palPack, nil
}

func SizeInBytes(palPack *PalPack) int {
	size := 0
	for _, palette := range palPack.Palettes {
		// Each color is 4 bytes (RGBA)
		size += len(palette) * 4
	}
	return size
}
