package fontpack

import (
	"io"

	"github.com/jurgen-kluft/go-datastream/codestream"
)

func WritePack(w io.Writer, fontPack *FontPack) error {

	if err := codestream.WriteToStream(w, fontPack); err != nil {
		return err
	}

	return nil
}
