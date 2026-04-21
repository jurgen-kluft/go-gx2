package fontpack

import (
	"image"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type BuiltGlyph struct {
	Rune     rune
	AdvanceX int16
	BearingX int16
	BearingY int16
	Width    uint16
	Height   uint16
	Bitmap   []byte
}

func fixedToInt(v fixed.Int26_6) int {
	return int(v.Round())
}

func NewTTFFace(options Options, data []byte) (font.Face, error) {
	f, err := opentype.Parse(data)
	if err != nil {
		return nil, err
	}

	return opentype.NewFace(f, &opentype.FaceOptions{
		Size:    float64(options.FontSize),
		DPI:     float64(options.DPI),
		Hinting: font.HintingFull,
	})
}

func ExtractFontMetrics(face font.Face) (ascent, descent, lineGap int16) {
	m := face.Metrics()
	a := fixedToInt(m.Ascent)
	d := fixedToInt(m.Descent)
	h := fixedToInt(m.Height)
	return int16(a), int16(-d), int16(h - (a + d))
}

func BuildGlyphTTF(face font.Face, r rune) (*BuiltGlyph, error) {
	// advance, ok := face.GlyphAdvance(r)
	// if !ok {
	// 	return nil, nil
	// }

	pb, advance, ok := face.GlyphBounds(r)
	if !ok {
		return nil, nil
	}

	w := pb.Max.X - pb.Min.X
	h := pb.Max.Y - pb.Min.Y

	if w <= 0 || h <= 0 {
		return &BuiltGlyph{
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

	return &BuiltGlyph{
		Rune:     r,
		AdvanceX: int16(fixedToInt(advance)),
		BearingX: int16(fixedToInt(pb.Min.X)),
		//BearingY: int16(fixedToInt(pb.Max.Y)),
		BearingY: int16(fixedToInt(-pb.Min.Y)),
		Width:    uint16(fixedToInt(w)),
		Height:   uint16(fixedToInt(h)),
		Bitmap:   bitmap,
	}, nil
}
