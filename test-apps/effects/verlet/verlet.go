package fx_verlet

import fx_common "github.com/jurgen-kluft/go-gx2/test-apps/effects/common"

type VerletEffect struct {
	PhysicsEngine *PhysicsEngine
	RestTime      float32
	Palette       [256]uint16
	Random        fx_common.RandU
}

func NewEffect(width, height, cellsize int32) *VerletEffect {
	ve := &VerletEffect{}
	ve.Random = fx_common.RandU(0xDEADBEEF)
	ve.PhysicsEngine = newPhysicsEngine(1024, width, height, cellsize)

	// Generate a dark-blue/black to light-blue/white gradient palette for the particles.
	for i := 0; i < 256; i++ {
		// Calculate the color components based on the index.
		r := uint8(i / 3)            // Red component (0-127)
		g := uint8(i / 3)            // Green component (0-127)
		b := uint8((255 - 32) + i/8) // Blue component (223-255)

		// Convert RGBA to RGB565 format.
		rgb565 := fx_common.ConvertToRGB565(r, g, b)
		ve.Palette[i] = rgb565
	}

	return ve
}

func (e *VerletEffect) ProcessFrame(deltaTime float32, frameBuffer *fx_common.FrameBuffer) {
	// Spawn a new particle at a random change at a random position at the top of the screen.
	// The particle will fall down due to gravity and bounce off the bottom of the screen.
	if (&e.Random).PseudoRand()&0xffff < 0xA000 {
		x := float32(4) + float32((&e.Random).PseudoRand()%uint32(e.PhysicsEngine.Width-8))
		y := float32(1) + float32((&e.Random).PseudoRand()%uint32(e.PhysicsEngine.Height/4))
		e.PhysicsEngine.SpawnParticle(x, y)
	}

	// The verlet physics engine needs to be ticked at 1000 fps, so we will have to tick
	// it multiple times per frame if the deltaTime is larger than 1/1000 seconds.
	const tickRate = 1.0 / 2000.0
	totalTicks := int((e.RestTime + deltaTime) / tickRate)
	for i := 0; i < totalTicks; i++ {
		e.PhysicsEngine.Tick(tickRate)
	}
	e.RestTime = (e.RestTime + deltaTime) - float32(totalTicks)*tickRate

	e.DrawParticles(frameBuffer)
}

func (e *VerletEffect) DrawParticles(fb *fx_common.FrameBuffer) {
	pe := e.PhysicsEngine

	i := int32(-1)
	for pe.ActiveParticles.Next(&i) {
		p := &pe.Particles[i]

		// Draw a point with 8 neighbors, making it look like a square.
		pX := int32(p.CurrPos.X)
		pY := int32(p.CurrPos.Y)

		// Clamp
		if pX < 2 {
			pX = 2
		} else if pX >= fb.Width-2 {
			pX = fb.Width - 3
		}
		if pY < 2 {
			pY = 2
		} else if pY >= fb.Height-2 {
			pY = fb.Height - 3
		}

		for dx := int32(-2); dx <= 2; dx++ {
			for dy := int32(-2); dy <= 2; dy++ {
				fb.Pixels[(pY+dy)*fb.Width+(pX+dx)] = e.Palette[p.ColorIdx]
			}
		}
	}
}
