package main

import (
	"math"
	"testing"
)

func TestMetaBallInitializesVerletRing(t *testing.T) {
	effect := NewMetaBallEffect(42, 1, 16, 480, 480)
	ball := &effect.balls[0]

	if got := len(ball.points); got != 16 {
		t.Fatalf("point count = %d, want 16", got)
	}
	wantChord := float32(2 * restRadius * math.Sin(math.Pi/16))
	for pointIdx, point := range ball.points {
		if got := point.position.length(); absFloat32(got-restRadius) > 0.001 {
			t.Fatalf("point %d radius = %v, want %v", pointIdx, got, restRadius)
		}
		if point.position != point.previousPosition {
			t.Fatalf("point %d has initial velocity: current=%v previous=%v", pointIdx, point.position, point.previousPosition)
		}
		next := ball.points[(pointIdx+1)%len(ball.points)]
		if got := next.position.sub(point.position).length(); absFloat32(got-wantChord) > 0.001 {
			t.Fatalf("edge %d length = %v, want %v", pointIdx, got, wantChord)
		}
	}
}

func TestMetaBallRejectsInvalidPointCount(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("NewMetaBallEffect did not reject fewer than three points")
		}
	}()
	NewMetaBallEffect(42, 1, 2, 480, 480)
}

func TestMetaBallVerletConstraintsRestoreDisplacedPoint(t *testing.T) {
	effect := NewMetaBallEffect(42, 1, 16, 480, 480)
	ball := &effect.balls[0]
	ball.points[0].position = Vec2{x: restRadius + 8, y: -8}
	ball.points[0].previousPosition = ball.points[0].position
	initialRadiusError := absFloat32(ball.points[0].position.length() - restRadius)
	initialNeighborError := absFloat32(ball.points[1].position.sub(ball.points[0].position).length() - effect.neighborRestLength)

	ball.updateVerlet()

	radiusError := absFloat32(ball.points[0].position.length() - restRadius)
	neighborError := absFloat32(ball.points[1].position.sub(ball.points[0].position).length() - effect.neighborRestLength)
	if radiusError >= initialRadiusError {
		t.Fatalf("spoke error = %v, want below initial %v", radiusError, initialRadiusError)
	}
	if neighborError >= initialNeighborError {
		t.Fatalf("neighbor error = %v, want below initial %v", neighborError, initialNeighborError)
	}
	if ball.points[0].position.y == 0 {
		t.Fatal("tangential displacement was erased; want free 2D motion")
	}
}

func TestMetaBallPaletteUsesRGB565(t *testing.T) {
	effect := NewMetaBallEffect(42, 1, 16, 480, 480)

	if got, want := effect.liquidPalette[127], uint16(0xF800); got != want {
		t.Fatalf("red palette entry = %#04x, want %#04x", got, want)
	}
	if got, want := effect.liquidPalette[200], uint16(0xFFE0); got != want {
		t.Fatalf("yellow palette entry = %#04x, want %#04x", got, want)
	}
}

