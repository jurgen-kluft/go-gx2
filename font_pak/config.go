package fontpack

import (
	"encoding/json"
	"os"
)

type Options struct {
	FontSize int
	DPI      int
}

type Config struct {
	Files []FontFile `json:"files"`
}

type FontFile struct {
	File  string       `json:"file"`
	Fonts []FontConfig `json:"fonts"`
}

type FontConfig struct {
	Name  string            `json:"name"`
	DPI   int               `json:"dpi"`
	Size  int               `json:"size"`
	Chars map[string]string `json:"chars"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
