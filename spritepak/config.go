package spritepack

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jurgen-kluft/go-gx2/common"
)

//
// ===== Configuration (JSON) =====
//

type SpritePackCfg struct {
	Files          []SpritePackFileCfg `json:"sprites"`
	SpriteMapping  map[string]int      `json:"sprites_mapping,omitempty"`
	PaletteMapping map[string]int      `json:"palettes_mapping,omitempty"`
}

type SpritePackFileCfg struct {
	ImageFile   string `json:"image_file"`
	SpritesFile string `json:"sprites_file"`
	PaletteName string `json:"palette_name,omitempty"`
}

type SpritesFileCfg struct {
	Sprites []SpriteEntryCfg `json:"sprites"`
}

type SpriteEntryCfg struct {
	Name  string       `json:"name"`
	Alpha int          `json:"alpha"`
	Rect  *common.Rect `json:"rect,omitempty"`
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
	for _, sprite := range cfg.Sprites {
		if sprite.Name == "" {
			return nil, fmt.Errorf("sprite name cannot be empty in file %q", path)
		}
		if sprite.Rect == nil {
			return nil, fmt.Errorf("sprite rect cannot be nil for sprite %q in file %q", sprite.Name, path)
		}
		if sprite.Rect.W <= 0 || sprite.Rect.H <= 0 {
			return nil, fmt.Errorf("sprite rect width and height must be positive for sprite %q in file %q", sprite.Name, path)
		}
	}

	return &cfg, nil
}

func loadPalette(path string) ([]common.ColorRGBA, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var palette []common.ColorRGBA
	if err := json.Unmarshal(data, &palette); err != nil {
		return nil, err
	}
	return palette, nil
}
