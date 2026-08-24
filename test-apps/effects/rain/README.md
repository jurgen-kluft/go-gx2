# Ultra-Low-Memory Multi-Drop Fixed-Point Ripple Engine

A highly optimized engine for rendering multiple, overlapping raindrop ripples using compressed **Q8.8 fixed-point math** (`int16`). Specially tailored for resource-constrained embedded systems and microcontrollers (such as the **ESP32-S3**), this architecture minimizes memory consumption and structures algorithms to match 16-bit register capabilities perfectly.

---

## 📐 Embedded & Mathematical Architecture

To deliver fluid frame rates on microcontrollers without melting the heap or exhausting data caches, the engine relies on the following design pillars:

1. **Q8.8 Fixed-Point Format**: By replacing standard `int32` elements with compact `int16` types, the global heightfield memory footprint is cut exactly in half. Fixed-point `1.0` is represented by `1 << 8 = 256`, preserving smooth fractional resolution across tight memory profiles.
2. **Hardware Register Alignment**: Modern microcontrollers (like the ESP32-S3) feature SIMD instruction sets capable of packing and executing two 16-bit operations simultaneously in a single clock cycle. This layout is structured to trigger those assembly optimizations.
3. **Sub-Pixel Phase Indexing**: Individual wave velocities are tracked as fractional 16-bit stepping vectors. Ripples expand smoothly across a small 256-entry sine table at granular fractional speeds (e.g., `90 / 256 ≈ 0.35` slots per frame) without aliasing or jagged stepping artifacts.
4. **Cache-Friendly 1D Layouts**: All dynamic canvases, heightfields, and shared static distance arrays use flat 1D layouts (`[]T`). This avoids continuous multi-dimensional pointer indirection, allowing hardware prefetchers to cache pixel rows efficiently.

---

## 💻 Engine Specification (Go)

```go
package ripple

import (
	"math"
)

const Shift = 8
const One = 1 << Shift      // 256 (Fixed Point 1.0 in Q8.8)
const DropSize = 256        // Bounding box dimensions (Width and Height)
const Center = DropSize / 2 // Center pixel offset (128)

// RainDrop tracks state metrics using highly efficient 16-bit values.
type RainDrop struct {
	X, Y        int          // Screen coordinates of the drop epicenter
	Age         int16        // Current frame lifecycle index
	MaxLifetime int16        // Total frames this drop runs before destruction
	Phase       int16        // Q8.8: Accumulates granular movement fractions over time
	Speed       int16        // Q8.8: Sub-pixel distance to step per frame (e.g., 90 for ~0.35 steps)
	RippleTable [256]int16   // 16-bit Local 1D curve mapping generated for the active frame
	IsActive    bool         // Life status flag for updater scheduling
}

// RainEffect groups configuration metrics, pre-computed tables, and object pools.
type RainEffect struct {
	ScreenWidth   int
	ScreenHeight  int
	SinLUT        [256]int16
	DampenLUT     [256]int16
	DistanceTable [DropSize * DropSize]uint8 // Shared 1D local coordinate grid
	DropsPool     []RainDrop
}

// NewRainEffect builds and optimizes a new weather simulation grid context.
func NewRainEffect(screenW, screenH, maxConcurrentDrops int) *RainEffect {
	re := &RainEffect{
		ScreenWidth:  screenW,
		ScreenHeight: screenH,
		DropsPool:    make([]RainDrop, maxConcurrentDrops),
	}
	re.initLUTs()
	return re
}

// initLUTs generates pure math matrices once on application lifecycle boot.
func (re *RainEffect) initLUTs() {
	// 1. Initialize Sine LUT (-1.0 to 1.0 in Q8.8 mapped across 256 steps)
	for i := 0; i < 256; i++ {
		angle := (float64(i) / 256.0) * 2.0 * math.Pi
		re.SinLUT[i] = int16(math.Sin(angle) * float64(One))
	}

	// 2. Initialize Physics-Accurate Dampen LUT (1/sqrt(x + epsilon) in Q8.8)
	epsilon := 1.0
	maxVal := 1.0 / math.Sqrt(0.0 + epsilon)
	for x := 0; x < 256; x++ {
		physicalDrop := 1.0 / math.Sqrt(float64(x) + epsilon)
		normalizedDrop := physicalDrop / maxVal
		re.DampenLUT[x] = int16(normalizedDrop * float64(One))
	}

	// 3. Initialize Shared Local 1D Distance Map
	for y := 0; y < DropSize; y++ {
		for x := 0; x < DropSize; x++ {
			dx := x - Center
			dy := y - Center
			dist := math.Sqrt(float64(dx*dx + dy*dy))
			
			if dist > 255 {
				dist = 255
			}
			
			flatIndex := (y * DropSize) + x
			re.DistanceTable[flatIndex] = uint8(dist)
		}
	}
}

// mulFP executes a safe Q8.8 multiplication using int32 casting to avoid bit overflow.
func mulFP(a, b int16) int16 {
	return int16((int32(a) * int32(b)) >> Shift)
}

// SpawnDrop searches the object pool to deploy a localized raindrop instance.
// speedFP must be passed as a Q8.8 value (e.g., 90 for ~0.35 index steps per frame).
func (re *RainEffect) SpawnDrop(x, y, maxLifetime, speedFP int16) {
	for i := range re.DropsPool {
		if !re.DropsPool[i].IsActive {
			re.DropsPool[i].X = int(x)
			re.DropsPool[i].Y = int(y)
			re.DropsPool[i].Age = 0
			re.DropsPool[i].MaxLifetime = maxLifetime
			re.DropsPool[i].Phase = 0
			re.DropsPool[i].Speed = speedFP
			re.DropsPool[i].IsActive = true
			return
		}
	}
}

// Update handles state machine evaluation and granular wave tables tracking.
func (re *RainEffect) Update() {
	for i := range re.DropsPool {
		drop := &re.DropsPool[i]
		if !drop.IsActive {
			continue
		}

		drop.Age++
		if drop.Age >= drop.MaxLifetime {
			drop.IsActive = false
			continue
		}

		// 1. Advance the wave phase using granular sub-pixel fixed-point adjustments
		drop.Phase += drop.Speed

		// 2. Compute individual time decay multipliers in Q8.8
		globalFade := int16(((drop.MaxLifetime - drop.Age) << Shift) / drop.MaxLifetime)

		// 3. Extract the clean integer lookup pointer for array step checking (Shift by 8)
		currentFrameOffset := int(drop.Phase >> Shift)

		for x := 0; x < 256; x++ {
			// 'x * 8' controls spatial wavelength density
			lutIndex := (x*8 - currentFrameOffset) & 255
			wave := -re.SinLUT[lutIndex] // Negative sine delivers physical downward craters

			spatialDampen := re.DampenLUT[x]
			combinedDampening := mulFP(spatialDampen, globalFade)
			drop.RippleTable[x] = mulFP(wave, combinedDampening)
		}
	}
}
```

