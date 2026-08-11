package main

import (
	"image/color"

	rl "github.com/gen2brain/raylib-go/raylib"
	fontpack "github.com/jurgen-kluft/go-gx2/fontpak"
)

// 16.16 Fixed-Point Configuration
const (
	FpFracBits = 16
	FpFracMask = (1 << FpFracBits) - 1
	FpOne      = 1 << FpFracBits
)

type FrameBuffer struct {
	Width  int
	Height int
	Pixels []color.RGBA
}

func toFp(f float32) int32 {
	return int32(f * float32(FpOne))
}

// Glyph matches our 8-byte C structure
type Font = fontpack.Font

// FontRenderContext tracks global scale configuration
type FontRenderContext struct {
	MinEdge    int32
	MaxEdge    int32
	ShiftBits  uint32
	InvScaleFp int32
}

// prepareFontContext calculates scaling boundaries exactly once per text block
func prepareFontContext(scale float32) FontRenderContext {
	var ctx FontRenderContext
	ctx.InvScaleFp = toFp(1.0 / scale)

	// Target a 5-pixel smoothing window scaled into 256 internal precision units
	targetWidth := 5.0 / scale
	widthUnits := int32(targetWidth * 256.0)
	if widthUnits < 256 {
		widthUnits = 256
	}

	// 1-cycle leading zero count emulation in Go using standard math bits package
	// or simple shift to find the next highest power of two
	var shift uint32
	shift = 8 // Start with 2^8 = 256 since widthUnits minimum is 256
	for (1 << shift) < widthUnits {
		shift++
	}

	finalWidthUnits := int32(1 << shift)
	sdfSpan := finalWidthUnits >> 8
	if sdfSpan < 2 {
		sdfSpan = 2
	}

	ctx.MinEdge = 128 - (sdfSpan >> 1)
	ctx.MaxEdge = ctx.MinEdge + sdfSpan
	ctx.ShiftBits = shift - 8

	return ctx
}

// blendRGBA performs clean, isolated 32-bit channel blending
func blendRGBA(dst rl.Color, src rl.Color, coverage uint32) rl.Color {
	// Simple, clean alpha blending math that doesn't suffer from type promotion traps
	// out = (src * coverage + dst * (255 - coverage)) / 255
	// To avoid division, we use the standard (val * 257) >> 16 trick for 255 normalization
	invCoverage := 255 - coverage

	outR := (((uint32(src.R) * coverage) + (uint32(dst.R) * invCoverage)) * 257) >> 16
	outG := (((uint32(src.G) * coverage) + (uint32(dst.G) * invCoverage)) * 257) >> 16
	outB := (((uint32(src.B) * coverage) + (uint32(dst.B) * invCoverage)) * 257) >> 16

	return rl.Color{R: uint8(outR), G: uint8(outG), B: uint8(outB), A: 255}
}

// RenderGlyph draws an isolated character using fixed-point nearest-neighbor sampling.
func RenderGlyph(fb *FrameBuffer, font *Font, glyphIndex uint8, ctx *FontRenderContext, color rl.Color, startX, startY int, scale float32) {
	glyphWidth := font.GlyphDims[glyphIndex].Width
	glyphHeight := font.GlyphDims[glyphIndex].Height
	glyphOffset := int(font.GlyphOffset[glyphIndex]) * 8
	glyphSize := int(glyphWidth) * int(glyphHeight)
	glyphData := font.Data[glyphOffset : glyphOffset+glyphSize]

	targetWidth := int(float32(glyphWidth) * scale)
	targetHeight := int(float32(glyphHeight-2) * scale)

	var srcYFp int32

	for dstY := 0; dstY < targetHeight; dstY++ {
		screenY := startY + dstY

		y0 := int(srcYFp >> FpFracBits)

		rowOffset0 := y0 * int(glyphWidth)
		rowOffset1 := rowOffset0 + int(glyphWidth)

		fy_fp := srcYFp & FpFracMask

		var srcXFp int32

		for dstX := 0; dstX < targetWidth && dstX < fb.Width; dstX++ {
			screenX := startX + dstX

			x0 := int(srcXFp >> FpFracBits)
			x1 := x0 + 1

			p00 := int32(glyphData[rowOffset0+x0])
			p10 := int32(glyphData[rowOffset0+x1])
			p01 := int32(glyphData[rowOffset1+x0])
			p11 := int32(glyphData[rowOffset1+x1])

			// Get the horizontal fractional part for interpolation
			fx_fp := srcXFp & FpFracMask

			// Bilinear interpolation in fixed-point arithmetic
			top := p00 + ((p10 - p00) * int32(fx_fp) >> FpFracBits)
			bottom := p01 + ((p11 - p01) * int32(fx_fp) >> FpFracBits)
			sdfVal := top + ((bottom - top) * int32(fy_fp) >> FpFracBits)

			// Apply Threshold Assignment and Draw
			if sdfVal >= ctx.MaxEdge {
				fb.Pixels[screenY*fb.Width+screenX] = color
			} else if sdfVal > ctx.MinEdge {
				pixelColor := fb.Pixels[screenY*fb.Width+screenX]
				coverage := uint32((sdfVal-ctx.MinEdge)*255) >> ctx.ShiftBits
				fb.Pixels[screenY*fb.Width+screenX] = blendRGBA(pixelColor, color, coverage)
				// fb.Pixels[screenY*fb.Width+screenX] = rl.Red
			}

			srcXFp += ctx.InvScaleFp
		}
		srcYFp += ctx.InvScaleFp
	}
}

