package fx_plasma

import (
	"math"

	fx_common "github.com/jurgen-kluft/go-gx2/test-apps/effects/common"
)

// Texture represents a generic offscreen intermediate render surface.
type Texture struct {
	Width  int32
	Height int32
	MaskX  int32    // Bitwise mask for fast texture X wrap-around (Width - 1)
	MaskY  int32    // Bitwise mask for fast texture Y wrap-around (Height - 1)
	Pixels []uint16 // RGB565 pixel data
}

// RotoZoomEffect isolates all logic, internal memory surfaces, and camera matrices.
type RotoZoomEffect struct {
	rotationAngle float32 // 16.16 fixed point rotation position counter
	rotationSpeed float32 // Rotation speed in degrees per frame
	texture       *Texture
	sinTable      [256]int32
}

// NewRotoZoomEffect initializes an isolated texture transformation engine.
func NewRotoZoomEffect(texW, texH int32) *RotoZoomEffect {
	effect := &RotoZoomEffect{
		rotationAngle: 0,
		rotationSpeed: 20.0, // Default rotation speed in degrees per second
		texture: &Texture{
			Width:  texW,
			Height: texH,
			MaskX:  texW - 1,
			MaskY:  texH - 1,
			Pixels: make([]uint16, texW*texH),
		},
	}

	// 1. Generate 16.16 fixed-point Sine Lookup Table
	for i := 0; i < 256; i++ {
		rad := (float64(i) * 2.0 * math.Pi) / 256.0
		effect.sinTable[i] = int32(math.Sin(rad) * 65536.0)
	}

	return effect
}

// SimulateStep advances camera positions by a fixed fraction step per logic tick.
func (rz *RotoZoomEffect) SimulateStep(dt float32) {
	rz.rotationAngle = rz.rotationAngle + (rz.rotationSpeed * dt)
	if rz.rotationAngle >= 360.0 {
		rz.rotationAngle = 0.0
	}
}

// GetTexture returns the internal texture handle so external modules can render directly to it.
func (rz *RotoZoomEffect) GetTexture() *Texture {
	return rz.texture
}

// Render steps through the framebuffer in screen-space while tracking
// and accumulating pure U and V coordinates across texture space.
func (rz *RotoZoomEffect) Render(fb *fx_common.FrameBuffer, plasmaPhaseY int32) {
	pSize := int32(2) // 2x2 macro block screen rendering

	// =========================================================================
	// INITIALIZATION STAGE: PURE FLOAT32 MATH
	// =========================================================================

	// Convert internal integer angle bounds into a real radian float value
	angleRad := float32((rz.rotationAngle * 2.0 * math.Pi) / 360.0)
	cosVal := float32(math.Cos(float64(angleRad)))
	sinVal := float32(math.Sin(float64(angleRad)))

	// Process the zoom modulation breathing wave using standard float values
	cy1 := (plasmaPhaseY >> 16) & 0xFF
	zoomWaveFloat := float32(rz.sinTable[cy1]) / 65536.0

	// 6.0x baseline magnification factor. Adjust this value to zoom in or out globally.
	baseZoomFactor := float32(8.0)
	zoomFactor := baseZoomFactor + (zoomWaveFloat * 2.5)
	if zoomFactor < 3.0 {
		zoomFactor = 3.0 // Safety limit to keep texture filling the frame
	}

	// COMPUTE STEP VECTORS IN TRUE ROTATED UV SPACE PER SCREEN PIXEL
	float_dUdX := cosVal / zoomFactor
	float_dVdX := sinVal / zoomFactor
	float_dUdY := -sinVal / zoomFactor
	float_dVdY := cosVal / zoomFactor

	// Target the center markers of both windows
	screenCenterX := float32(fb.Width) / 2.0
	screenCenterY := float32(fb.Height) / 2.0
	texCenterX := float32(rz.texture.Width) / 2.0
	texCenterY := float32(rz.texture.Height) / 2.0

	// FIND THE TRUE ROTATED STARTING ANCHOR POSITION AT THE TOP-LEFT CORNER (0,0)
	float_startU := texCenterX - (screenCenterX * float_dUdX) - (screenCenterY * float_dUdY)
	float_startV := texCenterY - (screenCenterX * float_dVdX) - (screenCenterY * float_dVdY)

	// =========================================================================
	// CONVERSION STAGE: FLOATS TO 16.16 FIXED-POINT INTEGERS
	// =========================================================================

	startU := int32(float_startU * 65536.0)
	startV := int32(float_startV * 65536.0)

	// Scale steps properly by the macro block step size jumping length
	stepUdX := int32((float_dUdX * float32(pSize)) * 65536.0)
	stepVdX := int32((float_dVdX * float32(pSize)) * 65536.0)
	stepUdY := int32((float_dUdY * float32(pSize)) * 65536.0)
	stepVdY := int32((float_dVdY * float32(pSize)) * 65536.0)

	// =========================================================================
	// DEEP LOOP STAGE: PURE 16.16 FIXED-POINT UV TRAVERSAL
	// =========================================================================
	gridXMax := fb.Width / pSize
	gridYMax := fb.Height / pSize

	for gy := int32(0); gy < gridYMax; gy++ {
		screenY := gy * pSize

		// Track active drawing positions across the screen column line
		u := startU
		v := startV

		for gx := int32(0); gx < gridXMax; gx++ {
			screenX := gx * pSize

			// Extract whole texture array indices from 16.16 accumulators
			tu := (u >> 16) & rz.texture.MaskX
			tv := (v >> 16) & rz.texture.MaskY

			// Fetch the pixel color from the decoupled background scratchpad
			color := rz.texture.Pixels[(tv*rz.texture.Width)+tu]

			// Write the macro block directly using linear bounds safety checks
			for blockY := int32(0); blockY < pSize; blockY++ {
				currY := screenY + blockY
				if currY >= fb.Height {
					break
				}

				// Calculate target pointer row using additions instead of inner loop mults
				destRow := (currY * fb.Width) + screenX

				for blockX := int32(0); blockX < pSize; blockX++ {
					if (screenX + blockX) < fb.Width {
						fb.Pixels[destRow+blockX] = color
					}
				}
			}

			// DDA horizontal step accumulation updates both coordinates across the row
			u += stepUdX
			v += stepVdX
		}

		// Advance the main left-hand anchor point downward to prepare for the next line
		startU += stepUdY
		startV += stepVdY
	}
}