func TestMetaBallIntensityHasThirtyFivePixelRadius(t *testing.T) {
	effect := NewMetaBallEffect(42, 1, 16, 480, 480)
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

func TestMetaBallsOutsideRepulsionRangeAttractGently(t *testing.T) {
	effect := NewMetaBallEffect(42, 2, 16, 480, 480)
	setStationaryBall(&effect.balls[0], 100, 100)
	setStationaryBall(&effect.balls[1], 200, 100)

	effect.updatePhysics()

	if got, want := effect.balls[0].dx, float32(0.00390625); got != want {
		t.Fatalf("left ball attraction = %v, want %v", got, want)
	}
	if got, want := effect.balls[1].dx, float32(-0.00390625); got != want {
		t.Fatalf("right ball attraction = %v, want %v", got, want)
	}
	if effect.balls[0].dy != 0 || effect.balls[1].dy != 0 {
		t.Fatalf("vertical velocities = (%v, %v), want no vertical force", effect.balls[0].dy, effect.balls[1].dy)
	}
}

func TestMetaBallPreservesInertialMotion(t *testing.T) {
	effect := NewMetaBallEffect(42, 1, 16, 480, 480)
	setStationaryBall(&effect.balls[0], 100, 100)
	effect.balls[0].dx = 0.5
	effect.balls[0].dy = -0.25

	effect.updatePhysics()

	if got, want := effect.balls[0].dx, float32(0.5); got != want {
		t.Fatalf("horizontal velocity = %v, want %v", got, want)
	}
	if got, want := effect.balls[0].dy, float32(-0.25); got != want {
		t.Fatalf("vertical velocity = %v, want %v", got, want)
	}
	if got, want := effect.balls[0].x, float32(100.5); got != want {
		t.Fatalf("x position = %v, want %v", got, want)
	}
	if got, want := effect.balls[0].y, float32(99.75); got != want {
		t.Fatalf("y position = %v, want %v", got, want)
	}
}

func TestMetaBallContactReducesStaticPenetration(t *testing.T) {
	effect := NewMetaBallEffect(42, 2, 16, 480, 480)
	setStationaryBall(&effect.balls[0], 100, 100)
	setStationaryBall(&effect.balls[1], 120, 100)
	initialDistance := effect.balls[1].x - effect.balls[0].x

	effect.updatePhysics()

	if got := effect.balls[1].x - effect.balls[0].x; got <= initialDistance {
		t.Fatalf("center distance = %v, want above initial %v", got, initialDistance)
	}
	if effect.balls[0].dx != 0 || effect.balls[1].dx != 0 {
		t.Fatalf("static overlap created velocities (%v, %v)", effect.balls[0].dx, effect.balls[1].dx)
	}
}

func TestMetaBallAttractionWaitsBeyondContactReleaseBand(t *testing.T) {
	effect := NewMetaBallEffect(42, 2, 16, 480, 480)
	setStationaryBall(&effect.balls[0], 100, 100)
	setStationaryBall(&effect.balls[1], 175, 100)

	effect.updatePhysics()

	if effect.balls[0].dx != 0 || effect.balls[1].dx != 0 {
		t.Fatalf("release-band velocities = (%v, %v), want no immediate attraction", effect.balls[0].dx, effect.balls[1].dx)
	}
}

func TestMetaBallCollisionDentsFacingVerletPoints(t *testing.T) {
	effect := NewMetaBallEffect(42, 2, 16, 480, 480)
	setStationaryBall(&effect.balls[0], 100, 100)
	setStationaryBall(&effect.balls[1], 165, 100)
	effect.balls[0].dx = 1
	effect.balls[1].dx = -1
	effect.frameCount = 1

	effect.updatePhysics()

	leftFront := effect.balls[0].points[0].position.length()
	leftRear := effect.balls[0].points[8].position.length()
	leftSide := effect.balls[0].points[4].position.length()
	rightFront := effect.balls[1].points[8].position.length()
	if leftFront > restRadius-1.5 || rightFront > restRadius-1.5 {
		t.Fatalf("facing radii = (%v, %v), want a visible dent below %v", leftFront, rightFront, restRadius-1.5)
	}
	if leftFront >= leftRear {
		t.Fatalf("left front radius = %v, want below rear radius %v", leftFront, leftRear)
	}
	if leftSide <= restRadius+0.05 {
		t.Fatalf("left side radius = %v, want area-preserving bulge above %v", leftSide, restRadius+0.05)
	}
	if effect.balls[0].dx >= 0 || effect.balls[1].dx <= 0 {
		t.Fatalf("center velocities = (%v, %v), want restitution to reverse closing motion", effect.balls[0].dx, effect.balls[1].dx)
	}
}

func TestMetaBallCoincidentCentersRemainFinite(t *testing.T) {
	effect := NewMetaBallEffect(42, 2, 16, 480, 480)
	setStationaryBall(&effect.balls[0], 100, 100)
	setStationaryBall(&effect.balls[1], 100, 100)
	effect.frameCount = 1

	effect.updatePhysics()

	for ballIdx, ball := range effect.balls {
		values := []float32{ball.x, ball.y, ball.dx, ball.dy}
		for _, point := range ball.points {
			values = append(values, point.position.x, point.position.y)
		}
		for _, value := range values {
			if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
				t.Fatalf("ball %d contains non-finite value %v", ballIdx, value)
			}
		}
	}
}

func TestMetaBallRebuildsContourFromVerletRing(t *testing.T) {
	effect := NewMetaBallEffect(42, 1, 16, 480, 480)
	ball := &effect.balls[0]

	ball.rebuildContour()
	if got := ball.contourRadii[0]; absFloat32(got-restRadius) > 0.001 {
		t.Fatalf("east contour radius = %v, want %v", got, restRadius)
	}
	if got := ball.contourRadii[contourBinCount/2]; absFloat32(got-restRadius) > 0.001 {
		t.Fatalf("west contour radius = %v, want %v", got, restRadius)
	}

	ball.points[0].position.x -= 8
	ball.rebuildContour()
	if got := ball.contourRadii[0]; got >= restRadius-4 {
		t.Fatalf("dented east contour radius = %v, want below %v", got, restRadius-4)
	}
	if got := ball.contourRadii[contourBinCount/2]; absFloat32(got-restRadius) > 0.001 {
		t.Fatalf("west contour changed to %v, want %v", got, restRadius)
	}
}

