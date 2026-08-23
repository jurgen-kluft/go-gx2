package coolfire

import (
	"math"
	"math/rand"

	fx_common "github.com/jurgen-kluft/go-gx2/test-apps/effects/common"
)

const (
	defaultWidth      int32 = 256
	defaultHeight     int32 = 256
	defaultSourceRows int32 = 2
)

type Effect struct {
	width          int32
	height         int32
	current        []uint8
	next           []uint8
	cooling        []uint8
	coolingOffsetY int32
	palette        [256]uint16
	yStart         float64
	sourceRows     int32
}

func NewEffect(width, height int32) *Effect {
	if width <= 0 {
		width = defaultWidth
	}
	if height <= 0 {
		height = defaultHeight
	}

	size := int(width * height)

	effect := &Effect{
		width:      width,
		height:     height,
		current:    make([]uint8, size),
		next:       make([]uint8, size),
		cooling:    make([]uint8, size),
		palette:    newClassicFirePalette(),
		sourceRows: defaultSourceRows,
	}

	effect.createCooling()

	return effect
}

func (e *Effect) ProcessFrame(deltaTime float32, frameBuffer *fx_common.FrameBuffer) {
	e.seedFire()
	e.propagate()
	e.render(frameBuffer)
	_ = deltaTime

	e.coolingOffsetY = (e.coolingOffsetY + 1) % e.height
}

func (e *Effect) createCooling() {

	// Sprinkle around 2000 points randomly in the cooling 2d array, and then
	// apply 20 passes of smoothing to create a more natural cooling effect.
	// The result should be that the cooling map contains values ranging from 0 to 10.

	numPoints := 4200
	numSmoothingPasses := 20

	current := make([]uint8, len(e.cooling))
	previous := make([]uint8, len(e.cooling))

	for i := 0; i < numPoints; i++ {
		x := int32(math.Floor(float64(e.width) * rand.Float64()))
		y := int32(math.Floor(float64(e.height) * rand.Float64()))
		previous[y*e.width+x] = 255
	}

	for pass := 0; pass < numSmoothingPasses; pass++ {
		for y := int32(1); y < e.height-1; y++ {
			for x := int32(1); x < e.width-1; x++ {
				sum := 0
				count := 0
				for dy := int32(-1); dy <= 1; dy++ {
					for dx := int32(-1); dx <= 1; dx++ {
						sum += int(previous[(y+dy)*e.width+(x+dx)])
						count++
					}
				}
				current[y*e.width+x] = uint8(sum / count)
			}
		}
		current, previous = previous, current
	}

	e.cooling = previous
}

func (e *Effect) seedFire() {
	width := int(e.width)
	height := int(e.height)
	rows := int(e.sourceRows)
	if rows > height {
		rows = height
	}

	for row := 0; row < rows; row++ {
		y := height - 1 - row
		offset := y * width
		for x := 0; x < width; x++ {
			e.current[offset+x] = 245 + uint8(rand.Intn(10))
		}
	}
}

func (e *Effect) readCooling(x, y int32) int32 {
	newY := (e.coolingOffsetY + y) % e.height
	return int32(e.cooling[newY*e.width+x])
}

func (e *Effect) propagate() {
	for i := range e.next {
		e.next[i] = 0
	}

	width := e.width
	height := e.height

	for y := int32(1); y < height-1; y++ {
		rowOffset := y * width
		for x := int32(1); x < width-1; x++ {
			index := rowOffset + x

			n1 := int32(e.current[index-width])
			n2 := int32(e.current[index+width])
			n3 := int32(e.current[index-1])
			n4 := int32(e.current[index+1])

			c := e.readCooling(x, y)

			neighbors := n1 + n2 + n3 + n4
			cooled := (neighbors / 4) - c
			if cooled < 0 {
				cooled = 0
			}

			target := index - width
			e.next[target] = uint8(cooled)
		}
	}

	e.current, e.next = e.next, e.current
}

func (e *Effect) render(frameBuffer *fx_common.FrameBuffer) {
	xOffset := (int(frameBuffer.Width) - int(e.width)) / 2
	yOffset := (int(frameBuffer.Height) - int(e.height)) / 2

	width := int(e.width)
	height := int(e.height)
	frameWidth := int(frameBuffer.Width)
	frameHeight := int(frameBuffer.Height)

	for y := 0; y < height; y++ {
		dstY := y + yOffset
		if dstY < 0 || dstY >= frameHeight {
			continue
		}
		srcRow := y * width
		dstRow := dstY * frameWidth
		for x := 0; x < width; x++ {
			dstX := x + xOffset
			if dstX < 0 || dstX >= frameWidth {
				continue
			}
			intensity := e.current[srcRow+x]
			frameBuffer.Pixels[dstRow+dstX] = e.palette[intensity]
		}
	}
}

func newClassicFirePalette() [256]uint16 {
	var palette [256]uint16

	for i := 0; i < 256; i++ {
		var r uint8
		var g uint8
		var b uint8

		switch {
		case i < 64:
			t := float64(i) / 63.0
			r = uint8(lerp(0, 255, t))
			g = 0
			b = 0
		case i < 128:
			t := float64(i-64) / 63.0
			r = 255
			g = uint8(lerp(0, 160, t))
			b = 0
		case i < 192:
			t := float64(i-128) / 63.0
			r = 255
			g = uint8(lerp(160, 255, t))
			b = uint8(lerp(0, 48, t))
		default:
			t := float64(i-192) / 63.0
			r = 255
			g = 255
			b = uint8(lerp(48, 255, t))
		}

		palette[i] = fx_common.ConvertToRGB565(r, g, b)
	}

	return palette
}

func clampToByte(value float64) uint8 {
	if value <= 0 {
		return 0
	}
	if value >= 255 {
		return 255
	}
	return uint8(value)
}

func lerp(start, end, t float64) float64 {
	return start + (end-start)*t
}

type perlin2D struct {
	perm [512]int
}

func newPerlin2D(seed uint32) perlin2D {
	base := [256]int{}
	for i := range base {
		base[i] = i
	}

	state := seed
	for i := len(base) - 1; i > 0; i-- {
		state = state*1664525 + 1013904223
		j := int(state % uint32(i+1))
		base[i], base[j] = base[j], base[i]
	}

	var perm [512]int
	for i := 0; i < 512; i++ {
		perm[i] = base[i&255]
	}

	return perlin2D{perm: perm}
}

func (p perlin2D) Noise2D(x, y float64) float64 {
	xi := int(math.Floor(x)) & 255
	yi := int(math.Floor(y)) & 255
	xf := x - math.Floor(x)
	yf := y - math.Floor(y)

	u := fade(xf)
	v := fade(yf)

	aa := p.perm[p.perm[xi]+yi]
	ab := p.perm[p.perm[xi]+yi+1]
	ba := p.perm[p.perm[xi+1]+yi]
	bb := p.perm[p.perm[xi+1]+yi+1]

	x1 := lerp(grad2D(aa, xf, yf), grad2D(ba, xf-1.0, yf), u)
	x2 := lerp(grad2D(ab, xf, yf-1.0), grad2D(bb, xf-1.0, yf-1.0), u)

	return (lerp(x1, x2, v) + 1.0) * 0.5
}

func fade(t float64) float64 {
	return t * t * t * (t*(t*6.0-15.0) + 10.0)
}

func grad2D(hash int, x, y float64) float64 {
	switch hash & 7 {
	case 0:
		return x + y
	case 1:
		return -x + y
	case 2:
		return x - y
	case 3:
		return -x - y
	case 4:
		return x
	case 5:
		return -x
	case 6:
		return y
	default:
		return -y
	}
}
