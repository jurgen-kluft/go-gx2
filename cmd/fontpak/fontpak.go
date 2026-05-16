package main

import (
	"fmt"
	"os"

	font_pak "github.com/jurgen-kluft/go-gx2/fontpak"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("usage: fontpack <config.json> <out.bin>")
		os.Exit(1)
	}
	if err, cfg := font_pak.LoadConfig(os.Args[1]); err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	} else {
		if err, fontPak := font_pak.BuildFontPak(cfg); err != nil {
			fmt.Printf("failed to build font pak: %v\n", err)
			os.Exit(1)
		} else {
			f, err := os.Create(os.Args[2])
			if err != nil {
				fmt.Printf("failed to create output file: %v\n", err)
				os.Exit(1)
			}
			defer f.Close()

			if err := font_pak.WritePack(f, fontPak); err != nil {
				fmt.Printf("failed to write font pak: %v\n", err)
				os.Exit(1)
			}
		}
	}
}
