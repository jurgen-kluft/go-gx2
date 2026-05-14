package fontpack

import (
	"encoding/binary"
	"io"
	"os"
)

// Glyph holds the decoded data for a single glyph.
type Glyph struct {
	AdvanceX int16
	BearingX int16
	BearingY int16
	Width    uint16
	Height   uint16
	Bitmap   []byte // Width × Height alpha/coverage bytes
}

// Font holds all glyphs and metrics for a single font.
type Font struct {
	Glyphs  []Glyph
	Map     [256]uint8 // maps ASCII code → glyph index, 0xFF = unsupported
	Ascent  int16
	Descent int16
	LineGap int16
}

// FontPack holds all fonts loaded from a fontpak file.
type FontPack struct {
	Fonts []Font
}

func readPack(filePath string) (*FontPack, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// --- Header ---
	var fontsOff uint64
	var fontCount uint32
	var reserved uint32
	if err := binary.Read(f, binary.LittleEndian, &fontsOff); err != nil {
		return nil, err
	}
	if err := binary.Read(f, binary.LittleEndian, &fontCount); err != nil {
		return nil, err
	}
	if err := binary.Read(f, binary.LittleEndian, &reserved); err != nil {
		return nil, err
	}

	// --- Font table ---
	if _, err := f.Seek(int64(fontsOff), io.SeekStart); err != nil {
		return nil, err
	}

	type rawFont struct {
		GlyphsArrayOffset       uint64
		GlyphsBitmapArrayOffset uint64
		Map                     [256]uint8
		Ascent                  int16
		Descent                 int16
		LineGap                 int16
		Reserved                int16
	}

	rawFonts := make([]rawFont, fontCount)
	for i := range rawFonts {
		r := &rawFonts[i]
		if err := binary.Read(f, binary.LittleEndian, &r.GlyphsArrayOffset); err != nil {
			return nil, err
		}
		if err := binary.Read(f, binary.LittleEndian, &r.GlyphsBitmapArrayOffset); err != nil {
			return nil, err
		}
		if err := binary.Read(f, binary.LittleEndian, &r.Map); err != nil {
			return nil, err
		}
		if err := binary.Read(f, binary.LittleEndian, &r.Ascent); err != nil {
			return nil, err
		}
		if err := binary.Read(f, binary.LittleEndian, &r.Descent); err != nil {
			return nil, err
		}
		if err := binary.Read(f, binary.LittleEndian, &r.LineGap); err != nil {
			return nil, err
		}
		if err := binary.Read(f, binary.LittleEndian, &r.Reserved); err != nil {
			return nil, err
		}
	}

	// --- Build Font slice ---
	fonts := make([]Font, fontCount)
	for i, r := range rawFonts {
		font := &fonts[i]
		font.Map = r.Map
		font.Ascent = r.Ascent
		font.Descent = r.Descent
		font.LineGap = r.LineGap

		// Determine glyph count from the highest valid index in Map.
		glyphCount := 0
		for _, idx := range r.Map {
			if idx != 0xFF && int(idx)+1 > glyphCount {
				glyphCount = int(idx) + 1
			}
		}
		if glyphCount == 0 {
			continue
		}

		// Read glyph structs.
		if _, err := f.Seek(int64(r.GlyphsArrayOffset), io.SeekStart); err != nil {
			return nil, err
		}
		glyphs := make([]Glyph, glyphCount)
		for j := range glyphs {
			g := &glyphs[j]
			if err := binary.Read(f, binary.LittleEndian, &g.AdvanceX); err != nil {
				return nil, err
			}
			if err := binary.Read(f, binary.LittleEndian, &g.BearingX); err != nil {
				return nil, err
			}
			if err := binary.Read(f, binary.LittleEndian, &g.BearingY); err != nil {
				return nil, err
			}
			if err := binary.Read(f, binary.LittleEndian, &g.Width); err != nil {
				return nil, err
			}
			if err := binary.Read(f, binary.LittleEndian, &g.Height); err != nil {
				return nil, err
			}
		}

		// Read bitmap pointer array.
		if _, err := f.Seek(int64(r.GlyphsBitmapArrayOffset), io.SeekStart); err != nil {
			return nil, err
		}
		bitmapOffsets := make([]uint64, glyphCount)
		if err := binary.Read(f, binary.LittleEndian, bitmapOffsets); err != nil {
			return nil, err
		}

		// Read each bitmap.
		for j := range glyphs {
			size := int(glyphs[j].Width) * int(glyphs[j].Height)
			if size == 0 {
				continue
			}
			glyphs[j].Bitmap = make([]byte, size)
			if _, err := f.Seek(int64(bitmapOffsets[j]), io.SeekStart); err != nil {
				return nil, err
			}
			if _, err := io.ReadFull(f, glyphs[j].Bitmap); err != nil {
				return nil, err
			}
		}

		font.Glyphs = glyphs
	}

	return &FontPack{Fonts: fonts}, nil
}
