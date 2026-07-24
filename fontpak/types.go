package fontpack

type GlyphBearing struct {
	X int8
	Y int8
}

type GlyphDimensions struct {
	Width  uint8
	Height uint8
}

type FontType uint8

const (
	FontTypeBitmap FontType = iota
	FontTypeSDF
)

// Font holds all glyphs and metrics for a single font.
type Font struct {
	Bitmap        []byte            // Bitmap (SDF or Coverage) data for all glyphs in the font
	GlyphAdvanceX []int8            // Advance X of each glyph
	GlyphBearing  []GlyphBearing    // X and Y bearing of each glyph
	GlyphDims     []GlyphDimensions // Width and height of each glyph
	GlyphOffset   []uint32          // offset into SDF data for each glyph
	Map           [128 - 3]uint8    // maps ASCII code → glyph index, 0xFF = unsupported
	Ascent        int8
	Descent       int8
	LineGap       int8
	FontType      FontType
}

type FontPack struct {
	Fonts []Font
}
