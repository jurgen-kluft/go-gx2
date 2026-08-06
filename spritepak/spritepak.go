package spritepack

import (
	"bufio"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/jurgen-kluft/go-datastream/codestream"
	"github.com/jurgen-kluft/go-gx2/tga"
)

//
// ===== Image loading =====
//

func loadImage(filePath string) (image.Image, error) {
	imgFile, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer imgFile.Close()

	// if the extension is .tga, use the TGA decoder
	if strings.HasSuffix(filePath, ".tga") {
		img, err := tga.Decode(bufio.NewReader(imgFile))
		if err != nil {
			return nil, err
		}
		return img, nil
	}

	// otherwise, use the standard image decoder for PNG and other supported formats
	img, _, err := image.Decode(imgFile)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func fullRect(img image.Image) Rect {
	b := img.Bounds()
	return Rect{
		X: 0,
		Y: 0,
		W: b.Dx(),
		H: b.Dy(),
	}
}

func analyzeAlpha(img image.Image, r Rect, alphaDisabled bool) AlphaFormat {
	if alphaDisabled {
		return FMT_ALPHA_NONE
	}

	alphas := make(map[uint8]bool, 256)
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			_, _, _, ca := img.At(r.X+x, r.Y+y).RGBA()
			a := uint8(ca >> 8)
			alphas[a] = true
		}
	}

	if len(alphas) == 0 {
		return FMT_ALPHA_NONE
	}

	if len(alphas) == 1 {
		for a := range alphas {
			if a == 0xFF {
				return FMT_ALPHA_MASK
			}
		}
		return FMT_ALPHA_NONE
	}

	if len(alphas) <= 16 {
		return FMT_ALPHA_A4
	}

	return FMT_ALPHA_A8
}

//
// ===== Pixel encoders =====
//

type paletteResult struct {
	indexedPixels []byte // len = w*h
	paletteRGBA   PaletteRGBA
}

// ColorNode represents a cluster of colors
type ColorNode struct {
	// Floating-point state kept exclusively for precise center of gravity math
	R, G, B float32
	Count   float32

	// Pre-computed 8-bit values updated after each merge pass
	// These are used for zero-cost, collision-free binary search keys
	R8, G8, B8 uint8

	// The node's permanent index position inside the main nodes array
	ID uint16
}

// makeSearchKey runs purely on integers. It compiles down to a single
// bitshift and OR operation on the M1 processor.
func quantMakeSearchKey(color uint8, id uint16) uint32 {
	return (uint32(color) << 16) | uint32(id)
}

func quantBuildHistogram(rgb565Data []uint16) []ColorNode {
	// A static 65,536 array allocated on the stack/heap.
	// This acts as our perfect 16-bit color compressed space.
	var hist [65536]uint32
	var used [65536]uint8 // To track which colors are used

	// Tight, sequential loop. The M1's hardware prefetcher
	// will load this contiguous memory into cache instantly.
	for _, pixel := range rgb565Data {
		hist[pixel]++
		used[pixel] = 1
	}

	usedColors := 0
	for _, u := range used {
		usedColors += int(u)
	}
	if usedColors == 0 {
		return nil
	}

	// Pre-allocate a slice with a sensible capacity to prevent
	// Go from constantly re-allocating memory during appends.
	nodes := make([]ColorNode, 0, usedColors)

	// Single pass over the histogram to unpack colors
	for pixel, count := range hist {
		if count > 0 {
			// Fast bit-shifting and scaling to 0-255 range
			r := float32((pixel>>11)&0x1F) * (255.0 / 31.0)
			g := float32((pixel>>5)&0x3F) * (255.0 / 63.0)
			b := float32(pixel&0x1F) * (255.0 / 31.0)

			nodes = append(nodes, ColorNode{
				R:     r,
				G:     g,
				B:     b,
				Count: float32(count),
				ID:    uint16(len(nodes)),
			})
		}
	}
	return nodes
}

