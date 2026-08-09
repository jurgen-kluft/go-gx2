package palpak

type ColorFormat uint8

const (
	FMT_COLOR_RGB565 ColorFormat = 0x01 // RGB565 (16-bit) with no alpha
	FMT_COLOR_RGB888 ColorFormat = 0x02 // RGB888 (24-bit)
)
