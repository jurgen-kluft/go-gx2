package fx_warp

import (
	"math/rand"

	fx_common "github.com/jurgen-kluft/go-gx2/test-apps/effects/common"
)

/*

Imaging your screen and we put a grid on top, each cell being NxN pixels.
For each grid location, corners of each cell, we can give it a dU/dV value.
This will then be used to render the image, by texture mapping each cell.

We can then create two cooling images, like we did for 'coolfire', and scroll
them over the grid, then modulate (and clamp) the dU/dV values of each cell
based on the pixel values of the cooling images.

These cooling images use int16, 2.14 fixed point values, so that we can have
negative values, which can warp dU/dV in positive and negative directions.

This creates a kind of fluid effect, where the image is warped based on the cooling
images.

Pseudo Code to render a cell:

            Variable Declarations
            FIXEDPOINT VLDx, VRDx, HDx
            FIXEDPOINT VLDy, VRDy, HDy
            FIXEDPOINT TX1, TY1, TX2, TY2, tx, ty
            INTEGER x, y

            Code Begins
            VLDx = (Cx - Ax) / 16          'Rate of change of X down the left side of the wonky square
            VRDx = (Dx - Bx) / 16          'Rate of change of X down the right side
            VLDy = (Cy - Ay) / 16          'Rate of change of Y down the left side
            VRDy = (Dy - By) / 16          'Rate of change of Y down the right side

            TX1  = Ax
            TY1  = Ay
            TX2  = Bx
            TY2  = By

            loop y from 0 to 15
                    HDx  = (TX2-TX1) / 16   'Rate of change of X across the wonky polygon
                    HDy  = (TY2-TY1) / 16   'Rate of change of Y across the wonky polygon
                    tx = TX1
                    ty = TY1

                    loop x from 0 to 15
                            Image2(x, y) = Image1( int(tx), int(ty) )
                            tx = tx + HDx
                            ty = ty + HDy
                    end of x loop

                    TX1 = TX1 + VLDx;
                    TY1 = TY1 + VLDy;
                    TX2 = TX2 + VRDx;
                    TY2 = TY2 + VRDy;
            end of y loop

*/

type GridLocation struct {
	dU int16 // 8.8 fixed point value (-128.0 to 127.99609375)
	dV int16 // 8.8 fixed point value (-128.0 to 127.99609375)
}

type WarpEffect struct {
	GridWidth  int32
	GridHeight int32
	CellSize   int32
	Grid       []GridLocation

	Image       []uint16 // The RGB565 image to warp
	ImageWidth  int32
	ImageHeight int32

	CoolMapU       []int16 // 2.14 fixed point value (-1.0 to 1.0)
	CoolMapUSpeed  float32 // Speed for scrolling the CoolMapU
	CoolMapUOffset float32 // Offset for scrolling the CoolMapU
	CoolMapV       []int16 // 2.14 fixed point value (-1.0 to 1.0)
	CoolMapVSpeed  float32 // Speed for scrolling the CoolMapV
	CoolMapVOffset float32 // Offset for scrolling the CoolMapV
}

func NewEffect(gridWidth, gridHeight, cellSize int32) *WarpEffect {
	gw := (gridWidth / cellSize) + 1
	gh := (gridHeight / cellSize) + 1
	image := make([]uint16, gridWidth*gridHeight)
	w := &WarpEffect{
		GridWidth:      gw,                          // +1 to account for the last row/column of corners
		GridHeight:     gh,                          // +1 to account for the last row/column of corners
		CellSize:       cellSize,                    //
		Grid:           make([]GridLocation, gw*gh), //
		Image:          image,                       //
		ImageWidth:     gridWidth,                   //
		ImageHeight:    gridHeight,                  //
		CoolMapU:       make([]int16, gw*gh),        //
		CoolMapUSpeed:  1.5,                         // Example speed for CoolMapU
		CoolMapUOffset: 0.0,                         // Initial offset for CoolMapU
		CoolMapV:       make([]int16, gw*gh),        //
		CoolMapVSpeed:  1.9,                         // Example speed for CoolMapV
		CoolMapVOffset: 0.0,                         // Initial offset for CoolMapV
	}

	if loadedImage, width, height, err := fx_common.ReadPngAsRGB565("warp/image.png"); err == nil {
		w.Image = loadedImage
		w.ImageWidth = width
		w.ImageHeight = height
	} else {
		// If the image fails to load, fill the image with a gradient pattern
		for y := int32(0); y < gridHeight; y++ {
			for x := int32(0); x < gridWidth; x++ {
				r := uint16((x * 31) / gridWidth)                      // Red channel (5 bits)
				g := uint16((y * 63) / gridHeight)                     // Green channel (6 bits)
				b := uint16(((x + y) * 31) / (gridWidth + gridHeight)) // Blue channel (5 bits)
				w.Image[y*gridWidth+x] = (r << 11) | (g << 5) | b
			}
		}
	}
	// Make the image a checkerboard pattern for better visual effect, cellSize is the size of each square in the checkerboard
	// for y := int32(0); y < gridHeight; y++ {
	// 	for x := int32(0); x < gridWidth; x++ {
	// 		if ((x/cellSize)+(y/cellSize))%2 == 0 {
	// 			w.Image[y*gridWidth+x] = 0xFFFF // White
	// 		} else {
	// 			w.Image[y*gridWidth+x] = 0x0000 // Black
	// 		}
	// 	}
	// }

	w.createCoolMaps()
	w.refreshGrid()
	return w
}

