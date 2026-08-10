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

func ImageIsIndexed(img image.Image) bool {
	_, colorMap, ok := BuildPaletteFromImage(img)
	if !ok {
		return false
	}
	if len(colorMap) > 256 {
		fmt.Printf("Image has %d colors which is more than 256, not indexed\n", len(colorMap))
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

// RGB565
func EncodeRGB565(img image.Image, r Rect) ([]byte, []byte) {
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

func EncodeIndexed(img image.Image, r Rect, colorMap map[uint32]uint8) (pixels []byte, alpha []byte) {
	pixels = make([]byte, 0, r.W*r.H)
	alpha = make([]byte, 0, r.W*r.H)

	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			pixelColor := NewColorFromImageColor(img.At(r.X+x, r.Y+y))
			rgb32 := pixelColor.ToRGB32()

			if idx, exists := colorMap[rgb32]; exists {
				pixels = append(pixels, idx)
			} else {
				return nil, nil // Color not found in the color map
			}

			alpha = append(alpha, pixelColor.A())
		}
	}

	return pixels, alpha
}

func ConvertAlpha(alpha []byte, width, height int, alphaFormat AlphaFormat) []byte {
	switch alphaFormat {
	case FMT_ALPHA_NONE:
		return nil
	case FMT_ALPHA_MASK:
		rowSize := (width + 7) / 8
		bits := make([]byte, rowSize*height)
		for h := 0; h < height; h++ {
			for w := 0; w < width; w++ {
				i := h*width + w
				a := alpha[i]
				if a > 0 {
					bits[h*rowSize+w/8] |= (1 << (7 - uint(w&7)))
				}
			}
		}
		return bits
	case FMT_ALPHA_A4:
		bits := make([]byte, (width*height+1)/2)
		rowSize := (width + 1) / 2
		for h := 0; h < height; h++ {
			for w := 0; w < width; w++ {
				i := h*width + w
				a := alpha[i] >> 4 // Convert 8-bit alpha to 4-bit
				if w&1 == 0 {
					bits[h*rowSize+w/2] = a << 4
				} else {
					bits[h*rowSize+w/2] |= a & 0xF
				}
			}
		}
		return bits
	case FMT_ALPHA_A8:
		return alpha
	default:
		return nil
	}
}
