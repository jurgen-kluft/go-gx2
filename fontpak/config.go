package fontpack

import (
	"encoding/json"
	"fmt"
	"os"
)

type Options struct {
	FontSize int // FontSize is the size of the font in points (pt).
	DPI      int // DPI is the dots per inch for rendering the font. Higher DPI means higher quality but larger bitmaps.
}

type Config struct {
	Files []*FontFile `json:"files"`
}

type FontFile struct {
	File  string        `json:"file"`
	Fonts []*FontConfig `json:"fonts"`
}

type FontConfig struct {
	Name    string       `json:"name"`
	Dpi     int          `json:"dpi"`
	Size    int          `json:"size"`
	CharMap []CharConfig `json:"chars"`
}

type CharConfig struct {
	ASCII string `json:"ascii"`
	Glyph string `json:"glyph"`
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

	// Validate the config
	for _, file := range cfg.Files {
		if file.File == "" {
			return nil, fmt.Errorf("font file path cannot be empty")
		}
		for _, font := range file.Fonts {
			if font.Name == "" {
				return nil, fmt.Errorf("font name cannot be empty in file %q", file.File)
			}
			if font.Size <= 0 {
				return nil, fmt.Errorf("font size must be positive in font %q of file %q", font.Name, file.File)
			}
			if font.Dpi == 0 {
				font.Dpi = 72 // default DPI
			}
			if font.Dpi < 0 || font.Dpi > 1000 {
				return nil, fmt.Errorf("DPI must be between 0 and 1000 in font %q of file %q", font.Name, file.File)
			}
			if len(font.CharMap) == 0 || len(font.CharMap) > 255 {
				return nil, fmt.Errorf("char map must contain between 1 and 255 characters in font %q of file %q", font.Name, file.File)
			}
		}
	}

	return &cfg, nil
}
