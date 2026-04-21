package main

import (
	"fmt"
	"go-gx2/sprite_pak"
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

	sprite_pak.Build(jsonPath, outPath)
	fmt.Printf("Built sprite pak: %s\n", outPath)
}
