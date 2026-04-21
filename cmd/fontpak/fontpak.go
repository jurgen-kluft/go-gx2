package main

import (
	"fmt"
	"os"

	fontpack "go-gx2/fontpak"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("usage: fontpack <config.json> <out.bin>")
		os.Exit(1)
	}
	if cfg, err := fontpack.LoadConfig(os.Args[1]); err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	} else {
		if err := fontpack.BuildFontPak(cfg, os.Args[2]); err != nil {
			fmt.Printf("failed to build font pak: %v\n", err)
			os.Exit(1)
		}
	}
}
