package main

import rl "github.com/gen2brain/raylib-go/raylib"

type Tile struct {
	R     uint16
	G     uint16
	B     uint16
	A     uint16
	Alive bool
	Age   uint8
}

// board alias
type Board struct {
	tiles [][]Tile
}

type RenderSettings struct {
	FadeOpacity bool
	FadeColor   bool
	FadeLength  int

	GridColorR uint8
	GridColorG uint8
	GridColorB uint8
	GridColorA uint8
}

type ConwayGameOfLife struct {
	ScreenWidth  int32
	ScreenHeight int32

	Settings RenderSettings

	SelectedR uint16
	SelectedG uint16
	SelectedB uint16
	SelectedA uint16

	Lapsed   float32
	Step     float32
	CellSize int32

	BoardA  *Board
	BoardB  *Board
	Current *Board
	Next    *Board
}

func NewConwayEffect(screenWidth, screenHeight int32) *ConwayGameOfLife {
	state := &ConwayGameOfLife{
		ScreenWidth:  screenWidth,
		ScreenHeight: screenHeight,
		CellSize:     int32(16),
		SelectedR:    255,
		SelectedG:    0,
		SelectedB:    0,
		SelectedA:    255,
	}

	settings := RenderSettings{
		FadeOpacity: false,
		FadeColor:   false,
		FadeLength:  5,
		GridColorR:  130,
		GridColorG:  130,
		GridColorB:  130,
		GridColorA:  255,
	}
	state.Settings = settings

	state.Step = 1.0 / float32(2.0)
	state.BoardA = newBoard(int(state.ScreenWidth)/int(state.CellSize), int(state.ScreenHeight)/int(state.CellSize))
	state.BoardB = newBoard(int(state.ScreenWidth)/int(state.CellSize), int(state.ScreenHeight)/int(state.CellSize))
	state.Current = state.BoardA
	state.Next = state.BoardB

	state.Current.addShapes() // Add initial shapes to the board

	return state
}

func newBoard(rows, cols int) *Board {
	board := &Board{
		tiles: make([][]Tile, rows),
	}
	for x := range board.tiles {
		board.tiles[x] = make([]Tile, cols)

		for y := range board.tiles[x] {
			board.tiles[x][y].Alive = false
			board.tiles[x][y].R = 0
			board.tiles[x][y].G = 0
			board.tiles[x][y].B = 0
			board.tiles[x][y].A = 255
			board.tiles[x][y].Age = uint8(255)
		}
	}

	return board
}

var (
	tileAliveBlue  = Tile{Alive: true, R: 0, G: 0, B: 255, A: 255}
	tileAliveGreen = Tile{Alive: true, R: 0, G: 255, B: 0, A: 255}
	tileAliveRed   = Tile{Alive: true, R: 255, G: 0, B: 0, A: 255}
)

func (b *Board) addAcorn() {
	b.tiles[14][3] = tileAliveRed
	b.tiles[15][5] = tileAliveBlue
	b.tiles[16][2] = tileAliveGreen
	b.tiles[16][3] = tileAliveGreen
	b.tiles[16][6] = tileAliveRed
	b.tiles[16][7] = tileAliveBlue
	b.tiles[16][8] = tileAliveRed
}

func (b *Board) addShapes() {
	b.addAcorn()
}

func (b *Board) clear() {
	for x := range b.tiles {
		for y := range b.tiles[x] {
			b.tiles[x][y].Alive = false
			b.tiles[x][y].R = 0
			b.tiles[x][y].G = 0
			b.tiles[x][y].B = 0
			b.tiles[x][y].A = 255
			b.tiles[x][y].Age = uint8(255)
		}
	}
}

func (s *ConwayGameOfLife) swapBoards() {
	s.Current, s.Next = s.Next, s.Current
}

