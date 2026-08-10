package common

import (
	"image"
	"image/color"
	"testing"
)

func TestBuildIndexed8PaletteIgnoresAlphaForMatching(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, G: 255, B: 255, A: 0})
	img.SetNRGBA(1, 0, color.NRGBA{R: 255, G: 255, B: 255, A: 128})
	img.SetNRGBA(2, 0, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	img.SetNRGBA(3, 0, color.NRGBA{R: 0, G: 0, B: 0, A: 128})

	palette := PaletteRGBA{
		NewColorFromR8G8B8A8(255, 255, 255, 255),
		NewColorFromR8G8B8A8(255, 0, 0, 0),
	}
	pixels, ok := BuildIndexed8Palette(img, ImageFullRect(img), palette)
	if !ok {
		t.Fatal("expected indexed palette conversion to succeed")
	}

	want := []byte{0, 0, 0, 1}
	if len(pixels) != len(want) {
		t.Fatalf("indexed pixel length: got %d, want %d", len(pixels), len(want))
	}
	for index := range want {
		if pixels[index] != want[index] {
			t.Fatalf("indexed pixel %d: got %d, want %d", index, pixels[index], want[index])
		}
	}
}

func TestBuildPaletteUsesStraightRGB(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, G: 0, B: 0, A: 0})
	img.SetNRGBA(1, 0, color.NRGBA{R: 255, G: 0, B: 0, A: 128})
	img.SetNRGBA(2, 0, color.NRGBA{R: 255, G: 0, B: 0, A: 255})

	palette, colorMap, ok := BuildPaletteFromImage(img)
	if !ok {
		t.Fatalf("buildPalette failed")
	}
	if len(palette) != 1 {
		t.Fatalf("palette length: got %d, want 1", len(palette))
	}
	if len(colorMap) != 1 {
		t.Fatalf("colorMap length: got %d, want 1", len(colorMap))
	}

	r, g, b, a := palette[0].ToR8G8B8A8()
	if r != 255 || g != 0 || b != 0 || a != 255 {
		t.Fatalf("palette color: got (%d,%d,%d,%d), want (255,0,0,255)", r, g, b, a)
	}
}
