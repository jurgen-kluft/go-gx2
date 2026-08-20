package fx_wormhole

import (
	"fmt"
	"math"

	fx_common "github.com/jurgen-kluft/go-gx2/test-apps/effects/common"
)

const motionPeriod float32 = 2 * math.Pi

const (
	defaultWidth     int32 = 256
	defaultHeight    int32 = 256
	defaultPixelSize int32 = 2
)

type WormholeEffect struct {
	Width     int32
	Height    int32
	PixelSize int32

	// Image should be a power-of-2
	Image    *ImageRgb565
	DistMap  []uint8 // Precomputed distance map for one quadrant
	AngleMap []uint8 // Precomputed base angle map for one quadrant
	FadeMap  []uint8 // Precomputed brightness map for one quadrant

	RotateSpeed float32 // Unit = radians per second
	MoveSpeed   float32 // Unit = pixels per second
	AngleOffset float32 // Unit = radians, used to rotate the wormhole effect over time
	DistOffset  float32 // Unit = pixels, used to move the wormhole effect over time
	MotionTime  float32 // Unit = seconds, used to animate the wormhole center
	ShiftX      float32 // Unit = pixels, used to shift the wormhole effect horizontally over time
	ShiftY      float32 // Unit = pixels, used to shift the wormhole effect vertically over time
}

type renderViewport struct {
	sourceX0 int32
	sourceY0 int32
	sourceX1 int32
	sourceY1 int32
	destX    int32
	destY    int32
}

func NewEffect() *WormholeEffect {
	return NewEffectWithSize(defaultWidth, defaultHeight, defaultPixelSize)
}

func NewEffectWithSize(width, height, pixelSize int32) *WormholeEffect {
	width = normalizeEffectSize(width, defaultWidth)
	height = normalizeEffectSize(height, defaultHeight)
	if pixelSize <= 0 {
		pixelSize = 1
	}

	fmt.Printf("Creating WormholeEffect with size %dx%d at pixel size %d\n", width, height, pixelSize)

	fx := &WormholeEffect{
		Width:     width,
		Height:    height,
		PixelSize: pixelSize,
		Image:     GetImage(),

		RotateSpeed: 32,  // Speed of rotation for the wormhole effect
		MoveSpeed:   32,  // Speed of movement for the wormhole effect
		AngleOffset: 0.0, // Initial angle offset for rotation
	}

	// Generate the distance and angle maps for one quadrant.
	fx.DistMap = make([]uint8, fx.Height*fx.Width)
	fx.AngleMap = make([]uint8, fx.Height*fx.Width)
	fx.FadeMap = make([]uint8, fx.Height*fx.Width)

	// First generate the distance map, which is done by calculating the distance from the center of the
	// screen to each pixel and mapping it to a value that can be used to sample the wormhole texture.
	// Knowing UV mapping, this is calculating the U coordinate for each pixel based on its distance from
	// the center of the screen.
	// The distance is calculated using the Pythagorean theorem, and then scaled to fit within the range of
	// the wormhole texture's width.

	distmap := fx.DistMap
	fademap := fx.FadeMap
	maxX := float64(fx.Width) - 0.5
	maxY := float64(fx.Height) - 0.5
	maxDistance := math.Sqrt(maxX*maxX + maxY*maxY)
	for y := float64(0.5); y < float64(fx.Height); y++ {
		for x := float64(0.5); x < float64(fx.Width); x++ {
			distance := math.Sqrt(x*x + y*y)
			mapIndex := int32(y)*fx.Width + int32(x)
			distmap[mapIndex] = uint8((int32)(float64(2*fx.Image.Width*(fx.Width/2))/distance) % fx.Image.Width)
			fademap[mapIndex] = fadeBrightness(distance, maxDistance)
		}
	}

	// Second generate the angle map, which is done by calculating the angle of each pixel relative to the center of the
	// screen and mapping it to a value that can be used to sample the wormhole texture.
	// Knowing UV mapping, this is calculating the V coordinate for each pixel based on its angle from the
	// center of the screen. The angle is calculated using the arctangent function (math.Atan2), which returns
	// the angle in radians between the positive x-axis and the point (xx, yy). This angle is then normalized
	// to a range of [0, 1] and scaled to fit within the range of the wormhole texture's width.

	anglemap := fx.AngleMap
	for y := float64(0.5); y < float64(fx.Height); y++ {
		for x := float64(0.5); x < float64(fx.Width); x++ {
			angle := math.Atan2(y, x) / math.Pi
			anglemap[int32(y)*fx.Width+int32(x)] = uint8(int32(math.Round(float64(8*fx.Image.Height)*angle)) % fx.Image.Height)
		}
	}

	return fx
}

