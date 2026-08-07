package spritepack

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"os"
)

//
// ===== Configuration (JSON) =====
//

type SpritePackCfg struct {
	Files   []SpritePackFileCfg `json:"sprites"`
	Mapping map[string]int      `json:"mapping,omitempty"`
}

type SpritePackFileCfg struct {
	ImageFile   string `json:"image_file"`
	PaletteFile string `json:"palette_file,omitempty"`
	SpritesFile string `json:"sprites_file"`

	Image   image.Image
	Palette []ColorRGBA
	Sprites SpritesFileCfg
}

type SpritesFileCfg struct {
	Sprites []SpriteEntryCfg `json:"sprites"`
}

type SpriteEntryCfg struct {
	Name  string      `json:"name"`
	Alpha int         `json:"alpha"`
	Rect  *SpriteRect `json:"rect,omitempty"`

	Index int
}

type SpriteRect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

func LoadConfig(jsonPath string) (*SpritePackCfg, error) {
	// Load the JSON file
	var config SpritePackCfg
	if jsonData, err := os.ReadFile(jsonPath); err != nil {
		fmt.Printf("Error reading JSON file: %v\n", err)
		os.Exit(1)
	} else {
		json.Unmarshal(jsonData, &config)
	}

	// Load the images
	for i, fileCfg := range config.Files {
		image, err := loadImage(fileCfg.ImageFile)
		if err != nil {
			fmt.Printf("Error loading image file %s: %v\n", fileCfg.ImageFile, err)
			os.Exit(1)
		}
		config.Files[i].Image = image
	}

	// Load the palettes if any palette files are specified
	for i, fileCfg := range config.Files {
		if fileCfg.PaletteFile != "" {
			palette, err := loadPalette(fileCfg.PaletteFile)
			if err != nil {
				fmt.Printf("Error loading palette file %s: %v\n", fileCfg.PaletteFile, err)
				os.Exit(1)
			}
			config.Files[i].Palette = palette
		}
	}

	// Load the sprites files
	for i, fileCfg := range config.Files {
		sprites, err := loadSpritesFile(fileCfg.SpritesFile)
		if err != nil {
			fmt.Printf("Error loading sprites file %s: %v\n", fileCfg.SpritesFile, err)
			os.Exit(1)
		}
		config.Files[i].Sprites = *sprites
	}

	return &config, nil
}

func loadSpritesFile(path string) (*SpritesFileCfg, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg SpritesFileCfg
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Validate the config
	for i, sprite := range cfg.Sprites {
		if sprite.Name == "" {
			return nil, fmt.Errorf("sprite name cannot be empty in file %q", path)
		}
		if sprite.Rect == nil {
			return nil, fmt.Errorf("sprite rect cannot be nil for sprite %q in file %q", sprite.Name, path)
		}
		if sprite.Rect.W <= 0 || sprite.Rect.H <= 0 {
			return nil, fmt.Errorf("sprite rect width and height must be positive for sprite %q in file %q", sprite.Name, path)
		}
		cfg.Sprites[i].Index = i
	}

	return &cfg, nil
}
