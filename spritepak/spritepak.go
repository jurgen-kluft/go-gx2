package sprite_pak

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"go-gx2/tga"
	"image"
	_ "image/png"
	"io"
	"os"
	"strings"
)

//
// ===== JSON structures =====
//

type PackDesc struct {
	Files []FileDesc `json:"files"`
}

type FileDesc struct {
	File    string       `json:"file"`
	Sprites []SpriteDesc `json:"sprites"`
}

type SpriteDesc struct {
	Name   string `json:"name"`
	Format string `json:"format"`
	Rect   *Rect  `json:"rect,omitempty"`
}

type Rect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// ===== Binary enums =====
const (
	FMT_ALPHA_A0 = 0
	FMT_ALPHA_A1 = 1
	FMT_ALPHA_A4 = 4
	FMT_ALPHA_A8 = 8
)

const (
	FMT_RGB565   = 0x0100
	FMT_RGB565A1 = 0x0100 | FMT_ALPHA_A1
	FMT_RGB565A4 = 0x0100 | FMT_ALPHA_A4
	FMT_RGB565A8 = 0x0100 | FMT_ALPHA_A8
	FMT_RGBA8888 = 0x0200
	FMT_I8       = 0x0300
	FMT_I8A1     = 0x0300 | FMT_ALPHA_A1
	FMT_I8A4     = 0x0300 | FMT_ALPHA_A4
	FMT_I8A8     = 0x0300 | FMT_ALPHA_A8
)

func formatStringToEnum(s string) (uint16, error) {
	switch s {
	case "RGB565":
		return FMT_RGB565, nil
	case "RGB565A1":
		return FMT_RGB565A1, nil
	case "RGB565A4":
		return FMT_RGB565A4, nil
	case "RGB565A8":
		return FMT_RGB565A8, nil
	case "RGBA8888":
		return FMT_RGBA8888, nil
	case "I8":
		return FMT_I8, nil
	case "I8A1":
		return FMT_I8A1, nil
	case "I8A4":
		return FMT_I8A4, nil
	case "I8A8":
		return FMT_I8A8, nil
	}
	return 0, fmt.Errorf("unsupported format: %s", s)
}

//
// ===== Sprite table entry =====
//

type spriteEntry struct {
	Width             uint16
	Height            uint16
	Format            uint16
	Reserved          uint16
	PixelDataSize     uint32
	AlphaDataSize     uint32
	PixelDataOffset   uint64
	AlphaDataOffset   uint64
	PaletteDataOffset uint64
}

func (s spriteEntry) WriteBinary(w io.Writer) {
	binary.Write(w, binary.LittleEndian, s.Width)
	binary.Write(w, binary.LittleEndian, s.Height)
	binary.Write(w, binary.LittleEndian, s.Format)
	binary.Write(w, binary.LittleEndian, s.Reserved)
	binary.Write(w, binary.LittleEndian, s.PixelDataSize)
	binary.Write(w, binary.LittleEndian, s.AlphaDataSize)
	binary.Write(w, binary.LittleEndian, s.PixelDataOffset)
	binary.Write(w, binary.LittleEndian, s.AlphaDataOffset)
	binary.Write(w, binary.LittleEndian, s.PaletteDataOffset)
}

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

func analyzeAlpha(img image.Image, r Rect, alphaDisabled bool) uint16 {
	if alphaDisabled {
		return FMT_ALPHA_A0
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
		return FMT_ALPHA_A0
	}

	if len(alphas) == 1 {
		for a := range alphas {
			if a == 0xFF {
				return FMT_ALPHA_A1
			}
		}
		return FMT_ALPHA_A0
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
	indexedPixels []byte   // len = w*h
	paletteRGBA   []uint32 // 0xRRGGBBAA
	paletteRGB565 []uint16 // 0xRGB565
}

func buildIndexed8Palette(img image.Image, r Rect) (*paletteResult, bool) {

	colorIndex := make(map[uint32]byte)
	palette8888 := make([]uint32, 0, 256)
	indexed := make([]byte, r.W*r.H)

	p := 0
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			cr, cg, cb, ca := img.At(r.X+x, r.Y+y).RGBA()

			// canonical RGBA8888
			c := uint32(cr>>8)<<24 |
				uint32(cg>>8)<<16 |
				uint32(cb>>8)<<8 |
				uint32(ca>>8)

			idx, ok := colorIndex[c]
			if !ok {
				if len(palette8888) >= 256 {
					return nil, false
				}
				idx = byte(len(palette8888))
				colorIndex[c] = idx
				palette8888 = append(palette8888, c)
			}
			indexed[p] = idx
			p++
		}
	}

	palette565 := make([]uint16, len(palette8888))
	for i, c := range palette8888 {
		r := (c >> 24) & 0xFF
		g := (c >> 16) & 0xFF
		b := (c >> 8) & 0xFF

		r5 := (r >> 3) & 0x1F
		g6 := (g >> 2) & 0x3F
		b5 := (b >> 3) & 0x1F

		palette565[i] = uint16(r5<<11 | g6<<5 | b5)
	}

	return &paletteResult{
		indexedPixels: indexed,
		paletteRGBA:   palette8888,
		paletteRGB565: palette565,
	}, true
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

