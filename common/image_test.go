package common

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestPaletteEditorExportDiagnostics(t *testing.T) {
	img, err := LoadImage("test.png")
	if err != nil {
		t.Fatalf("load image: %v", err)
	}
	colorCount := ImageRGBColorCount(img)
	t.Logf("decoded %T with %d unique RGB colors", img, colorCount)
	if colorCount != 256 {
		t.Fatalf("Palette Editor export RGB color count: got %d, want 256", colorCount)
	}
	if !ImageIsIndexed(img) {
		t.Fatal("Palette Editor export has more than 256 RGB colors")
	}
}

func TestRGB888IgnoresAlpha(t *testing.T) {
	want := uint32(0xC86430)
	for _, alpha := range []uint8{0, 1, 128, 255} {
		color := NewColorFromR8G8B8A8(alpha, 200, 100, 48)
		if got := color.ToRGB32(); got != want {
			t.Fatalf("alpha %d: got RGB key %#06x, want %#06x", alpha, got, want)
		}
	}
}

func TestImageIsIndexedIgnoresAlpha(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 256, 2))
	for x := 0; x < 256; x++ {
		img.SetNRGBA(x, 0, color.NRGBA{R: uint8(x), G: 40, B: 90, A: uint8(x)})
		img.SetNRGBA(x, 1, color.NRGBA{R: uint8(x), G: 40, B: 90, A: uint8(255 - x)})
	}

	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
	decoded, err := png.Decode(bytes.NewReader(encoded.Bytes()))
	if err != nil {
		t.Fatalf("decode PNG: %v", err)
	}

	if !ImageIsIndexed(decoded) {
		t.Fatal("256 straight RGB colors with varying alpha should be indexed")
	}
}

func TestImageIsIndexedRejectsMoreThan256StraightColors(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 257, 1))
	for x := 0; x < 256; x++ {
		img.SetNRGBA(x, 0, color.NRGBA{R: uint8(x), G: 40, B: 90, A: uint8(x)})
	}
	img.SetNRGBA(256, 0, color.NRGBA{R: 0, G: 41, B: 90, A: 128})

	if ImageIsIndexed(img) {
		t.Fatal("257 straight RGB colors should not be indexed")
	}
}

func TestEncodeRGBA8888PreservesStraightColorAndAlpha(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 200, G: 100, B: 48, A: 0})
	img.SetNRGBA(1, 0, color.NRGBA{R: 200, G: 100, B: 48, A: 128})
	img.SetNRGBA(2, 0, color.NRGBA{R: 200, G: 100, B: 48, A: 255})

	got := EncodeRGBA8888(img, ImageFullRect(img))
	want := []byte{
		200, 100, 48, 0,
		200, 100, 48, 128,
		200, 100, 48, 255,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("encoded RGBA mismatch: got %v, want %v", got, want)
	}
}

func TestEncodeRGB565AlphaFormatsPreserveStraightColor(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	for x, alpha := range []uint8{0, 128, 255} {
		img.SetNRGBA(x, 0, color.NRGBA{R: 200, G: 100, B: 48, A: alpha})
	}

	rgb565 := NewColorFromR8G8B8A8(255, 200, 100, 48).ToRGB16()
	wantPixels := []byte{
		byte(rgb565), byte(rgb565 >> 8),
		byte(rgb565), byte(rgb565 >> 8),
		byte(rgb565), byte(rgb565 >> 8),
	}

	tests := []struct {
		name        string
		encode      func(image.Image, Rect) ([]byte, []byte)
		alphaFormat AlphaFormat
		wantAlpha   []byte
	}{
		{name: "A0", encode: EncodeRGB565, alphaFormat: FMT_ALPHA_NONE, wantAlpha: []byte{}},
		{name: "A1", encode: EncodeRGB565, alphaFormat: FMT_ALPHA_A1, wantAlpha: []byte{0x06}},
		{name: "A4", encode: EncodeRGB565, alphaFormat: FMT_ALPHA_A4, wantAlpha: []byte{0x80, 0x0F}},
		{name: "A8", encode: EncodeRGB565, alphaFormat: FMT_ALPHA_A8, wantAlpha: []byte{0x00, 0x80, 0xFF}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pixels, alpha := test.encode(img, ImageFullRect(img))
			alpha = ConvertAlpha(alpha, 3, 1, test.alphaFormat)
			if !bytes.Equal(pixels, wantPixels) {
				t.Fatalf("encoded RGB565 mismatch: got %v, want %v", pixels, wantPixels)
			}
			if !bytes.Equal(alpha, test.wantAlpha) {
				t.Fatalf("encoded alpha mismatch: got %v, want %v", alpha, test.wantAlpha)
			}
		})
	}
}