func fadeBrightness(distance, maxDistance float64) uint8 {
	minDistance := math.Sqrt(0.5)
	if distance <= minDistance || maxDistance <= minDistance {
		return 0
	}
	if distance >= maxDistance {
		return 255
	}

	// 1. Linear normalization
	normalized := (distance - minDistance) / (maxDistance - minDistance)

	// 2. Apply curve (Example: Quarter Sine)
	//curved := math.Sin(normalized * math.Pi / 2)
	N := 3.0 // Adjust this value to control the curve's steepness
	curved := 1.0 - math.Pow(1.0-normalized, N)

	// 3. Convert to uint8
	fadingValue := uint8(math.Round(255.0 * curved))

	return fadingValue
}

func normalizeEffectSize(size, fallback int32) int32 {
	if size <= 0 {
		return fallback
	}

	result := int32(1)
	for result < size {
		result <<= 1
	}
	return result
}

func (fx *WormholeEffect) viewport(fb *fx_common.FrameBuffer) (renderViewport, bool) {
	if fb == nil || fx.PixelSize <= 0 || fb.Width <= 0 || fb.Height <= 0 ||
		fb.Width%fx.PixelSize != 0 || fb.Height%fx.PixelSize != 0 ||
		int64(len(fb.Pixels)) < int64(fb.Width)*int64(fb.Height) {
		return renderViewport{}, false
	}

	capacityX := fb.Width / fx.PixelSize
	capacityY := fb.Height / fx.PixelSize
	visibleWidth := min(fx.Width, capacityX)
	visibleHeight := min(fx.Height, capacityY)
	sourceX0 := (fx.Width - visibleWidth) / 2
	sourceY0 := (fx.Height - visibleHeight) / 2
	return renderViewport{
		sourceX0: sourceX0,
		sourceY0: sourceY0,
		sourceX1: sourceX0 + visibleWidth,
		sourceY1: sourceY0 + visibleHeight,
		destX:    (fb.Width - visibleWidth*fx.PixelSize) / 2,
		destY:    (fb.Height - visibleHeight*fx.PixelSize) / 2,
	}, true
}

func (fx *WormholeEffect) update(dt float32) {
	fx.MotionTime = float32(math.Mod(float64(fx.MotionTime+dt), float64(motionPeriod)))
	if fx.MotionTime < 0 {
		fx.MotionTime += motionPeriod
	}

	fx.AngleOffset += fx.RotateSpeed * dt
	if fx.AngleOffset < 0 {
		fx.AngleOffset += float32(fx.Image.Width)
	} else if fx.AngleOffset > float32(fx.Image.Width) {
		fx.AngleOffset -= float32(fx.Image.Width)
	}

	fx.DistOffset += fx.MoveSpeed * dt
	if fx.DistOffset < 0 {
		fx.DistOffset += float32(fx.Image.Width)
	} else if fx.DistOffset > float32(fx.Image.Width) {
		fx.DistOffset -= float32(fx.Image.Width)
	}

	motionTime := float64(fx.MotionTime)
	fx.ShiftX = (float32(fx.Width)/12.0)*float32(math.Sin(2.0*motionTime)) + (float32(fx.Width) / 2.0)
	fx.ShiftY = (float32(fx.Height)/12.0)*float32(math.Cos(3.0*motionTime)) + (float32(fx.Height) / 2.0)
}

