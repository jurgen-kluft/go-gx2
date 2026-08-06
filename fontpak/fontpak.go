package fontpack

import (
	"fmt"
	"io"

	"github.com/jurgen-kluft/go-datastream/codestream"
)

// ReadPack reads a font pack from the provided reader and returns a slice of Font objects.
func ReadPack(r io.Reader) ([]Font, error) {
	fontPack := &FontPack{}
	if err := codestream.ReadFromStream(r, fontPack); err != nil {
		return nil, err
	}
	return fontPack.Fonts, nil
}

// WritePack writes the provided FontPack to the provided writer.
func WritePack(w io.Writer, fonts []Font, names []string) error {

	fmt.Println("Writing font pack...")

	for i, font := range fonts {
		PrintFontInfo(&font, names[i])
	}

	fontPack := &FontPack{
		Fonts: fonts,
	}
	if err := codestream.WriteToStream(w, fontPack); err != nil {
		return err
	}
	return nil
}
