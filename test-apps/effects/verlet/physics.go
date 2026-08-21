package fx_verlet

import "math"

const (
	freeFlightDamping = float32(0.993)
	contactDamping    = float32(0.95)
	restThresholdSq   = float32(0.02 * 0.02)
)

type PhysicsEngine struct {
	Width    float32
	Height   float32
	FPS      int32
	Gravity  float32
	Radius   float32
	SweepLTR bool

	FreeParticles   []uint16   // Indices of free particles in the Particles Array
	ActiveParticles BitArray   // BitArray to track active particles in the Particles Array
	Particles       []Particle // Array of particles
	Grid            *Grid      // Spatial grid for collision detection
}

func newPhysicsEngine(maxParticles int32, width, height, cellsize int32) *PhysicsEngine {
	pe := &PhysicsEngine{}

	pe.Width = float32(width)
	pe.Height = float32(height)
	pe.FPS = 60
	pe.Gravity = 3000.0
	pe.Radius = 4.0 // 6 pixels diameter, 3 pixels radius
	pe.SweepLTR = true

	pe.Grid = NewGrid(width, height, cellsize)

	pe.ActiveParticles = NewBitArray(maxParticles)
	pe.FreeParticles = make([]uint16, maxParticles)
	pe.Particles = make([]Particle, maxParticles)

	for i := int32(0); i < maxParticles; i++ {
		pe.FreeParticles[i] = uint16(i)
		pe.Particles[i].Index = uint16(i)
		pe.Particles[i].CurrPos = Vector2{0, 0}
		pe.Particles[i].PrevPos = Vector2{0, 0}
		pe.Particles[i].Radius = pe.Radius
		pe.Particles[i].ColorIdx = uint8(i % 256)
		pe.Particles[i].ListNext = 0xFFFF
		pe.Particles[i].ListPrev = 0xFFFF
		pe.Particles[i].GridCellIdx = 0xFFFF
	}

	return pe
}

type Vector2 struct {
	X float32
	Y float32
}

type Particle struct {
	Index       uint16 // Index of the particle in the Particles Array
	CurrPos     Vector2
	PrevPos     Vector2
	Radius      float32
	InContact   bool
	ColorIdx    uint8
	GridCellIdx uint16 // Index of the grid cell the particle is currently in
	ListNext    uint16 // Circular Linked-List, Next
	ListPrev    uint16 // Circular Linked-List, Prev
}

type Grid struct {
	Width       int32
	Height      int32
	CellSize    float32
	InvCellSize float32
	Cells       []uint16
	CellCount   int32
}

func NewGrid(width, height, cellsize int32) *Grid {
	g := &Grid{}
	g.Width = width / cellsize
	g.Height = height / cellsize
	g.CellSize = float32(cellsize)
	g.InvCellSize = 1.0 / float32(cellsize)
	g.Cells = make([]uint16, g.Width*g.Height)
	for i := range g.Cells {
		g.Cells[i] = 0xFFFF // Initialize all cells to 0xFFFF (no particles)
	}
	return g
}

func (pe *PhysicsEngine) SpawnParticle(x float32, y float32) {
	if len(pe.FreeParticles) == 0 {
		return // No free particles available
	}

	// Pop a free particle index from the FreeParticles slice
	particleIndex := pe.FreeParticles[len(pe.FreeParticles)-1]
	pe.FreeParticles = pe.FreeParticles[:len(pe.FreeParticles)-1]

	p := &pe.Particles[particleIndex]

	// Add the particle index to the ActiveParticles BitArray
	pe.ActiveParticles.SetBit(int32(p.Index))

	// Initialize the particle's properties
	p.CurrPos = Vector2{x, y}
	p.PrevPos = Vector2{x, y}
	p.Radius = pe.Radius
	p.InContact = false
	p.ColorIdx = uint8(y) & 0xFF // Assign a color index based on the x position
	p.ListNext = 0xFFFF
	p.ListPrev = 0xFFFF

	// Add it to the grid
	cellX := int32(p.CurrPos.X * pe.Grid.InvCellSize)
	cellY := int32(p.CurrPos.Y * pe.Grid.InvCellSize)
	cellIndex := cellY*pe.Grid.Width + cellX
	pe.addParticleToGridCell(p, cellIndex)
}

func (pe *PhysicsEngine) FreeParticle(particleIndex uint16) {
	p := &pe.Particles[particleIndex]

	// Remove the particle from the grid cell's linked list
	if p.GridCellIdx != 0xFFFF {
		pe.removeParticleFromGridCell(p)
	}

	// Reset the particle's properties
	p.CurrPos = Vector2{0, 0}
	p.PrevPos = Vector2{0, 0}
	p.Radius = pe.Radius
	p.InContact = false

	// Add the particle index back to the FreeParticles slice
	pe.FreeParticles = append(pe.FreeParticles, particleIndex)

	// Remove it from the ActiveParticles BitArray
	pe.ActiveParticles.ClearBit(int32(p.Index))
}

