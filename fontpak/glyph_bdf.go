// glyph_bdf.go
package fontpack

import (
	"go-gx2/bdf"
)

// BuildGlyphBDF converts a BDF Character into a BuiltGlyph.
// The bitmap is returned as tightly packed 8‑bit alpha,
// row‑major, with origin at the top‑left of the glyph bitmap.
func BuildGlyphBDF(ch *bdf.Character) BuiltGlyph {
	img := ch.Alpha
	if img == nil {
		return BuiltGlyph{
			Rune:     ch.Encoding,
			AdvanceX: int16(ch.Advance[0]),
		}
	}

	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()

	// Tight copy of alpha bitmap
	bitmap := make([]byte, w*h)
	for y := 0; y < h; y++ {
		src := img.Pix[(y+b.Min.Y)*img.Stride+b.Min.X : (y+b.Min.Y)*img.Stride+b.Min.X+w]
		dst := bitmap[y*w : y*w+w]
		copy(dst, src)
	}

	// LowerPoint is the pixel coordinate of the lowest point
	// of the glyph relative to the baseline.
	//
	// bearing_y = distance from baseline to top of bitmap
	bearingY := h + ch.LowerPoint[1]

	return BuiltGlyph{
		Rune:     ch.Encoding,
		AdvanceX: int16(ch.Advance[0]),

		// Horizontal bearing: bitmap left relative to pen position
		BearingX: int16(ch.LowerPoint[0]),

		// Vertical bearing: baseline -> top of bitmap
		BearingY: int16(bearingY),

		Width:  uint16(w),
		Height: uint16(h),
		Bitmap: bitmap,
	}
}
