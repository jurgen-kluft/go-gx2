package fontpack

import (
	"encoding/json"
	"fmt"
	"os"
)

type FontPackCfg struct {
	Fonts []*FontCfg `json:"fonts"`
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
	FontSize       int     `json:"size"`             // FontSize is the size of the font in points (pt).
	SDF            bool    `json:"sdf"`              // SDF indicates whether Signed Distance Field rendering is enabled.
	SDFBuildBorder int16   `json:"sdf_build_border"` // SDFBuildBorder is the border size used for generating the SDF glyphs. It is typically larger than SDFFinalBorder to provide a margin for distance field calculations, but the stored glyphs are cropped to a smaller size to reduce memory usage.
	SDFFinalBorder int16   `json:"sdf_final_border"` // SDFFinalBorder is the buffer size used for storing the SDF glyphs in the font pack. It is typically smaller than SDFBuildBorder to reduce memory usage while still providing a margin for distance field calculations.
	SDFRadius      float64 `json:"sdf_radius"`       // SDFRadius is the radius used for generating the SDF glyphs.
	SDFCutoff      float64 `json:"sdf_cutoff"`       // SDFCutoff is the cutoff value used for generating the SDF glyphs.
}

const (
	defaultSDFBuffer = 3
	defaultSDFRadius = 8.0
	defaultSDFCutoff = 0.25
)

func (font *FontCfg) options() FontOptions {
	if font.Options.SDF {
		if font.Options.SDFBuildBorder < defaultSDFBuffer {
			font.Options.SDFBuildBorder = defaultSDFBuffer
		}
		if font.Options.SDFRadius <= 0.0 {
			font.Options.SDFRadius = defaultSDFRadius
		}
		if font.Options.SDFCutoff < 0.0 {
			font.Options.SDFCutoff = defaultSDFCutoff
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
		if len(font.Chars) == 0 || len(font.Chars) > (cMaxNumChars) {
			return nil, fmt.Errorf("char map must contain between 1 and %d characters in font %q of file %q", cMaxNumChars, font.Name, font.File)
		}

		options := font.options()
		if options.FontSize <= 0 {
			return nil, fmt.Errorf("font size must be positive in font %q of file %q", font.Name, font.File)
		}
	}

	return &cfg, nil
}
