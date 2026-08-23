package fx_fastfluid

import (
	"math"
	"math/bits"

	fx_common "github.com/jurgen-kluft/go-gx2/test-apps/effects/common"
)

type FluidEffect struct {
	requestedWidth  int32
	requestedHeight int32
	size            int32
	mask            int32
	shift           uint
	iterations      int32
	diffusion       float32
	viscosity       float32
	fadeRate        float32
	time            float32
	showVelocity    bool
	velocityStride  int32
	palette         [256]uint16
	scratch         []float32
	density         []float32
	vx              []float32
	vy              []float32
	vx0             []float32
	vy0             []float32
}

func NewEffect(width, height int32) *FluidEffect {
	limit := minInt32(width, height)
	size := nearestPowerOfTwoAtMost(limit)
	shift := uint(bits.TrailingZeros32(uint32(size)))
	cellCount := size * size

	palette := fx_common.ComputePalette(fx_common.PaletteConfigurations[1])

	return &FluidEffect{
		requestedWidth:  width,
		requestedHeight: height,
		size:            size,
		mask:            size - 1,
		shift:           shift,
		iterations:      16,
		diffusion:       0.00001,
		viscosity:       0.00008,
		fadeRate:        18.0,
		showVelocity:    false,
		velocityStride:  maxInt32(2, size/16),
		palette:         palette,
		scratch:         make([]float32, cellCount),
		density:         make([]float32, cellCount),
		vx:              make([]float32, cellCount),
		vy:              make([]float32, cellCount),
		vx0:             make([]float32, cellCount),
		vy0:             make([]float32, cellCount),
	}
}

func (f *FluidEffect) ProcessFrame(dt float32, fb *fx_common.FrameBuffer) {
	if f == nil || fb == nil || f.size < 4 || fb.Width <= 0 || fb.Height <= 0 || len(fb.Pixels) == 0 {
		return
	}

	f.time += dt
	f.injectSources(dt)
	f.step(dt)
	f.fadeDensity(dt)
	f.renderDensity(fb)
	if f.showVelocity {
		f.renderVelocity(fb)
	}
}

func (f *FluidEffect) ix(x, y int32) int32 {
	if x < 0 {
		x = 0
	} else if x >= f.size {
		x = f.size - 1
	}
	if y < 0 {
		y = 0
	} else if y >= f.size {
		y = f.size - 1
	}
	return ((y & f.mask) << f.shift) | (x & f.mask)
}

func (f *FluidEffect) addDensity(x, y int32, amount float32) {
	idx := f.ix(x, y)
	f.density[idx] = clampf(f.density[idx]+amount, 0, 255)
}

func (f *FluidEffect) addVelocity(x, y int32, amountX, amountY float32) {
	idx := f.ix(x, y)
	f.vx[idx] += amountX
	f.vy[idx] += amountY
}

func (f *FluidEffect) injectSources(dt float32) {
	frameScale := clampf(dt*60.0, 0.25, 2.0)
	center := float32(f.size-1) * 0.5
	radius := float32(f.size) * 0.22
	angleA := float32(math.Sin(float64(f.time * 0.85)))
	angleB := float32(math.Cos(float64(f.time * 1.15)))

	x1 := center + float32(math.Cos(float64(f.time*0.9)))*radius
	y1 := center + float32(math.Sin(float64(f.time*1.2)))*radius
	x2 := center + float32(math.Cos(float64(f.time*1.25+math.Pi)))*radius
	y2 := center + float32(math.Sin(float64(f.time*0.8+math.Pi)))*radius

	f.emitSplat(x1, y1, 128.0*frameScale, angleB*0.9, -angleA*0.9)
	f.emitSplat(x2, y2, 60*frameScale, -angleA*0.8, angleB*0.8)
	f.emitSplat(center, center, 30.0*frameScale, -angleB*0.35, angleA*0.35)
}

func (f *FluidEffect) emitSplat(x, y, densityAmount, vxAmount, vyAmount float32) {
	baseX := int32(math.Round(float64(x))) & f.mask
	baseY := int32(math.Round(float64(y))) & f.mask
	if baseX <= 0 {
		baseX = 1
	} else if baseX >= f.size-1 {
		baseX = f.size - 2
	}
	if baseY <= 0 {
		baseY = 1
	} else if baseY >= f.size-1 {
		baseY = f.size - 2
	}

	for oy := int32(-1); oy <= 1; oy++ {
		for ox := int32(-1); ox <= 1; ox++ {
			falloff := float32(1.0)
			if ox != 0 || oy != 0 {
				falloff = 0.35
			}
			f.addDensity(baseX+ox, baseY+oy, densityAmount*falloff)
			f.addVelocity(baseX+ox, baseY+oy, vxAmount*falloff, vyAmount*falloff)
		}
	}
}