func (e *PNNEngine) quantShiftAxis(axis []uint16, nodeA, nodeB uint16, oldColorA, oldColorB, newColorA uint8, getColor func(*ColorNode) uint8) {
	keyA := quantMakeSearchKey(oldColorA, nodeA)
	keyB := quantMakeSearchKey(oldColorB, nodeB)

	posA := e.quantFindUniqueNodePosition(axis, keyA, getColor)
	posB := e.quantFindUniqueNodePosition(axis, keyB, getColor)

	pos1, pos2 := posA, posB
	if pos1 > pos2 {
		pos1, pos2 = pos2, pos1
	}

	newKeyA := quantMakeSearchKey(newColorA, nodeA)
	insPos := e.quantBinarySearchFindInsertUnique(axis, pos1, pos2, newKeyA, getColor)

	// Execute your 3 clean structural block copies
	if insPos <= pos1 {
		copy(axis[insPos+1:pos1+1], axis[insPos:pos1])
		copy(axis[pos1:pos2], axis[pos1+1:pos2+1])
		copy(axis[pos2:len(axis)-1], axis[pos2+2:])
		axis[insPos] = nodeA
	} else if insPos <= pos2 {
		copy(axis[pos1:insPos-1], axis[pos1+1:insPos])
		copy(axis[insPos:pos2], axis[pos1+1:pos2+1])
		copy(axis[pos2:len(axis)-1], axis[pos2+2:])
		axis[insPos-1] = nodeA
	} else {
		copy(axis[pos1:pos2-1], axis[pos1+1:pos2])
		copy(axis[pos2-1:insPos-2], axis[pos2+1:insPos-1])
		copy(axis[insPos-1:len(axis)-1], axis[insPos:])
		axis[insPos-2] = nodeA
	}
}

func (e *PNNEngine) quantFindUniqueNodePosition(axis []uint16, targetKey uint32, getColor func(*ColorNode) uint8) int {
	low := 0
	high := len(axis) - 1

	for low <= high {
		mid := (low + high) >> 1
		midNode := &e.nodes[axis[mid]]
		midKey := quantMakeSearchKey(getColor(midNode), midNode.ID)

		if midKey < targetKey {
			low = mid + 1
		} else if midKey > targetKey {
			high = mid - 1
		} else {
			return mid
		}
	}
	return low
}

func (e *PNNEngine) quantBinarySearchFindInsertUnique(axis []uint16, pos1, pos2 int, targetKey uint32, getColor func(*ColorNode) uint8) int {
	low := 0
	high := len(axis) - 1

	for low <= high {
		mid := (low + high) >> 1

		adjMid := mid
		if adjMid >= pos1 {
			adjMid++
		}
		if adjMid >= pos2 {
			adjMid++
		}

		if adjMid >= len(axis) {
			return mid
		}

		midNode := &e.nodes[axis[adjMid]]
		midKey := quantMakeSearchKey(getColor(midNode), midNode.ID)
		if targetKey < midKey {
			high = mid - 1
		} else {
			low = mid + 1
		}
	}
	return low
}

type PNNEngine struct {
	nodes       []ColorNode
	axisR       []uint16
	axisG       []uint16
	axisB       []uint16
	activeCount int
}

