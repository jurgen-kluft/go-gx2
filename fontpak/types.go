package fontpack

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

type FontPack struct {
	Fonts []Font
}
