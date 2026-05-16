package fontpack

import (
	"io"

	"github.com/jurgen-kluft/go-datastream/codestream"
)

// Glyph holds the decoded data for a single glyph.
type Glyph struct {
	AdvanceX int16
	BearingX int16
	BearingY int16
	Width    uint16
	Height   uint16
	Bitmap   []byte // Width × Height alpha/coverage bytes
}

// Font holds all glyphs and metrics for a single font.
type Font struct {
	Glyphs  []Glyph
	Map     [256]uint8 // maps ASCII code → glyph index, 0xFF = unsupported
	Ascent  int16
	Descent int16
	LineGap int16
}

// FontPack holds all fonts loaded from a fontpak file.
type FontPack struct {
	Fonts []Font
}

func ReadPack(r io.Reader) (*FontPack, error) {
	fontPack := &FontPack{}
	if err := codestream.ReadFromStream(r, fontPack); err != nil {
		return nil, err
	}
	return fontPack, nil
}
