package fx_plasma

import (
	"math"

	fx_common "github.com/jurgen-kluft/go-gx2/test-apps/effects/common"
)

// Effect holds all state, lookup tables, and constants for the plasma.
type Effect struct {
	width      int32
	height     int32
	PixelSize  int32
	EffectType int32   // 0 = Linear, 1 = Radial
	Zoom       float32 // 1.0 = Default, 2.0 = 2x Zoomed In, 0.5 = Zoomed Out

	// Phase variables for the plasma movement (16.16 fixed-point)
	k1 int32
	k2 int32
	k3 int32
	k4 int32

	// Time accumulator for ensuring exactly 60 FPS logical steps
	timeAccumulator float32
	fixedTimeStep   float32 // Exactly 1.0 / 60.0 (0.01666667)

	// Precomputed tables
	sinTable        [256]int32
	palette         [256]uint16
	distTable       []uint8 // Compressed 0-255 normalized values
	distTableW      int32
	distTableH      int32
	distScaleFactor int32 // Global fixed-point scale factor (16.16 format)

	rotoZoom *RotoZoomEffect // Optional rotozoom effect for advanced transformations
}

// NewEffect initializes tables, creates the palette, and returns the Effect pointer.
func NewEffect() *Effect {
	pSize := int32(2) // 2x2 block rendering for ESP32-S3 performance

	e := &Effect{
		width:           480,
		height:          480,
		PixelSize:       pSize,
		EffectType:      0,   // Default to 1 (Radial)
		Zoom:            2.5, // Default zoom level
		k1:              0,
		k2:              0,
		k3:              0,
		k4:              0,
		timeAccumulator: 0.0,
		fixedTimeStep:   1.0 / 60.0, // Strict 60 FPS step interval (~16.67ms)
	}

	// 1. Generate 16.16 fixed-point Sine Lookup Table
	for i := 0; i < 256; i++ {
		rad := (float64(i) * 2.0 * math.Pi) / 256.0
		e.sinTable[i] = int32(math.Sin(rad) * 127.0 * 65536.0)
	}

	// 2. Generate a 256-color retro psychedelic palette (RGB565)
	for i := 0; i < 256; i++ {
		rad := (float64(i) * 2.0 * math.Pi) / 256.0
		r := uint16((math.Sin(rad) * 127) + 128)
		g := uint16((math.Sin(rad+(2.0*math.Pi/3.0)) * 127) + 128)
		b := uint16((math.Sin(rad+(4.0*math.Pi/3.0)) * 127) + 128)
		e.palette[i] = ((r >> 3) << 11) | ((g >> 2) << 5) | (b >> 3)
	}

	// 3. Precalculate Symmetrical Quadrant Distance Table with Normalization
	halfW := (e.width / 2) / pSize
	halfH := (e.height / 2) / pSize
	e.distTableW = halfW
	e.distTableH = halfH
	e.distTable = make([]uint8, halfW*halfH)

	maxDist := math.Sqrt(float64((e.width/2)*(e.width/2) + (e.height/2)*(e.height/2)))
	e.distScaleFactor = int32((maxDist / 255.0) * 65536.0)

	idx := 0
	for y := int32(0); y < halfH; y++ {
		realY := y * pSize
		dy := float64(realY - (e.height / 2))
		dySq := dy * dy

		for x := int32(0); x < halfW; x++ {
			realX := x * pSize
			dx := float64(realX - (e.width / 2))
			dist := math.Sqrt(dx*dx + dySq)

			normalizedDist := (dist / maxDist) * 255.0
			e.distTable[idx] = uint8(normalizedDist)
			idx++
		}
	}

	// 4. Initialize optional rotozoom effect for advanced transformations
	e.rotoZoom = NewRotoZoomEffect(128, 128) // 128x128 offscreen texture for rotozoom

	return e
}

// ProcessFrame steps the simulation and updates the display framebuffer.
func (e *Effect) ProcessFrame(dt float32, fb *fx_common.FrameBuffer) {
	if dt > 0.1 {
		dt = 0.1
	}

	// Step the wave positioning logic exactly at 60 FPS markers.
	// If a frame drops below 16.6ms on the hardware side, this catches up naturally.
	// If it renders too fast, it skips math updates until the next true step.
	e.simulate(dt)
	e.render(fb)
}

// simulate updates the movement phase offsets using fixed-point step increments.
func (e *Effect) simulate(dt float32) {
	n := 4

	e.k1 += int32(dt * 40.0 * 65536.0)
	if e.k1 > int32(n*256*65536) {
		e.k1 -= int32(n * 256 * 65536)
	}

	e.k2 += int32(dt * 30.0 * 65536.0)
	if e.k2 > int32(n*256*65536) {
		e.k2 -= int32(n * 256 * 65536)
	}

	e.k3 += int32(dt * 50.0 * 65536.0)
	if e.k3 > int32(n*256*65536) {
		e.k3 -= int32(n * 256 * 65536)
	}

	e.k4 += int32(dt * 20.0 * 65536.0)
	if e.k4 > int32(n*256*65536) {
		e.k4 -= int32(n * 256 * 65536)
	}

	if e.rotoZoom != nil {
		e.rotoZoom.SimulateStep(dt)
	}
}