---

## 🎬 16-Bit Localized Rendering & Blending Pipeline

The rendering function updates a linear **`[]int16`** height buffer. Because overlapping ripples additively accumulate heights via standard integer accumulation (`+=`), intersecting wave fronts pass through and interfere with each other natively.

```go
// Composite blends all active elements directly into your flat int16 screen buffer.
// globalBuffer dimensions must be completely configured to match (re.ScreenWidth * re.ScreenHeight).
func (re *RainEffect) Composite(globalBuffer []int16) {
	for i := range re.DropsPool {
		drop := &re.DropsPool[i]
		if !drop.IsActive {
			continue
		}

		// Calculate global top-left bounding tile start indices
		startX := drop.X - Center
		startY := drop.Y - Center

		for localY := 0; localY < DropSize; localY++ {
			screenY := startY + localY
			if screenY < 0 || screenY >= re.ScreenHeight {
				continue // Clip vertical canvas boundaries
			}

			// Cache row indices outside inner pixel loops
			localRowOffset := localY * DropSize
			screenRowOffset := screenY * re.ScreenWidth

			for localX := 0; localX < DropSize; localX++ {
				screenX := startX + localX
				if screenX < 0 || screenX >= re.ScreenWidth {
					continue // Clip horizontal canvas boundaries
				}

				// 1. Extract physical distance offset using continuous memory space
				localFlatIndex := localRowOffset + localX
				dist := re.DistanceTable[localFlatIndex]

				// 2. Fetch specific frame wave vector data from droplet context
				amplitude := drop.RippleTable[dist]

				// 3. Accumulate results cleanly inside 16-bit integer boundaries
				screenFlatIndex := screenRowOffset + screenX
				globalBuffer[screenFlatIndex] += amplitude
			}
		}
	}
}
```

---

## 🌊 Fast Environment Refraction Loop

To process the output into an environment refraction mapping effect, a 4-way finite difference slope filter is run across the heightfield. The slopes are converted directly into texture U/V offset displacements.

```go
// RenderWaterDistortion generates the final distorted screen output.
// - srcRGBA: Flat 1D byte array of your original background image (size: screenW * screenH * 4)
// - destRGBA: Flat 1D target buffer sent to your screen/window canvas (size: screenW * screenH * 4)
// - heightBuffer: Your flat 1D []int16 globalBuffer updated by the ripple engine
func (re *RainEffect) RenderWaterDistortion(srcRGBA, destRGBA []byte, heightBuffer []int16) {
	const DistortionStrength = 2 

	for y := 1; y < re.ScreenHeight-1; y++ {
		rowOffset := y * re.ScreenWidth
		prevRowOffset := (y - 1) * re.ScreenWidth
		nextRowOffset := (y + 1) * re.ScreenWidth

		for x := 1; x < re.ScreenWidth-1; x++ {
			currentIdx := rowOffset + x

			// Calculate Horizontal (dX) and Vertical (dY) slopes using 4-way neighbors
			hLeft  := heightBuffer[currentIdx-1]
			hRight := heightBuffer[currentIdx+1]
			hUp    := heightBuffer[prevRowOffset+x]
			hDown  := heightBuffer[nextRowOffset+x]

			// Shifting right by 8 cleanly drops the Q8.8 fixed point values back into scalar integers
			dX := int(hRight - hLeft) >> Shift
			dY := int(hDown - hUp) >> Shift

			// Map slopes directly to UV pixel texture coordinate offsets
			offsetX := dX * DistortionStrength
			offsetY := dY * DistortionStrength

			// Calculate target background coordinates with screen boundary clipping
			sampleX := x + offsetX
			sampleY := y + offsetY

			if sampleX < 0 { sampleX = 0 }
			if sampleX >= re.ScreenWidth { sampleX = re.ScreenWidth - 1 }
			if sampleY < 0 { sampleY = 0 }
			if sampleY >= re.ScreenHeight { sampleY = re.ScreenHeight - 1 }

			// Read from distorted source pixel, write to output destination
			srcPixelIdx  := (sampleY * re.ScreenWidth + sampleX) * 4
			destPixelIdx := currentIdx * 4

			destRGBA[destPixelIdx]   = srcRGBA[srcPixelIdx]   // Red
			destRGBA[destPixelIdx+1] = srcRGBA[srcPixelIdx+1] // Green
			destRGBA[destPixelIdx+2] = srcRGBA[srcPixelIdx+2] // Blue
			destRGBA[destPixelIdx+3] = 255                    // Alpha (Opaque)
		}
	}
}
```