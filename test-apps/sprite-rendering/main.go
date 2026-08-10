package main

import (
	"bytes"
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
	common "github.com/jurgen-kluft/go-gx2/common"
	palpak "github.com/jurgen-kluft/go-gx2/palpak"
	spritepak "github.com/jurgen-kluft/go-gx2/spritepak"
)

type FrameBuffer struct {
	Width  int
	Height int
	Pixels []rl.Color
}

type Sprite = spritepak.Sprite

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

func drawSpriteRGB565(frameBuffer *FrameBuffer, sprite *Sprite, x int, y int) {
	for sy := 0; sy < int(sprite.Height); sy++ {
		for sx := 0; sx < int(sprite.Width); sx++ {
			pixelIndex := sy*int(sprite.Width) + sx
			pixelColor := uint16(sprite.PixelData[pixelIndex*2])
			pixelColor |= uint16(sprite.PixelData[pixelIndex*2+1]) << 8
			frameBuffer.Pixels[(y+sy)*frameBuffer.Width+(x+sx)] = rgb565ToColor(pixelColor)
		}
	}
}

func main() {

	// -----------------------------------------------------------------------------
	// Load the sprite pack configuration
	// -----------------------------------------------------------------------------

	spritePak, err := spritepak.LoadConfig("SpritePack.json")
	if err != nil {
		panic(err)
	}

	sprites, err := spritepak.Build(spritePak)
	if err != nil {
		panic(err)
	}

	spritePackWriter := bytes.NewBuffer(nil)
	err = spritepak.WritePack(spritePackWriter, sprites)
	// Print the size of the sprite pack in bytes
	if err != nil {
		panic(err)
	}
	fmt.Printf("Sprite pack size: %d bytes\n", spritePackWriter.Len())

	// -----------------------------------------------------------------------------
	// Load the palette pack configuration
	// -----------------------------------------------------------------------------

	palpakCfg, err := palpak.LoadConfig("PalettePack.json")
	if err != nil {
		panic(err)
	}

	palPak, err := palpak.Build(palpakCfg)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Palette pack size: %d bytes\n", palpak.SizeInBytes(palPak))

	// -----------------------------------------------------------------------------

	screenWidth := 1920
	screenHeight := 1080

	rl.InitWindow(int32(screenWidth), int32(screenHeight), "SDF Font Renderer via Raylib Go")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)

	frameBuffer := &FrameBuffer{
		Width:  screenWidth,
		Height: screenHeight,
		Pixels: make([]rl.Color, screenWidth*screenHeight),
	}

	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)

		// Clear framebuffer pixels to black before rendering
		for i := range frameBuffer.Pixels {
			frameBuffer.Pixels[i] = rl.Black
		}

		// Draw sprites
		spriteX := 0
		spriteY := 0
		maxY := 0
		for _, sprite := range sprites {
			if sprite.PixelFormat == common.FMT_PIXEL_RGB565 {
				if spriteX+int(sprite.Width) > frameBuffer.Width {
					spriteX = 0
					spriteY += maxY + 10 // Move down for the next row of sprites
					maxY = 0
				}
				if spriteY+int(sprite.Height) > frameBuffer.Height {
					break // Stop drawing if we exceed the framebuffer height
				}

				drawSpriteRGB565(frameBuffer, &sprite, spriteX, spriteY)

				spriteX += int(sprite.Width) + 10 // Move to the right for the next sprite
				if int(sprite.Height) > maxY {
					maxY = int(sprite.Height)
				}
			}
		}

		// Draw framebuffer pixels to the screen
		for y := 0; y < frameBuffer.Height; y++ {
			for x := 0; x < frameBuffer.Width; x++ {
				rl.DrawPixel(int32(x), int32(y), frameBuffer.Pixels[y*frameBuffer.Width+x])
			}
		}

		rl.EndDrawing()
	}

}