func quantPNN(nodes []ColorNode, targetPaletteSize int) []ColorNode {
	if len(nodes) <= targetPaletteSize {
		return nodes
	}

	// Perceptual distance combined with PNN merge cost formula
	mergeCost := func(a, b *ColorNode) float32 {
		dr := a.R - b.R
		dg := a.G - b.G
		db := a.B - b.B
		distSq := (2.0 * dr * dr) + (4.0 * dg * dg) + (3.0 * db * db)
		return (a.Count * b.Count / (a.Count + b.Count)) * distSq
	}

	numNodes := len(nodes)
	engine := &PNNEngine{
		nodes:       nodes,
		axisR:       make([]uint16, numNodes),
		axisG:       make([]uint16, numNodes),
		axisB:       make([]uint16, numNodes),
		activeCount: numNodes,
	}

	for i := 0; i < numNodes; i++ {
		idx := uint16(i)
		engine.axisR[i], engine.axisG[i], engine.axisB[i] = idx, idx, idx
		engine.nodes[i].ID = idx
	}

	// 1. Initial Sorting using your integer search keys
	sort.Slice(engine.axisR, func(i, j int) bool {
		ni, nj := &engine.nodes[engine.axisR[i]], &engine.nodes[engine.axisR[j]]
		return quantMakeSearchKey(ni.R8, ni.ID) < quantMakeSearchKey(nj.R8, nj.ID)
	})
	sort.Slice(engine.axisG, func(i, j int) bool {
		ni, nj := &engine.nodes[engine.axisG[i]], &engine.nodes[engine.axisG[j]]
		return quantMakeSearchKey(ni.G8, ni.ID) < quantMakeSearchKey(nj.G8, nj.ID)
	})
	sort.Slice(engine.axisB, func(i, j int) bool {
		ni, nj := &engine.nodes[engine.axisB[i]], &engine.nodes[engine.axisB[j]]
		return quantMakeSearchKey(ni.B8, ni.ID) < quantMakeSearchKey(nj.B8, nj.ID)
	})

	// 2. CORE MUTATION LOOP
	for engine.activeCount > targetPaletteSize {
		minCost := float32(math.MaxFloat32)
		bestA, bestB := -1, -1

		// 3. SEAMLESS SEQUENTIAL SWEEP PASS
		for p := 0; p < engine.activeCount; p++ {
			i := int(engine.axisR[p])

			for q := p + 1; q < engine.activeCount; q++ {
				j := int(engine.axisR[q])

				dr := engine.nodes[j].R - engine.nodes[i].R
				if (2.0 * dr * dr) >= minCost {
					break
				}

				dg := engine.nodes[j].G - engine.nodes[i].G
				if (4.0 * dg * dg) >= minCost {
					continue
				}

				db := engine.nodes[j].B - engine.nodes[i].B
				if (3.0 * db * db) >= minCost {
					continue
				}

				cost := mergeCost(&engine.nodes[i], &engine.nodes[j])
				if cost < minCost {
					minCost = cost
					bestA = i
					bestB = j
				}
			}
		}

		if bestA == -1 {
			break
		}

		a := &engine.nodes[bestA]
		b := &engine.nodes[bestB]
		combinedCount := a.Count + b.Count

		// Record old keys before mutating coordinates
		oldR8A, oldG8A, oldB8A := a.R8, a.G8, a.B8
		oldR8B, oldG8B, oldB8B := b.R8, b.G8, b.B8

		// Precise floating point merge
		newR := ((a.R * a.Count) + (b.R * b.Count)) / combinedCount
		newG := ((a.G * a.Count) + (b.G * b.Count)) / combinedCount
		newB := ((a.B * a.Count) + (b.B * b.Count)) / combinedCount

		a.R, a.G, a.B, a.Count = newR, newG, newB, combinedCount

		// Update our fast integer indexing values with rounded floats
		a.R8 = uint8(newR + 0.5)
		a.G8 = uint8(newG + 0.5)
		a.B8 = uint8(newB + 0.5)

		// 4. BLOCK SHIFT MAINTENANCE PASS
		engine.axisR = engine.axisR[:engine.activeCount]
		engine.axisG = engine.axisG[:engine.activeCount]
		engine.axisB = engine.axisB[:engine.activeCount]

		engine.quantShiftAxis(engine.axisR, uint16(bestA), uint16(bestB), oldR8A, oldR8B, a.R8, func(n *ColorNode) uint8 { return n.R8 })
		engine.quantShiftAxis(engine.axisG, uint16(bestA), uint16(bestB), oldG8A, oldG8B, a.G8, func(n *ColorNode) uint8 { return n.G8 })
		engine.quantShiftAxis(engine.axisB, uint16(bestA), uint16(bestB), oldB8A, oldB8B, a.B8, func(n *ColorNode) uint8 { return n.B8 })

		engine.activeCount--
	}

	palette := make([]ColorNode, engine.activeCount)
	for i := 0; i < engine.activeCount; i++ {
		palette[i] = engine.nodes[engine.axisR[i]]
	}
	return palette
}

