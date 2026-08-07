package fontpack

import (
	"image"

	sdf_font "github.com/jurgen-kluft/go-gx2/fontpak/sdf"
)

func applySDF(glyph *builtGlyph, img image.Image, opts FontOptions) {
	if glyph == nil || !opts.SDF || glyph.Width == 0 || glyph.Height == 0 {
		return
	}

	bitmap, width, height := sdf_font.Generate(img, int(cSDFBuildBorder), opts.SDFRadius, opts.SDFCutoff)
	crop := int(cSDFBuildBorder - opts.SDFBorder)
	if crop > 0 {
		croppedWidth := width - crop*2
		croppedHeight := height - crop*2
		cropped := make([]byte, 0, croppedWidth*croppedHeight)
		for y := 0; y < croppedHeight; y++ {
			srcOffset := (y+crop)*width + crop
			cropped = append(cropped, bitmap[srcOffset:srcOffset+croppedWidth]...)
		}
		bitmap = cropped
		width = croppedWidth
		height = croppedHeight
	}
	glyph.Bitmap = bitmap
	glyph.Width = uint16(width)
	glyph.Height = uint16(height)
	glyph.BearingX -= opts.SDFBorder
	glyph.BearingY += opts.SDFBorder
}