func (w *WarpEffect) createCoolMaps() {
	size := len(w.CoolMapU)
	if size == 0 {
		return
	}

	numPoints := 4200
	numPasses := 20

	currentU := make([]float32, size)
	previousU := make([]float32, size)

	for i := 0; i < numPoints; i++ {
		x := rand.Int31n(w.GridWidth)
		y := rand.Int31n(w.GridHeight)
		index := y*w.GridWidth + x
		previousU[index] = 16 * (rand.Float32()*2 - 1)
	}

	currentV := make([]float32, size)
	previousV := make([]float32, size)

	for i := 0; i < numPoints; i++ {
		x := rand.Int31n(w.GridWidth)
		y := rand.Int31n(w.GridHeight)
		index := y*w.GridWidth + x
		previousV[index] = 16 * (rand.Float32()*2 - 1)
	}

	for pass := 0; pass < numPasses; pass++ {
		for y := int32(0); y < w.GridHeight; y++ {
			for x := int32(0); x < w.GridWidth; x++ {
				u1 := previousU[wrapIndex(x+1, w.GridWidth)+(y*w.GridWidth)]
				u2 := previousU[wrapIndex(x-1, w.GridWidth)+(y*w.GridWidth)]
				u3 := previousU[x+(wrapIndex(y+1, w.GridHeight)*w.GridWidth)]
				u4 := previousU[x+(wrapIndex(y-1, w.GridHeight)*w.GridWidth)]

				sumU := (u1 + u2 + u3 + u4) / 4

				v1 := previousV[wrapIndex(x+1, w.GridWidth)+(y*w.GridWidth)]
				v2 := previousV[wrapIndex(x-1, w.GridWidth)+(y*w.GridWidth)]
				v3 := previousV[x+(wrapIndex(y+1, w.GridHeight)*w.GridWidth)]
				v4 := previousV[x+(wrapIndex(y-1, w.GridHeight)*w.GridWidth)]

				sumV := (v1 + v2 + v3 + v4) / 4

				index := y*w.GridWidth + x
				currentU[index] = sumU
				currentV[index] = sumV
			}
		}
		currentU, previousU = previousU, currentU
		currentV, previousV = previousV, currentV
	}

	for index := range w.CoolMapU {
		w.CoolMapU[index] = int16(previousU[index] * (1 << 14)) // Convert to 2.14 fixed point
		w.CoolMapV[index] = int16(previousV[index] * (1 << 14)) // Convert to 2.14 fixed point
	}
}

func (w *WarpEffect) update(dt float32) {
	// Update the offsets for the cooling maps based on their speeds and the elapsed time
	w.CoolMapUOffset += w.CoolMapUSpeed * dt
	w.CoolMapVOffset += w.CoolMapVSpeed * dt

	if w.CoolMapUOffset >= float32(w.GridWidth) {
		w.CoolMapUOffset -= float32(w.GridWidth)
	}
	if w.CoolMapVOffset >= float32(w.GridHeight) {
		w.CoolMapVOffset -= float32(w.GridHeight)
	}

	w.refreshGrid()
}