func (pe *PhysicsEngine) addParticleToGridCell(p *Particle, cellIndex int32) {
	// Insert the particle into the grid cell's linked list
	// Remember, we need to build a circular linked list!
	if pe.Grid.Cells[cellIndex] == 0xFFFF {
		// This is the first particle in the cell, so it points to itself
		p.ListNext = p.Index
		p.ListPrev = p.Index
	} else {
		// Insert the particle into the existing circular linked list
		headIdx := pe.Grid.Cells[cellIndex]
		head := &pe.Particles[headIdx]

		p.ListNext = head.Index
		p.ListPrev = head.ListPrev

		pe.Particles[head.ListPrev].ListNext = p.Index
		head.ListPrev = p.Index
	}

	// Update the grid cell's head to point to the new particle
	pe.Grid.Cells[cellIndex] = p.Index
	p.GridCellIdx = uint16(cellIndex)
}

func (pe *PhysicsEngine) removeParticleFromGridCell(p *Particle) {
	cellIndex := int32(p.GridCellIdx)

	if p.ListNext == p.ListPrev {
		// This is the only particle in the list, so we can just set the head to 0xFFFF
		pe.Grid.Cells[cellIndex] = 0xFFFF
	} else {
		// Remove the particle from the list
		pe.Particles[p.ListPrev].ListNext = p.ListNext
		pe.Particles[p.ListNext].ListPrev = p.ListPrev

		// If this particle is the head of the list, update the head to the next particle
		if pe.Grid.Cells[cellIndex] == p.Index {
			pe.Grid.Cells[cellIndex] = p.ListNext
		}
	}

	p.ListNext = 0xFFFF
	p.ListPrev = 0xFFFF
	p.GridCellIdx = 0xFFFF
}

func (pe *PhysicsEngine) Tick(dt float32) {
	i := int32(-1)
	for pe.ActiveParticles.Next(&i) {
		p := &pe.Particles[i]

		vX := p.CurrPos.X - p.PrevPos.X
		vY := p.CurrPos.Y - p.PrevPos.Y

		damping := freeFlightDamping
		if p.InContact {
			damping = contactDamping
			if vX*vX+vY*vY < restThresholdSq {
				vX = 0
				vY = 0
				p.PrevPos = p.CurrPos
			}
		}

		vX *= damping
		vY *= damping
		p.InContact = false

		p.PrevPos = p.CurrPos

		p.CurrPos.X += vX
		p.CurrPos.Y += vY + pe.Gravity*dt*dt

		// ========================= Grid Cell Assignment Begin =========================
		// Is particle moving out of its current grid cell? If so, remove it from
		// the grid cell and reinsert it into the new grid cell.
		cellX := int32(p.CurrPos.X * pe.Grid.InvCellSize)
		cellY := int32(p.CurrPos.Y * pe.Grid.InvCellSize)
		cellIndex := cellY*pe.Grid.Width + cellX
		if cellIndex != int32(p.GridCellIdx) {
			pe.removeParticleFromGridCell(p)
			pe.addParticleToGridCell(p, cellIndex)
		}
		// ========================= Grid Cell Assignment End =========================
	}

	pe.handleCollisionSweep()

	i = int32(-1)
	for pe.ActiveParticles.Next(&i) {
		p := &pe.Particles[i]
		// ========================= Wall Collision Begin =========================
		{
			var CR float32 = 0.9

			vX := p.CurrPos.X - p.PrevPos.X
			vY := p.CurrPos.Y - p.PrevPos.Y

			if p.CurrPos.X < p.Radius {
				p.CurrPos.X = p.Radius
				p.PrevPos.X = p.CurrPos.X - (vX * CR)
			} else if p.CurrPos.X > pe.Width-p.Radius {
				p.CurrPos.X = pe.Width - p.Radius
				p.PrevPos.X = p.CurrPos.X - (vX * CR)
			}

			if p.CurrPos.Y < p.Radius {
				p.CurrPos.Y = p.Radius
				p.PrevPos.Y = p.CurrPos.Y - (vY * CR)
			} else if p.CurrPos.Y > pe.Height-p.Radius {
				p.CurrPos.Y = pe.Height - p.Radius
				p.PrevPos.Y = p.CurrPos.Y - (vY * CR)
			}
		}
		// ========================= Wall Collision End =========================
	}
}

// Note: These directions must have a unit length of 1
var cRandomDirections = []Vector2{
	{1, 0},           // right
	{0, 1},           // down
	{-1, 0},          // left
	{0, -1},          // up
	{0.707, 0.707},   // down-right
	{-0.707, 0.707},  // down-left
	{-0.707, -0.707}, // up-left
	{0.707, -0.707},  // up-right
}