func (fx *WormholeEffect) render(fb *fx_common.FrameBuffer) {
	viewport, ok := fx.viewport(fb)
	if !ok {
		return
	}

	offsetU := int32(fx.DistOffset)
	offsetV := int32(fx.AngleOffset)
	sx := int32(fx.ShiftX)
	sy := int32(fx.ShiftY)
	centerX := min(max(fx.Width-sx, 0), fx.Width)
	centerY := min(max(fx.Height-sy, 0), fx.Height)
	splitX := min(max(centerX, viewport.sourceX0), viewport.sourceX1)
	splitY := min(max(centerY, viewport.sourceY0), viewport.sourceY1)
	textureMaskU := fx.Image.Width - 1
	textureMaskV := fx.Image.Height - 1

	for y := viewport.sourceY0; y < splitY; y++ {
		mapRow := (centerY - y - 1) * fx.Width
		destY := viewport.destY + (y-viewport.sourceY0)*fx.PixelSize
		for x := viewport.sourceX0; x < splitX; x++ {
			mapIndex := mapRow + centerX - x - 1
			u := (int32(fx.DistMap[mapIndex]) + offsetU) & textureMaskU
			v := (int32(fx.AngleMap[mapIndex]) + offsetV) & textureMaskV
			destX := viewport.destX + (x-viewport.sourceX0)*fx.PixelSize
			color := scaleRGB565(fx.Image.Data[v*fx.Image.Width+u], fx.FadeMap[mapIndex])
			writePixelBlock(fb, destX, destY, fx.PixelSize, color)
		}
	}

	for y := viewport.sourceY0; y < splitY; y++ {
		mapRow := (centerY - y - 1) * fx.Width
		destY := viewport.destY + (y-viewport.sourceY0)*fx.PixelSize
		for x := splitX; x < viewport.sourceX1; x++ {
			mapIndex := mapRow + x - centerX
			u := (int32(fx.DistMap[mapIndex]) + offsetU) & textureMaskU
			v := (offsetV - int32(fx.AngleMap[mapIndex])) & textureMaskV
			destX := viewport.destX + (x-viewport.sourceX0)*fx.PixelSize
			color := scaleRGB565(fx.Image.Data[v*fx.Image.Width+u], fx.FadeMap[mapIndex])
			writePixelBlock(fb, destX, destY, fx.PixelSize, color)
		}
	}

	for y := splitY; y < viewport.sourceY1; y++ {
		mapRow := (y - centerY) * fx.Width
		destY := viewport.destY + (y-viewport.sourceY0)*fx.PixelSize
		for x := viewport.sourceX0; x < splitX; x++ {
			mapIndex := mapRow + centerX - x - 1
			u := (int32(fx.DistMap[mapIndex]) + offsetU) & textureMaskU
			v := (offsetV - int32(fx.AngleMap[mapIndex])) & textureMaskV
			destX := viewport.destX + (x-viewport.sourceX0)*fx.PixelSize
			color := scaleRGB565(fx.Image.Data[v*fx.Image.Width+u], fx.FadeMap[mapIndex])
			writePixelBlock(fb, destX, destY, fx.PixelSize, color)
		}
	}

	for y := splitY; y < viewport.sourceY1; y++ {
		mapRow := (y - centerY) * fx.Width
		destY := viewport.destY + (y-viewport.sourceY0)*fx.PixelSize
		for x := splitX; x < viewport.sourceX1; x++ {
			mapIndex := mapRow + x - centerX
			u := (int32(fx.DistMap[mapIndex]) + offsetU) & textureMaskU
			v := (int32(fx.AngleMap[mapIndex]) + offsetV) & textureMaskV
			destX := viewport.destX + (x-viewport.sourceX0)*fx.PixelSize
			color := scaleRGB565(fx.Image.Data[v*fx.Image.Width+u], fx.FadeMap[mapIndex])
			writePixelBlock(fb, destX, destY, fx.PixelSize, color)
		}
	}
}

func scaleRGB565(color uint16, brightness uint8) uint16 {
	red := ((uint32(color>>11) & 0x1f) * (uint32(brightness) + 1)) >> 8
	green := ((uint32(color>>5) & 0x3f) * (uint32(brightness) + 1)) >> 8
	blue := (uint32(color&0x1f) * (uint32(brightness) + 1)) >> 8
	return uint16(red<<11 | green<<5 | blue)
}

func writePixelBlock(fb *fx_common.FrameBuffer, x, y, pixelSize int32, color uint16) {
	for blockY := int32(0); blockY < pixelSize; blockY++ {
		row := (y + blockY) * fb.Width
		for blockX := int32(0); blockX < pixelSize; blockX++ {
			fb.Pixels[row+x+blockX] = color
		}
	}
}

func (fx *WormholeEffect) ProcessFrame(dt float32, fb *fx_common.FrameBuffer) {
	fx.update(dt) // Update the wormhole effect's state based on the elapsed time
	fx.render(fb) // Render the wormhole effect to the framebuffer
}
