package spritepack

import (
	"io"

	"github.com/jurgen-kluft/go-datastream/codestream"
)

// ReadPack reads a spritepak file and returns a SpritePack.
func ReadPack(r io.Reader) (*SpritePack, error) {
	spritePack := SpritePack{
		Sprites:  []SpriteEntry{},
		Count:    0,
		Reserved: 0,
	}

	if err := codestream.ReadFromStream(r, &spritePack); err != nil {
		return nil, err
	}

	return &spritePack, nil
}
