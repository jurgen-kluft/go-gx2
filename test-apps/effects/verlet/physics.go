package fx_verlet

import "math"

type PhysicsEngine struct {
	Width   float32
	Height  float32
	FPS     int32
	Gravity float32
	Radius  float32

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
	pe.Radius = 2.0

	pe.Grid = NewGrid(width, height, cellsize)

	pe.ActiveParticles = NewBitArray(maxParticles)
	pe.FreeParticles = make([]uint16, maxParticles)
	pe.Particles = make([]Particle, maxParticles)

	for i := int32(0); i < maxParticles; i++ {
		pe.FreeParticles[i] = uint16(i)
		pe.Particles[i].CurrPos = Vector2{0, 0}
		pe.Particles[i].PrevPos = Vector2{0, 0}
		pe.Particles[i].Accel = Vector2{0, pe.Gravity}
		pe.Particles[i].Radius = pe.Radius
		pe.Particles[i].ColorIdx = uint8(i % 256)
		pe.Particles[i].ListNext = 0xFFFF
		pe.Particles[i].ListPrev = 0xFFFF
		pe.Particles[i].Index = uint16(i)
		pe.Particles[i].GridCellIdx = 0xFFFF
	}

	return pe
}

type Vector2 struct {
	X float32
	Y float32
}

