package fontpack

import (
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
func WritePack(w io.Writer, fontPack *FontPack) error {
	if err := codestream.WriteToStream(w, fontPack); err != nil {
		return err
	}
	return nil
}
