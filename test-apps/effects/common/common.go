package fx_common

import "math"

type FrameBuffer struct {
	Width  int
	Height int
	Pixels []uint16
}

type Vec2 struct {
	x float32
	y float32
}

func (v Vec2) Sub(other Vec2) Vec2 {
	return Vec2{x: v.x - other.x, y: v.y - other.y}
}

func (v Vec2) Add(other Vec2) Vec2 {
	return Vec2{x: v.x + other.x, y: v.y + other.y}
}

func (v Vec2) Scale(factor float32) Vec2 {
	return Vec2{x: v.x * factor, y: v.y * factor}
}

func (v Vec2) Dot(other Vec2) float32 {
	return v.x*other.x + v.y*other.y
}

func Cross(a, b Vec2) float32 {
	return a.x*b.y - a.y*b.x
}

func (v Vec2) Length() float32 {
	return float32(math.Sqrt(float64(v.x*v.x + v.y*v.y)))
}

func Abs(value int) int {
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

func ConvertToRGB565(r, g, b uint8) uint16 {
	return (uint16(r>>3) << 11) | (uint16(g>>2) << 5) | uint16(b>>3)
}
