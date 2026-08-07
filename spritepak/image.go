package spritepack

import (
	"bufio"
	"image"
	_ "image/png"
	"os"
	"strings"

	"github.com/jurgen-kluft/go-gx2/tga"
)

func imageFullRect(img image.Image) Rect {
	bounds := img.Bounds()
	return Rect{
		X: 0,
		Y: 0,
		W: bounds.Dx(),
		H: bounds.Dy(),
	}
}

func loadImage(filePath string) (image.Image, error) {
	imgFile, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer imgFile.Close()

	// if the extension is .tga, use the TGA decoder
	if strings.HasSuffix(filePath, ".tga") {
		img, err := tga.Decode(bufio.NewReader(imgFile))
		if err != nil {
			return nil, err
		}
		return img, nil
	}

	// otherwise, use the standard image decoder for PNG and other supported formats
	img, _, err := image.Decode(imgFile)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func imageIsIndexed(img image.Image) bool {
	switch img.(type) {
	case *image.Paletted:
		return true
	default:
		return false
	}
}

func analyzeAlpha(img image.Image, r Rect, alphaDisabled bool) AlphaFormat {
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
func encodeRGB565A0(img image.Image, r Rect) ([]byte, []byte) {
	pixels := make([]byte, 0, r.W*r.H*2)

	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			cr, cg, cb, _ := img.At(r.X+x, r.Y+y).RGBA()

			r5 := (cr >> 11) & 0x1F
			g6 := (cg >> 10) & 0x3F
			b5 := (cb >> 11) & 0x1F

			v := uint16(r5<<11 | g6<<5 | b5)
			pixels = append(pixels, byte(v), byte(v>>8))
		}
	}
	return pixels, []byte{}
}

// RGB565 + A1 (separate alpha bitstream)
func encodeRGB565A1(img image.Image, r Rect) ([]byte, []byte) {
	pixels := make([]byte, 0, r.W*r.H*2)
	alpha := make([]byte, 0, (r.W*r.H+7)/8)

	var abit byte
	var acnt uint

	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			cr, cg, cb, ca := img.At(r.X+x, r.Y+y).RGBA()
			colorRGBA8888 := NewColorFromR8G8B8A8(uint8(ca>>8), uint8(cr>>8), uint8(cg>>8), uint8(cb>>8))
			colorRGB565 := colorRGBA8888.ToRGB565()

			pixels = append(pixels, byte(colorRGB565), byte(colorRGB565>>8))

			if ca >= 0x8000 {
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
func encodeRGB565A4(img image.Image, r Rect) ([]byte, []byte) {
	pixels := make([]byte, 0, r.W*r.H*2)
	alpha := make([]byte, 0, (r.W*r.H+1)/2)

	var abit byte
	var acnt uint

	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			cr, cg, cb, ca := img.At(r.X+x, r.Y+y).RGBA()
			colorRGBA8888 := NewColorFromR8G8B8A8(uint8(ca>>8), uint8(cr>>8), uint8(cg>>8), uint8(cb>>8))
			colorRGB565 := colorRGBA8888.ToRGB565()

			pixels = append(pixels, byte(colorRGB565), byte(colorRGB565>>8))

			abit |= (byte(ca>>4) & 0x0F) << (4 * acnt)
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
func encodeRGB565A8(img image.Image, r Rect) ([]byte, []byte) {
	pixels := make([]byte, 0, r.W*r.H*2)
	alpha := make([]byte, 0, r.W*r.H)

	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			cr, cg, cb, ca := img.At(r.X+x, r.Y+y).RGBA()
			colorRGBA8888 := NewColorFromR8G8B8A8(uint8(ca>>8), uint8(cr>>8), uint8(cg>>8), uint8(cb>>8))
			colorRGB565 := colorRGBA8888.ToRGB565()

			pixels = append(pixels, byte(colorRGB565), byte(colorRGB565>>8))
			alpha = append(alpha, byte(ca>>8))
		}
	}
	return pixels, alpha
}

// RGBA8888
func encodeRGBA8888(img image.Image, r Rect) []byte {
	pixels := make([]byte, 0, r.W*r.H*4)

	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			cr, cg, cb, ca := img.At(r.X+x, r.Y+y).RGBA()
			pixels = append(
				pixels,
				byte(cr>>8),
				byte(cg>>8),
				byte(cb>>8),
				byte(ca>>8),
			)
		}
	}
	return pixels
}
