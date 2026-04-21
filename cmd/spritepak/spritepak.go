package main

import (
	"fmt"
	sprite_pak "go-gx2/spritepak"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Printf("Usage: %s pack.json output.bin\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}

	jsonPath := os.Args[1]
	outPath := os.Args[2]

	if err := sprite_pak.Build(jsonPath, outPath); err != nil {
		fmt.Printf("Error building sprite pak: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Built sprite pak: %s\n", outPath)
}
