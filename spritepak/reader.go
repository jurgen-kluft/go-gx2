package spritepak

import (
	"encoding/binary"
	"io"
	"os"
)

// Sprite holds the decoded data for a single sprite.
// Exactly one of Indexed, RGB565, or RGBA8888 is populated, depending on Format.
type Sprite struct {
	Width  uint16
	Height uint16
	Format uint16

	// FMT_I8: one byte index per pixel
	Indexed []uint8
	// FMT_I8: 256 RGBA8888 palette entries (shared across sprites with the same palette)
	Palette []uint32

	// FMT_RGB565 / FMT_RGB565A1 / FMT_RGB565A4 / FMT_RGB565A8: one uint16 per pixel
	RGB565 []uint16

	// FMT_RGBA8888: one uint32 per pixel
	RGBA8888 []uint32

	// Alpha channel for FMT_RGB565A1/A4/A8 (packed bits, nil otherwise)
	AlphaData []byte
}

// SpritePack holds all sprites loaded from a spritepak file.
type SpritePack struct {
	Sprites []Sprite
}

// ReadPack reads a spritepak file and returns a SpritePack.
func ReadPack(filePath string) (*SpritePack, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// --- Read header ---
	var spritesArrayOffset uint64
	var spriteCount uint32
	var reserved uint32
	if err := binary.Read(f, binary.LittleEndian, &spritesArrayOffset); err != nil {
		return nil, err
	}
	if err := binary.Read(f, binary.LittleEndian, &spriteCount); err != nil {
		return nil, err
	}
	if err := binary.Read(f, binary.LittleEndian, &reserved); err != nil {
		return nil, err
	}

	// --- Read sprite entries from the sprite table ---
	if _, err := f.Seek(int64(spritesArrayOffset), io.SeekStart); err != nil {
		return nil, err
	}

	entries := make([]spriteEntry, spriteCount)
	for i := range entries {
		e := &entries[i]
		if err := binary.Read(f, binary.LittleEndian, &e.Width); err != nil {
			return nil, err
		}
		if err := binary.Read(f, binary.LittleEndian, &e.Height); err != nil {
			return nil, err
		}
		if err := binary.Read(f, binary.LittleEndian, &e.Format); err != nil {
			return nil, err
		}
		if err := binary.Read(f, binary.LittleEndian, &e.Reserved); err != nil {
			return nil, err
		}
		if err := binary.Read(f, binary.LittleEndian, &e.PixelDataSize); err != nil {
			return nil, err
		}
		if err := binary.Read(f, binary.LittleEndian, &e.AlphaDataSize); err != nil {
			return nil, err
		}
		if err := binary.Read(f, binary.LittleEndian, &e.PixelDataOffset); err != nil {
			return nil, err
		}
		if err := binary.Read(f, binary.LittleEndian, &e.AlphaDataOffset); err != nil {
			return nil, err
		}
		if err := binary.Read(f, binary.LittleEndian, &e.PaletteDataOffset); err != nil {
			return nil, err
		}
	}

	// --- Cache for already-loaded palettes (keyed by file offset) ---
	paletteCache := make(map[uint64][]uint32)

	// --- Build Sprite slice ---
	sprites := make([]Sprite, spriteCount)
	for i, e := range entries {
		s := &sprites[i]
		s.Width = e.Width
		s.Height = e.Height
		s.Format = e.Format

		// pixel data — seek once then decode into typed slice
		if _, err := f.Seek(int64(e.PixelDataOffset), io.SeekStart); err != nil {
			return nil, err
		}
		colorFormat := e.Format & 0xFF00
		switch colorFormat {
		case FMT_I8:
			s.Indexed = make([]uint8, e.PixelDataSize)
			if _, err := io.ReadFull(f, s.Indexed); err != nil {
				return nil, err
			}
		case FMT_RGB565:
			pixelCount := int(e.PixelDataSize) / 2
			s.RGB565 = make([]uint16, pixelCount)
			if err := binary.Read(f, binary.LittleEndian, s.RGB565); err != nil {
				return nil, err
			}
		case FMT_RGBA8888:
			pixelCount := int(e.PixelDataSize) / 4
			s.RGBA8888 = make([]uint32, pixelCount)
			if err := binary.Read(f, binary.LittleEndian, s.RGBA8888); err != nil {
				return nil, err
			}
		}

		// alpha data (only present when AlphaDataSize > 0 and offset is non-zero)
		if e.AlphaDataSize > 0 && e.AlphaDataOffset != 0 {
			s.AlphaData = make([]byte, e.AlphaDataSize)
			if _, err := f.Seek(int64(e.AlphaDataOffset), io.SeekStart); err != nil {
				return nil, err
			}
			if _, err := io.ReadFull(f, s.AlphaData); err != nil {
				return nil, err
			}
		}

		// palette data (only present when offset is non-zero, always 256 × uint32)
		if e.PaletteDataOffset != 0 {
			if pal, ok := paletteCache[e.PaletteDataOffset]; ok {
				s.Palette = pal
			} else {
				pal = make([]uint32, 256)
				if _, err := f.Seek(int64(e.PaletteDataOffset), io.SeekStart); err != nil {
					return nil, err
				}
				if err := binary.Read(f, binary.LittleEndian, pal); err != nil {
					return nil, err
				}
				paletteCache[e.PaletteDataOffset] = pal
				s.Palette = pal
			}
		}
	}

	return &SpritePack{Sprites: sprites}, nil
}
