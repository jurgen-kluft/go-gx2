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

func equalSpriteEntry(a, b Sprite) bool {
	if a.Width != b.Width ||
		a.Height != b.Height ||
		a.PixelFormat != b.PixelFormat ||
		a.AlphaFormat != b.AlphaFormat {
		return false
	}

	if !bytes.Equal(a.PixelData, b.PixelData) {
		return false
	}
	if !bytes.Equal(a.AlphaData, b.AlphaData) {
		return false
	}

	return true
}

func equalSpriteEntries(a, b []Sprite) bool {
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

	inSprites := []Sprite{
		{
			Width:       16,
			Height:      8,
			PixelFormat: FMT_PIXEL_RGB565,
			AlphaFormat: FMT_ALPHA_A4,
			PixelData:   []byte{1, 2, 3, 4, 5, 6},
			AlphaData:   []byte{0x3C, 0xC3},
		},
		{
			Width:       4,
			Height:      4,
			PixelFormat: FMT_PIXEL_I8,
			AlphaFormat: FMT_ALPHA_NONE,
			PixelData:   []byte{0, 1, 2, 3, 4, 5, 6, 7},
			AlphaData:   nil,
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

	inSprites := []Sprite{
		{
			Width:       512,
			Height:      512,
			PixelFormat: FMT_PIXEL_RGB888,
			AlphaFormat: FMT_ALPHA_A8,
			PixelData:   pixelData,
			AlphaData:   alphaData,
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
