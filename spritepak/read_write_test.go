package spritepack

import (
	"bytes"
	"testing"
)

func equalUint32Slice(a, b []uint32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalSpriteEntry(a, b SpriteEntry) bool {
	if a.Width != b.Width ||
		a.Height != b.Height ||
		a.Format != b.Format ||
		a.Reserved != b.Reserved ||
		a.PixelDataSize != b.PixelDataSize ||
		a.AlphaDataSize != b.AlphaDataSize {
		return false
	}

	if !bytes.Equal(a.PixelData, b.PixelData) {
		return false
	}
	if !bytes.Equal(a.AlphaData, b.AlphaData) {
		return false
	}
	if !equalUint32Slice(a.PaletteData, b.PaletteData) {
		return false
	}

	return true
}

func equalSpriteEntries(a, b []SpriteEntry) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !equalSpriteEntry(a[i], b[i]) {
			return false
		}
	}
	return true
}

func TestWriteReadPackRoundTrip(t *testing.T) {
	t.Parallel()

	inSprites := []SpriteEntry{
		{
			Width:         16,
			Height:        8,
			Format:        FMT_RGB565A1,
			Reserved:      0,
			PixelDataSize: 6,
			AlphaDataSize: 2,
			PixelData:     []byte{1, 2, 3, 4, 5, 6},
			AlphaData:     []byte{0x3C, 0xC3},
			PaletteData:   nil,
		},
		{
			Width:         4,
			Height:        4,
			Format:        FMT_I8,
			Reserved:      0,
			PixelDataSize: 8,
			AlphaDataSize: 0,
			PixelData:     []byte{0, 1, 2, 3, 4, 5, 6, 7},
			AlphaData:     nil,
			PaletteData:   []uint32{0x11223344, 0x55667788, 0x99AABBCC},
		},
	}

	var out bytes.Buffer
	if err := WritePack(&out, inSprites); err != nil {
		t.Fatalf("writePack failed: %v", err)
	}

	pack, err := ReadPack(bytes.NewReader(out.Bytes()))
	if err != nil {
		t.Fatalf("ReadPack failed: %v", err)
	}

	if pack.Count != uint32(len(inSprites)) {
		t.Fatalf("Count mismatch: got %d want %d", pack.Count, len(inSprites))
	}

	if !equalSpriteEntries(pack.Sprites, inSprites) {
		t.Fatalf("Sprites mismatch after round-trip:\n got: %#v\nwant: %#v", pack.Sprites, inSprites)
	}
}

func TestWriteReadPackEmpty(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	if err := WritePack(&out, nil); err != nil {
		t.Fatalf("writePack failed: %v", err)
	}

	pack, err := ReadPack(bytes.NewReader(out.Bytes()))
	if err != nil {
		t.Fatalf("ReadPack failed: %v", err)
	}

	if pack.Count != 0 {
		t.Fatalf("Count mismatch: got %d want 0", pack.Count)
	}

	if len(pack.Sprites) != 0 {
		t.Fatalf("Sprites length mismatch: got %d want 0", len(pack.Sprites))
	}
}

func TestWriteReadPackLargePayload(t *testing.T) {
	t.Parallel()

	pixelData := make([]byte, 200000)
	for i := range pixelData {
		pixelData[i] = byte(i % 251)
	}

	alphaData := make([]byte, 70000)
	for i := range alphaData {
		alphaData[i] = byte((i * 3) % 255)
	}

	inSprites := []SpriteEntry{
		{
			Width:         512,
			Height:        512,
			Format:        FMT_RGBA8888,
			PixelDataSize: uint32(len(pixelData)),
			AlphaDataSize: uint32(len(alphaData)),
			PixelData:     pixelData,
			AlphaData:     alphaData,
		},
	}

	var out bytes.Buffer
	if err := WritePack(&out, inSprites); err != nil {
		t.Fatalf("WritePack failed for large payload: %v", err)
	}

	pack, err := ReadPack(bytes.NewReader(out.Bytes()))
	if err != nil {
		t.Fatalf("ReadPack failed for large payload: %v", err)
	}

	if !equalSpriteEntries(pack.Sprites, inSprites) {
		t.Fatalf("Large payload round-trip mismatch")
	}
}
