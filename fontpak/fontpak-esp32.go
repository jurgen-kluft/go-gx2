package fontpack

import (
	"io"

	"github.com/jurgen-kluft/go-datastream/codestream"
)

type GlyphForEsp32 struct {
	AdvanceX int8
	BearingX int8
	BearingY int8
	Width    uint8
	Height   uint8
}

type FontForEsp32 struct {
	Coverage []byte          // All glyphs are packed here
	Glyphs   []GlyphForEsp32 // Array of glyphs
	Offsets  []uint32        // Offsets into Coverage for each glyph
	Map      [256]uint8      // maps ASCII code → glyph index, 0xFF = unsupported
	Bpp      uint16
	Ascent   int16
	Descent  int16
	LineGap  int16
}

func WriteFontForEsp32(w io.Writer, font *Font) error {

	bpp := 4

	totalCoverageSize := 0
	for _, glyph := range font.Glyphs {
		glyphCoverageRowSize := (int(glyph.Width)*8 + (bpp - 1)) / bpp
		glyphCoverageSize := glyphCoverageRowSize * int(glyph.Height)
		totalCoverageSize += glyphCoverageSize
	}

	esp32Font := &FontForEsp32{
		Coverage: make([]byte, 0, totalCoverageSize),
		Glyphs:   make([]GlyphForEsp32, 0, len(font.Glyphs)),
		Offsets:  make([]uint32, 0, len(font.Glyphs)),
		Map:      font.Map,
		Bpp:      4, // Set 4 bits per pixel alpha coverage for ESP32
		Ascent:   font.Ascent,
		Descent:  font.Descent,
		LineGap:  font.LineGap,
	}

	// Convert Font to FontForEsp32
	for _, glyph := range font.Glyphs {
		esp32Glyph := GlyphForEsp32{
			AdvanceX: int8(glyph.AdvanceX),
			BearingX: int8(glyph.BearingX),
			BearingY: int8(glyph.BearingY),
			Width:    uint8(glyph.Width),
			Height:   uint8(glyph.Height),
		}
		esp32Font.Glyphs = append(esp32Font.Glyphs, esp32Glyph)
		esp32Font.Offsets = append(esp32Font.Offsets, uint32(len(esp32Font.Coverage)))

		// Convert the glyph bitmap to match bpp and append to Coverage
		glyphCoverageRowSize := (int(glyph.Width)*8 + (bpp - 1)) / bpp
		glyphCoverage := make([]byte, glyphCoverageRowSize*int(glyph.Height))

		for y := 0; y < int(glyph.Height); y++ {
			bitPos := 0
			byteIndex := (y * glyphCoverageRowSize)
			for x := 0; x < int(glyph.Width); x++ {
				alpha := glyph.Bitmap[y*int(glyph.Width)+x]
				coverage := 0
				switch bpp {
				case 1:
					if alpha > 127 {
						coverage = 1
					}
					bitPos++
				case 2:
					coverage = int(alpha) >> 6 // 0-3
					bitPos += 2
				case 4:
					coverage = int(alpha) >> 4 // 0-15
					bitPos += 4
				default:
					coverage = int(alpha) // Keep as is for other bpp values
					bitPos += 8
				}

				if bitPos == 8 {
					bitPos = 0
					byteIndex++
				}

				glyphCoverage[byteIndex] |= byte(coverage << bitPos)
			}
		}
		esp32Font.Coverage = append(esp32Font.Coverage, glyphCoverage...)

	}

	if err := codestream.WriteToStream(w, esp32Font); err != nil {
		return err
	}

	return nil
}
