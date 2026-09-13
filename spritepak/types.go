package spritepack

import (
	"github.com/jurgen-kluft/go-gx2/common"
)

type SpritePack struct {
	Version uint32
	Sprites []Sprite
}

type Sprite struct {
	Width        uint16
	Height       uint16
	PixelFormat  common.PixelFormat
	AlphaFormat  common.AlphaFormat
	PaletteIndex uint8
	Reserved     uint8
	PixelData    []byte
	AlphaData    []byte
}
