package main

// Constants matching classic retro video specifications
const (
	LutRadius = 128
	LutSize   = LutRadius * 2

	// Fixed-point scaling shifts (16.16 format)
	FPShift     = 16
	FPOne       = 1 << FPShift
	maxVelocity = 4 * FPOne

	fieldStrength    = 180000
	surfaceThreshold = 140
)

// Ball structure tracking individual 2D velocity vectors
type Ball struct {
	x, y   int32
	dx, dy int32

	// Morphing Engine Data
	Radii      [8]int32 // Current offset from base radius (16.16 fixed-point)
	Velocities [8]int32 // Spring velocity of each point (16.16 fixed-point)
	BaseRadius int32    // Standard resting radius modifier (scaled around 256)
}

// MetaBallEffect encapsulates all buffers, configurations, and states for the effect.
type MetaBallEffect struct {
	// Static Lookup Tables and Render Buffers
	intensityTable  [LutSize * LutSize]uint8
	masterAccumGrid []uint8
	liquidPalette   [256]uint16

	numBalls int32
	gridSize int32
	width    int32
	height   int32

	widthFP  int32
	heightFP int32

	// Physics tuning constants
	repulsionDistSq int32 // Real pixel distance squared (30 pixels) where repulsion triggers
	repulsionForce  int32 // Aggressive speed thrust pushing balls apart

	// Dynamic State Data
	balls     []Ball
	randState uint32 // Serves as the mutable state variable for pseudoRand
}

// NewMetaBallEffect instantiates and safely initializes a new effect state.
func NewMetaBallEffect(seed uint32, numBalls int32, width, height int32) *MetaBallEffect {
	effect := &MetaBallEffect{
		numBalls:        numBalls,
		gridSize:        width * height,
		width:           width,
		height:          height,
		balls:           make([]Ball, numBalls),
		masterAccumGrid: make([]uint8, width*height),

		widthFP:  width << FPShift,
		heightFP: height << FPShift,

		// Physics tuning constants
		repulsionDistSq: 900,        // Real pixel distance squared (30 pixels) where repulsion triggers
		repulsionForce:  20 * FPOne, // Inverse-distance repulsion strength in fixed-point units

	}
	effect.randState = seed
	effect.init()
	return effect
}

// pseudoRand implements your exact PCG-XSH-RR hash mixing function
func (m *MetaBallEffect) pseudoRand() uint32 {
	state := m.randState*uint32(747796405) + uint32(2891336453)
	word := ((state >> ((state >> 28) + 4)) ^ state) * uint32(277803737)
	m.randState += 1
	return (word >> 22) ^ word
}

// customRandHelper scales the 32-bit pseudoRand output into a specific [min, max] range using raw integer math
func (m *MetaBallEffect) customRandHelper(min, max int32) int32 {
	val := m.pseudoRand()
	// Map the uint32 value cleanly inside our target boundaries
	return min + int32(val%uint32(max-min+1))
}

// init builds lookup tables and assigns initial coordinates upon creation
func (m *MetaBallEffect) init() {
	// 1. Generate Inverse-Distance Intensity LUT
	for y := 0; y < LutSize; y++ {
		for x := 0; x < LutSize; x++ {
			dx := int32(x - LutRadius)
			dy := int32(y - LutRadius)
			distSq := (dx * dx) + (dy * dy)

			if distSq == 0 {
				distSq = 1
			}

			// Classic 1/r field formula scaled to fit cleanly into a single byte
			intensity := fieldStrength / distSq
			if intensity > 255 {
				intensity = 255
			}
			m.intensityTable[y*LutSize+x] = uint8(intensity)
		}
	}

	// 2. Generate a metallic/electric liquid palette (Deep Red -> Bright Orange -> Yellow)
	for i := 0; i < 256; i++ {
		var r, g, b uint8
		if i < 128 { // Outer fringe: Crimson red
			r = uint8(i * 2)
		} else if i < 200 { // Mid-layer: Bright orange transition
			r = 255
			g = uint8((i - 128) * 3)
		} else { // Hot core: Vibrant yellow-white
			r = 255
			g = 255
			b = uint8((i - 200) * 4)
		}
		m.liquidPalette[i] = (uint16(r>>3) << 11) | (uint16(g>>2) << 5) | uint16(b>>3)
	}

	// 3. Populate physics fields with coordinates translated to Fixed-Point
	for i := 0; i < int(m.numBalls); i++ {
		// Store screenspace pixel position shifted left by 16 bits
		m.balls[i].x = m.customRandHelper(50, m.width-50) << FPShift
		m.balls[i].y = m.customRandHelper(50, m.height-50) << FPShift

		// Initial velocity fractional increments (e.g., 0.5 to 1.5 pixels per frame)
		m.balls[i].dx = m.customRandHelper(32768, 98304)
		m.balls[i].dy = m.customRandHelper(32768, 98304)

		if m.pseudoRand()%2 == 1 {
			m.balls[i].dx = -m.balls[i].dx
		}
		if m.pseudoRand()%2 == 1 {
			m.balls[i].dy = -m.balls[i].dy
		}
	}
}

