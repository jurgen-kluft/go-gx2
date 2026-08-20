package fx_wormhole

import (
	"math"
	"testing"

	fx_common "github.com/jurgen-kluft/go-gx2/test-apps/effects/common"
)

func TestNewEffectUsesDefaultPowerOfTwoGrid(t *testing.T) {
	fx := NewEffect()

	if fx.Width != 256 || fx.Height != 256 || fx.PixelSize != 2 {
		t.Fatalf("default effect = %dx%d at pixel size %d, want 256x256 at pixel size 2", fx.Width, fx.Height, fx.PixelSize)
	}
}

func TestNewEffectWithSizeRoundsToPowersOfTwo(t *testing.T) {
	fx := NewEffectWithSize(300, 129, 0)

	if fx.Width != 512 || fx.Height != 256 || fx.PixelSize != 1 {
		t.Fatalf("custom effect = %dx%d at pixel size %d, want 512x256 at pixel size 1", fx.Width, fx.Height, fx.PixelSize)
	}
}

func TestViewportCrops256GridTo480Framebuffer(t *testing.T) {
	fx := NewEffect()
	fb := &fx_common.FrameBuffer{
		Width:  480,
		Height: 480,
		Pixels: make([]uint16, 480*480),
	}

	viewport, ok := fx.viewport(fb)
	if !ok {
		t.Fatal("viewport rejected compatible framebuffer")
	}
	if viewport != (renderViewport{sourceX0: 8, sourceY0: 8, sourceX1: 248, sourceY1: 248}) {
		t.Fatalf("viewport = %+v, want source [8,248) on both axes at destination origin", viewport)
	}
}

func TestViewportCentersSmallerEffectInFramebuffer(t *testing.T) {
	fx := NewEffectWithSize(64, 32, 2)
	fb := &fx_common.FrameBuffer{
		Width:  160,
		Height: 96,
		Pixels: make([]uint16, 160*96),
	}

	viewport, ok := fx.viewport(fb)
	if !ok {
		t.Fatal("viewport rejected compatible framebuffer")
	}
	want := renderViewport{sourceX1: 64, sourceY1: 32, destX: 16, destY: 16}
	if viewport != want {
		t.Fatalf("viewport = %+v, want %+v", viewport, want)
	}
}

func TestRenderReplicatesLogicalPixelsAndPreservesPadding(t *testing.T) {
	fx, _ := newQuadrantTestEffect(2, 2, 1, 1)
	fx.PixelSize = 2
	fb := &fx_common.FrameBuffer{
		Width:  8,
		Height: 8,
		Pixels: make([]uint16, 8*8),
	}

	fx.render(fb)

	for y := int32(0); y < fb.Height; y++ {
		for x := int32(0); x < fb.Width; x++ {
			pixel := fb.Pixels[y*fb.Width+x]
			inside := x >= 2 && x < 6 && y >= 2 && y < 6
			if inside && pixel == 0 {
				t.Fatalf("scaled output pixel (%d,%d) was not rendered", x, y)
			}
			if !inside && pixel != 0 {
				t.Fatalf("padding pixel (%d,%d) was modified", x, y)
			}
		}
	}
	for blockY := int32(0); blockY < 2; blockY++ {
		for blockX := int32(0); blockX < 2; blockX++ {
			x := 2 + blockX*2
			y := 2 + blockY*2
			color := fb.Pixels[y*fb.Width+x]
			for py := int32(0); py < 2; py++ {
				for px := int32(0); px < 2; px++ {
					if actual := fb.Pixels[(y+py)*fb.Width+x+px]; actual != color {
						t.Fatalf("block at (%d,%d) is not uniform", x, y)
					}
				}
			}
		}
	}
}

