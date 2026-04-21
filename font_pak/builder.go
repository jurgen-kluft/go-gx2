// builder.go
package fontpack

import (
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf8"

	"go-gx2/bdf"
)

type BuiltFont struct {
	Name    string
	Glyphs  []BuiltGlyph
	CharMap [256]uint8
	Ascent  int16
	Descent int16
	LineGap int16
}

//
// Helpers
//

func parseCharMap(input map[string]string) (map[byte]rune, error) {
	out := make(map[byte]rune, len(input))

	for k, v := range input {
		if len(k) != 1 {
			return nil, fmt.Errorf("char map key %q is not a single character", k)
		}

		ascii := k[0]
		if ascii > 0x7F {
			return nil, fmt.Errorf("char map key %q is not ASCII", k)
		}

		r, size := utf8.DecodeRuneInString(v)
		if r == utf8.RuneError || size == 0 {
			return nil, fmt.Errorf("char map value %q is not a valid rune", v)
		}

		out[ascii] = r
	}

	return out, nil
}

//
// TTF builder
//

func buildTTFFont(fontPath string, options Options, name string, charSpec map[string]string) (*BuiltFont, error) {

	data, err := os.ReadFile(fontPath)
	if err != nil {
		return nil, err
	}

	face, err := NewTTFFace(options, data)
	if err != nil {
		return nil, err
	}
	defer face.Close()

	parsedChars, err := parseCharMap(charSpec)
	if err != nil {
		return nil, err
	}

	var font BuiltFont
	font.Name = name

	// init ASCII map
	for i := 0; i < 256; i++ {
		font.CharMap[i] = 0xFF
	}

	font.Ascent, font.Descent, font.LineGap =
		ExtractFontMetrics(face)

	for ascii, r := range parsedChars {
		glyph, err := BuildGlyphTTF(face, r)
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

// buildBDFFont reads a BDF font file and builds a BuiltFont struct based on the provided character mapping.
func buildBDFFont(fontPath string, name string, charSpec map[string]string) (*BuiltFont, error) {

	file, err := os.Open(fontPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bdfData, err := os.ReadFile(fontPath)
	if err != nil {
		return nil, err
	}

	bdfFont, err := bdf.Parse(bdfData)
	if err != nil {
		return nil, err
	}

	parsedChars, err := parseCharMap(charSpec)
	if err != nil {
		return nil, err
	}

	var font BuiltFont
	font.Name = name

	for i := 0; i < 256; i++ {
		font.CharMap[i] = 0xFF
	}

	font.Ascent = int16(bdfFont.Ascent)
	font.Descent = int16(-bdfFont.Descent)
	font.LineGap = 0

	for ascii, r := range parsedChars {
		g := bdfFont.CharMap[r]
		if g == nil {
			continue
		}

		bg := BuildGlyphBDF(g)
		index := uint8(len(font.Glyphs))
		font.CharMap[ascii] = index
		font.Glyphs = append(font.Glyphs, bg)
	}

	return &font, nil
}

//
// Public entry point
//

func BuildFontsFromConfig(cfg *Config) ([]BuiltFont, error) {
	var out []BuiltFont

	for _, file := range cfg.Files {
		ext := filepath.Ext(file.File)

		for _, f := range file.Fonts {
			var built *BuiltFont
			var err error

			options := Options{
				FontSize: f.Size,
				DPI:      f.DPI,
			}

			switch ext {
			case ".ttf", ".otf":
				built, err = buildTTFFont(
					file.File,
					options,
					f.Name,
					f.Chars,
				)

			case ".bdf":
				built, err = buildBDFFont(
					file.File,
					f.Name,
					f.Chars,
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

	return out, nil
}
