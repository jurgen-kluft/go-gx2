package main

import (
	"fmt"
	"os"
	"path/filepath"

	sprite_pak "github.com/jurgen-kluft/go-gx2/spritepak"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Printf("Usage: %s pack.json output.bin\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}

	jsonPath := os.Args[1]
	outPath := os.Args[2]

	if spritePackCfg, err := sprite_pak.LoadConfig(jsonPath); err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	} else if sprites, err := sprite_pak.Build(spritePackCfg); err != nil {
		fmt.Printf("Error building sprite pak: %v\n", err)
		os.Exit(1)
	} else {
		f, err := os.Create(outPath)
		if err != nil {
			fmt.Printf("Error creating output file: %v\n", err)
			os.Exit(1)
		}
		if err := sprite_pak.WritePack(f, sprites); err != nil {
			fmt.Printf("Error writing sprite pak: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Built sprite pak: %s\n", outPath)
	}
}
