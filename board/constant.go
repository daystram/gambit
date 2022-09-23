package board

import (
	"log"
	"time"

	"github.com/daystram/gambit/position"
)

const (
	Width      = position.MaxComponentScalar
	Height     = position.MaxComponentScalar
	TotalCells = Width * Height
)

var (
	DefaultStartingPositionFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	maskCell = [TotalCells]bitmap{}
	maskCol  = [Width]bitmap{
		0x_01_01_01_01_01_01_01_01,
		0x_02_02_02_02_02_02_02_02,
		0x_04_04_04_04_04_04_04_04,
		0x_08_08_08_08_08_08_08_08,
		0x_10_10_10_10_10_10_10_10,
		0x_20_20_20_20_20_20_20_20,
		0x_40_40_40_40_40_40_40_40,
		0x_80_80_80_80_80_80_80_80,
	}
	maskRow = [Height]bitmap{
		0x_00_00_00_00_00_00_00_FF,
		0x_00_00_00_00_00_00_FF_00,
		0x_00_00_00_00_00_FF_00_00,
		0x_00_00_00_00_FF_00_00_00,
		0x_00_00_00_FF_00_00_00_00,
		0x_00_00_FF_00_00_00_00_00,
		0x_00_FF_00_00_00_00_00_00,
		0x_FF_00_00_00_00_00_00_00,
	}
	maskDia    = [TotalCells]bitmap{}
	maskADia   = [TotalCells]bitmap{}
	maskKnight = [TotalCells]bitmap{}
	maskKing   = [TotalCells]bitmap{}

	maskCastling = [4 + 1]bitmap{}
	posCastling  = [4 + 1][6 + 1][2]position.Pos{
		CastleDirectionWhiteRight: {
			PieceKing: {4, 6},
			PieceRook: {7, 5},
		},
		CastleDirectionWhiteLeft: {
			PieceKing: {4, 2},
			PieceRook: {0, 3}},
		CastleDirectionBlackRight: {
			PieceKing: {4 + 7*Width, 6 + 7*Width},
			PieceRook: {7 + 7*Width, 5 + 7*Width},
		},
		CastleDirectionBlackLeft: {
			PieceKing: {4 + 7*Width, 2 + 7*Width},
			PieceRook: {0 + 7*Width, 3 + 7*Width}},
	}

	maskCastleRights = [5]CastleRights{
		CastleDirectionWhiteRight: 0b1000,
		CastleDirectionWhiteLeft:  0b0100,
		CastleDirectionBlackRight: 0b0010,
		CastleDirectionBlackLeft:  0b0001,
	}
)

func init() {
	start := time.Now()
	for pos := position.Pos(0); pos < TotalCells; pos++ {
		maskCell[pos] = 1 << pos
	}

	for pos := position.Pos(0); pos < TotalCells; pos++ {
		mask := bitmap(0)
		x, y := pos%Width, pos/Width
		x, y = x-min(x, y), y-min(x, y)
		for x < Width && y < Height {
			mask |= bitmap(1 << (y*Width + x))
			x++
			y++
		}
		maskDia[pos] = mask
	}

	for pos := position.Pos(0); pos < TotalCells; pos++ {
		mask := bitmap(0)
		x, y := pos%Width, pos/Width
		x, y = x-min(x, Height-y-1), y+min(x, Height-y-1)
		for x < Width && y >= 0 {
			mask |= bitmap(1 << (y*Width + x))
			x++
			y--
		}
		maskADia[pos] = mask
	}

	for pos := position.Pos(0); pos < TotalCells; pos++ {
		cell := maskCell[pos]
		mask := bitmap(0)
		mask |= ShiftN(ShiftN(ShiftE(cell &^ maskRow[7] &^ maskRow[6] &^ maskCol[7])))
		mask |= ShiftN(ShiftN(ShiftW(cell &^ maskRow[7] &^ maskRow[6] &^ maskCol[0])))
		mask |= ShiftS(ShiftS(ShiftE(cell &^ maskRow[0] &^ maskRow[1] &^ maskCol[7])))
		mask |= ShiftS(ShiftS(ShiftW(cell &^ maskRow[0] &^ maskRow[1] &^ maskCol[0])))
		mask |= ShiftE(ShiftE(ShiftN(cell &^ maskCol[7] &^ maskCol[6] &^ maskRow[7])))
		mask |= ShiftE(ShiftE(ShiftS(cell &^ maskCol[7] &^ maskCol[6] &^ maskRow[0])))
		mask |= ShiftW(ShiftW(ShiftN(cell &^ maskCol[0] &^ maskCol[1] &^ maskRow[7])))
		mask |= ShiftW(ShiftW(ShiftS(cell &^ maskCol[0] &^ maskCol[1] &^ maskRow[0])))
		maskKnight[pos] = mask
	}

	for pos := position.Pos(0); pos < TotalCells; pos++ {
		cell := maskCell[pos]
		mask := bitmap(0)
		mask |= ShiftN(cell &^ maskRow[7])
		mask |= ShiftNE(cell &^ maskRow[7] &^ maskCol[7])
		mask |= ShiftE(cell &^ maskCol[7])
		mask |= ShiftSE(cell &^ maskRow[0] &^ maskCol[7])
		mask |= ShiftS(cell &^ maskRow[0])
		mask |= ShiftSW(cell &^ maskRow[0] &^ maskCol[0])
		mask |= ShiftW(cell &^ maskCol[0])
		mask |= ShiftNW(cell &^ maskRow[7] &^ maskCol[0])
		maskKing[pos] = mask
	}

	maskCastling = [5]bitmap{
		CastleDirectionWhiteRight: maskRow[0] & (maskCol[5] | maskCol[6]),
		CastleDirectionWhiteLeft:  maskRow[0] & (maskCol[1] | maskCol[2] | maskCol[3]),
		CastleDirectionBlackRight: maskRow[7] & (maskCol[5] | maskCol[6]),
		CastleDirectionBlackLeft:  maskRow[7] & (maskCol[1] | maskCol[2] | maskCol[3]),
	}

	log.Printf("init board lookup: %s elapsed\n", time.Since(start))
}
