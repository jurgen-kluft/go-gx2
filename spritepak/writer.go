package spritepak

import (
	"encoding/binary"
	"io"
	"os"
)

func writePack(outPath string, sprites []spriteEntry, pixelData [][]byte, alphaData [][]byte, paletteRefArray []int, paletteDataArray [][]uint32) error {

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// --- Helper for 8-byte alignment ---
	paddingData := make([]byte, 8)
	alignTo8 := func(offset int64) int64 {
		padding := (8 - (offset % 8)) % 8
		if padding > 0 {
			f.Write(paddingData[:padding])
		}
		return offset + padding
	}

	// --- Header placeholder ---
	binary.Write(f, binary.LittleEndian, uint64(0))            // sprite array offset
	binary.Write(f, binary.LittleEndian, uint32(len(sprites))) // sprite count
	binary.Write(f, binary.LittleEndian, uint32(0))            // reserved

	var offset int64

	// --- Palette blocks ---
	paletteDataOffsetArray := make([]int64, len(paletteRefArray))
	for i, p := range paletteDataArray {
		palette := p
		offset, _ = f.Seek(0, io.SeekCurrent)
		paletteDataOffsetArray[i] = offset
		for _, c := range palette {
			binary.Write(f, binary.LittleEndian, c)
		}
	}

	// --- Data blocks ---
	for i := range sprites {

		// --- Align to 8 bytes before writing pixel data ---
		offset, _ = f.Seek(0, io.SeekCurrent)
		offset = alignTo8(offset)
		sprites[i].PixelDataOffset = uint64(offset)
		f.Write(pixelData[i])

		if alphaData[i] != nil {
			offset, _ = f.Seek(0, io.SeekCurrent)
			// --- Align to 8 bytes before writing alpha data ---
			offset = alignTo8(offset)
			sprites[i].AlphaDataOffset = uint64(offset)
			f.Write(alphaData[i])
		}

		if paletteRefArray[i] >= 0 {
			offset = paletteDataOffsetArray[paletteRefArray[i]]
			sprites[i].PaletteDataOffset = uint64(offset)
		} else {
			sprites[i].PaletteDataOffset = 0
		}
	}

	// --- Sprite table ---
	spritesArrayOffset, _ := f.Seek(0, io.SeekCurrent)

	for _, s := range sprites {
		s.writeBinary(f)
	}

	// --- Rewrite header + table ---
	f.Seek(0, io.SeekStart)
	binary.Write(f, binary.LittleEndian, uint64(spritesArrayOffset))
	binary.Write(f, binary.LittleEndian, uint32(len(sprites)))
	binary.Write(f, binary.LittleEndian, uint32(0))

	return nil
}
