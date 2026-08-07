package fontpack

const cMaxNumChars = 128 - 4

const cSDFBuildBorder = 3
const cDefaultSDFRadius = 4.0
const cDefaultSDFCutoff = 0.25

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

func (ft FontType) String() string {
	switch ft {
	case FontTypeBitmap:
		return "Bitmap"
	case FontTypeSDF:
		return "SDF"
	default:
		return "Unknown"
	}
}

// Font holds all glyphs and metrics for a single font.
type Font struct {
	Data          []byte              // Bitmap (SDF or Coverage) data for all glyphs in the font
	GlyphAdvanceX []int8              // Advance X of each glyph
	GlyphBearing  []GlyphBearing      // X and Y bearing of each glyph
	GlyphDims     []GlyphDimensions   // Width and height of each glyph
	GlyphOffset   []uint16            // Offset = (GlyphOffset[i] * 8) into SDF data for each glyph
	Map           [cMaxNumChars]uint8 // maps ASCII code → glyph index, 0xFF = unsupported
	Ascent        int8
	Descent       int8
	LineGap       int8
	FontType      FontType
}

type FontInfo struct {
	Name    string
	Options FontOptions
}

type FontPack struct {
	Fonts []Font
}
