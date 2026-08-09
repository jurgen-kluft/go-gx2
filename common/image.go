package common

import (
	"bufio"
	"fmt"
	"image"
	"image/png"
	_ "image/png"
	"os"
	"strings"

	"github.com/jurgen-kluft/go-gx2/tga"
)

func ImageFullRect(img image.Image) Rect {
	bounds := img.Bounds()
	return Rect{
		X: 0,
		Y: 0,
		W: bounds.Dx(),
		H: bounds.Dy(),
	}
}

func LoadImage(filePath string) (image.Image, error) {
	imgFile, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer imgFile.Close()

	// if the extension is .tga, use the TGA decoder
	if strings.HasSuffix(filePath, ".tga") {
		fmt.Println("Loading TGA image:", filePath)
		img, err := tga.Decode(bufio.NewReader(imgFile))
		if err != nil {
			return nil, err
		}
		return img, nil
	}

	if strings.HasSuffix(filePath, ".png") {
		fmt.Println("Loading PNG image:", filePath)
		img, err := png.Decode(imgFile)
		if err != nil {
			return nil, err
		}
		return img, nil
	}

	fmt.Println("Loading general image:", filePath)
	img, _, err := image.Decode(imgFile)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func ImageRGBColorCount(img image.Image) int {
	colorSet := make(map[uint32]uint8, 256)
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pixelColor := NewColorFromImageColor(img.At(x, y))
			colorSet[pixelColor.ToRGB32()] = 1
		}
	}
	return len(colorSet)
}

func ImageIsIndexed(img image.Image) bool {
	colorCount := ImageRGBColorCount(img)
	if colorCount > 256 {
		fmt.Printf("Image has %d colors which is more than 256, not indexed\n", colorCount)
		return false
	}
	return true
}

func AnalyzeAlpha(img image.Image, r Rect, alphaDisabled bool) AlphaFormat {
	if alphaDisabled {
		return FMT_ALPHA_NONE
	}

	alphas := make(map[uint8]bool, 256)
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			_, _, _, ca := img.At(r.X+x, r.Y+y).RGBA()
			a := uint8(ca >> 8)
			alphas[a] = true
		}
	}

	if len(alphas) == 0 {
		return FMT_ALPHA_NONE
	}

	if len(alphas) == 1 {
		for a := range alphas {
			if a == 0xFF {
				return FMT_ALPHA_MASK
			}
		}
		return FMT_ALPHA_NONE
	}

	if len(alphas) <= 16 {
		return FMT_ALPHA_A4
	}

	return FMT_ALPHA_A8
}

// RGB565 + A0 (no separate alpha bitstream)
func EncodeRGB565A0(img image.Image, r Rect) ([]byte, []byte) {
	pixels := make([]byte, 0, r.W*r.H*2)

	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			pixelColor := NewColorFromImageColor(img.At(r.X+x, r.Y+y))
			colorRGB565 := pixelColor.ToRGB16()
			pixels = append(pixels, byte(colorRGB565), byte(colorRGB565>>8))
		}
	}
	return pixels, []byte{}
}

// RGB565 + A1 (separate alpha bitstream)
func EncodeRGB565A1(img image.Image, r Rect) ([]byte, []byte) {
	pixels := make([]byte, 0, r.W*r.H*2)
	alpha := make([]byte, 0, (r.W*r.H+7)/8)

	var abit byte
	var acnt uint

	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			pixelColor := NewColorFromImageColor(img.At(r.X+x, r.Y+y))
			colorRGB565 := pixelColor.ToRGB16()

			pixels = append(pixels, byte(colorRGB565), byte(colorRGB565>>8))

			if pixelColor.A() > 0 {
				abit |= 1 << acnt
			}
			acnt++
			if acnt == 8 {
				alpha = append(alpha, abit)
				abit = 0
				acnt = 0
			}
		}
	}
	if acnt != 0 {
		alpha = append(alpha, abit)
	}
	return pixels, alpha
}

// RGB565 + A4 (separate alpha bitstream)
func EncodeRGB565A4(img image.Image, r Rect) ([]byte, []byte) {
	pixels := make([]byte, 0, r.W*r.H*2)
	alpha := make([]byte, 0, (r.W*r.H+1)/2)

	var abit byte
	var acnt uint

	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			pixelColor := NewColorFromImageColor(img.At(r.X+x, r.Y+y))
			colorRGB565 := pixelColor.ToRGB16()

			pixels = append(pixels, byte(colorRGB565), byte(colorRGB565>>8))

			abit |= (pixelColor.A() >> 4) << (4 * acnt)
			acnt++
			if acnt == 2 {
				alpha = append(alpha, abit)
				abit = 0
				acnt = 0
			}
		}
	}
	if acnt != 0 {
		alpha = append(alpha, abit)
	}
	return pixels, alpha
}

// RGB565 + A8 (separate alpha bitstream)
func EncodeRGB565A8(img image.Image, r Rect) ([]byte, []byte) {
	pixels := make([]byte, 0, r.W*r.H*2)
	alpha := make([]byte, 0, r.W*r.H)

	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			pixelColor := NewColorFromImageColor(img.At(r.X+x, r.Y+y))
			colorRGB565 := pixelColor.ToRGB16()

			pixels = append(pixels, byte(colorRGB565), byte(colorRGB565>>8))
			alpha = append(alpha, pixelColor.A())
		}
	}
	return pixels, alpha
}

// RGBA8888
func EncodeRGBA8888(img image.Image, r Rect) []byte {
	pixels := make([]byte, 0, r.W*r.H*4)

	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			pixelColor := NewColorFromImageColor(img.At(r.X+x, r.Y+y))
			pixels = append(
				pixels,
				pixelColor.R(),
				pixelColor.G(),
				pixelColor.B(),
				pixelColor.A(),
			)
		}
	}
	return pixels
}
