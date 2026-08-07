package fontpack

import (
	"os"
	"path/filepath"
	"testing"
)

func loadTestConfig(t *testing.T, data string) (*FontPackCfg, error) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "fontpack.json")
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	return LoadConfig(path)
}

func TestLoadConfigSDFDefaults(t *testing.T) {
	cfg, err := loadTestConfig(t, `{
		"fonts": [
            {"file": "font.ttf", "name":"bitmap","chars":[{"address":"A", "glyph":"A"}], "options":{"size":16}},
			{"file": "font.ttf", "name":"sdf","chars":[{"address":"A", "glyph":"A"}], "options":{"size":16,"sdf":true}}
		]
	}`)
	if err != nil {
		t.Fatal(err)
	}

	bitmap := cfg.Fonts[0]
	if bitmap.Options.SDF || bitmap.Options.SDFRadius != 0 || bitmap.Options.SDFCutoff != 0 {
		t.Fatalf("bitmap defaults changed: %+v", bitmap)
	}

	sdf := cfg.Fonts[1]
	if sdf.Options.SDFRadius != cDefaultSDFRadius || sdf.Options.SDFCutoff != cDefaultSDFCutoff {
		t.Fatalf("unexpected SDF defaults: %+v", sdf)
	}
}

func TestLoadConfigRejectsInvalidSDFSettings(t *testing.T) {
	_, err := loadTestConfig(t, `{
		"fonts": [{"file":"font.ttf","fonts":[{
			"name":"sdf","size":16,"chars":["A"],"glyphs":["A"],
			"sdf":true,"sdf_radius":-1
		}]}]
	}`)
	if err == nil {
		t.Fatal("expected invalid SDF radius to fail")
	}
}