type Particle struct {
	CurrPos     Vector2
	PrevPos     Vector2
	Accel       Vector2
	Radius      float32
	ColorIdx    uint8
	Index       uint16 // Index of the particle in the Particles Array
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

	// Initialize the particle's properties
	p := &pe.Particles[particleIndex]
	p.CurrPos = Vector2{x, y}
	p.PrevPos = Vector2{x, y}
	p.Accel = Vector2{0, pe.Gravity}
	p.Radius = pe.Radius
	p.ColorIdx = uint8(x) & 0xFF // Assign a color index based on the x position
	p.ListNext = 0xFFFF
	p.ListPrev = 0xFFFF

	// Add the particle index to the ActiveParticles BitArray
	pe.ActiveParticles.SetBit(int32(p.Index))

	// Add it to the grid
	cellX := int32(p.CurrPos.X * pe.Grid.InvCellSize)
	cellY := int32(p.CurrPos.Y * pe.Grid.InvCellSize)
	cellIndex := cellY*pe.Grid.Width + cellX
	pe.addParticleToGridCell(p, cellIndex)
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

func (pe *PhysicsEngine) FreeParticle(particleIndex uint16) {
	p := &pe.Particles[particleIndex]

	// Remove the particle from the grid cell's linked list
	if p.GridCellIdx != 0xFFFF {
		pe.removeParticleFromGridCell(p)
	}

	// Reset the particle's properties
	p.CurrPos = Vector2{0, 0}
	p.PrevPos = Vector2{0, 0}
	p.Accel = Vector2{0, 0}
	p.Radius = pe.Radius

	// Add the particle index back to the FreeParticles slice
	pe.FreeParticles = append(pe.FreeParticles, particleIndex)

	// Remove it from the ActiveParticles BitArray
	pe.ActiveParticles.ClearBit(int32(p.Index))
}

func (pe *PhysicsEngine) Tick(dt float32) {
	i := int32(-1)
	for pe.ActiveParticles.Next(&i) {
		p := &pe.Particles[i]

		vX := (p.CurrPos.X - p.PrevPos.X) * 0.995
		vY := (p.CurrPos.Y - p.PrevPos.Y) * 0.995

		p.PrevPos = p.CurrPos

		p.CurrPos.X += vX + p.Accel.X*dt*dt
		p.CurrPos.Y += vY + p.Accel.Y*dt*dt
		p.Accel.Y = pe.Gravity

		// Wall Colliision Begin
		{
			var CR float32 = 0.5

			vX := p.CurrPos.X - p.PrevPos.X
			vY := p.CurrPos.Y - p.PrevPos.Y

			if p.CurrPos.X < p.Radius {
				p.CurrPos.X = p.Radius
				p.PrevPos.X = p.CurrPos.X + (vX * CR)
			} else if p.CurrPos.X > pe.Width-p.Radius {
				p.CurrPos.X = pe.Width - p.Radius
				p.PrevPos.X = p.CurrPos.X + (vX * CR)
			}

			if p.CurrPos.Y < p.Radius {
				p.CurrPos.Y = p.Radius
				p.PrevPos.Y = p.CurrPos.Y + (vY * CR)
			} else if p.CurrPos.Y > pe.Height-p.Radius {
				p.CurrPos.Y = pe.Height - p.Radius
				p.PrevPos.Y = p.CurrPos.Y + (vY * CR)
			}
		}
		// Wall Colliision End

		// Grid Cell Assignment Begin

		// Is particle moving out of its current grid cell? If so, remove it from
		// the grid cell and reinsert it into the new grid cell.

		cellX := int32(p.CurrPos.X * pe.Grid.InvCellSize)
		cellY := int32(p.CurrPos.Y * pe.Grid.InvCellSize)

		cellIndex := cellY*pe.Grid.Width + cellX
		if cellIndex != int32(p.GridCellIdx) {
			if p.ListPrev == p.ListNext {
				// This is the only particle in the list, so we can just set the head to 0xFFFF
				pe.Grid.Cells[p.GridCellIdx] = 0xFFFF
			} else {
				// Remove the particle from the list
				pe.Particles[p.ListPrev].ListNext = p.ListNext
				pe.Particles[p.ListNext].ListPrev = p.ListPrev

				// If this particle is the head of the list, update the head to the next particle
				if pe.Grid.Cells[p.GridCellIdx] == p.Index {
					pe.Grid.Cells[p.GridCellIdx] = p.ListNext
				}
			}

			// Insert the particle into the new cell's list
			p.ListPrev = 0xFFFF
			p.ListNext = pe.Grid.Cells[cellIndex]
			if p.ListNext != 0xFFFF {
				pe.Particles[p.ListNext].ListPrev = p.Index
			}
			pe.Grid.Cells[cellIndex] = p.Index

			p.GridCellIdx = uint16(cellIndex)
		}

		// Grid Cell Assignment End

	}

	// Check for collisions between particles

	// A particle knows the grid cell it is in, we need to check for collisions with
	// particles in the same cell and neighboring cells.
	// Note: When a collision (overlap) is detected, we change the position of the particle
	// so as to move them apart, then we also reduce the acceleration of the particles by
	// a factor of 0.5, so that they don't keep colliding with each other in the next frame.

	// So our iteration will take a grid cell and 4 forward neighboring cells:
	// - east
	// - southeast
	// - south
	// - southwest

	// This way, we will check each pair of particles only once, and we will not miss any collisions.
	for cellY := int32(0); cellY < pe.Grid.Height-1; cellY++ {

		// Left border, we know that South-West doesn't exist
		cellIndex := cellY * pe.Grid.Width
		if pe.Grid.Cells[cellIndex] == 0xFFFF {
			continue
		}
		handleCollisions(cellIndex, cellIndex, pe)
		eastIndex := cellIndex + 1
		handleCollisions(cellIndex, eastIndex, pe)
		southeastIndex := cellIndex + pe.Grid.Width + 1
		handleCollisions(cellIndex, southeastIndex, pe)
		southIndex := cellIndex + pe.Grid.Width
		handleCollisions(cellIndex, southIndex, pe)

		southWestIndex := southIndex - 1
		for cellX := int32(1); cellX < pe.Grid.Width-1; cellX++ {
			cellIndex += 1
			if pe.Grid.Cells[cellIndex] == 0xFFFF {
				continue
			}
			handleCollisions(cellIndex, cellIndex, pe)

			eastIndex += 1
			handleCollisions(cellIndex, eastIndex, pe)
			southeastIndex += 1
			handleCollisions(cellIndex, southeastIndex, pe)
			southIndex += 1
			handleCollisions(cellIndex, southIndex, pe)
			southWestIndex += 1
			handleCollisions(cellIndex, southWestIndex, pe)
		}

		// Right border, we can only check with South and South-West,
		// but check only when the cell actually has particles.
		cellIndex += 1
		if pe.Grid.Cells[cellIndex] != 0xFFFF {
			handleCollisions(cellIndex, cellIndex, pe)

			southIndex += 1
			southWestIndex += 1
			handleCollisions(cellIndex, southIndex, pe)
			handleCollisions(cellIndex, southWestIndex, pe)
		}
	}

	// The bottom row, we know that only East exists.
	bottomRowIndex := (pe.Grid.Height - 1) * pe.Grid.Width
	for cellX := int32(0); cellX < pe.Grid.Width-1; cellX++ {
		cellIndex := bottomRowIndex + cellX
		if pe.Grid.Cells[cellIndex] == 0xFFFF {
			continue
		}
		handleCollisions(cellIndex, cellIndex, pe)

		eastIndex := cellIndex + 1
		handleCollisions(cellIndex, eastIndex, pe)
	}

	// The bottom-right corner, here we know that we don't have any
	// of the neighbors, just check for collisions within the cell itself.
	bottomRightIndex := (pe.Grid.Height-1)*pe.Grid.Width + (pe.Grid.Width - 1)
	if pe.Grid.Cells[bottomRightIndex] != 0xFFFF {
		handleCollisions(bottomRightIndex, bottomRightIndex, pe)
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

func handleCollisions(gridCellAIdx int32, gridCellBIdx int32, pe *PhysicsEngine) {
	// Get the head of the linked list for each grid cell
	headAIdx := pe.Grid.Cells[gridCellAIdx]
	headBIdx := pe.Grid.Cells[gridCellBIdx]

	// Iterate through all particles in grid cell A
	for pAIdx := headAIdx; pAIdx != 0xFFFF; pAIdx = pe.Particles[pAIdx].ListNext {
		pA := &pe.Particles[pAIdx]

		// Iterate through all particles in grid cell B
		for pBIdx := headBIdx; pBIdx != 0xFFFF; pBIdx = pe.Particles[pBIdx].ListNext {
			pB := &pe.Particles[pBIdx]
			if pA.Index == pB.Index {
				continue // Skip self-collision
			}

			// Check for collision between pA and pB
			dx := pB.CurrPos.X - pA.CurrPos.X
			dy := pB.CurrPos.Y - pA.CurrPos.Y
			distSq := dx*dx + dy*dy
			radiusSum := pA.Radius + pB.Radius

			if distSq < radiusSum*radiusSum {
				// Collision detected, resolve it
				dist := float32(math.Sqrt(float64(distSq)))
				var seperationVector Vector2
				if dist == 0 {
					// If the distance is zero, roll a dice and move the particles in a
					// random direction, choosing from 8 different directions.
					// This is to avoid any heavy sin/cos etc...
					seperationVector = cRandomDirections[(pA.Index+pB.Index)&7]
				} else {
					seperationVector = Vector2{dx / dist, dy / dist}
				}

				overlap := (radiusSum - dist) * 0.5
				offsetX := overlap * seperationVector.X
				offsetY := overlap * seperationVector.Y

				pA.Accel.X *= 0.5
				pA.Accel.Y *= 0.5
				pB.Accel.X *= 0.5
				pB.Accel.Y *= 0.5

				pA.CurrPos.X -= offsetX
				pA.CurrPos.Y -= offsetY
				pB.CurrPos.X += offsetX
				pB.CurrPos.Y += offsetY
			}
		}
	}
}
