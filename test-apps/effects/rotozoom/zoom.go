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

	ImageData   []uint8
	ImageWidth  int32
	ImageHeight int32
}

func NewEffect(speed float32, pixelSize int32) *Effect {
	f := &Effect{
		RotationAngle: 0,
		RotationSpeed: speed,
		PixelSize:     pixelSize,
	}
	f.ImageData = IMAGE_DATA[4:]
	f.ImageWidth = IMAGE_WIDTH
	f.ImageHeight = IMAGE_HEIGHT

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

	for x := int32(0); x < fb.Width; x += e.PixelSize {
		for y := int32(0); y < fb.Height; y += e.PixelSize {

			// Get a rotated pixel from the image
			u := int32((float32(x)*c-float32(y)*s)*z) % e.ImageWidth
			v := int32((float32(x)*s+float32(y)*c)*z) % e.ImageHeight

			if u < 0 {
				u += e.ImageWidth
			}
			if v < 0 {
				v += e.ImageHeight
			}

			imgByteIndex := v*e.ImageWidth*2 + (u * 2)
			color := (uint16(e.ImageData[imgByteIndex]) << 8) | (uint16(e.ImageData[imgByteIndex+1]) << 0)

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
