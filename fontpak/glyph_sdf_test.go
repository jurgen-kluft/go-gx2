package fontpack

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	sdf_font "github.com/jurgen-kluft/go-gx2/fontpak/sdf"
)

func TestApplySDFRetainsOnePixelBuffer(t *testing.T) {
	img := image.NewAlpha(image.Rect(0, 0, 3, 2))
	img.SetAlpha(1, 0, color.Alpha{A: 255})
	img.SetAlpha(2, 1, color.Alpha{A: 128})

	for _, generationBuffer := range []int{1, 3} {
		t.Run(string(rune('0'+generationBuffer)), func(t *testing.T) {
			opts := FontOptions{
				SDF:       true,
				SDFBorder: int16(generationBuffer),
				SDFRadius: 8,
				SDFCutoff: 0.25,
			}
			full, fullWidth, _ := sdf_font.Generate(img, cSDFBuildBorder, opts.SDFRadius, opts.SDFCutoff)
			crop := cSDFBuildBorder - int(opts.SDFBorder)
			wantWidth := img.Bounds().Dx() + 2*int(opts.SDFBorder)
			wantHeight := img.Bounds().Dy() + 2*int(opts.SDFBorder)
			want := make([]byte, wantWidth*wantHeight)
			for y := 0; y < wantHeight; y++ {
				srcOffset := (y+crop)*fullWidth + crop
				copy(want[y*wantWidth:(y+1)*wantWidth], full[srcOffset:srcOffset+wantWidth])
			}

			glyph := Glyph{Width: 3, Height: 2, BearingX: 4, BearingY: 5}
			applySDF(&glyph, img, opts)

			if glyph.Width != uint16(wantWidth) || glyph.Height != uint16(wantHeight) {
				t.Fatalf("unexpected dimensions: %dx%d, want %dx%d", glyph.Width, glyph.Height, wantWidth, wantHeight)
			}
			if glyph.BearingX != (4+opts.SDFBorder) || glyph.BearingY != (5+opts.SDFBorder) {
				t.Fatalf("unexpected bearings: (%d,%d), want (%d,%d)", glyph.BearingX, glyph.BearingY, 4+opts.SDFBorder, 5+opts.SDFBorder)
			}
			if !bytes.Equal(glyph.Bitmap, want) {
				t.Fatal("packed SDF bitmap is not the centered one-pixel-buffer crop")
			}
		})
	}
}