func (f *FluidEffect) step(dt float32) {
	f.diffuse(1, f.vx0, f.vx, f.viscosity, dt)
	f.diffuse(2, f.vy0, f.vy, f.viscosity, dt)
	f.project(f.vx0, f.vy0, f.vx, f.vy)

	f.advect(1, f.vx, f.vx0, f.vx0, f.vy0, dt)
	f.advect(2, f.vy, f.vy0, f.vx0, f.vy0, dt)
	f.project(f.vx, f.vy, f.vx0, f.vy0)

	f.diffuse(0, f.scratch, f.density, f.diffusion, dt)
	f.advect(0, f.density, f.scratch, f.vx, f.vy, dt)
}

func (f *FluidEffect) fadeDensity(dt float32) {
	fade := f.fadeRate * dt
	for i, value := range f.density {
		f.density[i] = clampf(value-fade, 0, 255)
	}
}

func (f *FluidEffect) diffuse(boundary int32, field, prev []float32, diff, dt float32) {
	a := dt * diff * float32((f.size-2)*(f.size-2))
	f.linSolve(boundary, field, prev, a, 1.0+4.0*a)
}

func (f *FluidEffect) linSolve(boundary int32, field, prev []float32, a, c float32) {
	invC := float32(1.0) / c
	for k := int32(0); k < f.iterations; k++ {
		for y := int32(1); y < f.size-1; y++ {
			for x := int32(1); x < f.size-1; x++ {
				field[f.ix(x, y)] = (prev[f.ix(x, y)] + a*(field[f.ix(x+1, y)]+field[f.ix(x-1, y)]+field[f.ix(x, y+1)]+field[f.ix(x, y-1)])) * invC
			}
		}
		f.setBoundaries(boundary, field)
	}
}

func (f *FluidEffect) project(velocX, velocY, pressure, divergence []float32) {
	sizeF := float32(f.size)
	for y := int32(1); y < f.size-1; y++ {
		for x := int32(1); x < f.size-1; x++ {
			divergence[f.ix(x, y)] = -0.5 * (velocX[f.ix(x+1, y)] - velocX[f.ix(x-1, y)] + velocY[f.ix(x, y+1)] - velocY[f.ix(x, y-1)]) / sizeF
			pressure[f.ix(x, y)] = 0
		}
	}

	f.setBoundaries(0, divergence)
	f.setBoundaries(0, pressure)
	f.linSolve(0, pressure, divergence, 1, 4)

	for y := int32(1); y < f.size-1; y++ {
		for x := int32(1); x < f.size-1; x++ {
			velocX[f.ix(x, y)] -= 0.5 * (pressure[f.ix(x+1, y)] - pressure[f.ix(x-1, y)]) * sizeF
			velocY[f.ix(x, y)] -= 0.5 * (pressure[f.ix(x, y+1)] - pressure[f.ix(x, y-1)]) * sizeF
		}
	}
	f.setBoundaries(1, velocX)
	f.setBoundaries(2, velocY)
}

func (f *FluidEffect) advect(boundary int32, field, prev, velocX, velocY []float32, dt float32) {
	dtx := dt * float32(f.size-2)
	dty := dt * float32(f.size-2)
	maxPos := float32(f.size) - 1.5

	for y := int32(1); y < f.size-1; y++ {
		yf := float32(y)
		for x := int32(1); x < f.size-1; x++ {
			xf := float32(x)
			backX := clampf(xf-dtx*velocX[f.ix(x, y)], 0.5, maxPos)
			backY := clampf(yf-dty*velocY[f.ix(x, y)], 0.5, maxPos)

			i0 := int32(math.Floor(float64(backX)))
			i1 := i0 + 1
			j0 := int32(math.Floor(float64(backY)))
			j1 := j0 + 1

			s1 := backX - float32(i0)
			s0 := 1.0 - s1
			t1 := backY - float32(j0)
			t0 := 1.0 - t1

			field[f.ix(x, y)] = s0*(t0*prev[f.ix(i0, j0)]+t1*prev[f.ix(i0, j1)]) +
				s1*(t0*prev[f.ix(i1, j0)]+t1*prev[f.ix(i1, j1)])
		}
	}

	f.setBoundaries(boundary, field)
}

func (f *FluidEffect) setBoundaries(boundary int32, field []float32) {
	last := f.size - 1
	for x := int32(1); x < last; x++ {
		if boundary == 2 {
			field[f.ix(x, 0)] = -field[f.ix(x, 1)]
			field[f.ix(x, last)] = -field[f.ix(x, last-1)]
		} else {
			field[f.ix(x, 0)] = field[f.ix(x, 1)]
			field[f.ix(x, last)] = field[f.ix(x, last-1)]
		}
	}
	for y := int32(1); y < last; y++ {
		if boundary == 1 {
			field[f.ix(0, y)] = -field[f.ix(1, y)]
			field[f.ix(last, y)] = -field[f.ix(last-1, y)]
		} else {
			field[f.ix(0, y)] = field[f.ix(1, y)]
			field[f.ix(last, y)] = field[f.ix(last-1, y)]
		}
	}

	field[f.ix(0, 0)] = 0.5 * (field[f.ix(1, 0)] + field[f.ix(0, 1)])
	field[f.ix(0, last)] = 0.5 * (field[f.ix(1, last)] + field[f.ix(0, last-1)])
	field[f.ix(last, 0)] = 0.5 * (field[f.ix(last-1, 0)] + field[f.ix(last, 1)])
	field[f.ix(last, last)] = 0.5 * (field[f.ix(last-1, last)] + field[f.ix(last, last-1)])
}