// updatePhysics applies close-range separation, inertial movement, and wall bounces.
func (m *MetaBallEffect) updatePhysics() {
	// 1. Separate centers that get too close.
	for i := 0; i < int(m.numBalls); i++ {
		for j := 0; j < int(m.numBalls); j++ {
			if i == j {
				continue
			}

			// Calculate vector distance in raw integer screenspace pixels to prevent overlow
			// To convert 16.16 back to regular pixels, shift right by FPShift (16)
			ix, iy := m.balls[i].x>>FPShift, m.balls[i].y>>FPShift
			jx, jy := m.balls[j].x>>FPShift, m.balls[j].y>>FPShift

			dx := jx - ix
			dy := jy - iy
			distSq := (dx * dx) + (dy * dy)

			if distSq == 0 {
				distSq = 1
			}

			// Repel centers that are close enough to overlap.
			if distSq < m.repulsionDistSq {
				// Calculate push direction vector away from ball J
				// Multiply by fixed-point scalar force to smoothly change velocity curves
				m.balls[i].dx -= (dx * m.repulsionForce) / distSq
				m.balls[i].dy -= (dy * m.repulsionForce) / distSq
			}
		}
	}

	// 2. Apply velocities and handle fixed-point window boundary collisions
	for i := 0; i < int(m.numBalls); i++ {
		// Velocity drag clamping to prevent kinetic explosions from intense close-up repulsions
		if m.balls[i].dx > maxVelocity {
			m.balls[i].dx = maxVelocity
		}
		if m.balls[i].dx < -maxVelocity {
			m.balls[i].dx = -maxVelocity
		}
		if m.balls[i].dy > maxVelocity {
			m.balls[i].dy = maxVelocity
		}
		if m.balls[i].dy < -maxVelocity {
			m.balls[i].dy = -maxVelocity
		}

		// Update position values using raw 16.16 addition
		m.balls[i].x += m.balls[i].dx
		m.balls[i].y += m.balls[i].dy

		// Dynamic padding bounds checked against fixed-point dimensions
		padding := int32(15 << FPShift)

		if m.balls[i].x <= padding {
			m.balls[i].x = padding
			m.balls[i].dx = -m.balls[i].dx
		} else if m.balls[i].x >= m.widthFP-padding {
			m.balls[i].x = m.widthFP - padding
			m.balls[i].dx = -m.balls[i].dx
		}

		if m.balls[i].y <= padding {
			m.balls[i].y = padding
			m.balls[i].dy = -m.balls[i].dy
		} else if m.balls[i].y >= m.heightFP-padding {
			m.balls[i].y = m.heightFP - padding
			m.balls[i].dy = -m.balls[i].dy
		}
	}
}

// ProcessFrame runs one physics update and rendering step directly into the target slice
func (m *MetaBallEffect) update() {
	// Step 1: Execute fixed-point movement and collision physics
	m.updatePhysics()

	// Step 2: Clear screen grid
	for i := 0; i < int(m.gridSize); i++ {
		m.masterAccumGrid[i] = 0
	}

	// Step 3: Map field overlays onto screenspace coordinates
	for b := 0; b < int(m.numBalls); b++ {
		// Convert fixed-point coordinate maps down to normal integer pixel offsets for rendering lookups
		bx := m.balls[b].x >> FPShift
		by := m.balls[b].y >> FPShift

		for y := int32(0); y < m.height; y++ {
			dy := y - by
			if dy < -LutRadius || dy >= LutRadius {
				continue
			}
			lutY := (dy + LutRadius) * LutSize
			gridY := y * m.width

			for x := int32(0); x < m.width; x++ {
				dx := x - bx
				if dx < -LutRadius || dx >= LutRadius {
					continue
				}
				lutX := dx + LutRadius

				energy := m.intensityTable[lutY+lutX]
				gridIdx := gridY + x

				if uint32(m.masterAccumGrid[gridIdx])+uint32(energy) > 255 {
					m.masterAccumGrid[gridIdx] = 255
				} else {
					m.masterAccumGrid[gridIdx] += energy
				}
			}
		}
	}
}

func (m *MetaBallEffect) render(frameBuffer *FrameBuffer) {
	for y := int32(0); y < m.height; y++ {
		for x := int32(0); x < m.width; x++ {
			idx := y*m.width + x
			totalEnergy := m.masterAccumGrid[idx]
			if totalEnergy > surfaceThreshold {
				frameBuffer.Pixels[idx] = m.liquidPalette[totalEnergy]
			} else {
				frameBuffer.Pixels[idx] = 0
			}
		}
	}
}

func (m *MetaBallEffect) ProcessFrame(frameBuffer *FrameBuffer) {
	m.update()
	m.render(frameBuffer)
}
