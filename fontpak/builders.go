// builder.go
package fontpack

import (
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/jurgen-kluft/go-gx2/bdf"
)

const cSDFBuildBorder = 3
const cDefaultSDFRadius = 4.0
const cDefaultSDFCutoff = 0.25

type Font struct {
	Name     string
	Options  FontOptions
	Glyphs   []Glyph
	CharMap  []uint8
	Ascent   int16
	Descent  int16
	LineGap  int16
	Reserved int16
	FontType FontType
}

type Glyph struct {
	Rune     rune
	AdvanceX int16
	BearingX int16
	BearingY int16
	Width    uint16
	Height   uint16
	Bitmap   []byte
}

func Build(cfg *FontPackCfg) ([]Font, error) {

	fonts := make([]Font, 0, len(cfg.Fonts))

	// build the charmap
	glyphRunes := make([]rune, 0, 256)

	for _, fontCfg := range cfg.Fonts {
		ext := filepath.Ext(fontCfg.File)

		var err error

		options := fontCfg.options()

		numChars := len(fontCfg.Chars)
		if numChars == 0 {
			return nil, fmt.Errorf("font file \"%q\" has an empty char map", fontCfg.File)
		}
		if numChars > cap(glyphRunes) {
			return nil, fmt.Errorf("font file \"%q\" has too many chars in char map (max %d)", fontCfg.File, cap(glyphRunes))
		}
		for range fontCfg.Chars {
			glyphRunes = append(glyphRunes, 0)
		}

		charMap := make([]uint8, 0, 256)

		for i, character := range fontCfg.Chars {
			ascii := character.Address
			glyph := character.Glyph
			if len(character.Address) != 1 {
				return nil, fmt.Errorf("font file \"%q\" has invalid char \"%s\" with glyph \"%s\"", fontCfg.File, ascii, glyph)
			}
			if utf8.RuneCountInString(glyph) != 1 {
				return nil, fmt.Errorf("font file \"%q\" has more than 1 rune in \"%s\" for char \"%s\"", fontCfg.File, glyph, ascii)
			}
			asciiCode := ascii[0]
			for int(asciiCode) >= len(charMap) {
				charMap = append(charMap, 0xFF)
			}
			charMap[asciiCode] = uint8(i)
			glyphRunes[i], _ = utf8.DecodeRuneInString(glyph)
		}

		var font *Font

		switch ext {
		case ".ttf", ".otf":
			font, err = buildTTFFont(
				fontCfg.File,
				options,
				fontCfg.Name,
				glyphRunes,
			)

		case ".bdf":
			font, err = buildBDFFont(
				fontCfg.File,
				options,
				fontCfg.Name,
				glyphRunes,
			)

		default:
			return nil, fmt.Errorf("unsupported font type: %s", ext)
		}

		if err != nil {
			return nil, err
		}

		font.Options = options
		font.CharMap = charMap
		fonts = append(fonts, *font)

	}

	return fonts, nil
}

// buildTTFFont reads a TTF font file and builds a builtFont struct based on the provided character mapping.
func buildTTFFont(fontPath string, ops FontOptions, name string, glyphRunes []rune) (*Font, error) {

	data, err := os.ReadFile(fontPath)
	if err != nil {
		return nil, err
	}

	face, err := newTTFFace(ops, data)
	if err != nil {
		return nil, err
	}
	defer face.Close()

	font := &Font{}
	font.Name = name
	if ops.SDF {
		font.FontType = FontTypeSDF
	}

	font.Ascent, font.Descent, font.LineGap = extractFontMetrics(face)
	font.Glyphs = make([]Glyph, len(glyphRunes))

	for i, r := range glyphRunes {
		glyph, err := buildGlyphTTF(face, r, ops)
		if err != nil {
			font.Glyphs[i] = Glyph{
				Rune:     r,
				AdvanceX: 0,
				BearingX: 0,
				BearingY: 0,
				Width:    0,
				Height:   0,
				Bitmap:   nil,
			}
		} else {
			font.Glyphs[i] = glyph
		}
	}

	return font, nil
}

// buildBDFFont reads a BDF font file and builds a builtFont struct based on the provided character mapping.
func buildBDFFont(fontPath string, ops FontOptions, name string, glyphRunes []rune) (*Font, error) {

	bdfData, err := os.ReadFile(fontPath)
	if err != nil {
		return nil, err
	}

	bdfFont, err := bdf.Parse(bdfData)
	if err != nil {
		return nil, err
	}

	font := &Font{}
	font.Name = name
	if ops.SDF {
		font.FontType = FontTypeSDF
	}

	font.Ascent = int16(bdfFont.Ascent)
	font.Descent = int16(-bdfFont.Descent)
	font.LineGap = 0
	font.Glyphs = make([]Glyph, len(glyphRunes))

	for i, r := range glyphRunes {
		g := bdfFont.CharMap[r]
		if g == nil {
			font.Glyphs[i] = Glyph{
				Rune:     r,
				AdvanceX: 0,
				BearingX: 0,
				BearingY: 0,
				Width:    0,
				Height:   0,
				Bitmap:   nil,
			}
		} else {
			font.Glyphs[i] = buildGlyphBDF(g, ops)
		}
	}

	return font, nil
}

func (g *Glyph) PrintInfo() {
	// Print binary grid that represents the glyph's bitmap data

	width := g.Width
	height := g.Height

	fmt.Printf("Glyph Rune: %s\n", string(g.Rune))
	fmt.Printf("Width: %d, Height: %d\n", width, height)

	fmt.Println("Glyph Bitmap Data (Hex):")
	for y := 0; y < int(height); y++ {
		fmt.Printf("    ")
		for x := 0; x < int(width); x++ {
			byteIndex := y*int(width) + x
			b := g.Bitmap[byteIndex]
			// print double character HEX representation of the byte
			fmt.Printf("%02X ", b)
		}
		fmt.Println()
	}
}

func PrintFontInfo(font *Font) {

	size := len(font.CharMap)        // size of the bitmap data
	for _, bg := range font.Glyphs { // add the size of each glyph's bitmap data
		size += len(bg.Bitmap)
	}
	size += len(font.Glyphs) * 1  // size of the glyph advance X data
	size += len(font.Glyphs) * 2  // size of the glyph bearing data
	size += len(font.Glyphs) * 2  // size of the glyph dimensions data
	size += len(font.Glyphs) * 2  // size of the glyph offset data
	size += len(font.CharMap) * 1 // size of the char map data

	println("Font Info: ", font.Name)
	println("    Glyphs: ", len(font.Glyphs))
	println("    Border: ", font.Options.SDFBorder, " pixels")
	println("    Font Size: ", font.Options.FontSize)
	println("    Ascent: ", font.Ascent)
	println("    Descent: ", font.Descent)
	println("    LineGap: ", font.LineGap)
	println("    FontType: ", font.FontType.String())
	println("    Font Data Size: ", size, " bytes")
	println("        Data: ", size, " bytes")
	println()
	println("    Glyph Average Size: ~", size/len(font.Glyphs), " bytes")
}
