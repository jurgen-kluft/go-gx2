package spritepack

import (
	"fmt"
	_ "image"
	"io"

	"github.com/jurgen-kluft/go-datastream/codestream"
	"github.com/jurgen-kluft/go-gx2/common"
)

func Build(cfgs *SpritePackCfg) ([]Sprite, error) {

	spritesArray := make([]Sprite, 0, 1024)

	for _, cfg := range cfgs.Files {

		img, err := common.LoadImage(cfg.ImageFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load image %s: %w", cfg.ImageFile, err)
		}

		// Based on the image, determine the pixel format.
		// If the number of colors in the image is less than or equal to 256, we can use I8 format.
		// Otherwise, we can use RGB565 format.

		pixelFormat := common.FMT_PIXEL_RGB565
		if common.ImageIsIndexed(img) {
			fmt.Println("Image is indexed, using I8 format:", cfg.ImageFile)
			pixelFormat = common.FMT_PIXEL_I8
		}

		spritesCfg, err := loadSpritesFile(cfg.SpritesFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load sprites file %s: %w", cfg.SpritesFile, err)
		}

		for _, s := range spritesCfg.Sprites {
			r := common.ImageFullRect(img)
			if s.Rect != nil {
				r = *s.Rect
			}

			alphaFormat, err := common.AlphaFormatFromInt(s.Alpha)
			if err != nil {
				return nil, err
			}

			var px []byte
			var al []byte

			px = nil
			al = nil

			switch pixelFormat {
			case common.FMT_PIXEL_RGB565:
				fmt.Println("Encoding RGB565 format:", cfg.ImageFile)
				px, al = common.EncodeRGB565(img, r)
			case common.FMT_PIXEL_RGB888:
				fmt.Println("Encoding RGB888 format:", cfg.ImageFile)
				px = common.EncodeRGBA8888(img, r)
				alphaFormat = common.FMT_ALPHA_NONE
			case common.FMT_PIXEL_I8:
				fmt.Println("Encoding I8 format:", cfg.ImageFile)
				px, al = common.EncodeIndexed(img, r)
			default:
				return nil, fmt.Errorf("unsupported format: %v", pixelFormat)
			}

			al = common.ConvertAlpha(al, r.W, r.H, alphaFormat)

			spritesArray = append(spritesArray, Sprite{
				Width:       uint16(r.W),
				Height:      uint16(r.H),
				PixelFormat: pixelFormat,
				AlphaFormat: alphaFormat,
				PixelData:   px,
				AlphaData:   al,
			})
		}
	}

	return spritesArray, nil
}

// ReadPack reads a binary spritepak file and returns a SpritePack.
func ReadPack(r io.Reader) (*SpritePack, error) {
	spritePack := SpritePack{
		Sprites: []Sprite{},
	}

	if err := codestream.ReadFromStream(r, &spritePack); err != nil {
		return nil, err
	}

	return &spritePack, nil
}

// WritePack writes a SpritePack to a binary spritepak file.
func WritePack(w io.Writer, sprites []Sprite) error {
	spritePack := SpritePack{
		Sprites: sprites,
	}

	if err := codestream.WriteToStream(w, spritePack); err != nil {
		return err
	}

	return nil
}