func TestRenderLeavesIncompatibleFramebufferUnchanged(t *testing.T) {
	fx, _ := newQuadrantTestEffect(4, 4, 2, 2)
	fx.PixelSize = 2

	for _, test := range []struct {
		name   string
		width  int32
		height int32
		pixels int
	}{
		{name: "non-divisible width", width: 7, height: 8, pixels: 56},
		{name: "undersized storage", width: 8, height: 8, pixels: 63},
	} {
		t.Run(test.name, func(t *testing.T) {
			pixels := make([]uint16, test.pixels)
			for index := range pixels {
				pixels[index] = 0x1234
			}
			fb := &fx_common.FrameBuffer{Width: test.width, Height: test.height, Pixels: pixels}

			fx.render(fb)

			for index, pixel := range pixels {
				if pixel != 0x1234 {
					t.Fatalf("pixel %d changed to %x", index, pixel)
				}
			}
		})
	}
}

func TestNewEffectBuildsCompactByteMaps(t *testing.T) {
	const width, height = int32(7), int32(5)

	fx := NewEffectWithSize(width, height, 1)

	if len(fx.DistMap) != int(fx.Width*fx.Height) {
		t.Fatalf("distance map length = %d, want %d", len(fx.DistMap), fx.Width*fx.Height)
	}
	if len(fx.AngleMap) != int(fx.Width*fx.Height) {
		t.Fatalf("angle map length = %d, want %d", len(fx.AngleMap), fx.Width*fx.Height)
	}
	if len(fx.FadeMap) != int(fx.Width*fx.Height) {
		t.Fatalf("fade map length = %d, want %d", len(fx.FadeMap), fx.Width*fx.Height)
	}
	for index := range fx.DistMap {
		if int32(fx.DistMap[index]) >= fx.Image.Width {
			t.Fatalf("distance map value %d at %d exceeds texture width", fx.DistMap[index], index)
		}
		if int32(fx.AngleMap[index]) >= fx.Image.Height {
			t.Fatalf("angle map value %d at %d exceeds texture height", fx.AngleMap[index], index)
		}
	}
}

func TestFadeMapUsesSquareRootWholeQuadrantBrightness(t *testing.T) {
	fx := NewEffectWithSize(16, 16, 1)

	if fx.FadeMap[0] != 0 {
		t.Fatalf("center brightness = %d, want 0", fx.FadeMap[0])
	}
	last := len(fx.FadeMap) - 1
	if fx.FadeMap[last] != 255 {
		t.Fatalf("corner brightness = %d, want 255", fx.FadeMap[last])
	}

	previous := uint8(0)
	for coordinate := int32(0); coordinate < fx.Width; coordinate++ {
		brightness := fx.FadeMap[coordinate*fx.Width+coordinate]
		if brightness < previous {
			t.Fatalf("brightness decreased at diagonal coordinate %d: %d < %d", coordinate, brightness, previous)
		}
		previous = brightness
	}

	maxCoordinate := float64(fx.Width) - 0.5
	maxDistance := math.Sqrt(2.0 * maxCoordinate * maxCoordinate)
	minDistance := math.Sqrt(0.5)
	midpoint := fadeBrightness(minDistance+(maxDistance-minDistance)/2.0, maxDistance)
	if midpoint < 180 || midpoint > 181 {
		t.Fatalf("half-distance brightness = %d, want approximately 180", midpoint)
	}
}

func TestScaleRGB565(t *testing.T) {
	const color = uint16(0xfbef)

	if actual := scaleRGB565(color, 0); actual != 0 {
		t.Fatalf("zero brightness = %04x, want 0000", actual)
	}
	if actual := scaleRGB565(color, 255); actual != color {
		t.Fatalf("full brightness = %04x, want %04x", actual, color)
	}

	half := scaleRGB565(0xffff, 128)
	red := (half >> 11) & 0x1f
	green := (half >> 5) & 0x3f
	blue := half & 0x1f
	if red != 16 || green != 32 || blue != 16 {
		t.Fatalf("half-brightness white channels = (%d,%d,%d), want (16,32,16)", red, green, blue)
	}

	if actual := scaleRGB565(0xffff, 5); actual == 0 {
		t.Fatal("low brightness rounded all white channels to zero")
	}
}

