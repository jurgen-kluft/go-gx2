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
	attractionForce int32 // Gentle pull to keep balls from drifting too far apart

	// Dynamic State Data
	balls      []Ball
	randState  uint32 // Serves as the mutable state variable for pseudoRand
	frameCount uint32
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
		attractionForce: 0x100,      // Gentle pull to keep balls from drifting too far apart

		randState:  seed,
		frameCount: 0,
	}
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

		// Initialize deformation variables
		m.balls[i].BaseRadius = 256 // Default standard size scaler
		for p := 0; p < 8; p++ {
			m.balls[i].Radii[p] = 0      // At rest perfectly on the circle radius
			m.balls[i].Velocities[p] = 0 // Static state
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

			// If balls smash into each other, deform their radial points on that side!
			if distSq < m.repulsionDistSq {
				m.balls[i].dx -= (dx * m.repulsionForce) / distSq
				m.balls[i].dy -= (dy * m.repulsionForce) / distSq

				// Deform calculation: inject energy into the spring points facing the collision
				for p := 0; p < 8; p++ {
					// Add a little chaotic squish when they collide
					if m.pseudoRand()%100 < 20 {
						m.balls[i].Velocities[p] -= m.customRandHelper(2000, 8000)
					}
				}
			} else if distSq < 25000 {
				m.balls[i].dx += (dx * m.attractionForce) / 100
				m.balls[i].dy += (dy * m.attractionForce) / 100
			}
		}
	}

	// 2. Update Spring-Mass Oscillations for the morphing borders
	for i := 0; i < int(m.numBalls); i++ {
		// Procedural idle jiggle: feed continuous micro-distortions over time
		if m.frameCount%4 == 0 {
			targetPoint := m.pseudoRand() % 8
			m.balls[i].Velocities[targetPoint] += m.customRandHelper(-4000, 4000)
		}

		for p := 0; p < 8; p++ {
			// Hooke's Law: Force = -k * displacement
			// Pulls the point back toward its resting circle radius (0)
			displacement := m.balls[i].Radii[p]
			springForce := (-displacement * 15) >> 8 // '15' determines stiffness

			// Apply force to velocity, add damping to slow it down over time
			m.balls[i].Velocities[p] += springForce
			m.balls[i].Velocities[p] = (m.balls[i].Velocities[p] * 248) >> 8 // Damping factor

			// Apply velocity to position
			m.balls[i].Radii[p] += m.balls[i].Velocities[p]
		}
	}
	m.frameCount++

	// 3. Apply velocities and handle fixed-point window boundary collisions
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
			gridY := y * m.width

			for x := int32(0); x < m.width; x++ {
				dx := x - bx
				if dx < -LutRadius || dx >= LutRadius {
					continue
				}
				// --- 90s Morphing Optimization Engine ---
				// Determine the angle index (0 to 7) of this pixel relative to ball center.
				// A fast octant selector avoiding expensive math.atan2:
				var angleIdx int
				absX := dx
				if absX < 0 {
					absX = -absX
				}
				absY := dy
				if absY < 0 {
					absY = -absY
				}

				if dx >= 0 && dy >= 0 { // Quadrant 1
					if absX > absY {
						angleIdx = 0
					} else {
						angleIdx = 1
					}
				} else if dx < 0 && dy >= 0 { // Quadrant 2
					if absX < absY {
						angleIdx = 2
					} else {
						angleIdx = 3
					}
				} else if dx < 0 && dy < 0 { // Quadrant 3
					if absX > absY {
						angleIdx = 4
					} else {
						angleIdx = 5
					}
				} else { // Quadrant 4
					if absX < absY {
						angleIdx = 6
					} else {
						angleIdx = 7
					}
				}

				// Fetch the spring deformation value for this angle segment
				deformation := m.balls[b].Radii[angleIdx] >> 12 // Scale down to manageable range

				// Apply the deformation to warp the spatial lookup coordinates.
				// If deformation is positive (extended), it pulls smaller table cells,
				// pushing the threshold perimeter further outward!
				distX := dx - (dx * deformation / 256)
				distY := dy - (dy * deformation / 256)

				lutX := distX + LutRadius
				lutY := distY + LutRadius

				// Bounds safety checking
				if lutX >= 0 && lutX < LutSize && lutY >= 0 && lutY < LutSize {
					energy := m.intensityTable[lutY*LutSize+lutX]
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
