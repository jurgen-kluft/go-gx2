package fx_fluid

func fabs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

func clampf(value, min, max float32) float32 {
	if value < min {
		return min
	} else if value > max {
		return max
	}
	return value
}
