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

		ok := false
		colorMap := map[uint32]uint8{}
		paletteIndex := uint8(0xFF)
		if _, colorMap, ok = common.BuildPaletteFromImage(img); ok {
			fmt.Println("Image is indexed, using I8 format:", cfg.ImageFile)
			pixelFormat = common.FMT_PIXEL_I8
			if pi, ok := cfgs.PaletteMapping[cfg.PaletteName]; ok {
				paletteIndex = uint8(pi)
			}
		} else {
			colorMap = nil
		}

		spritesCfg, err := loadSpritesFile(cfg.SpritesFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load sprites file %s: %w", cfg.SpritesFile, err)
		}

		if pixelFormat == common.FMT_PIXEL_I8 && paletteIndex == 0xFF {
			return nil, fmt.Errorf("palette index not found for indexed image: %s", cfg.ImageFile)
		}

		for _, s := range spritesCfg.Sprites {
			spriteIndex := 0
			if spriteIndex, ok = cfgs.SpriteMapping[s.Name]; !ok {
				fmt.Printf("Warning: Sprite name '%s' not found in mapping, skipping.\n", s.Name)
				continue // Skip sprites that are not in the mapping
			}
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
				px, al = common.EncodeIndexed(img, r, colorMap)
			}

			al = common.ConvertAlpha(al, r.W, r.H, alphaFormat)

			// Ensure the spritesArray has enough capacity for the spriteIndex
			for spriteIndex >= len(spritesArray) {
				spritesArray = append(spritesArray, Sprite{})
			}

			// Assign the sprite data to the correct index in the spritesArray
			spritesArray[spriteIndex] = Sprite{
				Width:        uint16(r.W),
				Height:       uint16(r.H),
				PixelFormat:  pixelFormat,
				AlphaFormat:  alphaFormat,
				PixelData:    px,
				AlphaData:    al,
				PaletteIndex: paletteIndex,
			}
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