func buildIndexed8Palette(img image.Image, r Rect, pal PaletteRGBA) (pixels []byte, ok bool) {

	// Step 1: Convert the image to the rectangle defined size with RGB565 format
	rgb565Img := make([]uint16, 0, r.W*r.H*2)
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			cr, cg, cb, _ := img.At(r.X+x, r.Y+y).RGBA()
			rgbaColor := NewColorFromR8G8B8A8(255, uint8(cr>>8), uint8(cg>>8), uint8(cb>>8))
			rgb565Img = append(rgb565Img, rgbaColor.ToRGB565())
		}
	}

	// Step 2: Build a list of palette nodes for fast nearest neighbor search
	paletteNodes := make([]ColorNode, len(pal))
	for i, color := range pal {
		r, g, b, _ := color.ToR8G8B8A8()
		paletteNodes[i] = ColorNode{
			R:  float32(r),
			G:  float32(g),
			B:  float32(b),
			R8: r,
			G8: g,
			B8: b,
			ID: uint16(i),
		}
	}

	// 1. Create a Lookup Table (LUT) for all 65,536 possible RGB565 values.
	// We use 0xFF as an uninitialized marker flag.
	lut := make([]uint8, 65536)
	for i := range lut {
		lut[i] = 0xFF
	}

	// 2. Pre-allocate the output slice for the indexed image pixels.
	indexedImage := make([]uint8, len(rgb565Img))

	// 3. Process the image sequentially.
	// The M1's hardware branch predictor will optimize this loop perfectly.
	for i, pixel := range rgb565Img {
		// If we have already solved the nearest palette index for this color, reuse it!
		if lut[pixel] != 0xFF {
			indexedImage[i] = lut[pixel]
			continue
		}

		// Otherwise, unpack the RGB565 pixel to find its nearest palette match
		r := float32((pixel>>11)&0x1F) * (255.0 / 31.0)
		g := float32((pixel>>5)&0x3F) * (255.0 / 63.0)
		b := float32(pixel&0x1F) * (255.0 / 31.0)

		var minDistance float32 = 1e30 // Start with a massive float number
		var bestIndex uint8 = 0

		// Find the closest palette color using Perceptual Euclidean Distance
		for palIdx, color := range paletteNodes {
			dr := r - color.R
			dg := g - color.G
			db := b - color.B

			// Perceptual weighting: Red=2, Green=4, Blue=3
			distanceSq := (2.0 * dr * dr) + (4.0 * dg * dg) + (3.0 * db * db)

			if distanceSq < minDistance {
				minDistance = distanceSq
				bestIndex = uint8(palIdx)
			}
		}

		// Cache the result in our LUT and assign it to the image array
		lut[pixel] = bestIndex
		indexedImage[i] = bestIndex
	}

	return indexedImage, true
}

func buildPalette(img image.Image) (pal PaletteRGBA, err error) {

	// Step 1: Convert the image to the rectangle defined size with RGB565 format
	rgb565Img := make([]uint16, 0, img.Bounds().Dx()*img.Bounds().Dy()*2)
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			cr, cg, cb, _ := img.At(img.Bounds().Min.X+x, img.Bounds().Min.Y+y).RGBA()
			rgbaColor := NewColorFromR8G8B8A8(255, uint8(cr>>8), uint8(cg>>8), uint8(cb>>8))
			rgb565Img = append(rgb565Img, rgbaColor.ToRGB565())
		}
	}

	// Step 2: Build a histogram of unique colors in the RGB565 data
	histogram := quantBuildHistogram(rgb565Img)

	// Step 3: Reduce the histogram to a maximum of 256 colors using PNN
	paletteNodes := quantPNN(histogram, 256)

	if len(rgb565Img) == 0 || len(paletteNodes) == 0 {
		return nil, fmt.Errorf("empty image or palette nodes")
	}

	pal = make([]ColorRGBA, len(paletteNodes))
	for i, node := range paletteNodes {
		pal[i] = NewColorFromR8G8B8A8(255, uint8(node.R), uint8(node.G), uint8(node.B))
	}

	return pal, nil
}

