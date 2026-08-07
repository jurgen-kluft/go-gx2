// builder.go
package fontpack

import (
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/jurgen-kluft/go-gx2/bdf"
)

type builtFont struct {
	Name          string
	GlyphsOffset  int64
	BitmapsOffset int64
	Glyphs        []builtGlyph
	CharMap       [cMaxNumChars]uint8
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

func Build(cfg *FontPackCfg) ([]Font, []FontInfo, error) {

	infos := make([]FontInfo, 0, len(cfg.Fonts))
	fonts := make([]Font, 0, len(cfg.Fonts))

	// build the charmap
	glyphRunes := [cMaxNumChars]rune{}

	for _, fontCfg := range cfg.Fonts {
		ext := filepath.Ext(fontCfg.File)

		var err error

		options := fontCfg.options()

		numChars := len(fontCfg.Chars)
		if numChars == 0 {
			return nil, nil, fmt.Errorf("font file \"%q\" has an empty char map", fontCfg.File)
		}
		if numChars > cMaxNumChars {
			return nil, nil, fmt.Errorf("font file \"%q\" has too many chars in char map (max %d)", fontCfg.File, cMaxNumChars)
		}
		for i := range glyphRunes {
			glyphRunes[i] = 0
		}
		for _, character := range fontCfg.Chars {
			ascii := character.Address
			glyph := character.Glyph
			if len(character.Address) != 1 {
				return nil, nil, fmt.Errorf("font file \"%q\" has invalid char \"%s\" with glyph \"%s\"", fontCfg.File, ascii, glyph)
			}
			if utf8.RuneCountInString(glyph) != 1 {
				return nil, nil, fmt.Errorf("font file \"%q\" has more than 1 rune in \"%s\" for char \"%s\"", fontCfg.File, glyph, ascii)
			}
			ascii_index := int(ascii[0])
			glyphRunes[ascii_index], _ = utf8.DecodeRuneInString(glyph)
		}

		var bf *builtFont

		switch ext {
		case ".ttf", ".otf":
			bf, err = buildTTFFont(
				fontCfg.File,
				options,
				fontCfg.Name,
				glyphRunes,
			)

		case ".bdf":
			bf, err = buildBDFFont(
				fontCfg.File,
				options,
				fontCfg.Name,
				glyphRunes,
			)

		default:
			return nil, nil, fmt.Errorf("unsupported font type: %s", ext)
		}

		if err != nil {
			return nil, nil, err
		}

		font := Font{
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
		infos = append(infos, FontInfo{
			Name:    fontCfg.Name,
			Options: fontCfg.options(),
		})
	}

	return fonts, infos, nil
}

// buildTTFFont reads a TTF font file and builds a builtFont struct based on the provided character mapping.
func buildTTFFont(fontPath string, ops FontOptions, name string, glyphRunes [cMaxNumChars]rune) (*builtFont, error) {

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

	font.Ascent, font.Descent, font.LineGap = extractFontMetrics(face)

	for ascii, r := range glyphRunes {
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
func buildBDFFont(fontPath string, ops FontOptions, name string, glyphRunes [cMaxNumChars]rune) (*builtFont, error) {

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

	for ascii, r := range glyphRunes {
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
