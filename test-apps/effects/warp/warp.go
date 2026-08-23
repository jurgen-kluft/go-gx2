package warp

import fx_common "github.com/jurgen-kluft/go-gx2/test-apps/effects/common"

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

	Image []uint16 // The RGB565 image to warp

	CoolMapU       []int16 // 2.14 fixed point value (-1.0 to 1.0)
	CoolMapUSpeed  float32 // Speed for scrolling the CoolMapU
	CoolMapUOffset float32 // Offset for scrolling the CoolMapU
	CoolMapV       []int16 // 2.14 fixed point value (-1.0 to 1.0)
	CoolMapVSpeed  float32 // Speed for scrolling the CoolMapV
	CoolMapVOffset float32 // Offset for scrolling the CoolMapV
}

func NewWarpEffect(gridWidth, gridHeight, cellSize int32) *WarpEffect {
	gw := (gridWidth / cellSize) + 1
	gh := (gridHeight / cellSize) + 1
	w := &WarpEffect{
		GridWidth:      gw,                          // +1 to account for the last row/column of corners
		GridHeight:     gh,                          // +1 to account for the last row/column of corners
		CellSize:       cellSize,                    //
		Grid:           make([]GridLocation, gw*gh), //
		Image:          make([]uint16, gw*gh),       //
		CoolMapU:       make([]int16, gw*gh),        //
		CoolMapUSpeed:  0.1,                         // Example speed for CoolMapU
		CoolMapUOffset: 0.0,                         // Initial offset for CoolMapU
		CoolMapV:       make([]int16, gw*gh),        //
		CoolMapVSpeed:  0.1,                         // Example speed for CoolMapV
		CoolMapVOffset: 0.0,                         // Initial offset for CoolMapV
	}

	w.Image, _, _, _ = fx_common.ReadPngAsRGB565("image.png")

	w.createCoolMaps()
	return w
}

func (w *WarpEffect) createCoolMaps() {
	// to implement
}

func (w *WarpEffect) update(dt float32) {
	// Update the offsets for the cooling maps based on their speeds and the elapsed time
	w.CoolMapUOffset += w.CoolMapUSpeed * dt
	w.CoolMapVOffset += w.CoolMapVSpeed * dt
}

func (w *WarpEffect) drawCell(cellX, cellY int32, fb *fx_common.FrameBuffer) {
	// to implement
}

func (w *WarpEffect) draw(fb *fx_common.FrameBuffer) {
	for y := int32(0); y < w.GridHeight; y++ {
		for x := int32(0); x < w.GridWidth; x++ {
			w.drawCell(x, y, fb)
		}
	}
}

func (w *WarpEffect) ProcessFrame(dt float32, fb *fx_common.FrameBuffer) {
	w.update(dt)
	w.draw(fb)
}
