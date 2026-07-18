package sdf_font

import (
	"fmt"
	"os"
	"testing"

	"github.com/golang/freetype/truetype"
)

func builderFor(fontFamily string) *SDFBuilder {
	ttf, err := os.ReadFile("./testdata/" + fontFamily + ".ttf")
	if err != nil {
		panic(err)
	}

	font, err := truetype.Parse(ttf)

	if err != nil {
		panic(err)
	}

	return NewSDFBuilder(font, SDFBuilderOpt{FontSize: 26, Buffer: 3})
}

func TestSDFBuilder_Glyph(t *testing.T) {
	builder := builderFor("NotoSans-Regular")

	size := 0
	for i := 0; i < 126; i++ {
		g := builder.Glyph(rune(i))
		if g != nil {
			size += len(g.Bitmap)
			fmt.Printf("%d %d\n", i, g.Top)
			img := DrawGlyph(g, true)
			SavePNG(fmt.Sprintf("./testdata/NotoSans/%d.png", i), img)
		}
	}
	fmt.Printf("Total size of glyphs: %d bytes\n", size)
}

func TestSDFBuilder(t *testing.T) {
	t.Run("#Glyphs", func(t *testing.T) {
		builder := builderFor("NotoSans-Regular")

		for _, rng := range [][]int{
			{0, 255},
			{20224, 20479},
			{22784, 23039},
		} {
			glyphs := builder.Glyphs(rng[0], rng[1])
			if len(glyphs) != rng[1]-rng[0] {
				t.Fatalf("failed to marshal glyphs: expected %d, got %d", rng[1]-rng[0], len(glyphs))
			}
		}
	})
}