func TestRenderAppliesSameFadeAcrossQuadrants(t *testing.T) {
	fx, _ := newQuadrantTestEffect(4, 4, 2, 2)
	fx.PixelSize = 2
	for index := range fx.Image.Data {
		fx.Image.Data[index] = 0xffff
	}
	for index := range fx.FadeMap {
		fx.FadeMap[index] = 128
	}
	fb := &fx_common.FrameBuffer{
		Width:  8,
		Height: 8,
		Pixels: make([]uint16, 8*8),
	}
	value := scaleRGB565(0xffff, 128)

	fx.render(fb)

	for _, point := range [][2]int32{{0, 0}, {3, 0}, {0, 3}, {3, 3}} {
		logicalX, logicalY := point[0], point[1]
		x := logicalX * fx.PixelSize
		y := logicalY * fx.PixelSize
		for py := int32(0); py < fx.PixelSize; py++ {
			for px := int32(0); px < fx.PixelSize; px++ {
				actual := fb.Pixels[(y+py)*fb.Width+x+px]
				if actual != value {
					t.Fatalf("faded block for logical pixel (%d,%d) contains %04x, want %04x", logicalX, logicalY, actual, value)
				}
			}
		}
	}
}

func TestRenderAppliesFixedQuadrantAngleDirections(t *testing.T) {
	fx, fb := newQuadrantTestEffect(5, 5, 2, 2)

	fx.render(fb)

	assertPixel(t, fb, 0, 0, textureCoordinateColor(3, 5))
	assertPixel(t, fb, 4, 0, textureCoordinateColor(3, 27))
	assertPixel(t, fb, 0, 4, textureCoordinateColor(3, 27))
	assertPixel(t, fb, 4, 4, textureCoordinateColor(3, 5))
	assertPixel(t, fb, 2, 2, textureCoordinateColor(3, 5))

	for index, pixel := range fb.Pixels {
		if pixel == 0 {
			t.Fatalf("framebuffer pixel %d was not rendered", index)
		}
	}
}

func TestRenderClampsCenterSplitToMapCoverage(t *testing.T) {
	for _, test := range []struct {
		name    string
		centerX int32
		centerY int32
	}{
		{name: "above left", centerX: -20, centerY: -20},
		{name: "below right", centerX: 20, centerY: 20},
	} {
		t.Run(test.name, func(t *testing.T) {
			fx, fb := newQuadrantTestEffect(5, 3, test.centerX, test.centerY)

			fx.render(fb)

			for index, pixel := range fb.Pixels {
				if pixel == 0 {
					t.Fatalf("framebuffer pixel %d was not rendered", index)
				}
			}
		})
	}
}

func TestUpdateKeepsCenterContinuousWhenDistanceOffsetWraps(t *testing.T) {
	fx := &WormholeEffect{
		Width:      960,
		Height:     960,
		PixelSize:  1,
		Image:      &ImageRgb565{Width: 32, Height: 32},
		MoveSpeed:  32,
		MotionTime: 0.99,
		DistOffset: 31.75,
	}

	fx.update(0.01)

	assertClose(t, "distance offset", float64(fx.DistOffset), 0.07, 0.00001)
	assertClose(t, "motion time", float64(fx.MotionTime), 1.0, 0.000001)
	assertClose(t, "shift x", float64(fx.ShiftX), 480.0+80.0*math.Sin(2.0), 0.0001)
	assertClose(t, "shift y", float64(fx.ShiftY), 480.0+80.0*math.Cos(3.0), 0.0001)
}

func TestUpdateWrapsMotionTimeWithoutCenterJump(t *testing.T) {
	const distanceToBoundary = float32(0.0001)

	fx := &WormholeEffect{
		Width:      960,
		Height:     720,
		PixelSize:  1,
		Image:      &ImageRgb565{Width: 32, Height: 32},
		MotionTime: motionPeriod - distanceToBoundary,
	}

	fx.update(0)
	beforeX, beforeY := fx.ShiftX, fx.ShiftY
	fx.update(2 * distanceToBoundary)

	assertClose(t, "wrapped motion time", float64(fx.MotionTime), float64(distanceToBoundary), 0.000001)
	if math.Abs(float64(fx.ShiftX-beforeX)) > 0.05 {
		t.Fatalf("shift x jumped across motion wrap: before=%f after=%f", beforeX, fx.ShiftX)
	}
	if math.Abs(float64(fx.ShiftY-beforeY)) > 0.05 {
		t.Fatalf("shift y jumped across motion wrap: before=%f after=%f", beforeY, fx.ShiftY)
	}
}