func TestMetaBallWallCollisionDeformsRing(t *testing.T) {
	tests := []struct {
		name             string
		x, y             float32
		dx, dy           float32
		contactDirection Vec2
	}{
		{name: "left", x: 36, y: 240, dx: -3, dy: 0.5, contactDirection: Vec2{x: -1}},
		{name: "right", x: 444, y: 240, dx: 3, dy: 0.5, contactDirection: Vec2{x: 1}},
		{name: "top", x: 240, y: 36, dx: 0.5, dy: -3, contactDirection: Vec2{y: -1}},
		{name: "bottom", x: 240, y: 444, dx: 0.5, dy: 3, contactDirection: Vec2{y: 1}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			effect := NewMetaBallEffect(42, 1, 16, 480, 480)
			ball := &effect.balls[0]
			ball.x, ball.y = test.x, test.y
			ball.dx, ball.dy = test.dx, test.dy
			effect.frameCount = 1

			effect.updatePhysics()

			if got := ball.supportRadius(test.contactDirection); got > restRadius-1 {
				t.Fatalf("wall-facing support = %v, want visible dent below %v", got, restRadius-1)
			}
			perpendicular := Vec2{x: -test.contactDirection.y, y: test.contactDirection.x}
			if got := ball.supportRadius(perpendicular); got <= restRadius {
				t.Fatalf("side support = %v, want bulge above %v", got, restRadius)
			}
			if test.contactDirection.x != 0 {
				if ball.dx*test.contactDirection.x >= 0 {
					t.Fatalf("normal velocity = %v, want bounce away from wall", ball.dx)
				}
				if ball.dy != test.dy {
					t.Fatalf("tangential velocity = %v, want %v", ball.dy, test.dy)
				}
			} else {
				if ball.dy*test.contactDirection.y >= 0 {
					t.Fatalf("normal velocity = %v, want bounce away from wall", ball.dy)
				}
				if ball.dx != test.dx {
					t.Fatalf("tangential velocity = %v, want %v", ball.dx, test.dx)
				}
			}
			for pointIdx, point := range ball.points {
				worldX := ball.x + point.position.x
				worldY := ball.y + point.position.y
				if worldX < 0 || worldX > float32(effect.width) || worldY < 0 || worldY > float32(effect.height) {
					t.Fatalf("point %d outside viewport at (%v, %v)", pointIdx, worldX, worldY)
				}
			}
		})
	}
}

func TestMetaBallVelocityIsClampedBeforeIntegration(t *testing.T) {
	effect := NewMetaBallEffect(42, 1, 16, 480, 480)
	setStationaryBall(&effect.balls[0], 100, 100)
	effect.balls[0].dx = 10.0

	effect.updatePhysics()

	if got, want := effect.balls[0].dx, float32(maxVelocity); got != want {
		t.Fatalf("clamped velocity = %v, want %v", got, want)
	}
	if got, want := effect.balls[0].x, float32(104.0); got != want {
		t.Fatalf("integrated position = %v, want %v", got, want)
	}
}

func TestMetaBallFieldsMergeAtOneHundredPixels(t *testing.T) {
	effect := NewMetaBallEffect(42, 2, 16, 480, 480)
	setStationaryBall(&effect.balls[0], 100, 100)
	setStationaryBall(&effect.balls[1], 200, 100)

	effect.update()

	midpoint := 100*effect.width + 150
	if got := effect.masterAccumGrid[midpoint]; got <= surfaceThreshold {
		t.Fatalf("midpoint intensity = %d, want above %d", got, surfaceThreshold)
	}
}

func TestMetaBallRadialDisplacementExpandsRenderedContour(t *testing.T) {
	effect := NewMetaBallEffect(42, 1, 16, 480, 480)
	setStationaryBall(&effect.balls[0], 100, 100)
	effect.frameCount = 1
	effect.balls[0].points[0].position.x += 15
	effect.balls[0].points[0].previousPosition = effect.balls[0].points[0].position

	effect.update()

	sample := 100*effect.width + 140
	if got := effect.masterAccumGrid[sample]; got <= surfaceThreshold {
		t.Fatalf("deformed contour intensity = %d, want above %d", got, surfaceThreshold)
	}
}

func TestMetaBallContourIsContinuousAcrossAngleWrap(t *testing.T) {
	effect := NewMetaBallEffect(42, 1, 16, 480, 480)
	ball := &effect.balls[0]
	ball.points[0].position.x += 10
	ball.rebuildContour()

	before := ball.contourRadii[contourBinCount-1]
	at := ball.contourRadii[0]
	after := ball.contourRadii[1]
	if absFloat32(before-at) > 0.6 || absFloat32(after-at) > 0.6 {
		t.Fatalf("contour wrap is discontinuous: before=%v at=%v after=%v", before, at, after)
	}
}

