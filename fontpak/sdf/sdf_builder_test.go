package sdf_font

import (
	"image"
	"image/color"
	"os"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

func builderFor(t *testing.T, fontFamily string) *SDFBuilder {
	t.Helper()

	ttf, err := os.ReadFile("./testdata/" + fontFamily + ".ttf")
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := opentype.Parse(ttf)
	if err != nil {
		t.Fatal(err)
	}

	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    26,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = face.Close() })

	return NewSDFBuilder(face, SDFBuilderOpt{Buffer: 3})
}

func TestGenerateAddsPadding(t *testing.T) {
	img := image.NewAlpha(image.Rect(0, 0, 3, 3))
	img.SetAlpha(1, 1, color.Alpha{A: 255})

	bitmap, width, height := Generate(img, 2, 8, 0.25)
	if width != 7 || height != 7 || len(bitmap) != width*height {
		t.Fatalf("unexpected generated dimensions: %dx%d, %d bytes", width, height, len(bitmap))
	}

	center := bitmap[3+3*width]
	corner := bitmap[0]
	if center <= corner {
		t.Fatalf("inside distance %d must exceed outside distance %d", center, corner)
	}
}

func TestSDFBuilderGlyphMetadataMatchesBitmap(t *testing.T) {
	builder := builderFor(t, "NotoSans-Regular")

	glyph := builder.Glyph('A')
	if glyph == nil {
		t.Fatal("expected glyph A")
	}
	if got, want := len(glyph.Bitmap), int(glyph.Width*glyph.Height); got != want {
		t.Fatalf("bitmap length %d does not match dimensions %d", got, want)
	}
}