func (f *FluidEffect) renderDensity(fb *fx_common.FrameBuffer) {
	for py := int32(0); py < fb.Height; py++ {
		sy := (py * f.size) / fb.Height
		if sy >= f.size {
			sy = f.size - 1
		}
		row := py * fb.Width
		for px := int32(0); px < fb.Width; px++ {
			sx := (px * f.size) / fb.Width
			if sx >= f.size {
				sx = f.size - 1
			}

			// density := clampf(f.density[f.ix(sx, sy)], 0, 255)

			// sample neighboring cells for smoother rendering
			density := (f.density[f.ix(sx, sy)] +
				f.density[f.ix(sx+1, sy)] +
				f.density[f.ix(sx-1, sy)] +
				f.density[f.ix(sx, sy+1)] +
				f.density[f.ix(sx, sy-1)]) * 0.2

			r := uint8(density)
			fb.Pixels[row+px] = f.palette[r]
		}
	}
}

func (f *FluidEffect) renderVelocity(fb *fx_common.FrameBuffer) {
	if f.velocityStride <= 0 {
		return
	}

	scaleX := float32(fb.Width) / float32(f.size)
	scaleY := float32(fb.Height) / float32(f.size)
	velocityColor := fx_common.ConvertToRGB565(255, 255, 255)

	for y := int32(1); y < f.size-1; y += f.velocityStride {
		for x := int32(1); x < f.size-1; x += f.velocityStride {
			vx := f.vx[f.ix(x, y)]
			vy := f.vy[f.ix(x, y)]
			if absf(vx) < 0.025 && absf(vy) < 0.025 {
				continue
			}

			startX := (float32(x) + 0.5) * scaleX
			startY := (float32(y) + 0.5) * scaleY
			endX := startX + vx*scaleX*4.0
			endY := startY + vy*scaleY*4.0
			drawLine(fb, startX, startY, endX, endY, velocityColor)
		}
	}
}

func drawLine(fb *fx_common.FrameBuffer, x0, y0, x1, y1 float32, color uint16) {
	ix0 := int32(math.Round(float64(x0)))
	iy0 := int32(math.Round(float64(y0)))
	ix1 := int32(math.Round(float64(x1)))
	iy1 := int32(math.Round(float64(y1)))

	dx := absInt32(ix1 - ix0)
	dy := absInt32(iy1 - iy0)
	sx := int32(-1)
	if ix0 < ix1 {
		sx = 1
	}
	sy := int32(-1)
	if iy0 < iy1 {
		sy = 1
	}
	err := dx - dy

	for {
		if ix0 >= 0 && ix0 < fb.Width && iy0 >= 0 && iy0 < fb.Height {
			fb.Pixels[iy0*fb.Width+ix0] = color
		}
		if ix0 == ix1 && iy0 == iy1 {
			break
		}
		err2 := err * 2
		if err2 > -dy {
			err -= dy
			ix0 += sx
		}
		if err2 < dx {
			err += dx
			iy0 += sy
		}
	}
}

func hsvToRGB(h, s, v float32) (uint8, uint8, uint8) {
	if s <= 0 {
		value := uint8(clampf(v*255.0, 0, 255))
		return value, value, value
	}

	h = float32(math.Mod(float64(h), 360))
	if h < 0 {
		h += 360
	}

	sector := h / 60.0
	i := int32(math.Floor(float64(sector)))
	f := sector - float32(i)
	p := v * (1.0 - s)
	q := v * (1.0 - s*f)
	t := v * (1.0 - s*(1.0-f))

	var r, g, b float32
	switch i % 6 {
	case 0:
		r, g, b = v, t, p
	case 1:
		r, g, b = q, v, p
	case 2:
		r, g, b = p, v, t
	case 3:
		r, g, b = p, q, v
	case 4:
		r, g, b = t, p, v
	default:
		r, g, b = v, p, q
	}

	return uint8(clampf(r*255.0, 0, 255)), uint8(clampf(g*255.0, 0, 255)), uint8(clampf(b*255.0, 0, 255))
}

func nearestPowerOfTwoAtMost(value int32) int32 {
	if value <= 1 {
		return 2
	}
	return 1 << (bits.Len32(uint32(value)) - 1)
}

func clampf(value, low, high float32) float32 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func minInt32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func maxInt32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func absInt32(value int32) int32 {
	if value < 0 {
		return -value
	}
	return value
}

func absf(value float32) float32 {
	if value < 0 {
		return -value
	}
	return value
}