// RGB565
func encodeRGB565(img image.Image, r Rect) []byte {
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
	return pixels
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

			r5 := (cr >> 11) & 0x1F
			g6 := (cg >> 10) & 0x3F
			b5 := (cb >> 11) & 0x1F

			v := uint16(r5<<11 | g6<<5 | b5)
			pixels = append(pixels, byte(v), byte(v>>8))

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

// ===== Main writer =====
func writePack(outPath string, sprites []spriteEntry, pixelData [][]byte, alphaData [][]byte) error {

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// --- Helper for 8-byte alignment ---
	paddingData := make([]byte, 8)
	alignTo8 := func(offset int64) int64 {
		padding := (8 - (offset % 8)) % 8
		if padding > 0 {
			f.Write(paddingData[:padding])
		}
		return offset + padding
	}

	// --- Header placeholder ---
	binary.Write(f, binary.LittleEndian, uint64(0))            // sprite array offset
	binary.Write(f, binary.LittleEndian, uint32(len(sprites))) // sprite count
	binary.Write(f, binary.LittleEndian, uint32(0))            // reserved

	// --- Sprite table placeholder ---
	spriteArrayOffset, _ := f.Seek(0, io.SeekCurrent)
	for _, s := range sprites {
		s.WriteBinary(f)
	}

	var offset int64

	// --- Data blocks ---
	for i := range sprites {

		// --- Align to 8 bytes before writing pixel data ---
		offset, _ = f.Seek(0, io.SeekCurrent)
		offset = alignTo8(offset)
		sprites[i].PixelDataOffset = uint64(offset)
		f.Write(pixelData[i])

		if sprites[i].AlphaDataSize != 0 {
			offset, _ = f.Seek(0, io.SeekCurrent)
			// --- Align to 8 bytes before writing alpha data ---
			offset = alignTo8(offset)
			sprites[i].AlphaDataOffset = uint64(offset)
			f.Write(alphaData[i])
		}
	}

	// --- Rewrite header + table ---
	f.Seek(0, io.SeekStart)
	binary.Write(f, binary.LittleEndian, uint64(spriteArrayOffset))
	binary.Write(f, binary.LittleEndian, uint32(len(sprites)))
	binary.Write(f, binary.LittleEndian, uint32(0))

	for _, s := range sprites {
		s.WriteBinary(f)
	}

	return nil
}

func Build(jsonPath, outPath string) error {
	jdata, err := os.ReadFile(jsonPath)
	if err != nil {
		return err
	}

	var pack PackDesc
	if err := json.Unmarshal(jdata, &pack); err != nil {
		return err
	}

	var sprites []spriteEntry
	var pixelData [][]byte
	var alphaData [][]byte

	for _, f := range pack.Files {
		img, err := loadImage(f.File)
		if err != nil {
			return err
		}

		for _, s := range f.Sprites {
			r := fullRect(img)
			if s.Rect != nil {
				r = *s.Rect
			}

			formatEnum, err := formatStringToEnum(s.Format)
			if err != nil {
				return err
			}

			var px []byte
			var al []byte

			px = nil
			al = nil

			switch formatEnum {
			case FMT_RGB565:
				px = encodeRGB565(img, r)
			case FMT_RGB565A1:
				px, al = encodeRGB565A1(img, r)
			case FMT_RGB565A4:
				fmt.Printf("Warning: format %s not implemented yet, falling back to RGBA8888\n", s.Format)
				px = encodeRGBA8888(img, r)
				formatEnum = FMT_RGBA8888
			case FMT_RGB565A8:
				fmt.Printf("Warning: format %s not implemented yet, falling back to RGBA8888\n", s.Format)
				px = encodeRGBA8888(img, r)
				formatEnum = FMT_RGBA8888
			case FMT_RGBA8888:
				px = encodeRGBA8888(img, r)
			case FMT_I8:
				fmt.Printf("Warning: format %s not implemented yet, falling back to RGBA8888\n", s.Format)
				px = encodeRGBA8888(img, r)
				formatEnum = FMT_RGBA8888
			case FMT_I8A1:
				fmt.Printf("Warning: format %s not implemented yet, falling back to RGBA8888\n", s.Format)
				px = encodeRGBA8888(img, r)
				formatEnum = FMT_RGBA8888
			case FMT_I8A4:
				fmt.Printf("Warning: format %s not implemented yet, falling back to RGBA8888\n", s.Format)
				px = encodeRGBA8888(img, r)
				formatEnum = FMT_RGBA8888
			case FMT_I8A8:
				fmt.Printf("Warning: format %s not implemented yet, falling back to RGBA8888\n", s.Format)
				px = encodeRGBA8888(img, r)
				formatEnum = FMT_RGBA8888
			default:
				return fmt.Errorf("unsupported format: %s", s.Format)
			}

			sprites = append(sprites, spriteEntry{
				Width:         uint16(r.W),
				Height:        uint16(r.H),
				Format:        formatEnum,
				PixelDataSize: uint32(len(px)),
				AlphaDataSize: uint32(len(al)),
			})
			pixelData = append(pixelData, px)
			alphaData = append(alphaData, al)
		}
	}

	if err := writePack(outPath, sprites, pixelData, alphaData); err != nil {
		return err
	}

	fmt.Printf("Wrote %d sprites to %s\n", len(sprites), outPath)
	return nil
}