func (pe *PhysicsEngine) handleCollisionSweep() {
	stepX := int32(1)
	startX := int32(0)
	endX := pe.Grid.Width
	if !pe.SweepLTR {
		stepX = -1
		startX = pe.Grid.Width - 1
		endX = -1
	}

	for cellY := int32(0); cellY < pe.Grid.Height; cellY++ {
		for cellX := startX; cellX != endX; cellX += stepX {
			cellIndex := cellY*pe.Grid.Width + cellX
			if pe.Grid.Cells[cellIndex] == 0xFFFF {
				continue
			}

			handleCollisionSingleCell(cellIndex, pe)

			neighborX := cellX + stepX
			if neighborX >= 0 && neighborX < pe.Grid.Width {
				handleCollisions(cellIndex, cellIndex+stepX, pe)
			}

			if cellY+1 >= pe.Grid.Height {
				continue
			}

			southIndex := cellIndex + pe.Grid.Width
			handleCollisions(cellIndex, southIndex, pe)

			if neighborX >= 0 && neighborX < pe.Grid.Width {
				handleCollisions(cellIndex, southIndex+stepX, pe)
			}

			oppositeX := cellX - stepX
			if oppositeX >= 0 && oppositeX < pe.Grid.Width {
				handleCollisions(cellIndex, southIndex-stepX, pe)
			}
		}
	}

	pe.SweepLTR = !pe.SweepLTR
}

func resolveParticleCollision(pA *Particle, pB *Particle) {
	dx := pB.CurrPos.X - pA.CurrPos.X
	dy := pB.CurrPos.Y - pA.CurrPos.Y
	distSq := dx*dx + dy*dy
	radiusSum := pA.Radius + pB.Radius

	if distSq >= radiusSum*radiusSum {
		return
	}

	dist := float32(math.Sqrt(float64(distSq)))
	var separationVector Vector2
	overlap := radiusSum * 0.5
	if dist >= float32(0.0001) {
		separationVector = Vector2{dx / dist, dy / dist}
		overlap = (radiusSum - dist) * 0.8
	} else {
		directionIdx := (int(pA.Index) + int(pB.Index)) & 7
		separationVector = cRandomDirections[directionIdx]
		overlap *= 0.8
	}

	offsetX := overlap * separationVector.X
	offsetY := overlap * separationVector.Y

	pA.CurrPos.X -= offsetX
	pA.CurrPos.Y -= offsetY
	pB.CurrPos.X += offsetX
	pB.CurrPos.Y += offsetY
	pA.InContact = true
	pB.InContact = true

	// Apply the same positional correction to PrevPos so overlap resolution
	// does not get converted into extra velocity on the next Verlet step.
	pA.PrevPos.X -= offsetX
	pA.PrevPos.Y -= offsetY
	pB.PrevPos.X += offsetX
	pB.PrevPos.Y += offsetY
}

func handleCollisions(gridCellAIdx int32, gridCellBIdx int32, pe *PhysicsEngine) {
	// Get the head of the linked list for each grid cell
	headAIdx := pe.Grid.Cells[gridCellAIdx]
	headBIdx := pe.Grid.Cells[gridCellBIdx]

	if headAIdx == 0xFFFF || headBIdx == 0xFFFF {
		return // No particles in one of the cells, so no collisions to check
	}

	// Iterate through all particles in grid cell A
	pAIdx := headAIdx
	for true {
		pA := &pe.Particles[pAIdx]

		// Iterate through all particles in grid cell B
		pBIdx := headBIdx
		for true {
			pB := &pe.Particles[pBIdx]
			if pA.Index == pB.Index {
				pBIdx = pe.Particles[pBIdx].ListNext
				if pBIdx == headBIdx {
					break // We've looped back to the head of the list
				}
				continue // Skip self-collision
			}

			resolveParticleCollision(pA, pB)

			pBIdx = pe.Particles[pBIdx].ListNext
			if pBIdx == headBIdx {
				break // We've looped back to the head of the list
			}
		}

		pAIdx = pe.Particles[pAIdx].ListNext
		if pAIdx == headAIdx {
			break // We've looped back to the head of the list
		}
	}
}

func handleCollisionSingleCell(gridCellIdx int32, pe *PhysicsEngine) {
	// Get the head of the linked list for each grid cell
	headIdx := pe.Grid.Cells[gridCellIdx]

	if headIdx == 0xFFFF {
		return // No particles in one of the cells, so no collisions to check
	}

	// Iterate through each unordered pair in the circular list exactly once.
	pAIdx := headIdx
	for true {
		nextAIdx := pe.Particles[pAIdx].ListNext
		if nextAIdx == headIdx {
			break
		}

		pA := &pe.Particles[pAIdx]

		// Iterate through the remaining particles after pA.
		pBIdx := nextAIdx
		for true {
			pB := &pe.Particles[pBIdx]

			resolveParticleCollision(pA, pB)

			pBIdx = pe.Particles[pBIdx].ListNext
			if pBIdx == headIdx {
				break // We've looped back to the head of the list
			}
		}

		pAIdx = nextAIdx
	}
}
