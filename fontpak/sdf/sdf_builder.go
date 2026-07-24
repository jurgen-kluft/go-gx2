package sdf_font

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func NewSDFBuilder(face font.Face, opts ...SDFBuilderOpt) *SDFBuilder {
	sdfBuilder := &SDFBuilder{
		Face: face,
	}

	for _, opt := range opts {
		if opt.Buffer != 0 {
			sdfBuilder.Buffer = opt.Buffer
		}
		if opt.Radius != 0 {
			sdfBuilder.Radius = opt.Radius
		}
		if opt.Cutoff != 0 {
			sdfBuilder.Cutoff = opt.Cutoff
		}
	}

	if sdfBuilder.Buffer == 0 {
		sdfBuilder.Buffer = 3
	}
	if sdfBuilder.Radius == 0 {
		sdfBuilder.Radius = 8
	}
	if sdfBuilder.Cutoff == 0 {
		sdfBuilder.Cutoff = 0.25
	}
	return sdfBuilder
}

type SDFBuilderOpt struct {
	Buffer int
	Radius float64
	Cutoff float64
}

type SDFBuilder struct {
	Face font.Face
	SDFBuilderOpt
}

func (b *SDFBuilder) Glyphs(min int, max int) []*Glyph {
	glyphs := make([]*Glyph, 0, max-min)
	for i := min; i < max; i++ {
		g := b.Glyph(rune(i))
		if g != nil {
			glyphs = append(glyphs, g)
		}
	}
	return glyphs
}

func (b *SDFBuilder) Glyph(x rune) *Glyph {
	if x == 0 {
		return nil
	}

	bounds, advance, ok := b.Face.GlyphBounds(x)
	if !ok {
		return nil
	}

	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y
	if width <= 0 || height <= 0 {
		return nil
	}

	mask := image.NewAlpha(image.Rect(0, 0, width.Ceil(), height.Ceil()))
	drawer := font.Drawer{
		Dst:  mask,
		Src:  image.White,
		Face: b.Face,
		Dot: fixed.Point26_6{
			X: -bounds.Min.X,
			Y: -bounds.Min.Y,
		},
	}
	drawer.DrawString(string(x))

	bitmap, sdfWidth, sdfHeight := Generate(mask, b.Buffer, b.Radius, b.Cutoff)

	g := &Glyph{
		Width:   uint32(sdfWidth),
		Height:  uint32(sdfHeight),
		Left:    int32(bounds.Min.X.Round() - b.Buffer),
		Top:     int32((-bounds.Min.Y).Round() + b.Buffer),
		Advance: uint32(advance.Round()),
		Bitmap:  bitmap,
	}

	return g
}

func Generate(img image.Image, buffer int, radius float64, cutoff float64) ([]byte, int, int) {
	bounds := img.Bounds()
	width := bounds.Dx() + buffer*2
	height := bounds.Dy() + buffer*2
	padded := image.NewAlpha(image.Rect(0, 0, width, height))
	draw.Draw(padded, image.Rect(buffer, buffer, buffer+bounds.Dx(), buffer+bounds.Dy()), img, bounds.Min, draw.Src)
	return CalcSDF(padded, radius, cutoff), width, height
}

const INF = 1e20

func CalcSDF(img image.Image, radius float64, cutoff float64) []uint8 {
	size := img.Bounds().Size()
	w, h := size.X, size.Y

	gridOuter := make([]float64, w*h)
	gridInner := make([]float64, w*h)

	scratchSize := max(w, h) + 1
	f := make([]float64, scratchSize)
	d := make([]float64, scratchSize)
	v := make([]float64, scratchSize)
	z := make([]float64, scratchSize)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := x + y*w
			_, _, _, a := img.At(x, y).RGBA()

			alpha := float64(a) / math.MaxUint16

			outer := float64(0)
			inner := INF

			if alpha != 1 {
				if alpha == 0 {
					outer = INF
					inner = 0
				} else {
					outer = math.Pow(math.Max(0, 0.5-alpha), 2)
					inner = math.Pow(math.Max(0, alpha-0.5), 2)
				}
			}

			gridOuter[i] = outer
			gridInner[i] = inner
		}
	}

	euclideanDistanceTransform(gridOuter, w, h, f, d, v, z)
	euclideanDistanceTransform(gridInner, w, h, f, d, v, z)

	alphas := make([]uint8, w*h)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := x + y*w
			d := gridOuter[i] - gridInner[i]

			a := math.Max(0, math.Min(255, math.Round(255-255*(d/radius+cutoff))))

			alphas[i] = uint8(a)
		}
	}

	return alphas
}

