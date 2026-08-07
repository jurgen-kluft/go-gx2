package fontpack

import (
	"image"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

func fixedToInt(v fixed.Int26_6) int {
	return int(v.Round())
}

func newTTFFace(opts FontOptions, data []byte) (font.Face, error) {
	f, err := opentype.Parse(data)
	if err != nil {
		return nil, err
	}

	return opentype.NewFace(f, &opentype.FaceOptions{
		Size:    float64(opts.FontSize),
		DPI:     float64(72), // Default DPI
		Hinting: font.HintingFull,
	})
}

func extractFontMetrics(face font.Face) (ascent, descent, lineGap int16) {
	m := face.Metrics()
	a := fixedToInt(m.Ascent)
	d := fixedToInt(m.Descent)
	h := fixedToInt(m.Height)
	return int16(a), int16(-d), int16(h - (a + d))
}

func buildGlyphTTF(face font.Face, r rune, opts FontOptions) (*builtGlyph, error) {
	pb, advance, ok := face.GlyphBounds(r)
	if !ok {
		return nil, nil
	}

	w := pb.Max.X - pb.Min.X
	h := pb.Max.Y - pb.Min.Y

	if w <= 0 || h <= 0 {
		return &builtGlyph{
			Rune:     r,
			AdvanceX: int16(fixedToInt(advance)),
		}, nil
	}

	img := image.NewAlpha(image.Rect(0, 0, w.Ceil(), h.Ceil()))

	d := font.Drawer{
		Dst:  img,
		Src:  image.White,
		Face: face,
		Dot: fixed.Point26_6{
			X: -pb.Min.X,
			Y: -pb.Min.Y,
		},
	}
	d.DrawString(string(r))

	bitmap := make([]byte, len(img.Pix))
	copy(bitmap, img.Pix)

	glyph := &builtGlyph{
		Rune:     r,
		AdvanceX: int16(fixedToInt(advance)),
		BearingX: int16(fixedToInt(pb.Min.X)),
		BearingY: int16(fixedToInt(-pb.Min.Y)),
		Width:    uint16(img.Bounds().Dx()),
		Height:   uint16(img.Bounds().Dy()),
		Bitmap:   bitmap,
	}

	applySDF(glyph, img, opts)

	return glyph, nil
}
