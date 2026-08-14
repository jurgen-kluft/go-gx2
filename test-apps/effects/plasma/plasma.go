package main

import (
	"math"

	fx_common "github.com/jurgen-kluft/go-gx2/test-apps/effects/common"
)

const (
	renderScale   = 8
	animationStep = 0.17
)

type PlasmaEffect struct {
	width  int
	height int
	lowW   int
	lowH   int

	cx        float32
	cy        float32
	invRadius float32

	grid      []uint16
	band      []uint8
	positions []plasmaPoint
}

type plasmaPoint struct {
	x    float32
	y    float32
	mask float32
}

type colorBlob struct {
	x float32
	y float32
	r float32
	g float32
	b float32
	s float32
}

func NewPlasmaEffect(width, height int16) *PlasmaEffect {

	w := int(width)
	h := int(height)
	lowW := (w + renderScale - 1) / renderScale
	lowH := (h + renderScale - 1) / renderScale
	radius := float32(w)
	if h < w {
		radius = float32(h)
	}
	radius *= 0.5

	r := &PlasmaEffect{
		width:     w,
		height:    h,
		lowW:      lowW,
		lowH:      lowH,
		cx:        float32(w-1) * 0.5,
		cy:        float32(h-1) * 0.5,
		invRadius: 1.0 / radius,
		grid:      make([]uint16, lowW*lowH),
		band:      make([]uint8, w*renderScale*2),
		positions: make([]plasmaPoint, lowW*lowH),
	}

	for gy := 0; gy < lowH; gy++ {
		sy := float32(gy*renderScale) + float32(renderScale)*0.5
		ny := (sy - r.cy) * r.invRadius
		for gx := 0; gx < lowW; gx++ {
			sx := float32(gx*renderScale) + float32(renderScale)*0.5
			nx := (sx - r.cx) * r.invRadius
			d2 := nx*nx + ny*ny
			mask := clamp((1.0-d2)*2.1, 0, 1)
			mask *= 0.38 + 0.62*mask
			r.positions[gy*lowW+gx] = plasmaPoint{
				x:    nx,
				y:    ny,
				mask: mask,
			}
		}
	}

	return r
}

func (r *PlasmaEffect) draw(t float32, fb *fx_common.FrameBuffer) {
	r.generateField(t)
	r.writeBands(fb)
}

func (r *PlasmaEffect) generateField(t float32) {
	blobs := [5]colorBlob{
		{
			x: 0.42 * sin(t*0.73+0.4),
			y: 0.34 * sin(t*0.58+2.1),
			r: 255, g: 108, b: 24, s: 6.7,
		},
		{
			x: 0.38 * sin(t*0.51+2.6),
			y: 0.46 * sin(t*0.67+0.9),
			r: 255, g: 36, b: 156, s: 7.0,
		},
		{
			x: 0.48 * sin(t*0.44+4.0),
			y: 0.36 * sin(t*0.79+3.2),
			r: 86, g: 82, b: 255, s: 6.4,
		},
		{
			x: 0.36 * sin(t*0.69+5.7),
			y: 0.44 * sin(t*0.47+4.4),
			r: 24, g: 224, b: 255, s: 7.6,
		},
		{
			x: 0.18 * sin(t*0.93+1.7),
			y: 0.18 * sin(t*0.88+5.1),
			r: 255, g: 224, b: 48, s: 11.0,
		},
	}

	for i, p := range r.positions {
		if p.mask <= 0 {
			r.grid[i] = 0
			continue
		}

		wave := 0.92 + 0.08*sin(p.x*7.0+p.y*4.0+t*1.4)
		var red, green, blue float32
		for _, b := range blobs {
			dx := p.x - b.x
			dy := p.y - b.y
			m := reciprocal(1.0 + (dx*dx+dy*dy)*b.s)
			m = (m*m + m*0.42) * wave
			red += b.r * m
			green += b.g * m
			blue += b.b * m
		}

		glow := p.mask * 1.24
		red, green, blue = toneMapBloom(red*glow, green*glow, blue*glow)
		r.grid[i] = rgb565(
			clamp(red, 0, 255),
			clamp(green, 0, 255),
			clamp(blue, 0, 255),
		)
	}
}

func (r *PlasmaEffect) writeBands(fb *fx_common.FrameBuffer) {
	beginRGB565(fb, 0, 0, int16(r.width), int16(r.height))

	for gy := 0; gy < r.lowH; gy++ {
		y := gy * renderScale
		rows := minInt(renderScale, r.height-y)
		rowBytes := r.width * 2

		out := 0
		for gx := 0; gx < r.lowW; gx++ {
			x := gx * renderScale
			cols := minInt(renderScale, r.width-x)
			c := r.grid[gy*r.lowW+gx]
			hi := uint8(c >> 8)
			lo := uint8(c)
			for i := 0; i < cols; i++ {
				r.band[out] = hi
				r.band[out+1] = lo
				out += 2
			}
		}

		for row := 1; row < rows; row++ {
			copy(r.band[row*rowBytes:(row+1)*rowBytes], r.band[:rowBytes])
		}

	}
}

func writeRGB565(fb *fx_common.FrameBuffer, x, y, w, h int16, data []uint8) {
}

func beginRGB565(fb *fx_common.FrameBuffer, x, y, w, h int16) {
}

func rgb565(r, g, b float32) uint16 {
	ri := uint16(r) >> 3
	gi := uint16(g) >> 2
	bi := uint16(b) >> 3
	return ri<<11 | gi<<5 | bi
}

func toneMapBloom(r, g, b float32) (float32, float32, float32) {
	maxc := r
	if g > maxc {
		maxc = g
	}
	if b > maxc {
		maxc = b
	}
	if maxc > 245 {
		scale := 245 / maxc
		r *= scale
		g *= scale
		b *= scale
	}

	avg := (r + g + b) * 0.33333334
	const saturation = 1.38
	r = avg + (r-avg)*saturation
	g = avg + (g-avg)*saturation
	b = avg + (b-avg)*saturation

	return r, g, b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func sin(x float32) float32 {
	const (
		pi       float32 = 3.1415926535
		twoPi    float32 = 6.283185307
		invTwoPi float32 = 0.15915494309
		c        float32 = 0.40528473456
	)

	y := x * invTwoPi
	y = (y - floor(y)) * twoPi

	var sign float32 = 1.0
	if y > pi {
		y = y - pi
		sign = -1.0
	}

	return sign * c * y * (pi - y)
}

func floor(x float32) float32 {
	i := int32(x)
	y := float32(i)
	if y > x {
		return y - 1
	}
	return y
}

func reciprocal(x float32) float32 {
	y := math.Float32frombits(0x7EF311C3 - math.Float32bits(x))
	return y * (2.0 - x*y)
}

func clamp(v, min, max float32) float32 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
