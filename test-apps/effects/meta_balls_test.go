package main

import "testing"

func TestMetaBallPaletteUsesRGB565(t *testing.T) {
	effect := NewMetaBallEffect(42, 1, 480, 480)

	if got, want := effect.liquidPalette[127], uint16(0xF800); got != want {
		t.Fatalf("red palette entry = %#04x, want %#04x", got, want)
	}
	if got, want := effect.liquidPalette[200], uint16(0xFFE0); got != want {
		t.Fatalf("yellow palette entry = %#04x, want %#04x", got, want)
	}
}

func TestMetaBallIntensityHasThirtyFivePixelRadius(t *testing.T) {
	effect := NewMetaBallEffect(42, 1, 480, 480)
	center := LutRadius*LutSize + LutRadius

	if got := effect.intensityTable[center]; got != 255 {
		t.Fatalf("center intensity = %d, want 255", got)
	}
	if got := effect.intensityTable[center+35]; got <= surfaceThreshold {
		t.Fatalf("intensity at 35 pixels = %d, want above %d", got, surfaceThreshold)
	}
	if got := effect.intensityTable[center+36]; got > surfaceThreshold {
		t.Fatalf("intensity at 36 pixels = %d, want at most %d", got, surfaceThreshold)
	}
}

func TestMetaBallsOutsideRepulsionRangeApplyNoForce(t *testing.T) {
	effect := NewMetaBallEffect(42, 2, 480, 480)
	setStationaryBall(&effect.balls[0], 100, 100)
	setStationaryBall(&effect.balls[1], 200, 100)

	effect.updatePhysics()

	for i, ball := range effect.balls {
		if ball.dx != 0 || ball.dy != 0 {
			t.Fatalf("ball %d velocity = (%d, %d), want no force", i, ball.dx, ball.dy)
		}
	}
}

func TestMetaBallPreservesInertialMotion(t *testing.T) {
	effect := NewMetaBallEffect(42, 1, 480, 480)
	setStationaryBall(&effect.balls[0], 100, 100)
	effect.balls[0].dx = FPOne / 2
	effect.balls[0].dy = -FPOne / 4

	effect.updatePhysics()

	if got, want := effect.balls[0].dx, int32(FPOne/2); got != want {
		t.Fatalf("horizontal velocity = %d, want %d", got, want)
	}
	if got, want := effect.balls[0].dy, int32(-FPOne/4); got != want {
		t.Fatalf("vertical velocity = %d, want %d", got, want)
	}
	if got, want := effect.balls[0].x, int32(100*FPOne+FPOne/2); got != want {
		t.Fatalf("x position = %d, want %d", got, want)
	}
	if got, want := effect.balls[0].y, int32(100*FPOne-FPOne/4); got != want {
		t.Fatalf("y position = %d, want %d", got, want)
	}
}

func TestMetaBallRepulsionUsesFixedPointVelocityUnits(t *testing.T) {
	effect := NewMetaBallEffect(42, 2, 480, 480)
	setStationaryBall(&effect.balls[0], 100, 100)
	setStationaryBall(&effect.balls[1], 120, 100)

	effect.updatePhysics()

	if got, want := effect.balls[0].dx, int32(-FPOne); got != want {
		t.Fatalf("left ball repulsion = %d, want %d", got, want)
	}
	if got, want := effect.balls[1].dx, int32(FPOne); got != want {
		t.Fatalf("right ball repulsion = %d, want %d", got, want)
	}
}

func TestMetaBallVelocityIsClampedBeforeIntegration(t *testing.T) {
	effect := NewMetaBallEffect(42, 1, 480, 480)
	setStationaryBall(&effect.balls[0], 100, 100)
	effect.balls[0].dx = 10 * FPOne

	effect.updatePhysics()

	if got, want := effect.balls[0].dx, int32(maxVelocity); got != want {
		t.Fatalf("clamped velocity = %d, want %d", got, want)
	}
	if got, want := effect.balls[0].x, int32(104*FPOne); got != want {
		t.Fatalf("integrated position = %d, want %d", got, want)
	}
}

func TestMetaBallFieldsMergeAtOneHundredPixels(t *testing.T) {
	effect := NewMetaBallEffect(42, 2, 480, 480)
	setStationaryBall(&effect.balls[0], 100, 100)
	setStationaryBall(&effect.balls[1], 200, 100)

	effect.update()

	midpoint := 100*effect.width + 150
	if got := effect.masterAccumGrid[midpoint]; got <= surfaceThreshold {
		t.Fatalf("midpoint intensity = %d, want above %d", got, surfaceThreshold)
	}
}

func setStationaryBall(ball *Ball, x, y int32) {
	ball.x = x * FPOne
	ball.y = y * FPOne
	ball.dx = 0
	ball.dy = 0
}
