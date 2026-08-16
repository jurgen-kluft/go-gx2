package fx_ripple

import (
	"math"
)

func absf(a float32) float32 {
	if a < 0 {
		return -a
	}
	return a
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func clampf(value, min, max float32) float32 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func clamp(value, min, max int32) int32 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func floorf(a float32) float32 {
	return float32(int(a))
}

func ceilf(a float32) float32 {
	return float32(int(a) + 1)
}

func hypotf(x, y float32) float32 {
	return float32(math.Hypot(float64(x), float64(y)))
}

func distanceSquaredf(x1, y1, x2, y2 float32) float32 {
	dx := x2 - x1
	dy := y2 - y1
	return dx*dx + dy*dy
}

func sqrtf(a float32) float32 {
	return float32(math.Sqrt(float64(a)))
}

func lerpf(a, b, t float32) float32 {
	return a + (b-a)*t
}

func inverseLerpf(a, b, value float32) float32 {
	if a == b {
		return 0
	}

	return (value - a) / (b - a)
}
