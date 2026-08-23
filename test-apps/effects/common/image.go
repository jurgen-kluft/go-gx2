package fx_common

import "github.com/jurgen-kluft/go-gx2/common"

func ReadPngAsRGB565(path string) ([]uint16, int32, int32, error) {
	img, err := common.LoadImage(path)
	if err != nil {
		return nil, 0, 0, err
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	pixels := make([]uint16, width*height)

	index := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pixels[index] = common.NewColorFromImageColor(img.At(x, y)).ToRGB16()
			index++
		}
	}

	return pixels, int32(width), int32(height), nil
}