func TestUpdateKeepsMotionTimeBoundedAndCenterPreciseOverTime(t *testing.T) {
	const frameCount = 60 * 60 * 10
	const dt = float32(1.0 / 60.0)

	fx := &WormholeEffect{
		Width:     960,
		Height:    720,
		PixelSize: 1,
		Image:     &ImageRgb565{Width: 32, Height: 32},
		MoveSpeed: 32,
	}

	for range frameCount {
		fx.update(dt)
		if fx.MotionTime < 0 || fx.MotionTime >= motionPeriod {
			t.Fatalf("motion time outside range: %f", fx.MotionTime)
		}
	}

	expectedTime := float64(fx.MotionTime)
	expectedX := 480.0 + 80.0*math.Sin(2.0*expectedTime)
	expectedY := 360.0 + 60.0*math.Cos(3.0*expectedTime)

	assertClose(t, "shift x", float64(fx.ShiftX), expectedX, 0.0001)
	assertClose(t, "shift y", float64(fx.ShiftY), expectedY, 0.0001)

	if fx.ShiftX < 400 || fx.ShiftX > 560 {
		t.Fatalf("shift x outside orbit bounds: %f", fx.ShiftX)
	}
	if fx.ShiftY < 300 || fx.ShiftY > 420 {
		t.Fatalf("shift y outside orbit bounds: %f", fx.ShiftY)
	}
}

func TestUpdateNormalizesLargeMotionDelta(t *testing.T) {
	fx := &WormholeEffect{
		Width:     960,
		Height:    720,
		PixelSize: 1,
		Image:     &ImageRgb565{Width: 32, Height: 32},
	}
	dt := 1000*motionPeriod + 0.25

	fx.update(dt)

	expectedTime := math.Mod(float64(dt), float64(motionPeriod))
	assertClose(t, "motion time", float64(fx.MotionTime), expectedTime, 0.000001)
	if fx.MotionTime < 0 || fx.MotionTime >= motionPeriod {
		t.Fatalf("motion time outside range: %f", fx.MotionTime)
	}
}

func assertClose(t *testing.T, name string, actual, expected, tolerance float64) {
	t.Helper()
	if math.Abs(actual-expected) > tolerance {
		t.Fatalf("%s = %f, want %f (+/- %f)", name, actual, expected, tolerance)
	}
}

func newQuadrantTestEffect(width, height, centerX, centerY int32) (*WormholeEffect, *fx_common.FrameBuffer) {
	image := &ImageRgb565{
		Width:  32,
		Height: 32,
		Data:   make([]uint16, 32*32),
	}
	for v := int32(0); v < image.Height; v++ {
		for u := int32(0); u < image.Width; u++ {
			image.Data[v*image.Width+u] = textureCoordinateColor(u, v)
		}
	}

	fx := &WormholeEffect{
		Width:     width,
		Height:    height,
		PixelSize: 1,
		Image:     image,
		DistMap:   make([]uint8, width*height),
		AngleMap:  make([]uint8, width*height),
		FadeMap:   make([]uint8, width*height),
		ShiftX:    float32(width - centerX),
		ShiftY:    float32(height - centerY),
	}
	for index := range fx.DistMap {
		fx.DistMap[index] = 3
		fx.AngleMap[index] = 5
		fx.FadeMap[index] = 255
	}

	fb := &fx_common.FrameBuffer{
		Width:  width,
		Height: height,
		Pixels: make([]uint16, width*height),
	}
	return fx, fb
}

func textureCoordinateColor(u, v int32) uint16 {
	return uint16(v*32 + u + 1)
}

func assertPixel(t *testing.T, fb *fx_common.FrameBuffer, x, y int32, expected uint16) {
	t.Helper()
	if actual := fb.Pixels[y*fb.Width+x]; actual != expected {
		t.Fatalf("pixel (%d,%d) = %d, want %d", x, y, actual, expected)
	}
}
