package fontpack

const cMaximumNumberOfChars = 128 - 4

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
	Data          []byte                       // Bitmap (SDF or Coverage) data for all glyphs in the font
	GlyphAdvanceX []int8                       // Advance X of each glyph
	GlyphBearing  []GlyphBearing               // X and Y bearing of each glyph
	GlyphDims     []GlyphDimensions            // Width and height of each glyph
	GlyphOffset   []uint16                     // Offset = (GlyphOffset[i] * 8) into SDF data for each glyph
	Map           [cMaximumNumberOfChars]uint8 // maps ASCII code → glyph index, 0xFF = unsupported
	Ascent        int8
	Descent       int8
	LineGap       int8
	FontType      FontType
}

type FontPack struct {
	Fonts []Font
}

func PrintFontInfo(font *Font, name string) {
	size := len(font.Data)              // size of the bitmap data
	size += len(font.GlyphAdvanceX) * 1 // int8
	size += len(font.GlyphBearing) * 2  // 2 * x+y
	size += len(font.GlyphDims) * 2     // 2 * width+height
	size += len(font.GlyphOffset) * 2   // uint16
	size += len(font.Map) * 1           // uint8
	size += 4                           // Ascent, Descent, LineGap, FontType

	println("Font Info: ", name)
	println("    Font Size: ", size, " bytes")
	println("    Ascent: ", font.Ascent)
	println("    Descent: ", font.Descent)
	println("    LineGap: ", font.LineGap)
	println("    FontType: ", font.FontType)
	println("    Number of Glyphs: ", len(font.GlyphAdvanceX))
	println("    Glyph Advance X: ", font.GlyphAdvanceX)
	println("    Glyph Bearing: ", font.GlyphBearing)
	println("    Glyph Dimensions: ", font.GlyphDims)
	println("    Glyph Offsets: ", font.GlyphOffset)
	println()
}
