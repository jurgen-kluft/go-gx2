package fx_lens

import (
	"math"

	fx_common "github.com/jurgen-kluft/go-gx2/test-apps/effects/common"
)

type LensEffect struct {
	ImageWidth    int32
	ImageHeight   int32
	Image         []uint16
	DistortionMap []Offset
	Lens          []*Lens
}

type Lens struct {
	LensSize          int32
	LensLUT           []Offset
	LensAnimateRadius float32
	LensAnimateSpeed  float32
	LensAnimateTime   float32
}

func NewEffect(lensSize int32) *LensEffect {
	lens1 := &Lens{
		LensSize:          lensSize,
		LensLUT:           make([]Offset, lensSize*lensSize),
		LensAnimateRadius: 100.0, // Adjust the radius as needed
		LensAnimateSpeed:  1.0,   // Adjust the speed as needed
		LensAnimateTime:   0.0,
	}
	lens1.precalculateLens(0.6) // Adjust the strength as needed

	lens2 := &Lens{
		LensSize:          lensSize,
		LensLUT:           make([]Offset, lensSize*lensSize),
		LensAnimateRadius: 105.0, // Adjust the radius as needed
		LensAnimateSpeed:  1.2,   // Adjust the speed as needed
		LensAnimateTime:   5.0,
	}
	lens2.precalculateLens(0.7) // Adjust the strength as needed

	image, imageWidth, imageHeight, _ := fx_common.ReadPngAsRGB565("lens/image.png")

	effect := &LensEffect{
		Lens:          []*Lens{lens1, lens2},
		Image:         image,
		ImageWidth:    imageWidth,
		ImageHeight:   imageHeight,
		DistortionMap: make([]Offset, imageWidth*imageHeight),
	}

	return effect
}

type Offset struct {
	X int8
	Y int8
}

func (e *Lens) precalculateLens(strength float32) {
	center := e.LensSize / 2
	maxRadius := float32(center)

	for y := int32(0); y < e.LensSize; y++ {
		for x := int32(0); x < e.LensSize; x++ {
			// Find relative distance from the center of the lens
			nx := float32(x - center)
			ny := float32(y - center)
			r := float32(math.Sqrt(float64(nx*nx + ny*ny)))

			// Distort inside the circle; outside remains unaffected
			if r < maxRadius {
				normR := r / maxRadius

				// Barrel distortion curve
				distortion := 1.0 + strength*(normR*normR)

				// Calculate targeted source coordinate mapping
				srcX := int32(float32(center) + nx*distortion)
				srcY := int32(float32(center) + ny*distortion)

				// Assign relative vector to the look-up table
				e.LensLUT[y*e.LensSize+x] = Offset{
					X: int8(srcX - x),
					Y: int8(srcY - y),
				}
			} else {
				e.LensLUT[y*e.LensSize+x] = Offset{X: 0, Y: 0}
			}
		}
	}
}

func (e *Lens) getOrbitingPosition(time float32, radius float32, speed float32, width, height int) (int, int) {
	// 1. Find the exact geometric center of the screen
	screenCenterX := float32(width) / 2.0
	screenCenterY := float32(height) / 2.0

	// 2. Compute orbital angle using time and speed
	angle := time * speed

	dynamicRadius := radius + (50.0 * float32(math.Sin(float64(time)*0.643)))

	// 3. Calculate target position orbiting the center
	// Change one of the radius multipliers to make an oval/ellipse trajectory instead of a perfect circle
	targetCenterX := screenCenterX + (dynamicRadius * float32(math.Cos(float64(angle))))
	targetCenterY := screenCenterY + (dynamicRadius * float32(math.Sin(float64(angle))))

	// 4. Offset by half the lens size (64 pixels) so the LENS CENTER aligns with the path
	// This prevents the lens from feeling off-center during rotation
	halfLens := float32(e.LensSize) / 2.0
	lensX := int(targetCenterX - halfLens)
	lensY := int(targetCenterY - halfLens)

	return lensX, lensY
}