// renderPlasmaDirect handles direct rendering onto the main screen viewport via the lookup tables.
func (e *Effect) renderToTexture(targetWidth, targetHeight, pSize int32, targetPixels []uint16) {
	cx1 := (e.k1 >> 16) & 0xFF
	cx2 := (e.k2 >> 16) & 0xFF
	cy1 := (e.k3 >> 16) & 0xFF
	cy2 := (e.k4 >> 16) & 0xFF

	centerX := targetWidth / 2
	centerY := targetHeight / 2
	gridXMax := targetWidth / pSize
	gridYMax := targetHeight / pSize

	zFactor := e.Zoom
	if zFactor < 0.01 {
		zFactor = 0.01
	}
	invZoomFP := int32((1.0 / zFactor) * 65536.0)

	if e.EffectType == 1 {
		// ==========================================
		// CLEAN RADIAL DIRECT PLASMA LOOP (BRANCH-FREE)
		// ==========================================
		for gy := int32(0); gy < gridYMax; gy++ {
			y := gy * pSize
			rowStartIdx := y * targetWidth

			lutY := abs((y - centerY) / pSize)
			if lutY >= e.distTableH {
				lutY = e.distTableH - 1
			}
			lutRowStart := lutY * e.distTableW

			for gx := int32(0); gx < gridXMax; gx++ {
				x := gx * pSize

				lutX := abs((x - centerX) / pSize)
				if lutX >= e.distTableW {
					lutX = e.distTableW - 1
				}

				normDist := int32(e.distTable[lutRowStart+lutX])
				baseDist := (normDist * e.distScaleFactor) >> 16
				dist := (baseDist * invZoomFP) >> 16

				r1 := e.sinTable[(dist+cx1)&0xFF] >> 16
				r2 := e.sinTable[((dist/2)+cy1)&0xFF] >> 16
				r3 := e.sinTable[((dist*2)+cx2)&0xFF] >> 16
				paletteIdx := uint8(r1 + r2 + r3 + cy2)

				color := e.palette[paletteIdx]

				for blockY := int32(0); blockY < pSize && (y+blockY) < targetHeight; blockY++ {
					destRow := (rowStartIdx + (blockY * targetWidth)) + x
					for blockX := int32(0); blockX < pSize && (x+blockX) < targetWidth; blockX++ {
						targetPixels[destRow+blockX] = color
					}
				}
			}
		}
	} else {
		// ==========================================
		// CLEAN LINEAR DIRECT PLASMA LOOP (BRANCH-FREE)
		// ==========================================
		for gy := int32(0); gy < gridYMax; gy++ {
			y := gy * pSize
			rowStartIdx := y * targetWidth
			scaledY := (y * invZoomFP) >> 16

			s2 := e.sinTable[(scaledY+cy1)&0xFF] >> 16
			s4 := e.sinTable[((scaledY*2)+cy2)&0xFF] >> 16

			for gx := int32(0); gx < gridXMax; gx++ {
				x := gx * pSize
				scaledX := (x * invZoomFP) >> 16

				s1 := e.sinTable[(scaledX+cx1)&0xFF] >> 16
				s3 := e.sinTable[((scaledX*3)+cx2)&0xFF] >> 16
				paletteIdx := uint8(s1 + s2 + s3 + s4)

				color := e.palette[paletteIdx]

				for blockY := int32(0); blockY < pSize && (y+blockY) < targetHeight; blockY++ {
					destRow := (rowStartIdx + (blockY * targetWidth)) + x
					for blockX := int32(0); blockX < pSize && (x+blockX) < targetWidth; blockX++ {
						targetPixels[destRow+blockX] = color
					}
				}
			}
		}
	}
}

func abs(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

// render calculates individual pixel colors via fixed-point plasma algorithms.
func (e *Effect) render(fb *fx_common.FrameBuffer) {
	if e.rotoZoom != nil {
		// Target isolated offscreen scratching texture buffer (forces pixel size grain block to 1)
		tex := e.rotoZoom.GetTexture()
		e.renderToTexture(tex.Width, tex.Height, 1, tex.Pixels)
		//e.checkerBoardRender(tex.Width, tex.Height, tex.Pixels) // For testing purposes, render a checkerboard pattern

		// Run affine transformation to stamp scratch vector pixels onto hardware layout
		e.rotoZoom.Render(fb, e.k3)
	} else {
		// Fallback: draw directly to screen viewport dimensions applying custom macro pixel bounds
		e.renderToTexture(fb.Width, fb.Height, e.PixelSize, fb.Pixels)
	}
}

func (e *Effect) checkerBoardRender(targetWidth, targetHeight int32, targetPixels []uint16) {
	color := uint16(0xFFFF) // White color for checkerboard
	for y := int32(0); y < targetHeight; y++ {
		if (y & 63) == 0 {
			color ^= 0xFFFF // Toggle color every 64 rows
		}

		rowColor := color
		for x := int32(0); x < targetWidth; x++ {
			if (x & 63) == 0 {
				rowColor ^= 0xFFFF // Toggle color every 64 columns
			}
			targetPixels[y*targetWidth+x] = rowColor
		}
	}
}
