package palpak

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jurgen-kluft/go-gx2/common"
)

//
// ===== Configuration (JSON) =====
//

type PalPackCfg struct {
	Palettes []PalEntryCfg `json:"palettes"`
}

type PalEntryCfg struct {
	ImageFile   string `json:"image_file"`
	PaletteFile string `json:"palette_file,omitempty"`
}

type PalFileCfg struct {
	Colors []uint32 `json:"colors"`
}

func LoadConfig(jsonPath string) (*PalPackCfg, error) {
	// Load the JSON file
	var config PalPackCfg
	if jsonData, err := os.ReadFile(jsonPath); err != nil {
		fmt.Printf("Error reading JSON file: %v\n", err)
		os.Exit(1)
	} else {
		json.Unmarshal(jsonData, &config)
	}

	return &config, nil
}

func LoadPaletteFile(path string) (*PalFileCfg, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg PalFileCfg
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func LoadPalette(path string) ([]common.ColorRGBA, error) {
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
