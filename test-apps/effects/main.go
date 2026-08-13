package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

type FrameBuffer struct {
	Width  int
	Height int
	Pixels []uint16
}

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

	screenWidth := 480
	screenHeight := 480

	rl.InitWindow(int32(screenWidth), int32(screenHeight), "SDF Font Rendering")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)

	frameBuffer := &FrameBuffer{
		Width:  screenWidth,
		Height: screenHeight,
		Pixels: make([]uint16, screenWidth*screenHeight),
	}

	//fireEffect := NewFireEffect(640, 480) // Create a fire effect with a width of 320 and height of 240
	metaballs := NewMetaBallEffect(12345, 10, int32(screenWidth), int32(screenHeight)) // Create a metaball effect with 10 balls

	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)

		// Clear framebuffer pixels to black before rendering
		for i := range frameBuffer.Pixels {
			frameBuffer.Pixels[i] = 0x3333
		}

		//fireEffect.ProcessFrame(frameBuffer)
		metaballs.ProcessFrame(frameBuffer)

		// Draw framebuffer pixels to the screen
		for y := 0; y < frameBuffer.Height; y++ {
			for x := 0; x < frameBuffer.Width; x++ {
				pixel := frameBuffer.Pixels[y*frameBuffer.Width+x]
				color := rgb565ToColor(pixel)
				rl.DrawPixel(int32(x), int32(y), color)
			}
		}

		rl.EndDrawing()
	}

}