func (w *WarpEffect) drawCell(cellX, cellY int32, fb *fx_common.FrameBuffer) {
	if len(w.Image) == 0 || w.ImageWidth == 0 || w.ImageHeight == 0 {
		return
	}

	x0 := cellX * w.CellSize
	y0 := cellY * w.CellSize
	x1 := x0 + w.CellSize
	y1 := y0 + w.CellSize

	if x0 >= fb.Width || y0 >= fb.Height {
		return
	}
	if x1 > fb.Width {
		x1 = fb.Width
	}
	if y1 > fb.Height {
		y1 = fb.Height
	}

	cellWidth := x1 - x0
	cellHeight := y1 - y0
	if cellWidth <= 0 || cellHeight <= 0 {
		return
	}

	baseAX := screenToImageCoord(x0, fb.Width, w.ImageWidth)
	baseAY := screenToImageCoord(y0, fb.Height, w.ImageHeight)
	baseBX := screenToImageCoord(x1, fb.Width, w.ImageWidth)
	baseBY := baseAY
	baseCX := baseAX
	baseCY := screenToImageCoord(y1, fb.Height, w.ImageHeight)
	baseDX := baseBX
	baseDY := baseCY

	gridIndex := cellY*w.GridWidth + cellX
	gridA := w.Grid[gridIndex]
	gridB := w.Grid[gridIndex+1]
	gridC := w.Grid[gridIndex+w.GridWidth]
	gridD := w.Grid[gridIndex+w.GridWidth+1]

	ax := baseAX + fixed8p8ToFloat32(gridA.dU)

	ay := baseAY + fixed8p8ToFloat32(gridA.dV)
	bx := baseBX + fixed8p8ToFloat32(gridB.dU)
	by := baseBY + fixed8p8ToFloat32(gridB.dV)
	cx := baseCX + fixed8p8ToFloat32(gridC.dU)
	cy := baseCY + fixed8p8ToFloat32(gridC.dV)
	dx := baseDX + fixed8p8ToFloat32(gridD.dU)
	dy := baseDY + fixed8p8ToFloat32(gridD.dV)

	heightScale := float32(cellHeight)
	if cellHeight > 1 {
		heightScale = float32(cellHeight - 1)
	}
	leftDeltaX := (cx - ax) / heightScale
	leftDeltaY := (cy - ay) / heightScale
	rightDeltaX := (dx - bx) / heightScale
	rightDeltaY := (dy - by) / heightScale

	leftX := ax
	leftY := ay
	rightX := bx
	rightY := by

	for row := int32(0); row < cellHeight; row++ {
		widthScale := float32(cellWidth)
		if cellWidth > 1 {
			widthScale = float32(cellWidth - 1)
		}
		horizontalDeltaX := (rightX - leftX) / widthScale
		horizontalDeltaY := (rightY - leftY) / widthScale

		sampleX := leftX
		sampleY := leftY
		dstOffset := (y0+row)*fb.Width + x0
		for column := int32(0); column < cellWidth; column++ {
			fb.Pixels[dstOffset+column] = w.sampleImage(sampleX, sampleY)
			sampleX += horizontalDeltaX
			sampleY += horizontalDeltaY
		}

		leftX += leftDeltaX
		leftY += leftDeltaY
		rightX += rightDeltaX
		rightY += rightDeltaY
	}
}

func (w *WarpEffect) draw(fb *fx_common.FrameBuffer) {

	for y := int32(0); y < w.GridHeight-1; y++ {
		for x := int32(0); x < w.GridWidth-1; x++ {
			w.drawCell(x, y, fb)
		}
	}

	// DEBUG
	drawCoolMapU := false
	if drawCoolMapU {
		// draw cool map U
		for y := int32(0); y < w.GridHeight-1; y++ {
			for x := int32(0); x < w.GridWidth-1; x++ {
				index := y*w.GridWidth + x
				uValue := (int32(w.CoolMapU[index]) * 255) >> 14 // Convert from 2.14 fixed point to 0-255 range
				r := uint8(uValue & 0xFF)
				for yc := int32(0); yc < w.CellSize; yc++ {
					for xc := int32(0); xc < w.CellSize; xc++ {
						fbX := x*w.CellSize + xc
						fbY := y*w.CellSize + yc
						fb.Pixels[fbY*fb.Width+fbX] = fx_common.ConvertToRGB565(r, 0, 0)
					}
				}
			}
		}
	}

	// DEBUG
	drawCoolMapV := false
	if drawCoolMapV {
		// draw cool map V
		for y := int32(0); y < w.GridHeight-1; y++ {
			for x := int32(0); x < w.GridWidth-1; x++ {
				index := y*w.GridWidth + x
				vValue := (int32(w.CoolMapV[index]) * 255) >> 14 // Convert from 2.14 fixed point to 0-255 range
				b := uint8(vValue & 0xFF)
				for yc := int32(0); yc < w.CellSize; yc++ {
					for xc := int32(0); xc < w.CellSize; xc++ {
						fbX := x*w.CellSize + xc
						fbY := y*w.CellSize + yc
						fb.Pixels[fbY*fb.Width+fbX] = fx_common.ConvertToRGB565(0, 0, b)
					}
				}
			}
		}
	}
}

