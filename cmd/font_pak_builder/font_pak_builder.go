package main

import (
	"fmt"
	"os"

	fontpack "go-gx2/font_pak"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("usage: fontpack <config.json> <out.bin>")
		os.Exit(1)
	}

	cfg, err := fontpack.LoadConfig(os.Args[1])
	if err != nil {
		panic(err)
	}

	_ = cfg // wire into builder
	_ = os.WriteFile(os.Args[2], []byte{}, 0644)
}
