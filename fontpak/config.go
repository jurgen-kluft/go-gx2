package fontpack

import (
	"encoding/json"
	"fmt"
	"os"
)

type Options struct {
	FontSize        int     // FontSize is the size of the font in points (pt).
	SDF             bool    // SDF indicates whether Signed Distance Field rendering is enabled.
	SDFBuffer       int16   // SDFBuffer is the buffer size used for generating the SDF glyphs. It is typically larger than SDFBufferStored to provide a margin for distance field calculations, but the stored glyphs are cropped to a smaller size to reduce memory usage.
	SDFBufferStored int16   // SDFBufferStored is the buffer size used for storing the SDF glyphs in the font pack. It is typically smaller than SDFBuffer to reduce memory usage while still providing a margin for distance field calculations.
	SDFRadius       float64 // SDFRadius is the radius used for generating the SDF glyphs.
	SDFCutoff       float64 // SDFCutoff is the cutoff value used for generating the SDF glyphs.
}

const (
	defaultSDFBuffer = 3
	defaultSDFRadius = 8.0
	defaultSDFCutoff = 0.25
)

type FontPackCfg struct {
	Fonts []*FontCfg `json:"fonts"`
}

type FontChar struct {
	Address string `json:"address"`
	Glyph   string `json:"glyph"`
}

type FontCfg struct {
	File      string     `json:"file"`
	Name      string     `json:"name"`
	Id        int        `json:"id"`
	Size      int        `json:"size"`
	Chars     []FontChar `json:"chars"`
	SDF       bool       `json:"sdf"`
	SDFBuffer int16      `json:"sdf_buffer"`
	SDFRadius float64    `json:"sdf_radius"`
	SDFCutoff float64    `json:"sdf_cutoff"`
}

func (font *FontCfg) options() Options {
	options := Options{
		FontSize:  font.Size,
		SDF:       font.SDF,
		SDFBuffer: font.SDFBuffer,
		SDFRadius: font.SDFRadius,
		SDFCutoff: font.SDFCutoff,
	}
	if options.SDF {
		if options.SDFBuffer == 0 {
			options.SDFBuffer = defaultSDFBuffer
		}
		if options.SDFRadius == 0 {
			options.SDFRadius = defaultSDFRadius
		}
		if options.SDFCutoff == 0 {
			options.SDFCutoff = defaultSDFCutoff
		}
	}
	return options
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
		if font.Size <= 0 {
			return nil, fmt.Errorf("font size must be positive in font %q of file %q", font.Name, font.File)
		}
		// if font.Dpi == 0 {
		// 	font.Dpi = 72 // default DPI
		// }
		// if font.Dpi < 0 || font.Dpi > 1000 {
		// 	return fmt.Errorf("DPI must be between 0 and 1000 in font %q of file %q", font.Name, font.File), nil
		// }
		if len(font.Chars) == 0 || len(font.Chars) > 255 {
			return nil, fmt.Errorf("char map must contain between 1 and 255 characters in font %q of file %q", font.Name, font.File)
		}
		if font.SDF {
			if font.SDFBuffer == 0 {
				font.SDFBuffer = defaultSDFBuffer
			}
			if font.SDFRadius == 0 {
				font.SDFRadius = defaultSDFRadius
			}
			if font.SDFCutoff == 0 {
				font.SDFCutoff = defaultSDFCutoff
			}
			if font.SDFBuffer < 0 {
				return nil, fmt.Errorf("SDF buffer must be non-negative in font %q of file %q", font.Name, font.File)
			}
			if font.SDFRadius <= 0 {
				return nil, fmt.Errorf("SDF radius must be positive in font %q of file %q", font.Name, font.File)
			}
			if font.SDFCutoff < 0 || font.SDFCutoff > 1 {
				return nil, fmt.Errorf("SDF cutoff must be between 0 and 1 in font %q of file %q", font.Name, font.File)
			}
		}
	}

	return &cfg, nil
}