// 2D Euclidean distance transform by Felzenszwalb & Huttenlocher https://cs.brown.edu/~pff/papers/dt-final.pdf
func euclideanDistanceTransform(data []float64, width int, height int, f []float64, d []float64, v []float64, z []float64) {
	for x := range width {
		for y := range height {
			f[y] = data[y*width+x]
		}
		euclideanDistanceTransform1d(f, d, v, z, height)
		for y := range height {
			data[y*width+x] = d[y]
		}
	}

	for y := range height {
		for x := range width {
			f[x] = data[y*width+x]
		}
		euclideanDistanceTransform1d(f, d, v, z, width)
		for x := range width {
			data[y*width+x] = math.Sqrt(d[x])
		}
	}
}

// 1D squared distance transform
func euclideanDistanceTransform1d(f []float64, d []float64, v []float64, z []float64, n int) {
	v[0] = 0
	z[0] = -INF
	z[1] = +INF

	for q, k := 1, 0; q < (n); q++ {
		getS := func() float64 {
			return ((f[q] + float64(q)*float64(q)) - (f[int(v[k])] + v[k]*v[k])) / (2*float64(q) - 2*v[k])
		}

		s := getS()
		for s <= float64(z[k]) {
			k--
			s = getS()
		}

		k++

		v[k] = float64(q)
		z[k] = float64(s)
		z[k+1] = +INF
	}

	for q, k := 0, 0; q < n; q++ {
		for z[k+1] < float64(q) {
			k++
		}
		d[q] = (float64(q)-v[k])*(float64(q)-v[k]) + f[int(v[k])]
	}
}

type Glyph struct {
	Bitmap  []byte // A signed distance field of the glyph with a border of 3 pixels.
	Width   uint32 // Glyph metrics.
	Height  uint32
	Left    int32
	Top     int32
	Advance uint32
}

func (m *Glyph) Reset() { *m = Glyph{} }

func (m *Glyph) GetBitmap() []byte {
	if m != nil {
		return m.Bitmap
	}
	return nil
}

func (m *Glyph) GetWidth() uint32 {
	if m != nil {
		return m.Width
	}
	return 0
}

func (m *Glyph) GetHeight() uint32 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *Glyph) GetLeft() int32 {
	if m != nil {
		return m.Left
	}
	return 0
}

func (m *Glyph) GetTop() int32 {
	if m != nil {
		return m.Top
	}
	return 0
}

func (m *Glyph) GetAdvance() uint32 {
	if m != nil {
		return m.Advance
	}
	return 0
}

func DrawGlyph(glyph *Glyph, smoothstep bool) image.Image {
	width := int(glyph.Width)
	height := int(glyph.Height)

	img := image.NewRGBA(image.Rectangle{Min: image.Point{0, 0}, Max: image.Point{width, height}})

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := x + y*width
			a := glyph.Bitmap[i]

			if smoothstep {
				img.Set(x, y, color.RGBA{0, 0, 0,
					alpha(136, 168, float64(a)),
				})
			} else {
				img.Set(x, y, color.RGBA{0, 0, 0,
					uint8(a),
				})
			}

		}
	}

	return img
}

func alpha(e0 float64, e1 float64, x float64) uint8 {
	a := math.Max(math.Min((x-e1)/(e1-e0), 1), 0)
	return uint8((a * a * (3 - 2*a)) * float64(x))
}

func SavePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	err = png.Encode(f, img)
	if err != nil {
		log.Fatal(err)
	}
}
