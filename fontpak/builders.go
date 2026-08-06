// builder.go
package fontpack

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jurgen-kluft/go-gx2/bdf"
)

type builtFont struct {
	Name          string
	GlyphsOffset  int64
	BitmapsOffset int64
	Glyphs        []builtGlyph
	CharMap       [cMaximumNumberOfChars]uint8
	Ascent        int16
	Descent       int16
	LineGap       int16
	Reserved      int16
	FontType      FontType
}

type builtGlyph struct {
	Rune     rune
	AdvanceX int16
	BearingX int16
	BearingY int16
	Width    uint16
	Height   uint16
	Bitmap   []byte
}

func Build(cfg *FontPackCfg) ([]Font, []string, error) {
	var out []builtFont

	// build the charmap
	parsedChars := [cMaximumNumberOfChars]rune{}

	for _, font := range cfg.Fonts {
		ext := filepath.Ext(font.File)

		var built *builtFont
		var err error

		options := font.options()

		numChars := len(font.Chars)
		if numChars == 0 {
			return nil, nil, fmt.Errorf("font file %q has an empty char map", font.File)
		}
		if numChars > cMaximumNumberOfChars {
			return nil, nil, fmt.Errorf("font file %q has too many chars in char map (max %d)", font.File, cMaximumNumberOfChars)
		}
		for i := range parsedChars {
			parsedChars[i] = 0
		}
		for _, character := range font.Chars {
			ascii := character.Address
			glyph := character.Glyph
			if len(character.Address) != 1 {
				return nil, nil, fmt.Errorf("font file %q has invalid char %s with glyph %s", font.File, ascii, glyph)
			}
			if len(glyph) != 1 {
				return nil, nil, fmt.Errorf("font file %q has invalid char %s with glyph %s", font.File, ascii, glyph)
			}
			ascii_index := int(ascii[0])
			parsedChars[ascii_index] = rune(glyph[0])
		}

		switch ext {
		case ".ttf", ".otf":
			built, err = buildTTFFont(
				font.File,
				options,
				font.Name,
				parsedChars,
			)

		case ".bdf":
			built, err = buildBDFFont(
				font.File,
				options,
				font.Name,
				parsedChars,
			)

		default:
			return nil, nil, fmt.Errorf("unsupported font type: %s", ext)
		}

		if err != nil {
			return nil, nil, err
		}

		out = append(out, *built)
	}

	// Conver the builtFont structs to Font structs, which is the format expected by the font pack reader and writer.
	names := make([]string, len(out))
	fonts := make([]Font, len(out))
	for _, bf := range out {
		var font Font
		font.Ascent = int8(bf.Ascent)
		font.Descent = int8(bf.Descent)
		font.LineGap = int8(bf.LineGap)
		font.Map = bf.CharMap
		font.FontType = bf.FontType
		font.Data = make([]byte, 0)

		names = append(names, bf.Name)

		for _, bg := range bf.Glyphs {
			font.GlyphAdvanceX = append(font.GlyphAdvanceX, int8(bg.AdvanceX))
			font.GlyphBearing = append(font.GlyphBearing, GlyphBearing{
				X: int8(bg.BearingX),
				Y: int8(bg.BearingY),
			})
			font.GlyphDims = append(font.GlyphDims, GlyphDimensions{
				Width:  uint8(bg.Width),
				Height: uint8(bg.Height),
			})

			offset := uint16(len(font.Data) >> 3) // Each glyph's offset is in units of 8 bytes
			font.GlyphOffset = append(font.GlyphOffset, offset)
			font.Data = append(font.Data, bg.Bitmap...)

			// align bitmap data to 8 bytes for each glyph
			for len(font.Data)&7 != 0 {
				font.Data = append(font.Data, 0)
			}
		}

		fonts = append(fonts, font)
	}

	return fonts, names, nil
}

// buildTTFFont reads a TTF font file and builds a builtFont struct based on the provided character mapping.
func buildTTFFont(fontPath string, ops Options, name string, parsedChars [cMaximumNumberOfChars]rune) (*builtFont, error) {

	data, err := os.ReadFile(fontPath)
	if err != nil {
		return nil, err
	}

	face, err := newTTFFace(ops, data)
	if err != nil {
		return nil, err
	}
	defer face.Close()

	var font builtFont
	font.Name = name
	if ops.SDF {
		font.FontType = FontTypeSDF
	}

	for i := 0; i < len(font.CharMap); i++ {
		font.CharMap[i] = 0xFF
	}

	font.Ascent, font.Descent, font.LineGap =
		extractFontMetrics(face)

	for ascii, r := range parsedChars {
		if r == 0 {
			// skip chars that were not set in the char map
			continue
		}

		glyph, err := buildGlyphTTF(face, r, ops)
		if err != nil {
			return nil, err
		}
		if glyph == nil {
			continue
		}

		index := uint8(len(font.Glyphs))
		font.CharMap[ascii] = index
		font.Glyphs = append(font.Glyphs, *glyph)
	}

	return &font, nil
}

// buildBDFFont reads a BDF font file and builds a builtFont struct based on the provided character mapping.
func buildBDFFont(fontPath string, ops Options, name string, parsedChars [cMaximumNumberOfChars]rune) (*builtFont, error) {

	bdfData, err := os.ReadFile(fontPath)
	if err != nil {
		return nil, err
	}

	bdfFont, err := bdf.Parse(bdfData)
	if err != nil {
		return nil, err
	}

	var font builtFont
	font.Name = name
	if ops.SDF {
		font.FontType = FontTypeSDF
	}

	for i := 0; i < len(font.CharMap); i++ {
		font.CharMap[i] = 0xFF
	}

	font.Ascent = int16(bdfFont.Ascent)
	font.Descent = int16(-bdfFont.Descent)
	font.LineGap = 0

	for ascii, r := range parsedChars {
		if r == 0 {
			// skip chars that were not set in the char map
			continue
		}

		g := bdfFont.CharMap[r]
		if g == nil {
			continue
		}

		bg := buildGlyphBDF(g, ops)
		index := uint8(len(font.Glyphs))
		font.CharMap[ascii] = index
		font.Glyphs = append(font.Glyphs, bg)
	}

	return &font, nil
}
