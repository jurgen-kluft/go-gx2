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

type FontPackCfg struct {
	Files []*FontFileCfg `json:"fonts"`
}

type FontFileCfg struct {
	File  string     `json:"file"`
	Fonts []*FontCfg `json:"fonts"`
}

type FontCfg struct {
	Name   string   `json:"name"`
	Id     int      `json:"id"`
	Dpix   int      `json:"dpi"`
	Size   int      `json:"size"`
	Chars  []string `json:"chars"`
	Glyphs []string `json:"glyphs"`
}

func LoadConfig(path string) (error, *FontPackCfg) {
	data, err := os.ReadFile(path)
	if err != nil {
		return err, nil
	}
	var cfg FontPackCfg
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err, nil
	}

	// Validate the config
	for _, file := range cfg.Files {
		if file.File == "" {
			return fmt.Errorf("font file path cannot be empty"), nil
		}
		for _, font := range file.Fonts {
			if font.Name == "" {
				return fmt.Errorf("font name cannot be empty in file %q", file.File), nil
			}
			if font.Size <= 0 {
				return fmt.Errorf("font size must be positive in font %q of file %q", font.Name, file.File), nil
			}
			// if font.Dpi == 0 {
			// 	font.Dpi = 72 // default DPI
			// }
			// if font.Dpi < 0 || font.Dpi > 1000 {
			// 	return fmt.Errorf("DPI must be between 0 and 1000 in font %q of file %q", font.Name, file.File), nil
			// }
			if len(font.Chars) == 0 || len(font.Chars) > 255 {
				return fmt.Errorf("char map must contain between 1 and 255 characters in font %q of file %q", font.Name, file.File), nil
			}
		}
	}

	return nil, &cfg
}
