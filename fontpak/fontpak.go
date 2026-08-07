package fontpack

import (
	"fmt"
	"io"

	"github.com/jurgen-kluft/go-datastream/codestream"
)

// ReadPack reads a font pack from the provided reader and returns a slice of Font objects.
func ReadPack(r io.Reader) ([]Font, error) {
	fontPack := &FontPack{}
	if err := codestream.ReadFromStream(r, fontPack); err != nil {
		return nil, err
	}
	return fontPack.Fonts, nil
}

// WritePack writes the provided fonts and infos to the provided writer as a FontPack.
func WritePack(w io.Writer, fonts []Font, infos []FontInfo) error {

	fmt.Println("Writing font pack...")

	for i, font := range fonts {
		PrintFontInfo(&font, infos[i])
	}

	fontPack := &FontPack{
		Fonts: fonts,
	}
	if err := codestream.WriteToStream(w, fontPack); err != nil {
		return err
	}
	return nil
}

func PrintFontInfo(font *Font, info FontInfo) {
	size := len(font.Data)              // size of the bitmap data
	size += len(font.GlyphAdvanceX) * 1 // int8
	size += len(font.GlyphBearing) * 2  // 2 * x+y
	size += len(font.GlyphDims) * 2     // 2 * width+height
	size += len(font.GlyphOffset) * 2   // uint16
	size += len(font.Map) * 1           // uint8
	size += 4                           // Ascent, Descent, LineGap, FontType

	println("Font Info: ", info.Name)
	println("    Glyphs: ", len(font.GlyphAdvanceX))
	println("    Border: ", info.Options.SDFFinalBorder, " pixels")
	println("    Font Size: ", info.Options.FontSize)
	println("    Ascent: ", font.Ascent)
	println("    Descent: ", font.Descent)
	println("    LineGap: ", font.LineGap)
	println("    FontType: ", font.FontType.String())
	println("    Font Data Size: ", size, " bytes")
	println("        Data: ", len(font.Data), " bytes")
	println("        Glyph Advance X: ", len(font.GlyphAdvanceX)*1, " bytes")
	println("        Glyph Bearing: ", len(font.GlyphBearing)*2, " bytes")
	println("        Glyph Dimensions: ", len(font.GlyphDims)*2, " bytes")
	println("        Glyph Offset: ", len(font.GlyphOffset)*2, " bytes")
	println("        Char Map: ", len(font.Map)*1, " bytes")
	println()
	println("    Glyph Average Size: ~", len(font.Data)/len(font.GlyphAdvanceX), " bytes")
}
