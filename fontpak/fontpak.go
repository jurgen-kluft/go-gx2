package fontpack

import (
	"fmt"
	"io"
	"slices"

	"github.com/jurgen-kluft/go-datastream/codestream"
)

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
type BinaryFont struct {
	Data          []byte            // Bitmap (SDF or Coverage) data for all glyphs in the font
	GlyphAdvanceX []int8            // Advance X of each glyph
	GlyphBearing  []GlyphBearing    // X and Y bearing of each glyph
	GlyphDims     []GlyphDimensions // Width and height of each glyph
	GlyphOffset   []uint16          // Offset = (GlyphOffset[i] * 8) into SDF data for each glyph
	Map           []uint8           // maps ASCII code → glyph index, 0xFF = unsupported
	Ascent        int8
	Descent       int8
	LineGap       int8
	FontType      FontType
}

type BinaryFontPack struct {
	Fonts []BinaryFont
}

// ReadPack reads a font pack from the provided reader and returns a slice of Font objects.
func ReadPack(r io.Reader) ([]BinaryFont, error) {
	fontPack := &BinaryFontPack{}

	options := codestream.NewOptions()
	stream := codestream.NewCodeStream(options)
	stream.ReadStream(r, fontPack)
	if stream.HasErrors() {
		stream.Report()
		return nil, fmt.Errorf("codestream: failed to read font pack")
	}
	return fontPack.Fonts, nil
}

// WritePack writes the provided fonts and infos to the provided writer as a FontPack.
func WritePack(w io.Writer, fonts []Font) error {

	fmt.Println("Writing font pack...")

	binaryFonts := make([]BinaryFont, 0, len(fonts))
	for _, bf := range fonts {
		PrintFontInfo(&bf)
		binaryFont := bf.ConvertToBinaryFont()
		binaryFonts = append(binaryFonts, binaryFont)
	}

	fontPack := &BinaryFontPack{
		Fonts: binaryFonts,
	}

	options := codestream.NewOptions()
	stream := codestream.NewCodeStream(options)
	stream.WriteStream(w, fontPack)

	if stream.HasErrors() {
		stream.Report()
		return fmt.Errorf("codestream: failed to write font pack")
	}
	return nil
}

func (bf *BinaryFont) ConvertToFont() Font {
	font := Font{
		Ascent:   int16(bf.Ascent),
		Descent:  int16(bf.Descent),
		LineGap:  int16(bf.LineGap),
		CharMap:  slices.Clone(bf.Map),
		FontType: bf.FontType,
		Glyphs:   make([]Glyph, 0, len(bf.GlyphAdvanceX)),
	}

	for i := range bf.GlyphAdvanceX {
		offset := int(bf.GlyphOffset[i]) * 8
		width := int(bf.GlyphDims[i].Width)
		height := int(bf.GlyphDims[i].Height)
		bitmapSize := (width * height) / 8

		glyph := Glyph{
			AdvanceX: int16(bf.GlyphAdvanceX[i]),
			BearingX: int16(bf.GlyphBearing[i].X),
			BearingY: int16(bf.GlyphBearing[i].Y),
			Width:    uint16(width),
			Height:   uint16(height),
			Bitmap:   slices.Clone(bf.Data[offset : offset+bitmapSize]),
		}
		font.Glyphs = append(font.Glyphs, glyph)
	}

	return font
}

func (bf *Font) ConvertToBinaryFont() BinaryFont {
	binaryFont := BinaryFont{
		Ascent:        int8(bf.Ascent),
		Descent:       int8(bf.Descent),
		LineGap:       int8(bf.LineGap),
		Map:           bf.CharMap,
		FontType:      bf.FontType,
		GlyphAdvanceX: make([]int8, 0, len(bf.Glyphs)),
		GlyphBearing:  make([]GlyphBearing, 0, len(bf.Glyphs)),
		GlyphDims:     make([]GlyphDimensions, 0, len(bf.Glyphs)),
		GlyphOffset:   make([]uint16, 0, len(bf.Glyphs)),
		Data:          make([]byte, 0, 65536),
	}

	for _, bg := range bf.Glyphs {
		binaryFont.GlyphAdvanceX = append(binaryFont.GlyphAdvanceX, int8(bg.AdvanceX))
		binaryFont.GlyphBearing = append(binaryFont.GlyphBearing, GlyphBearing{
			X: int8(bg.BearingX),
			Y: int8(bg.BearingY),
		})
		binaryFont.GlyphDims = append(binaryFont.GlyphDims, GlyphDimensions{
			Width:  uint8(bg.Width),
			Height: uint8(bg.Height),
		})

		offset := uint16(len(binaryFont.Data) >> 3) // Each glyph's offset is in units of 8 bytes
		binaryFont.GlyphOffset = append(binaryFont.GlyphOffset, offset)
		binaryFont.Data = append(binaryFont.Data, bg.Bitmap...)

		// align bitmap data to 8 bytes for each glyph
		for len(binaryFont.Data)&7 != 0 {
			binaryFont.Data = append(binaryFont.Data, 0)
		}
	}

	return binaryFont
}