func TestMetaBallFloatSimulationRemainsStable(t *testing.T) {
	effect := NewMetaBallEffect(42, 10, 16, 480, 480)

	for frame := 0; frame < 2000; frame++ {
		effect.updatePhysics()
	}

	for ballIdx, ball := range effect.balls {
		values := []float32{ball.x, ball.y, ball.dx, ball.dy}
		for _, point := range ball.points {
			values = append(values, point.position.x, point.position.y, point.previousPosition.x, point.previousPosition.y)
		}
		for valueIdx, value := range values {
			if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
				t.Fatalf("ball %d value %d is not finite: %v", ballIdx, valueIdx, value)
			}
		}
		for pointIdx, point := range ball.points {
			radius := point.position.length()
			if radius < 5 || radius > 80 {
				t.Fatalf("ball %d point %d radius = %v, want between 5 and 80", ballIdx, pointIdx, radius)
			}
			next := ball.points[(pointIdx+1)%len(ball.points)]
			if edgeLength := next.position.sub(point.position).length(); edgeLength > effect.neighborRestLength*2 {
				t.Fatalf("ball %d edge %d length = %v, want at most %v", ballIdx, pointIdx, edgeLength, effect.neighborRestLength*2)
			}
			if winding := cross(point.position, next.position); winding <= 0 {
				t.Fatalf("ball %d edge %d reversed or collapsed with winding %v", ballIdx, pointIdx, winding)
			}
		}
	}
}

func TestMetaBallEnergyRestorationBoostsSlowestBallFirst(t *testing.T) {
	effect := NewMetaBallEffect(42, 3, 16, 480, 480)
	setStationaryBall(&effect.balls[0], 100, 100)
	setStationaryBall(&effect.balls[1], 200, 100)
	setStationaryBall(&effect.balls[2], 300, 100)
	effect.balls[0].dx = 0.5
	effect.balls[1].dx = 1
	effect.balls[2].dx = 2
	effect.targetTranslationalEnergy = 10
	effect.frameCount = energyRestoreDelay

	effect.restoreTranslationalEnergy()

	if effect.balls[0].dx <= 0.5 || effect.balls[0].dy != 0 {
		t.Fatalf("slowest velocity = (%v, %v), want heading-preserving boost", effect.balls[0].dx, effect.balls[0].dy)
	}
	if effect.balls[1].dx != 1 || effect.balls[2].dx != 2 {
		t.Fatalf("faster velocities changed to (%v, %v), want slowest-first restoration", effect.balls[1].dx, effect.balls[2].dx)
	}
}

func TestMetaBallEnergyRestorationConvergesWithoutOvershoot(t *testing.T) {
	effect := NewMetaBallEffect(42, 3, 16, 480, 480)
	for ballIdx := range effect.balls {
		setStationaryBall(&effect.balls[ballIdx], int32(100+ballIdx*100), 100)
		effect.balls[ballIdx].dx = 0.5
	}
	effect.targetTranslationalEnergy = 6
	effect.frameCount = energyRestoreDelay

	for frame := 0; frame < 2000; frame++ {
		effect.restoreTranslationalEnergy()
		effect.frameCount++
	}

	energy := effect.translationalEnergy()
	minimum := effect.targetTranslationalEnergy * (1 - energyRestoreDeadband)
	if energy < minimum || energy > effect.targetTranslationalEnergy+0.0001 {
		t.Fatalf("restored energy = %v, want between %v and %v", energy, minimum, effect.targetTranslationalEnergy)
	}
}

func setStationaryBall(ball *Ball, x, y int32) {
	ball.x = float32(x)
	ball.y = float32(y)
	ball.dx = 0
	ball.dy = 0
}

func BenchmarkMetaBallPhysics(b *testing.B) {
	for _, pointCount := range []int{8, 16, 32} {
		name := map[int]string{8: "N8", 16: "N16", 32: "N32"}[pointCount]
		b.Run(name, func(b *testing.B) {
			effect := NewMetaBallEffect(42, 10, pointCount, 480, 480)
			b.ReportAllocs()
			b.ResetTimer()
			for iteration := 0; iteration < b.N; iteration++ {
				effect.updatePhysics()
			}
		})
	}
}

func BenchmarkMetaBallProcessFrame(b *testing.B) {
	for _, pointCount := range []int{8, 16, 32} {
		name := map[int]string{8: "N8", 16: "N16", 32: "N32"}[pointCount]
		b.Run(name, func(b *testing.B) {
			effect := NewMetaBallEffect(42, 10, pointCount, 480, 480)
			frameBuffer := &FrameBuffer{Width: 480, Height: 480, Pixels: make([]uint16, 480*480)}
			b.ReportAllocs()
			b.ResetTimer()
			for iteration := 0; iteration < b.N; iteration++ {
				effect.ProcessFrame(1.0/60.0, frameBuffer)
			}
		})
	}
}