// RGB565 + A0 (no separate alpha bitstream)
func encodeRGB565A0(img image.Image, r Rect) ([]byte, []byte) {
	pixels := make([]byte, 0, r.W*r.H*2)

	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			cr, cg, cb, _ := img.At(r.X+x, r.Y+y).RGBA()

			r5 := (cr >> 11) & 0x1F
			g6 := (cg >> 10) & 0x3F
			b5 := (cb >> 11) & 0x1F

			v := uint16(r5<<11 | g6<<5 | b5)
			pixels = append(pixels, byte(v), byte(v>>8))
		}
	}
	return pixels, []byte{}
}

// RGB565 + A1 (separate alpha bitstream)
func encodeRGB565A1(img image.Image, r Rect) ([]byte, []byte) {
	pixels := make([]byte, 0, r.W*r.H*2)
	alpha := make([]byte, 0, (r.W*r.H+7)/8)

	var abit byte
	var acnt uint

	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			cr, cg, cb, ca := img.At(r.X+x, r.Y+y).RGBA()
			colorRGBA8888 := NewColorFromR8G8B8A8(uint8(ca>>8), uint8(cr>>8), uint8(cg>>8), uint8(cb>>8))
			colorRGB565 := colorRGBA8888.ToRGB565()

			pixels = append(pixels, byte(colorRGB565), byte(colorRGB565>>8))

			if ca >= 0x8000 {
				abit |= 1 << acnt
			}
			acnt++
			if acnt == 8 {
				alpha = append(alpha, abit)
				abit = 0
				acnt = 0
			}
		}
	}
	if acnt != 0 {
		alpha = append(alpha, abit)
	}
	return pixels, alpha
}

// RGB565 + A4 (separate alpha bitstream)
func encodeRGB565A4(img image.Image, r Rect) ([]byte, []byte) {
	pixels := make([]byte, 0, r.W*r.H*2)
	alpha := make([]byte, 0, (r.W*r.H+1)/2)

	var abit byte
	var acnt uint

	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			cr, cg, cb, ca := img.At(r.X+x, r.Y+y).RGBA()
			colorRGBA8888 := NewColorFromR8G8B8A8(uint8(ca>>8), uint8(cr>>8), uint8(cg>>8), uint8(cb>>8))
			colorRGB565 := colorRGBA8888.ToRGB565()

			pixels = append(pixels, byte(colorRGB565), byte(colorRGB565>>8))

			abit |= (byte(ca>>4) & 0x0F) << (4 * acnt)
			acnt++
			if acnt == 2 {
				alpha = append(alpha, abit)
				abit = 0
				acnt = 0
			}
		}
	}
	if acnt != 0 {
		alpha = append(alpha, abit)
	}
	return pixels, alpha
}

// RGB565 + A8 (separate alpha bitstream)
func encodeRGB565A8(img image.Image, r Rect) ([]byte, []byte) {
	pixels := make([]byte, 0, r.W*r.H*2)
	alpha := make([]byte, 0, r.W*r.H)

	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			cr, cg, cb, ca := img.At(r.X+x, r.Y+y).RGBA()
			colorRGBA8888 := NewColorFromR8G8B8A8(uint8(ca>>8), uint8(cr>>8), uint8(cg>>8), uint8(cb>>8))
			colorRGB565 := colorRGBA8888.ToRGB565()

			pixels = append(pixels, byte(colorRGB565), byte(colorRGB565>>8))
			alpha = append(alpha, byte(ca>>8))
		}
	}
	return pixels, alpha
}

// RGBA8888
func encodeRGBA8888(img image.Image, r Rect) []byte {
	pixels := make([]byte, 0, r.W*r.H*4)

	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			cr, cg, cb, ca := img.At(r.X+x, r.Y+y).RGBA()
			pixels = append(
				pixels,
				byte(cr>>8),
				byte(cg>>8),
				byte(cb>>8),
				byte(ca>>8),
			)
		}
	}
	return pixels
}

