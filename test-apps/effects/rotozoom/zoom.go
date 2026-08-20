package fx_rotozoom

import (
	"math"

	fx_common "github.com/jurgen-kluft/go-gx2/test-apps/effects/common"
)

// TODO: Get an image that is a power-of-two in both dimensions, so we can use bitwise AND instead of modulo for wrapping.

type Effect struct {
	RotationAngle float32
	RotationSpeed float32
	PixelSize     int32
	Image         *ImageRgb565
}

func NewEffect(speed float32, pixelSize int32) *Effect {
	f := &Effect{
		RotationAngle: 0,
		RotationSpeed: speed,
		PixelSize:     pixelSize,
		Image:         GetImage(),
	}

	return f
}

func (e *Effect) animate(dt float32) {
	e.RotationAngle = (e.RotationAngle + e.RotationSpeed*dt)
	if e.RotationAngle >= 360 {
		e.RotationAngle -= 360
	}
}

func (e *Effect) render(fb *fx_common.FrameBuffer) {
	s := float32(math.Sin(float64(e.RotationAngle) * math.Pi / 180))
	c := float32(math.Cos(float64(e.RotationAngle) * math.Pi / 180))
	z := s * 1.1

	// keep the point that we rotate around in the center of the screen
	cx := float32(fb.Width) / 2
	cy := float32(fb.Height) / 2

	// what would be the UV starting point when rotating around the center of the screen?
	centerU := cx*c - cy*s
	centerV := cx*s + cy*c

	for centerU < 0 {
		centerU += float32(e.Image.Width)
	}
	for centerV < 0 {
		centerV += float32(e.Image.Height)
	}

	centerU = float32(int32(centerU) % e.Image.Width)
	centerV = float32(int32(centerV) % e.Image.Height)

	for x := int32(0); x < fb.Width; x += e.PixelSize {
		for y := int32(0); y < fb.Height; y += e.PixelSize {

			// Get a rotated pixel from the image
			u := (int32(centerU) + int32((float32(x)*c-float32(y)*s)*z)) % e.Image.Width
			v := (int32(centerV) + int32((float32(x)*s+float32(y)*c)*z)) % e.Image.Height

			if u < 0 {
				u += e.Image.Width
			}
			if v < 0 {
				v += e.Image.Height
			}

			pixelIndex := v*e.Image.Width + u
			color := e.Image.Data[pixelIndex]

			for px := int32(0); px < e.PixelSize; px++ {
				for py := int32(0); py < e.PixelSize; py++ {
					fb.Pixels[(y+py)*fb.Width+(x+px)] = color
				}
			}
		}
	}
}

func (e *Effect) ProcessFrame(dt float32, fb *fx_common.FrameBuffer) {
	e.animate(dt)
	e.render(fb)
}