func (w *WarpEffect) ProcessFrame(dt float32, fb *fx_common.FrameBuffer) {
	w.update(dt)
	w.draw(fb)
}

func (w *WarpEffect) refreshGrid() {
	if len(w.Grid) == 0 || len(w.CoolMapU) == 0 || len(w.CoolMapV) == 0 {
		return
	}

	amplitude := int32(8)
	fixedAmplitude := amplitude << 8

	for y := int32(0); y < w.GridHeight; y++ {
		for x := int32(0); x < w.GridWidth; x++ {
			u := w.sampleCoolMapU(x, y)
			v := w.sampleCoolMapV(x, y)

			index := y*w.GridWidth + x
			dU := int32(w.Grid[index].dU)
			dV := int32(w.Grid[index].dV)

			// Add (u * fixedAmplitude) as 8.8 fixed point value to dU
			// Grid values are 8.8, cool values are 2.14, so we need to shift right by 6 to convert from 2.14 to 8.8
			dU += (u * amplitude) >> 14
			dV += (v * amplitude) >> 14

			if dU > fixedAmplitude {
				dU = fixedAmplitude
			} else if dU < -fixedAmplitude {
				dU = -fixedAmplitude
			}

			if dV > fixedAmplitude {
				dV = fixedAmplitude
			} else if dV < -fixedAmplitude {
				dV = -fixedAmplitude
			}

			w.Grid[index].dU = int16(dU)
			w.Grid[index].dV = int16(dV)

			// w.Grid[index].dU = clampInt16(int32(w.Grid[index].dU) + ((u * fixedAmplitude) / 16384))
			// w.Grid[index].dV = clampInt16(int32(w.Grid[index].dV) + ((v * fixedAmplitude) / 16384))
		}
	}
}

func (w *WarpEffect) sampleCoolMapU(x, y int32) int32 {
	return sampleCoolMapAxis(w.CoolMapU, w.GridWidth, w.GridHeight, x, y, w.CoolMapUOffset)
}

func (w *WarpEffect) sampleCoolMapV(x, y int32) int32 {
	return sampleCoolMapAxis(w.CoolMapV, w.GridWidth, w.GridHeight, x, y, w.CoolMapVOffset)
}

func sampleCoolMapAxis(data []int16, width, height, x, y int32, offset float32) int32 {
	position := float32(x) + offset
	index0 := int32(position)
	if index0 >= width {
		index0 -= width
	}
	return int32(data[(y*width)+index0])
}

func (w *WarpEffect) sampleImage(u, v float32) uint16 {
	imageX := wrapIndex(int32(u), w.ImageWidth)
	imageY := wrapIndex(int32(v), w.ImageHeight)
	return w.Image[imageY*w.ImageWidth+imageX]
}

func fixed8p8ToFloat32(value int16) float32 {
	return float32(value) / 256.0
}

func screenToImageCoord(value, screenSize, imageSize int32) float32 {
	if screenSize <= 0 || imageSize <= 0 {
		return 0
	}
	return (float32(value) / float32(screenSize)) * float32(imageSize)
}

func wrapIndex(value, max int32) int32 {
	if value <= 0 {
		return 0
	}
	wrapped := value % max
	if wrapped < 0 {
		wrapped += max
	}
	return wrapped
}

func clampInt16(value int32) int16 {
	if value < -32768 {
		return -32768
	}
	if value > 32767 {
		return 32767
	}
	return int16(value)
}
