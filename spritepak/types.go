package spritepack

import (
	"github.com/jurgen-kluft/go-gx2/common"
)

type SpritePack struct {
	Sprites []Sprite
}

type Sprite struct {
	Width        uint16
	Height       uint16
	PixelFormat  common.PixelFormat
	AlphaFormat  common.AlphaFormat
	Reserved     uint8
	PaletteIndex uint8
	PixelData    []byte
	AlphaData    []byte
}
