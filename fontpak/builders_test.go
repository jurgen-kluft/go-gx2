package fontpack

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func singleFontConfig(path string, sdf bool) *FontPackCfg {
	return &FontPackCfg{Fonts: []*FontCfg{{
		File:  path,
		Name:  "test",
		Chars: []FontChar{{Address: "A", Glyph: "A"}},
		Options: FontOptions{
			FontSize:  16,
			SDF:       sdf,
			SDFBorder: 0,
		},
	}}}
}

func requireSingleGlyph(t *testing.T, pack []Font) Font {
	t.Helper()
	if len(pack) != 1 {
		t.Fatalf("expected one font, got %d", len(pack))
	}
	font := pack[0]
	if len(font.Glyphs) != 1 {
		t.Fatalf("expected one packed glyph: %+v", font)
	}

	return font
}

func compareBitmapAndSDF(t *testing.T, path string) Font {
	t.Helper()

	bitmapPack, err := Build(singleFontConfig(path, false))
	if err != nil {
		t.Fatal(err)
	}
	sdfPack, err := Build(singleFontConfig(path, true))
	if err != nil {
		t.Fatal(err)
	}
	if len(bitmapPack) != 1 || len(sdfPack) != 1 || bitmapPack[0].Name != sdfPack[0].Name {
		t.Fatalf("font names do not match: bitmap=%v sdf=%v", bitmapPack, sdfPack)
	}

	storedSDFBuffer := 0

	bitmap := requireSingleGlyph(t, bitmapPack)
	sdf := requireSingleGlyph(t, sdfPack)
	if bitmap.FontType != FontTypeBitmap || sdf.FontType != FontTypeSDF {
		t.Fatalf("unexpected font types: bitmap=%d sdf=%d", bitmap.FontType, sdf.FontType)
	}
	if got, want := int(sdf.Glyphs[0].Width), int(bitmap.Glyphs[0].Width)+2*storedSDFBuffer; got != want {
		t.Fatalf("SDF width %d, want %d", got, want)
	}
	if got, want := int(sdf.Glyphs[0].Height), int(bitmap.Glyphs[0].Height)+2*storedSDFBuffer; got != want {
		t.Fatalf("SDF height %d, want %d", got, want)
	}
	if got, want := int(sdf.Glyphs[0].BearingX), int(bitmap.Glyphs[0].BearingX)-storedSDFBuffer; got != want {
		t.Fatalf("SDF X bearing %d, want %d", got, want)
	}
	if got, want := int(sdf.Glyphs[0].BearingY), int(bitmap.Glyphs[0].BearingY)+storedSDFBuffer; got != want {
		t.Fatalf("SDF Y bearing %d, want %d", got, want)
	}
	if sdf.Glyphs[0].AdvanceX != bitmap.Glyphs[0].AdvanceX {
		t.Fatalf("SDF advance %d changed from %d", sdf.Glyphs[0].AdvanceX, bitmap.Glyphs[0].AdvanceX)
	}
	return sdf
}

func TestBuildTTFSDF(t *testing.T) {
	path := filepath.Join("sdf", "testdata", "NotoSans-Regular.ttf")
	sdf := compareBitmapAndSDF(t, path)

	var encoded bytes.Buffer
	if err := WritePack(&encoded, []Font{sdf}); err != nil {
		t.Fatal(err)
	}
	fonts, err := ReadPack(bytes.NewReader(encoded.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if len(fonts) != 1 || fonts[0].FontType != FontTypeSDF {
		t.Fatalf("SDF font type did not survive round trip: %+v", fonts)
	}
}

func TestBuildBDFSDF(t *testing.T) {
	const fixture = `STARTFONT 2.1
FONT test
SIZE 8 75 75
FONT_ASCENT 7
FONT_DESCENT 1
CHARS 1
STARTCHAR A
ENCODING 65
DWIDTH 6 0
BBX 5 7 0 0
BITMAP
70
88
88
F8
88
88
88
ENDCHAR
ENDFONT
`
	path := filepath.Join(t.TempDir(), "test.bdf")
	if err := os.WriteFile(path, []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}
	compareBitmapAndSDF(t, path)
}