func (s *ConwayGameOfLife) toggleCell(x, y int32) {
	s.Current.tiles[x][y].Alive = !s.Current.tiles[x][y].Alive
	s.Current.tiles[x][y].R = s.SelectedR
	s.Current.tiles[x][y].G = s.SelectedG
	s.Current.tiles[x][y].B = s.SelectedB
	s.Current.tiles[x][y].A = s.SelectedA
}

func (s *ConwayGameOfLife) resetBoard() {
	s.BoardA.clear()
	s.BoardB.clear()
	s.Current = s.BoardA
	s.Next = s.BoardB
}

func (gs *ConwayGameOfLife) stepBoard() {
	gs.Next.clear()

	for x := range gs.Current.tiles {
		for y := range gs.Current.tiles[x] {
			tile := gs.Current.tiles[x][y]
			neighbors := mooreNeighbors(gs.Current, x, y)

			//We pass the color from one board to the other
			gs.Next.tiles[x][y].R = tile.R
			gs.Next.tiles[x][y].G = tile.G
			gs.Next.tiles[x][y].B = tile.B
			gs.Next.tiles[x][y].A = tile.A

			if neighbors < 2 && tile.Alive {
				gs.Next.tiles[x][y].Alive = false
				gs.Next.tiles[x][y].Age = 1
			} else if neighbors >= 2 && neighbors <= 3 && tile.Alive {
				gs.Next.tiles[x][y].Alive = true
				gs.Next.tiles[x][y].Age = 0 // Reset age if cell is alive
			} else if neighbors > 3 && tile.Alive {
				gs.Next.tiles[x][y].Alive = false
				gs.Next.tiles[x][y].Age = 1
			} else if neighbors == 3 && !tile.Alive {
				// New cell born, blend color from neighbors
				gs.Next.tiles[x][y].Alive = true
				r, g, b := blendNeighborColors(gs.Current, x, y)
				gs.Next.tiles[x][y].R = uint16(r)
				gs.Next.tiles[x][y].G = uint16(g)
				gs.Next.tiles[x][y].B = uint16(b)
				gs.Next.tiles[x][y].Age = 0
			} else if !tile.Alive && tile.Age < 255 {
				gs.Next.tiles[x][y].Age = tile.Age + 1
			}
		}
	}
}

func mooreNeighbors(board *Board, x, y int) (count int) {
	//North
	if y > 0 && board.tiles[x][y-1].Alive {
		count++
	}
	//South
	if y < len(board.tiles[x])-1 && board.tiles[x][y+1].Alive {
		count++
	}
	//West
	if x > 0 && board.tiles[x-1][y].Alive {
		count++
	}
	//East
	if x < len(board.tiles)-1 && board.tiles[x+1][y].Alive {
		count++
	}
	//NorthWest
	if x > 0 && y > 0 && board.tiles[x-1][y-1].Alive {
		count++
	}
	//NorthEast
	if x < len(board.tiles)-1 && y > 0 && board.tiles[x+1][y-1].Alive {
		count++
	}
	//SouthWest
	if x > 0 && y < len(board.tiles[x])-1 && board.tiles[x-1][y+1].Alive {
		count++
	}
	//SouthEast
	if x < len(board.tiles)-1 && y < len(board.tiles[x])-1 && board.tiles[x+1][y+1].Alive {
		count++
	}
	return count
}

