package spritepack

import (
	"fmt"
	_ "image"
	"io"

	"github.com/jurgen-kluft/go-datastream/codestream"
	"github.com/jurgen-kluft/go-gx2/common"
)

func Build(cfgs *SpritePackCfg) ([]Sprite, []common.PaletteRGBA, error) {

	palettes := make([]common.PaletteRGBA, 0, 16)
	palettesMap := map[string]int{}

	// Go through all the sprites and build the palettes first, then build the sprites.
	for _, cfg := range cfgs.Files {
		if cfg.PaletteFile != "" {
			paletteImage, err := common.LoadImage(cfg.PaletteFile)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to load palette image %s: %w", cfg.PaletteFile, err)
			}
			pal, err := buildPalette(paletteImage)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to build palette %s: %w", cfg.PaletteFile, err)
			}
			palettesMap[cfg.PaletteFile] = len(palettes)
			palettes = append(palettes, pal)
		}
	}

	spritesArray := make([]Sprite, 0, 1024)

	for _, cfg := range cfgs.Files {

		img, err := common.LoadImage(cfg.ImageFile)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load image %s: %w", cfg.ImageFile, err)
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
			return nil, nil, fmt.Errorf("failed to load sprites file %s: %w", cfg.SpritesFile, err)
		}

		for _, s := range spritesCfg.Sprites {
			r := common.ImageFullRect(img)
			if s.Rect != nil {
				r = *s.Rect
			}

			alphaFormat, err := common.AlphaFormatFromInt(s.Alpha)
			if err != nil {
				return nil, nil, err
			}

			var px []byte
			var al []byte

			px = nil
			al = nil

			switch pixelFormat {
			case common.FMT_PIXEL_RGB565:
				switch alphaFormat {
				case common.FMT_ALPHA_NONE:
					px, al = common.EncodeRGB565A0(img, r)
				case common.FMT_ALPHA_MASK:
					// fmt.Println("Encoding RGB565 with alpha mask:", cfg.ImageFile)
					px, al = common.EncodeRGB565A1(img, r)
				case common.FMT_ALPHA_A4:
					px, al = common.EncodeRGB565A4(img, r)
				case common.FMT_ALPHA_A8:
					px, al = common.EncodeRGB565A8(img, r)
				default:
					return nil, nil, fmt.Errorf("unsupported alpha format for RGB565: %s", alphaFormat.String())
				}
			case common.FMT_PIXEL_RGB888:
				px = common.EncodeRGBA8888(img, r)
				// Alpha is embedded in RGBA8888, so we can ignore the alphaFormatEnum here
				alphaFormat = common.FMT_ALPHA_NONE
			case common.FMT_PIXEL_I8:
				fmt.Println("Encoding I8 format with palette:", cfg.ImageFile)
				palIdx, ok := palettesMap[cfg.PaletteFile]
				if !ok {
					return nil, nil, fmt.Errorf("palette not found for I8 format: %s", cfg.PaletteFile)
				}
				pal := palettes[palIdx]
				px, ok = common.BuildIndexed8Palette(img, r, pal)
				if !ok {
					return nil, nil, fmt.Errorf("failed to build indexed 8-bit image %s", cfg.ImageFile)
				}
			default:
				return nil, nil, fmt.Errorf("unsupported format: %v", pixelFormat)
			}

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

	return spritesArray, palettes, nil
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
