package palpak

import (
	"github.com/jurgen-kluft/go-gx2/common"
)

type PalPack struct {
	Palettes []common.PaletteRGBA
}

func Build(cfgs *PalPackCfg) (*PalPack, error) {

	return nil, nil
}

func SizeInBytes(palPack *PalPack) int {
	size := 0
	for _, palette := range palPack.Palettes {
		// Each color is 4 bytes (RGBA)
		size += len(palette) * 4
	}
	return size
}
