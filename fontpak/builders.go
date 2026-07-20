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
	CharMap       [256]uint8
	Ascent        int16
	Descent       int16
	LineGap       int16
	Reserved      int16
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

// buildTTFFont reads a TTF font file and builds a builtFont struct based on the provided character mapping.
func buildTTFFont(fontPath string, ops Options, name string, parsedChars [256]rune) (*builtFont, error) {

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

	// init ASCII map
	for i := 0; i < 256; i++ {
		font.CharMap[i] = 0xFF
	}

	font.Ascent, font.Descent, font.LineGap =
		extractFontMetrics(face)

	for ascii, r := range parsedChars {
		if r == 0 {
			// skip chars that were not set in the char map
			continue
		}

		glyph, err := buildGlyphTTF(face, r)
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
func buildBDFFont(fontPath string, name string, parsedChars [256]rune) (*builtFont, error) {

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

	for i := 0; i < 256; i++ {
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

		bg := buildGlyphBDF(g)
		index := uint8(len(font.Glyphs))
		font.CharMap[ascii] = index
		font.Glyphs = append(font.Glyphs, bg)
	}

	return &font, nil
}

func Build(cfg FontPackCfg) (*FontPack, error) {
	var out []builtFont

	// build the charmap
	parsedChars := [256]rune{}

	for _, file := range cfg.Files {
		ext := filepath.Ext(file.File)

		for _, f := range file.Fonts {
			var built *builtFont
			var err error

			options := Options{
				FontSize: f.Size,
			}

			numChars := len(f.Chars)
			if numChars == 0 {
				return nil, fmt.Errorf("font file %q has an empty char map", file.File)
			}
			if numChars > 255 {
				return nil, fmt.Errorf("font file %q has too many chars in char map (max 255)", file.File)
			}
			for i := range parsedChars {
				parsedChars[i] = 0
			}
			for i, ascii := range f.Chars {
				glyph := f.Glyphs[i]
				if len(ascii) != 1 {
					return nil, fmt.Errorf("font file %q has invalid char %s with glyph %s", file.File, ascii, glyph)
				}
				if len(glyph) != 1 {
					return nil, fmt.Errorf("font file %q has invalid char %s with glyph %s", file.File, ascii, glyph)
				}
				ascii_index := int(ascii[0])
				parsedChars[ascii_index] = rune(glyph[0])
			}

			switch ext {
			case ".ttf", ".otf":
				built, err = buildTTFFont(
					file.File,
					options,
					f.Name,
					parsedChars,
				)

			case ".bdf":
				built, err = buildBDFFont(
					file.File,
					f.Name,
					parsedChars,
				)

			default:
				return nil, fmt.Errorf("unsupported font type: %s", ext)
			}

			if err != nil {
				return nil, err
			}

			out = append(out, *built)
		}
	}

	// Conver the builtFont structs to Font structs, which is the format expected by the font pack reader and writer.
	var result []Font
	for _, bf := range out {
		var font Font
		font.Ascent = bf.Ascent
		font.Descent = bf.Descent
		font.LineGap = bf.LineGap
		font.Map = bf.CharMap

		for _, bg := range bf.Glyphs {
			var glyph Glyph
			glyph.AdvanceX = bg.AdvanceX
			glyph.BearingX = bg.BearingX
			glyph.BearingY = bg.BearingY
			glyph.Width = bg.Width
			glyph.Height = bg.Height
			glyph.Bitmap = bg.Bitmap
			font.Glyphs = append(font.Glyphs, glyph)
		}

		result = append(result, font)
	}

	return &FontPack{
		Fonts: result,
	}, nil
}
