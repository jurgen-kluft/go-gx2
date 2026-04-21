package fontpack

import (
	"bytes"
	"encoding/binary"
)

type fileFontContext struct {
	FontsOff  uint64
	FontCount uint32
	Reserved  uint32
}

func (ctx *fileFontContext) writeTo(buf *bytes.Buffer) {
	binary.Write(buf, binary.LittleEndian, ctx.FontsOff)
	binary.Write(buf, binary.LittleEndian, ctx.FontCount)
	binary.Write(buf, binary.LittleEndian, ctx.Reserved)
}

type fileFont struct {
	GlyphsArrayOffset        uint64
	GlyphsArray              []fileGlyph
	GlyphsBitmapArrayOffset  uint64
	GlyphsBitmapArrayOffsets []uint64
	GlyphsBitmapArray        [][]byte
	Map                      [256]uint8
	Ascent                   int16
	Descent                  int16
	LineGap                  int16
	Reserved                 int16
}

// C Structure
//      struct font_t
//      {
//          glyph_t*   glyphs;    // array of glyphs, indexed by glyph index (not ASCII code)
//          const u8** bitmaps;   // alpha or coverage bitmap
//          u8         map[256];  // maps ASCII character codes to glyph indices in the glyphs array, or 0xFF if the character is not supported
//          i16        ascent;    // distance from baseline to top of font
//          i16        descent;   // distance from baseline to bottom of font (negative value)
//          i16        line_gap;  // distance from bottom of one line to top of next line (can be negative)
//          i16        reserved;  // padding to make sizeof(font_t) a multiple of 8
//      };

func (f *fileFont) writeTo(buf *bytes.Buffer) {
	binary.Write(buf, binary.LittleEndian, f.GlyphsArrayOffset)
	binary.Write(buf, binary.LittleEndian, f.GlyphsBitmapArrayOffset)
	binary.Write(buf, binary.LittleEndian, f.Map)
	binary.Write(buf, binary.LittleEndian, f.Ascent)
	binary.Write(buf, binary.LittleEndian, f.Descent)
	binary.Write(buf, binary.LittleEndian, f.LineGap)
	binary.Write(buf, binary.LittleEndian, f.Reserved)
}

func (f *fileFont) writeArrayOfGlyphs(buf *bytes.Buffer) {
	f.GlyphsArrayOffset = uint64(buf.Len())
	for _, g := range f.GlyphsArray {
		g.writeTo(buf)
	}
}

func (f *fileFont) writeArrayOfBitmapPtrs(buf *bytes.Buffer) {
	f.GlyphsBitmapArrayOffset = uint64(buf.Len())
	for _, offset := range f.GlyphsBitmapArrayOffsets {
		binary.Write(buf, binary.LittleEndian, offset)
	}
}

func (f *fileFont) writeEachBitmap(buf *bytes.Buffer) {
	for i, b := range f.GlyphsBitmapArray {
		f.GlyphsBitmapArrayOffsets[i] = uint64(buf.Len())
		buf.Write(b)
	}
}

type fileGlyph struct {
	AdvanceX int16
	BearingX int16
	BearingY int16
	Width    uint16
	Height   uint16
}

func (g *fileGlyph) writeTo(buf *bytes.Buffer) {
	binary.Write(buf, binary.LittleEndian, g.AdvanceX)
	binary.Write(buf, binary.LittleEndian, g.BearingX)
	binary.Write(buf, binary.LittleEndian, g.BearingY)
	binary.Write(buf, binary.LittleEndian, g.Width)
	binary.Write(buf, binary.LittleEndian, g.Height)
}

func writeFontPack(fonts []builtFont) ([]byte, error) {
	var buf bytes.Buffer

	ctx := fileFontContext{
		FontsOff:  uint64(binary.Size(fileFontContext{})),
		FontCount: uint32(len(fonts)),
		Reserved:  uint32(0),
	}

	ctx.writeTo(&buf)

	// Build the file font array with empty offsets, we'll fill them in later
	fileFontArray := make([]*fileFont, 0, len(fonts))
	for _, f := range fonts {
		fileFont := &fileFont{
			GlyphsArrayOffset:        uint64(0),
			GlyphsArray:              make([]fileGlyph, 0, len(f.Glyphs)),
			GlyphsBitmapArrayOffset:  uint64(0),
			GlyphsBitmapArrayOffsets: make([]uint64, 0, len(f.Glyphs)),
			GlyphsBitmapArray:        make([][]byte, 0, len(f.Glyphs)),
			Map:                      f.CharMap,
			Ascent:                   f.Ascent,
			Descent:                  f.Descent,
			LineGap:                  f.LineGap,
		}

		// Convert built glyphs to file glyphs and collect bitmaps
		for _, g := range f.Glyphs {
			fileFont.GlyphsArray = append(fileFont.GlyphsArray, fileGlyph{
				AdvanceX: g.AdvanceX,
				BearingX: g.BearingX,
				BearingY: g.BearingY,
				Width:    g.Width,
				Height:   g.Height,
			})
			fileFont.GlyphsBitmapArray = append(fileFont.GlyphsBitmapArray, g.Bitmap)
			fileFont.GlyphsBitmapArrayOffsets = append(fileFont.GlyphsBitmapArrayOffsets, uint64(0)) // Placeholder, will be filled in later
		}

		fileFontArray = append(fileFontArray, fileFont)
	}

	for _, fileFont := range fileFontArray {
		fileFont.writeArrayOfGlyphs(&buf)
	}

	for _, fileFont := range fileFontArray {
		fileFont.writeEachBitmap(&buf)
	}

	for _, fileFont := range fileFontArray {
		fileFont.writeArrayOfBitmapPtrs(&buf)
	}

	fontTableOff := buf.Len()
	for _, fileFont := range fileFontArray {
		fileFont.writeTo(&buf)
	}

	storage := buf.Bytes()

	// Write the header and file font table again with the correct offsets
	// No need to write the rest since they do not contain offsets
	rewrite := bytes.NewBuffer(storage[:0])
	ctx.FontsOff = uint64(fontTableOff)
	ctx.writeTo(rewrite)

	return storage, nil
}
