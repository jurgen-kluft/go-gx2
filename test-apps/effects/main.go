package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	_ "github.com/jurgen-kluft/go-gx2/test-apps/effects/common"
	fx_common "github.com/jurgen-kluft/go-gx2/test-apps/effects/common"
	fx_conway "github.com/jurgen-kluft/go-gx2/test-apps/effects/conway"
)

func rgb565ToColor(c uint16) rl.Color {
	r := uint8((c >> 11) & 0x1F)
	g := uint8((c >> 5) & 0x3F)
	b := uint8(c & 0x1F)
	return rl.Color{
		R: r << 3,
		G: g << 2,
		B: b << 3,
		A: 255,
	}
}

func main() {

	screenWidth := int32(480)
	screenHeight := int32(480)

	rl.InitWindow(int32(screenWidth), int32(screenHeight), "SDF Font Rendering")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)

	frameBuffer := &fx_common.FrameBuffer{
		Width:  screenWidth,
		Height: screenHeight,
		Pixels: make([]uint16, screenWidth*screenHeight),
	}

	//effect := NewFireEffect(640, 480) // Create a fire effect with a width of 320 and height of 240
	//effect := NewMetaBallEffect(12345, 10, 16, int32(screenWidth), int32(screenHeight)) // Create a metaball effect with 10 balls
	//effect := NewMetaball2Effect()
	//effect := NewStarFieldEffect(2000) // Create a star field effect with 1000 stars
	effect := fx_conway.NewEffect(int32(screenWidth), int32(screenHeight)) // Create a Conway's Game of Life effect

	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)

		delta := rl.GetFrameTime() // Get the time elapsed since the last frame

		// Clear framebuffer pixels to black before rendering
		for i := range frameBuffer.Pixels {
			frameBuffer.Pixels[i] = 0
		}

		effect.ProcessFrame(delta, frameBuffer)

		// Draw framebuffer pixels to the screen
		for y := int32(0); y < frameBuffer.Height; y++ {
			for x := int32(0); x < frameBuffer.Width; x++ {
				pixel := frameBuffer.Pixels[y*frameBuffer.Width+x]
				color := rgb565ToColor(pixel)
				rl.DrawPixel(x, y, color)
			}
		}

		rl.EndDrawing()
	}

}
