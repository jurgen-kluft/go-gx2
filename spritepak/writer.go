package spritepack

import (
	"io"

	"github.com/jurgen-kluft/go-datastream/codestream"
)

type SpritePack struct {
	Sprites  []SpriteEntry
	Count    uint32
	Reserved uint32
}

func WritePack(w io.Writer, sprites []SpriteEntry) error {
	spritePack := SpritePack{
		Sprites:  sprites,
		Count:    uint32(len(sprites)),
		Reserved: 0,
	}

	if err := codestream.WriteToStream(w, spritePack); err != nil {
		return err
	}

	return nil
}
