package fontpack

import (
	"bytes"
	"encoding/binary"
)

type fileFontContext struct {
	FontCount uint32
	Reserved  uint32
	FontsOff  uint32
}

type fileFont struct {
	GlyphsOff uint32
	CharsOff  uint32
	Map       [256]uint8
	Ascent    int16
	Descent   int16
	LineGap   int16
	Reserved  int16
}

type fileGlyph struct {
	AdvanceX int16
	BearingX int16
	BearingY int16
	Width    uint16
	Height   uint16
}

func WriteFontPack(fonts []BuiltFont) ([]byte, error) {
	var buf bytes.Buffer

	ctx := fileFontContext{
		FontCount: uint32(len(fonts)),
		FontsOff:  uint32(binary.Size(fileFontContext{})),
	}
	binary.Write(&buf, binary.LittleEndian, ctx)

	fontTableOff := buf.Len()
	for range fonts {
		binary.Write(&buf, binary.LittleEndian, fileFont{})
	}

	glyphTableOff := buf.Len()
	var glyphs []fileGlyph
	var bitmaps [][]byte

	for _, f := range fonts {
		for _, g := range f.Glyphs {
			glyphs = append(glyphs, fileGlyph{
				AdvanceX: g.AdvanceX,
				BearingX: g.BearingX,
				BearingY: g.BearingY,
				Width:    g.Width,
				Height:   g.Height,
			})
			bitmaps = append(bitmaps, g.Bitmap)
		}
	}

	for _, g := range glyphs {
		binary.Write(&buf, binary.LittleEndian, g)
	}

	bitmapOffsetOff := buf.Len()
	offset := 0
	for _, b := range bitmaps {
		binary.Write(&buf, binary.LittleEndian, uint32(offset))
		offset += len(b)
	}

	for _, b := range bitmaps {
		buf.Write(b)
	}

	data := buf.Bytes()
	for i, f := range fonts {
		ff := fileFont{
			GlyphsOff: uint32(glyphTableOff),
			CharsOff:  uint32(bitmapOffsetOff),
			Map:       f.CharMap,
			Ascent:    f.Ascent,
			Descent:   f.Descent,
			LineGap:   f.LineGap,
		}
		var tmp bytes.Buffer
		binary.Write(&tmp, binary.LittleEndian, ff)
		copy(
			data[fontTableOff+i*binary.Size(fileFont{}):],
			tmp.Bytes(),
		)
	}

	return data, nil
}
