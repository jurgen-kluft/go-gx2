package fontpack

import (
	"image"

	sdf_font "github.com/jurgen-kluft/go-gx2/fontpak/sdf"
)

func applySDF(glyph *builtGlyph, img image.Image, opts Options) {
	if glyph == nil || !opts.SDF || glyph.Width == 0 || glyph.Height == 0 {
		return
	}

	bitmap, width, height := sdf_font.Generate(img, opts.SDFBuffer, opts.SDFRadius, opts.SDFCutoff)
	glyph.Bitmap = bitmap
	glyph.Width = uint16(width)
	glyph.Height = uint16(height)
	glyph.BearingX -= int16(opts.SDFBuffer)
	glyph.BearingY += int16(opts.SDFBuffer)
}