func Build(cfgs *SpritePackCfg) ([]Sprite, []PaletteRGBA, error) {

	palettes := make([]PaletteRGBA, 0, 16)
	palettesMap := map[string]int{}

	// Go through all the sprites and build the palettes first, then build the sprites.
	for _, cfg := range cfgs.Files {
		if cfg.PaletteFile != "" {
			paletteImage, err := loadImage(cfg.PaletteFile)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to load palette image %s: %w", cfg.PaletteFile, err)
			}
			pal, err := buildPalette(paletteImage)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to build palette %s: %w", cfg.PaletteFile, err)
			}
			palettesMap[cfg.PaletteFile] = len(palettes)
			palettes = append(palettes, pal)
		}
	}

	sprites := make([]Sprite, 0, 1024)

	for _, cfg := range cfgs.Files {

		img, err := loadImage(cfg.ImageFile)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load image %s: %w", cfg.ImageFile, err)
		}

		for _, s := range cfg.Sprites {
			r := fullRect(img)
			if s.Rect != nil {
				r = *s.Rect
			}

			pixelFormatEnum, err := PixelFormatFromString(s.PixelFormat)
			if err != nil {
				return nil, nil, err
			}
			alphaFormatEnum, err := AlphaFormatFromString(s.AlphaFormat)
			if err != nil {
				return nil, nil, err
			}

			var px []byte
			var al []byte

			px = nil
			al = nil

			switch pixelFormatEnum {
			case FMT_PIXEL_RGB565:
				switch alphaFormatEnum {
				case FMT_ALPHA_NONE:
					px, al = encodeRGB565A0(img, r)
				case FMT_ALPHA_MASK:
					px, al = encodeRGB565A1(img, r)
				case FMT_ALPHA_A4:
					px, al = encodeRGB565A4(img, r)
				case FMT_ALPHA_A8:
					px, al = encodeRGB565A8(img, r)
				default:
					return nil, nil, fmt.Errorf("unsupported alpha format for RGB565: %s", s.AlphaFormat)
				}
			case FMT_PIXEL_RGB888:
				px = encodeRGBA8888(img, r)
				// Alpha is embedded in RGBA8888, so we can ignore the alphaFormatEnum here
				alphaFormatEnum = FMT_ALPHA_NONE
			case FMT_PIXEL_I8:
				palIdx, ok := palettesMap[cfg.PaletteFile]
				if !ok {
					return nil, nil, fmt.Errorf("palette not found for I8 format: %s", cfg.PaletteFile)
				}
				pal := palettes[palIdx]
				px, ok = buildIndexed8Palette(img, r, pal)
				if !ok {
					return nil, nil, fmt.Errorf("failed to build indexed 8-bit image %s", cfg.ImageFile)
				}
			default:
				return nil, nil, fmt.Errorf("unsupported format: %s", s.PixelFormat)
			}

			sprites = append(sprites, Sprite{
				Width:       uint16(r.W),
				Height:      uint16(r.H),
				PixelFormat: pixelFormatEnum,
				AlphaFormat: alphaFormatEnum,
				PixelData:   px,
				AlphaData:   al,
			})
		}
	}

	return sprites, palettes, nil
}

// ReadPack reads a binary spritepak file and returns a SpritePack.
func ReadPack(r io.Reader) (*SpritePack, error) {
	spritePack := SpritePack{
		Sprites: []Sprite{},
	}

	if err := codestream.ReadFromStream(r, &spritePack); err != nil {
		return nil, err
	}

	return &spritePack, nil
}

// WritePack writes a SpritePack to a binary spritepak file.
func WritePack(w io.Writer, sprites []Sprite) error {
	spritePack := SpritePack{
		Sprites: sprites,
	}

	if err := codestream.WriteToStream(w, spritePack); err != nil {
		return err
	}

	return nil
}