func (e *LensEffect) update(dt float32) {
	for _, lens := range e.Lens {
		lens.LensAnimateTime += dt
	}
}

func (e *LensEffect) applyLens(distortionMap []Offset, lens *Lens, lensX, lensY int) {
	// 1. Iterate over the lens area
	for y := int32(0); y < lens.LensSize; y++ {
		for x := int32(0); x < lens.LensSize; x++ {
			// 2. Calculate the corresponding framebuffer coordinates
			fbX := lensX + int(x)
			fbY := lensY + int(y)

			// 3. Ensure we are within framebuffer bounds
			//    The distortion map is the same size as the framebuffer, so we need to check
			//    bounds here.
			if fbX < 0 || fbX >= int(e.ImageWidth) || fbY < 0 || fbY >= int(e.ImageHeight) {
				continue
			}

			// 4. Get the offset from the LUT
			offset := lens.LensLUT[y*lens.LensSize+x]

			// 7. Update the distortion map with the calculated offset, so we need to add
			//    the offset to the distortion map since we might apply multiple lenses and
			//    we want to accumulate the effect.
			distortionMap[fbY*int(e.ImageWidth)+fbX].X += offset.X
			distortionMap[fbY*int(e.ImageWidth)+fbX].Y += offset.Y
		}
	}
}

func (e *LensEffect) applyDistortionMap(fb *fx_common.FrameBuffer) {
	for y := int32(0); y < e.ImageHeight; y++ {
		for x := int32(0); x < e.ImageWidth; x++ {
			// Get the distortion offset for this pixel
			offset := e.DistortionMap[y*e.ImageWidth+x]

			// Calculate the source pixel coordinates
			srcX := x + int32(offset.X)
			srcY := y + int32(offset.Y)

			// Ensure the source coordinates are within bounds
			if srcX < 0 || srcX >= e.ImageWidth || srcY < 0 || srcY >= e.ImageHeight {
				continue
			}

			// Copy the pixel from the source to the framebuffer
			fb.Pixels[y*e.ImageWidth+x] = e.Image[srcY*e.ImageWidth+srcX]
		}
	}
}

func (e *LensEffect) renderImage(fb *fx_common.FrameBuffer) {
	// Draw the image on the screen, scale it up or down, making sure to center it properly

	dX := (float32(e.ImageWidth) / float32(fb.Width))
	dY := (float32(e.ImageHeight) / float32(fb.Height))

	for y := 0; y < int(fb.Height); y++ {
		for x := 0; x < int(fb.Width); x++ {
			// Calculate the corresponding pixel in the source image
			srcX := int(float32(x) * dX)
			srcY := int(float32(y) * dY)

			// Ensure we are within the source image bounds
			if srcX < 0 || srcX >= int(e.ImageWidth) || srcY < 0 || srcY >= int(e.ImageHeight) {
				continue
			}

			// Copy the pixel from the source image to the framebuffer
			fb.Pixels[y*int(fb.Width)+x] = e.Image[srcY*int(e.ImageWidth)+srcX]
		}
	}
}

func (e *LensEffect) render(fb *fx_common.FrameBuffer) {

	e.renderImage(fb)

	// Clear the distortion map
	// TODO we can optimize this by only clearing the areas where the
	// lenses were applied, but for now we will clear the whole map
	for i := range e.DistortionMap {
		e.DistortionMap[i] = Offset{}
	}

	for _, lens := range e.Lens {
		lensX, lensY := lens.getOrbitingPosition(lens.LensAnimateTime, lens.LensAnimateRadius, lens.LensAnimateSpeed, int(e.ImageWidth), int(e.ImageHeight))
		e.applyLens(e.DistortionMap, lens, lensX, lensY)
	}

	e.applyDistortionMap(fb)
}

func (e *LensEffect) ProcessFrame(dt float32, fb *fx_common.FrameBuffer) {
	e.update(dt)
	e.render(fb)
}
