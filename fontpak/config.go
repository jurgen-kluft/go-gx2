package fontpack

import (
	"encoding/json"
	"fmt"
	"os"
)

type FontPackCfg struct {
	Fonts   []*FontCfg     `json:"fonts"`
	Mapping map[string]int `json:"mapping"`
}

type FontChar struct {
	Address string `json:"address"`
	Glyph   string `json:"glyph"`
}

type FontCfg struct {
	File    string      `json:"file"`
	Name    string      `json:"name"`
	Chars   []FontChar  `json:"chars"`
	Options FontOptions `json:"options"`
}

type FontOptions struct {
	FontSize  int     `json:"size"`       // FontSize is the size of the font in points (pt).
	SDF       bool    `json:"sdf"`        // SDF indicates whether Signed Distance Field rendering is enabled.
	SDFBorder int16   `json:"sdf_border"` // SDFBorder is the buffer size used for storing the SDF glyphs in the font pack. It is typically smaller than SDFBuildBorder to reduce memory usage while still providing a margin for distance field calculations.
	SDFRadius float64 `json:"sdf_radius"` // SDFRadius is the radius used for generating the SDF glyphs.
	SDFCutoff float64 `json:"sdf_cutoff"` // SDFCutoff is the cutoff value used for generating the SDF glyphs.
}

func (font *FontCfg) options() FontOptions {
	if font.Options.SDF {
		if font.Options.SDFRadius <= 0.0 {
			font.Options.SDFRadius = cDefaultSDFRadius
		}
		if font.Options.SDFCutoff <= 0.0 {
			font.Options.SDFCutoff = cDefaultSDFCutoff
		}
	}
	return font.Options
}

func LoadConfig(path string) (*FontPackCfg, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg FontPackCfg
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Validate the config
	for _, font := range cfg.Fonts {
		if font.File == "" {
			return nil, fmt.Errorf("font file path cannot be empty")
		}

		if font.Name == "" {
			return nil, fmt.Errorf("font name cannot be empty in file %q", font.File)
		}
		if len(font.Chars) == 0 || len(font.Chars) > (255) {
			return nil, fmt.Errorf("char map must contain between 1 and %d characters in font %q of file %q", 255, font.Name, font.File)
		}

		options := font.options()
		if options.FontSize <= 0 {
			return nil, fmt.Errorf("font size must be positive in font %q of file %q", font.Name, font.File)
		}
	}

	return &cfg, nil
}
