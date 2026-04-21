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
	Alpha  string `json:"alpha"`
	Rect   *Rect  `json:"rect,omitempty"`
}

type Rect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

//
// ===== Binary enums =====
//

const (
	FMT_RGB565   = 1
	FMT_RGBA8888 = 2
)

//
// ===== Sprite table entry =====
//

type SpriteEntry struct {
	Width         uint16
	Height        uint16
	Format        uint16
	AlphaFormat   uint16
	PixelSize     uint32
	PixelOffset   uint64
	AlphaSize     uint32
	AlphaOffset   uint64
	PaletteOffset uint64
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

// ==== Alpha utilities =====
const (
	ALPHA_A0 = 0
	ALPHA_A1 = 1
	ALPHA_A4 = 4
	ALPHA_A8 = 8
)

func AnalyzeAlpha(img image.Image, r Rect, alphaDisabled bool) uint16 {
	if alphaDisabled {
		return ALPHA_A0
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
		return ALPHA_A0
	}

	if len(alphas) == 1 {
		for a := range alphas {
			if a == 0xFF {
				return ALPHA_A1
			}
		}
		return ALPHA_A0
	}

	if len(alphas) <= 16 {
		return ALPHA_A4
	}

	return ALPHA_A8
}

//
// ===== Pixel encoders =====
//

type PaletteResult struct {
	IndexedPixels []byte   // len = w*h
	PaletteRGBA   []uint32 // 0xRRGGBBAA
	PaletteRGB565 []uint16 // 0xRGB565
}

func BuildIndexed8Palette(img image.Image, r Rect) (*PaletteResult, bool) {

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

	return &PaletteResult{
		IndexedPixels: indexed,
		PaletteRGBA:   palette8888,
		PaletteRGB565: palette565,
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
func writePack(outPath string, sprites []SpriteEntry, pixelData [][]byte, alphaData [][]byte) error {

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// --- Header placeholder ---
	binary.Write(f, binary.LittleEndian, uint64(0))            // sprite array offset
	binary.Write(f, binary.LittleEndian, uint32(len(sprites))) // sprite count
	binary.Write(f, binary.LittleEndian, uint32(0))            // padding

	spriteArrayOffset, _ := f.Seek(0, io.SeekCurrent)

	// --- Sprite table placeholder ---
	for _, s := range sprites {
		binary.Write(f, binary.LittleEndian, s)
	}

	// --- Data blocks ---
	for i := range sprites {
		offset, _ := f.Seek(0, io.SeekCurrent)
		sprites[i].PixelOffset = uint64(offset)
		f.Write(pixelData[i])

		if sprites[i].AlphaSize != 0 {
			offset, _ := f.Seek(0, io.SeekCurrent)
			sprites[i].AlphaOffset = uint64(offset)
			f.Write(alphaData[i])
		}
	}

	// --- Rewrite header + table ---
	f.Seek(0, io.SeekStart)
	binary.Write(f, binary.LittleEndian, uint64(spriteArrayOffset))
	binary.Write(f, binary.LittleEndian, uint32(len(sprites)))
	binary.Write(f, binary.LittleEndian, uint32(0))

	for _, s := range sprites {
		binary.Write(f, binary.LittleEndian, s)
	}

	return nil
}

func Build(jsonPath, outPath string) error {
	jdata, err := os.ReadFile(jsonPath)
	if err != nil {
		panic(err)
	}

	var pack PackDesc
	if err := json.Unmarshal(jdata, &pack); err != nil {
		panic(err)
	}

	var sprites []SpriteEntry
	var pixelData [][]byte
	var alphaData [][]byte

	for _, f := range pack.Files {
		img, err := loadImage(f.File)
		if err != nil {
			panic(err)
		}

		for _, s := range f.Sprites {
			r := fullRect(img)
			if s.Rect != nil {
				r = *s.Rect
			}

			switch s.Format {
			case "RGB565":
				px, al := encodeRGB565A1(img, r)
				sprites = append(sprites, SpriteEntry{
					Width:       uint16(r.W),
					Height:      uint16(r.H),
					Format:      FMT_RGB565,
					AlphaFormat: ALPHA_A1,
					PixelSize:   uint32(len(px)),
					AlphaSize:   uint32(len(al)),
				})
				pixelData = append(pixelData, px)
				alphaData = append(alphaData, al)

			case "RGBA8888":
				px := encodeRGBA8888(img, r)
				sprites = append(sprites, SpriteEntry{
					Width:       uint16(r.W),
					Height:      uint16(r.H),
					Format:      FMT_RGBA8888,
					AlphaFormat: ALPHA_A0,
					PixelSize:   uint32(len(px)),
				})
				pixelData = append(pixelData, px)
				alphaData = append(alphaData, nil)

			default:
				panic("unsupported format: " + s.Format)
			}
		}
	}

	if err := writePack(outPath, sprites, pixelData, alphaData); err != nil {
		panic(err)
	}

	fmt.Printf("Wrote %d sprites to %s\n", len(sprites), outPath)

	return nil
}