func blendNeighborColors(board *Board, x, y int) (r, g, b uint8) {
	count := 0

	rr := 0
	gg := 0
	bb := 0

	//North
	if y > 0 && board.tiles[x][y-1].Alive {
		rr += int(board.tiles[x][y-1].R)
		gg += int(board.tiles[x][y-1].G)
		bb += int(board.tiles[x][y-1].B)
		count++
	}
	//South
	if y < len(board.tiles[x])-1 && board.tiles[x][y+1].Alive {
		rr += int(board.tiles[x][y+1].R)
		gg += int(board.tiles[x][y+1].G)
		bb += int(board.tiles[x][y+1].B)
		count++
	}
	//West
	if x > 0 && board.tiles[x-1][y].Alive {
		rr += int(board.tiles[x-1][y].R)
		gg += int(board.tiles[x-1][y].G)
		bb += int(board.tiles[x-1][y].B)
		count++
	}
	//East
	if x < len(board.tiles)-1 && board.tiles[x+1][y].Alive {
		rr += int(board.tiles[x+1][y].R)
		gg += int(board.tiles[x+1][y].G)
		bb += int(board.tiles[x+1][y].B)
		count++
	}
	//NorthWest
	if x > 0 && y > 0 && board.tiles[x-1][y-1].Alive {
		rr += int(board.tiles[x-1][y-1].R)
		gg += int(board.tiles[x-1][y-1].G)
		bb += int(board.tiles[x-1][y-1].B)
		count++
	}
	//NorthEast
	if x < len(board.tiles)-1 && y > 0 && board.tiles[x+1][y-1].Alive {
		rr += int(board.tiles[x+1][y-1].R)
		gg += int(board.tiles[x+1][y-1].G)
		bb += int(board.tiles[x+1][y-1].B)
		count++
	}
	//SouthWest
	if x > 0 && y < len(board.tiles[x])-1 && board.tiles[x-1][y+1].Alive {
		rr += int(board.tiles[x-1][y+1].R)
		gg += int(board.tiles[x-1][y+1].G)
		bb += int(board.tiles[x-1][y+1].B)
		count++
	}
	//SouthEast
	if x < len(board.tiles)-1 && y < len(board.tiles[x])-1 && board.tiles[x+1][y+1].Alive {
		rr += int(board.tiles[x+1][y+1].R)
		gg += int(board.tiles[x+1][y+1].G)
		bb += int(board.tiles[x+1][y+1].B)
		count++
	}

	return uint8(rr / count), uint8(gg / count), uint8(bb / count)
}

func (s *ConwayGameOfLife) ProcessFrame(deltaTime float32, frameBuffer *FrameBuffer) {
	s.Lapsed += deltaTime

	if s.Lapsed >= s.Step {
		s.stepBoard()
		s.swapBoards()
		s.Lapsed = 0
	}

	s.drawBoard(frameBuffer)
}

func (s *ConwayGameOfLife) drawBoard(frameBuffer *FrameBuffer) {
	for x, column := range s.Current.tiles {
		for y, tile := range column {
			rect := rl.Rectangle{
				X:      float32(int32(x) * s.CellSize),
				Y:      float32(int32(y) * s.CellSize),
				Width:  float32(s.CellSize),
				Height: float32(s.CellSize),
			}

			tileColor := getTileColor(tile, &s.Settings)
			drawRectangle(frameBuffer, rect, tileColor)
		}
	}
}

func drawRectangle(frameBuffer *FrameBuffer, rect rl.Rectangle, color uint16) {
	startX := int(rect.X)
	startY := int(rect.Y)
	endX := int(rect.X + rect.Width)
	endY := int(rect.Y + rect.Height)

	for y := startY; y < endY; y++ {
		for x := startX; x < endX; x++ {
			if x >= 0 && x < frameBuffer.Width && y >= 0 && y < frameBuffer.Height {
				frameBuffer.Pixels[y*frameBuffer.Width+x] = color
			}
		}
	}
}

func getTileColor(tile Tile, settings *RenderSettings) uint16 {
	r := tile.R
	g := tile.G
	b := tile.B
	a := tile.A

	// Only apply fading if the tile is dead
	if !tile.Alive {
		// Apply opacity fade if enabled
		if settings.FadeOpacity && tile.Age < uint8(settings.FadeLength) {
			// Apply opacity fade based on the tile's age
			a = uint16(255 - (tile.Age * 255 / uint8(settings.FadeLength)))
		} else {
			a = 0
		}
	}

	r8 := uint8((r * a) / 255)
	g8 := uint8((g * a) / 255)
	b8 := uint8((b * a) / 255)
	return convertToRGB565(r8, g8, b8)
}