// RenderText loops over text strings, processing character offsets
func RenderText(fb *FrameBuffer, text string, font *Font, x, y int, scale float32, color rl.Color) {
	ctx := prepareFontContext(scale)
	currentX := x

	for _, char := range text {
		glyphIndex := font.Map[char]

		if glyphIndex == 0xFF {
			// Advance pen position horizontally
			currentX += int(float32(font.GlyphAdvanceX[0]) * scale)
			continue
		}

		// Apply the glyph's layout metrics.
		renderX := currentX + int(float32(font.GlyphBearing[glyphIndex].X)*scale)
		renderY := y - int(float32(font.GlyphBearing[glyphIndex].Y)*scale)

		RenderGlyph(fb, font, glyphIndex, &ctx, color, renderX, renderY, scale)

		// Advance pen position horizontally
		currentX += int(float32(font.GlyphAdvanceX[glyphIndex]) * scale)
	}
}

func main() {
	screenWidth := 800
	screenHeight := 600

	rl.InitWindow(int32(screenWidth), int32(screenHeight), "SDF Font Renderer via Raylib Go")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)

	fontPackCfg, err := fontpack.LoadConfig("FontPack.json")
	if err != nil {
		panic(err)
	}

	fonts, infos, err := fontpack.Build(fontPackCfg)
	if err != nil {
		panic(err)
	}

	if len(fonts) > 0 {
		font := &fonts[0]

		fontpack.PrintFontInfo(font, infos[0])
		for _, gi := range font.Map {
			if gi != 0xFF { // Only print valid glyphs
				//font.PrintGlyphInfo(int(gi))
			}
		}

		frameBuffer := &FrameBuffer{
			Width:  screenWidth,
			Height: screenHeight,
			Pixels: make([]rl.Color, screenWidth*screenHeight),
		}

		scale1 := float32(1.2)
		scale2 := float32(2.5)
		scale3 := float32(4.0)

		scaler := float32(-1.0)
		scaler_inc := float32(0.004)

		for !rl.WindowShouldClose() {
			rl.BeginDrawing()
			rl.ClearBackground(rl.RayWhite)

			// Clear framebuffer pixels to black before rendering
			for i := range frameBuffer.Pixels {
				frameBuffer.Pixels[i] = rl.Black
			}

			// Draw smooth scalable text string using our custom SDF logic
			RenderText(frameBuffer, "Sophia g j", font, 100, 100, scale1+scaler, rl.SkyBlue)
			RenderText(frameBuffer, "Sophia g j", font, 100, 200, scale2+scaler, rl.White)
			RenderText(frameBuffer, "Sophia g j", font, 100, 300, scale3+scaler, rl.White)

			// Draw framebuffer pixels to the screen
			for y := 0; y < frameBuffer.Height; y++ {
				for x := 0; x < frameBuffer.Width; x++ {
					rl.DrawPixel(int32(x), int32(y), frameBuffer.Pixels[y*frameBuffer.Width+x])
				}
			}

			scaler += scaler_inc
			if scaler > 0.1 {
				scaler_inc = -scaler_inc
			} else if scaler < -1.0 {
				scaler_inc = -scaler_inc
			}

			rl.EndDrawing()
		}
	}
}
