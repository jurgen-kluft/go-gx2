package spritepack

import (
	"encoding/json"
	"fmt"
	_ "image/png"
	"os"
)

//
// ===== Configuration (JSON) =====
//

type SpritePackCfg struct {
	Files []SpriteFileCfg `json:"sprites"`
}

type SpriteFileCfg struct {
	ImageFile   string           `json:"image_file"`
	PaletteFile string           `json:"palette_file,omitempty"`
	Sprites     []SpriteEntryCfg `json:"sprites"`
}

type SpriteEntryCfg struct {
	Name        string `json:"name"`
	Id          int    `json:"id"`
	PixelFormat string `json:"pixel_format"`
	AlphaFormat string `json:"alpha_format"`
	Rect        *Rect  `json:"rect,omitempty"`
}

type Rect struct {
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

	return &config, nil
}
