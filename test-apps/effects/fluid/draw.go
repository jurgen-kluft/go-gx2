package fx_fluid

import (
	"math"

	fx_common "github.com/jurgen-kluft/go-gx2/test-apps/effects/common"
)

func (f *FluidEffect) DrawVelocityField(fb *fx_common.FrameBuffer, c color) {
	cellCountX := float32(f.CellCountX)
	halfCellSizeX := 0.5 * cellCountX
	cellCountY := float32(f.CellCountY)
	halfCellSizeY := 0.5 * cellCountY

	yIdx := int32(0)
	for y := int32(1); y <= f.CellCountY; y++ {
		xIdx := yIdx + 1
		for x := int32(1); x <= f.CellCountX; x++ {
			xv := f.CurrentVelocity.XVel[xIdx] * cellCountX
			yv := f.CurrentVelocity.YVel[xIdx] * cellCountY

			posX := float32(x)*cellCountX + halfCellSizeX
			posY := float32(y)*cellCountY + halfCellSizeY
			endX := posX + xv
			endY := posY + yv

			drawLineV(fb, posX, posY, endX, endY, c)

			xIdx++
		}
		yIdx += f.PaddedCellCountX
	}
}

func drawLineV(fb *fx_common.FrameBuffer, x1, y1, x2, y2 float32, c color) {

}

// func drawDensityField() {
// 	baseColor := r.ColorToHSV(gui.FluidColor)

// 	createColor := func(hueOffset, valueOffset float32) r.Color {
// 		// hue 0-360, saturation 0-1, value 0-1
// 		hue := float32(math.Mod(float64(baseColor.X+hueOffset), 360))
// 		value := baseColor.Z * valueOffset
// 		return r.ColorFromHSV(hue, baseColor.Y, value)
// 	}

// 	for x := range macFluid.Size + 1 {
// 		for y := range macFluid.Size + 1 {
// 			density := macFluid.DensityField[x][y]
// 			density1 := macFluid.DensityField[x+1][y]
// 			density2 := macFluid.DensityField[x][y+1]
// 			density3 := macFluid.DensityField[x+1][y+1]

// 			var vel, vel1, vel2, vel3 float32 = 0, 0, 0, 0
// 			if gui.ColorizeTurbulence {
// 				vel = (macFluid.XVelocities[x][y] + macFluid.YVelocities[x][y]) * 4
// 				vel1 = (macFluid.XVelocities[x+1][y] + macFluid.YVelocities[x+1][y]) * 4
// 				vel2 = (macFluid.XVelocities[x][y+1] + macFluid.YVelocities[x][y+1]) * 4
// 				vel3 = (macFluid.XVelocities[x+1][y+1] + macFluid.YVelocities[x+1][y+1]) * 4
// 			}

// 			topLeftColor := createColor(vel, density)
// 			bottomLeftColor := createColor(vel2, density2)
// 			topRightColor := createColor(vel3, density3)
// 			bottomRightColor := createColor(vel1, density1)

// 			pos := r.NewVector2(float32(x*cellSize), float32(y*cellSize))
// 			size := r.NewVector2(cellSize, cellSize)

// 			r.DrawRectangleGradientEx(r.Rectangle{
// 				X:      pos.X,
// 				Y:      pos.Y,
// 				Width:  size.X,
// 				Height: size.Y,
// 			}, topLeftColor, bottomLeftColor, topRightColor, bottomRightColor)
// 		}
// 	}
// }

func (f *FluidEffect) DrawDensityField(fb *fx_common.FrameBuffer, c color, colorizeTurbulence bool) {
	h, s, v := rgbToHSV(c)

	createColor := func(hueOffset, valueOffset float32) color {
		// hue 0-360, saturation 0-1, value 0-1
		hue := float32(math.Mod(float64(h+hueOffset), 360))
		value := v * valueOffset
		return hsvToRGB(hue, s, value)
	}

	sizeX := float32(f.CellCountX)
	sizeY := float32(f.CellCountY)

	y0Idx := int32(0)
	for y := int32(0); y < f.CellCountY; y++ {
		y1Idx := y0Idx + f.PaddedCellCountX
		for x := int32(0); x < f.CellCountX; x++ {
			x0Idx := y0Idx
			x1Idx := x0Idx + x
			x2Idx := y1Idx
			x3Idx := x2Idx + x

			density := f.CurrentDensity.Density[x0Idx]
			density1 := f.CurrentDensity.Density[x1Idx]
			density2 := f.CurrentDensity.Density[x2Idx]
			density3 := f.CurrentDensity.Density[x3Idx]

			var vel, vel1, vel2, vel3 float32 = 0, 0, 0, 0
			if colorizeTurbulence {
				vel = (f.CurrentVelocity.XVel[x0Idx] + f.CurrentVelocity.YVel[x0Idx]) * 4
				vel1 = (f.CurrentVelocity.XVel[x1Idx] + f.CurrentVelocity.YVel[x1Idx]) * 4
				vel2 = (f.CurrentVelocity.XVel[x2Idx] + f.CurrentVelocity.YVel[x2Idx]) * 4
				vel3 = (f.CurrentVelocity.XVel[x3Idx] + f.CurrentVelocity.YVel[x3Idx]) * 4
			}

			topLeftColor := createColor(vel, density)
			bottomLeftColor := createColor(vel2, density2)
			topRightColor := createColor(vel3, density3)
			bottomRightColor := createColor(vel1, density1)

			posX := int32(float32(x) * sizeX)
			posY := int32(float32(y) * sizeY)

			drawRectangleGradient(fb, posX, posY, int32(sizeX), int32(sizeY), topLeftColor, bottomLeftColor, topRightColor, bottomRightColor)
		}
		y0Idx += f.PaddedCellCountX
	}
}

func drawRectangleGradient(fb *fx_common.FrameBuffer, rx, ry, width, height int32, topLeftColor, bottomLeftColor, topRightColor, bottomRightColor color) {
}
