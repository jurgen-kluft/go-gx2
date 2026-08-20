package fx_common

import "math"

type FrameBuffer struct {
	Width  int32
	Height int32
	Pixels []uint16
}

// 888     888 8888888888 .d8888b. 88888888888 .d88888b.  8888888b.        .d8888b.  8888888b.
// 888     888 888       d88P  Y88b    888    d88P" "Y88b 888   Y88b      d88P  Y88b 888  "Y88b
// 888     888 888       888    888    888    888     888 888    888             888 888    888
// Y88b   d88P 8888888   888           888    888     888 888   d88P           .d88P 888    888
//  Y88b d88P  888       888           888    888     888 8888888P"        .od888P"  888    888
//   Y88o88P   888       888    888    888    888     888 888 T88b        d88P"      888    888
//    Y888P    888       Y88b  d88P    888    Y88b. .d88P 888  T88b       888"       888  .d88P
//     Y8P     8888888888 "Y8888P"     888     "Y88888P"  888   T88b      888888888  8888888P"

type Vec2f struct {
	x float32
	y float32
}

func (v Vec2f) Sub(other Vec2f) Vec2f {
	return Vec2f{x: v.x - other.x, y: v.y - other.y}
}

func (v Vec2f) Add(other Vec2f) Vec2f {
	return Vec2f{x: v.x + other.x, y: v.y + other.y}
}

func (v Vec2f) Scale(factor float32) Vec2f {
	return Vec2f{x: v.x * factor, y: v.y * factor}
}

func (v Vec2f) Dot(other Vec2f) float32 {
	return v.x*other.x + v.y*other.y
}

func Cross(a, b Vec2f) float32 {
	return a.x*b.y - a.y*b.x
}

func (v Vec2f) Length() float32 {
	return float32(math.Sqrt(float64(v.x*v.x + v.y*v.y)))
}

func Abs(value int32) int32 {
	if value < 0 {
		return -value
	}
	return value
}

func AbsFloat32(value float32) float32 {
	if value < 0 {
		return -value
	}
	return value
}

// 8888888b.         d8888 888b    888 8888888b.   .d88888b.  888b     d888
// 888   Y88b       d88888 8888b   888 888  "Y88b d88P" "Y88b 8888b   d8888
// 888    888      d88P888 88888b  888 888    888 888     888 88888b.d88888
// 888   d88P     d88P 888 888Y88b 888 888    888 888     888 888Y88888P888
// 8888888P"     d88P  888 888 Y88b888 888    888 888     888 888 Y888P 888
// 888 T88b     d88P   888 888  Y88888 888    888 888     888 888  Y8P  888
// 888  T88b   d8888888888 888   Y8888 888  .d88P Y88b. .d88P 888   "   888
// 888   T88b d88P     888 888    Y888 8888888P"   "Y88888P"  888       888

type RandU uint32

// pseudoRand implements your exact PCG-XSH-RR hash mixing function
func (m *RandU) PseudoRand() uint32 {
	state := uint32(*m)*uint32(747796405) + uint32(2891336453)
	word := ((state >> ((state >> 28) + 4)) ^ state) * uint32(277803737)
	*m += 1
	return (word >> 22) ^ word
}

// customRandFloat scales pseudoRand output into a specific [min, max) range.
func (m *RandU) CustomRandFloat(min, max float32) float32 {
	unit := float32(m.PseudoRand()>>8) / float32(1<<24)
	return min + unit*(max-min)
}

//  .d8888b.   .d88888b.  888      .d88888b.  8888888b.
// d88P  Y88b d88P" "Y88b 888     d88P" "Y88b 888   Y88b
// 888    888 888     888 888     888     888 888    888
// 888        888     888 888     888     888 888   d88P
// 888        888     888 888     888     888 8888888P"
// 888    888 888     888 888     888     888 888 T88b
// Y88b  d88P Y88b. .d88P 888     Y88b. .d88P 888  T88b
//  "Y8888P"   "Y88888P"  88888888 "Y88888P"  888   T88b

func ConvertToRGB565(r, g, b uint8) uint16 {
	return (uint16(r>>3) << 11) | (uint16(g>>2) << 5) | uint16(b>>3)
}

// 8888888888 8888888 Y88b   d88P 8888888888 8888888b.  8888888b.   .d88888b. 8888888 888b    888 88888888888
// 888          888    Y88b d88P  888        888  "Y88b 888   Y88b d88P" "Y88b  888   8888b   888     888
// 888          888     Y88o88P   888        888    888 888    888 888     888  888   88888b  888     888
// 8888888      888      Y888P    8888888    888    888 888   d88P 888     888  888   888Y88b 888     888
// 888          888      d888b    888        888    888 8888888P"  888     888  888   888 Y88b888     888
// 888          888     d88888b   888        888    888 888        888     888  888   888  Y88888     888
// 888          888    d88P Y88b  888        888  .d88P 888        Y88b. .d88P  888   888   Y8888     888
// 888        8888888 d88P   Y88b 8888888888 8888888P"  888         "Y88888P" 8888888 888    Y888     888

type Fp32 int32

const (
	FpOne  Fp32 = 1 << 16 // Represents the fixed-point value 1.0
	FpHalf Fp32 = 1 << 15 // Represents the fixed-point value 0.5
)

func Float32ToFp32(value float32) Fp32 {
	return Fp32(value * float32(FpOne))
}

func (f Fp32) Multiply(other Fp32) Fp32 {
	return Fp32((int64(f) * int64(other)) >> 16)
}

func (f Fp32) Divide(other Fp32) Fp32 {
	return Fp32((int64(f) << 16) / int64(other))
}

func (f Fp32) Add(other Fp32) Fp32 {
	return Fp32(int32(int64(f) + int64(other)))
}

func (f Fp32) Subtract(other Fp32) Fp32 {
	return Fp32(int32(int64(f) - int64(other)))
}

func (f Fp32) ToInt32() int32 {
	return int32(f >> 16)
}
