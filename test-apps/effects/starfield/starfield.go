package main

import (
	fx_common "github.com/jurgen-kluft/go-gx2/test-apps/effects/common"
)

type StarFieldEffect struct {
	rand_state    fx_common.RandU // Random number generator state for deterministic randomness
	starX         []float32       // X positions of stars
	starY         []float32       // Y positions of stars
	starZ         []float32       // Z positions of stars (depth)
	starPreviousZ []float32       // Previous Z positions of stars (for motion blur)
	focus         float32         // Focal length for perspective projection
	numStars      int             // Number of stars in the field
	palette       []uint16        // color palette mapping distance to RGB565 colors
}

func NewStarFieldEffect(numStars int) *StarFieldEffect {
	sfe := &StarFieldEffect{
		rand_state:    fx_common.RandU(12345), // Initialize with a default seed
		starX:         make([]float32, numStars),
		starY:         make([]float32, numStars),
		starZ:         make([]float32, numStars),
		starPreviousZ: make([]float32, numStars),
		focus:         0.7, // Set a default focal length for perspective projection
		numStars:      numStars,
		palette:       make([]uint16, 256),
	}

	// Initialize the color palette (e.g., white to gray gradient)
	for i := range sfe.palette {
		r := uint8(255 - (i / 2)) // Decrease red component
		g := uint8(255 - (i / 2)) // Decrease green component
		b := uint8(255 - (i / 2)) // Decrease blue component
		sfe.palette[i] = fx_common.ConvertToRGB565(r, g, b)
	}

	// Initialize star positions and depths
	for i := 0; i < numStars; i++ {
		sfe.starX[i] = (&sfe.rand_state).CustomRandFloat(-sfe.focus, sfe.focus)
		sfe.starY[i] = (&sfe.rand_state).CustomRandFloat(-sfe.focus, sfe.focus)
		sfe.starZ[i] = (&sfe.rand_state).CustomRandFloat(0.2, 1.0) // Z position (depth)
		sfe.starPreviousZ[i] = sfe.starZ[i]                        // Initialize previous Z to current Z
	}

	return sfe
}

func (sfe *StarFieldEffect) update(speed float32) {
	for i := 0; i < sfe.numStars; i++ {
		sfe.starPreviousZ[i] = sfe.starZ[i]
		sfe.starZ[i] -= speed
		if sfe.starZ[i] <= float32(0.0001) {
			sfe.starX[i] = (&sfe.rand_state).CustomRandFloat(-sfe.focus, sfe.focus)
			sfe.starY[i] = (&sfe.rand_state).CustomRandFloat(-sfe.focus, sfe.focus)
			sfe.starZ[i] = float32(1.0)
			sfe.starPreviousZ[i] = sfe.starZ[i]
		}
	}
}

func (sfe *StarFieldEffect) render(frameBuffer *fx_common.FrameBuffer) {
	width := float32(frameBuffer.Width)
	height := float32(frameBuffer.Height)

	for i := 0; i < sfe.numStars; i++ {
		// Project 3D coordinates to 2D screen space
		x := (sfe.starX[i]/sfe.starZ[i])*(width/2) + (width / 2)
		y := (sfe.starY[i]/sfe.starZ[i])*(height/2) + (height / 2)

		// Calculate previous screen position for motion blur effect
		prevX := (sfe.starX[i]/sfe.starPreviousZ[i])*(width/2) + (width / 2)
		prevY := (sfe.starY[i]/sfe.starPreviousZ[i])*(height/2) + (height / 2)

		// Determine color based on depth
		// Palette is from white (close) to gray (far), so we can use the Z value to index into the palette
		colorIndex := int(sfe.starZ[i] * 255)
		if colorIndex < 0 {
			colorIndex = 0
		} else if colorIndex > 255 {
			colorIndex = 255
		}
		color := sfe.palette[colorIndex]

		// Draw the star as a line from previous position to current position for motion blur
		drawLine(frameBuffer, int32(prevX), int32(prevY), int32(x), int32(y), color)
	}
}

func drawLine(frameBuffer *fx_common.FrameBuffer, x0, y0, x1, y1 int32, color uint16) {
	dx := fx_common.Abs(x1 - x0)
	dy := fx_common.Abs(y1 - y0)
	sx := int32(1)
	if x0 > x1 {
		sx = -1
	}
	sy := int32(1)
	if y0 > y1 {
		sy = -1
	}
	err := dx - dy

	if dx == 0 && dy == 0 {
		// If the line is a single point, draw it directly
		if x0 >= 0 && x0 < frameBuffer.Width && y0 >= 0 && y0 < frameBuffer.Height {
			frameBuffer.Pixels[y0*frameBuffer.Width+x0] = color
		}
		return
	}

	// Draw the line using Bresenham's algorithm
	for {
		if x0 >= 0 && x0 < frameBuffer.Width && y0 >= 0 && y0 < frameBuffer.Height {
			frameBuffer.Pixels[y0*frameBuffer.Width+x0] = color
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		err2 := err * 2
		if err2 > -dy {
			err -= dy
			x0 += sx
		}
		if err2 < dx {
			err += dx
			y0 += sy
		}
	}
}

func (sfe *StarFieldEffect) ProcessFrame(deltaTime float32, frameBuffer *fx_common.FrameBuffer) {
	sfe.update(float32(0.01)) // Update star positions with a speed factor
	sfe.render(frameBuffer)   // Render the updated star field
}
