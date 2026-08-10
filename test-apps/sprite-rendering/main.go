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
	Pixels []uint16
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

			if sprite.AlphaData != nil {
				if sprite.AlphaFormat == common.FMT_ALPHA_MASK {
					// 1 bit per pixel alpha, rows are aligned to byte boundaries
					rowSize := (int(sprite.Width) + 7) / 8
					alphaByte := sprite.AlphaData[sy*rowSize+(sx/8)]
					alphaBit := (alphaByte >> (7 - uint(sx%8))) & 1
					if alphaBit != 0 {
						frameBuffer.Pixels[(y+sy)*frameBuffer.Width+(x+sx)] = pixelColor
					}
				}
			} else {
				frameBuffer.Pixels[(y+sy)*frameBuffer.Width+(x+sx)] = pixelColor
			}
		}
	}
}

func drawSpriteI8PaletteRGB565(frameBuffer *FrameBuffer, sprite *Sprite, x int, y int, palette common.PaletteRGB565) {
	for sy := 0; sy < int(sprite.Height); sy++ {
		for sx := 0; sx < int(sprite.Width); sx++ {
			pixelIndex := sy*int(sprite.Width) + sx
			paletteIndex := sprite.PixelData[pixelIndex]
			if int(paletteIndex) < len(palette) {
				pixelColor := palette[paletteIndex]

				if sprite.AlphaData != nil {
					if sprite.AlphaFormat == common.FMT_ALPHA_MASK {
						// 1 bit per pixel alpha, rows are aligned to byte boundaries
						rowSize := (int(sprite.Width) + 7) / 8
						alphaByte := sprite.AlphaData[sy*rowSize+(sx/8)]
						alphaBit := (alphaByte >> (7 - uint(sx%8))) & 1
						if alphaBit != 0 {
							frameBuffer.Pixels[(y+sy)*frameBuffer.Width+(x+sx)] = pixelColor.ToRGB16()
						}
					}
				} else {
					frameBuffer.Pixels[(y+sy)*frameBuffer.Width+(x+sx)] = pixelColor.ToRGB16()
				}
			}
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

	palpakCfg, err := palpak.LoadConfig("SpritePack.json")
	if err != nil {
		panic(err)
	}

	palPak, err := palpak.Build(palpakCfg)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Palette pack memory size: %d bytes\n", palPak.MemSize())

	// -----------------------------------------------------------------------------

	screenWidth := 1920
	screenHeight := 1080

	rl.InitWindow(int32(screenWidth), int32(screenHeight), "SDF Font Rendering")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)

	frameBuffer := &FrameBuffer{
		Width:  screenWidth,
		Height: screenHeight,
		Pixels: make([]uint16, screenWidth*screenHeight),
	}

	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)

		// Clear framebuffer pixels to black before rendering
		for i := range frameBuffer.Pixels {
			frameBuffer.Pixels[i] = 0x3333
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
			} else if sprite.PixelFormat == common.FMT_PIXEL_I8 {
				if int(sprite.PaletteIndex) < len(palPak.PaletteColorRGB565) && palPak.PaletteColorRGB565[sprite.PaletteIndex] != nil {
					palette := palPak.PaletteColorRGB565[sprite.PaletteIndex]
					if spriteX+int(sprite.Width) > frameBuffer.Width {
						spriteX = 0
						spriteY += maxY + 10 // Move down for the next row of sprites
						maxY = 0
					}
					if spriteY+int(sprite.Height) > frameBuffer.Height {
						break // Stop drawing if we exceed the framebuffer height
					}

					drawSpriteI8PaletteRGB565(frameBuffer, &sprite, spriteX, spriteY, palette)

					spriteX += int(sprite.Width) + 10 // Move to the right for the next sprite
					if int(sprite.Height) > maxY {
						maxY = int(sprite.Height)
					}
				}
			}
		}

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
