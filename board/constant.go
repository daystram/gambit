package board

import (
	"math/rand"

	"github.com/daystram/gambit/position"
)

const (
	Width      = position.MaxComponentScalar
	Height     = position.MaxComponentScalar
	TotalCells = Width * Height
)

var (
	DefaultStartingPositionFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	maskCol = [Width]bitmap{
		position.FileA: 0x_01_01_01_01_01_01_01_01,
		position.FileB: 0x_02_02_02_02_02_02_02_02,
		position.FileC: 0x_04_04_04_04_04_04_04_04,
		position.FileD: 0x_08_08_08_08_08_08_08_08,
		position.FileE: 0x_10_10_10_10_10_10_10_10,
		position.FileF: 0x_20_20_20_20_20_20_20_20,
		position.FileG: 0x_40_40_40_40_40_40_40_40,
		position.FileH: 0x_80_80_80_80_80_80_80_80,
	}
	maskRow = [Height]bitmap{
		position.Rank1: 0x_00_00_00_00_00_00_00_FF,
		position.Rank2: 0x_00_00_00_00_00_00_FF_00,
		position.Rank3: 0x_00_00_00_00_00_FF_00_00,
		position.Rank4: 0x_00_00_00_00_FF_00_00_00,
		position.Rank5: 0x_00_00_00_FF_00_00_00_00,
		position.Rank6: 0x_00_00_FF_00_00_00_00_00,
		position.Rank7: 0x_00_FF_00_00_00_00_00_00,
		position.Rank8: 0x_FF_00_00_00_00_00_00_00,
	}
	maskCell   [TotalCells]bitmap
	maskDia    [TotalCells]bitmap
	maskADia   [TotalCells]bitmap
	maskKnight [TotalCells]bitmap
	maskKing   [TotalCells]bitmap

	maskCastling = [4 + 1]bitmap{}
	posCastling  = [4 + 1][6 + 1][2]position.Pos{
		CastleDirectionWhiteRight: {
			PieceKing: {position.E1, position.G1},
			PieceRook: {position.H1, position.F1},
		},
		CastleDirectionWhiteLeft: {
			PieceKing: {position.E1, position.C1},
			PieceRook: {position.A1, position.D1},
		},
		CastleDirectionBlackRight: {
			PieceKing: {position.E8, position.G8},
			PieceRook: {position.H8, position.F8},
		},
		CastleDirectionBlackLeft: {
			PieceKing: {position.E8, position.C8},
			PieceRook: {position.A8, position.D8},
		},
	}

	maskCastleRights = [5]CastleRights{
		CastleDirectionWhiteRight: 0b1000,
		CastleDirectionWhiteLeft:  0b0100,
		CastleDirectionBlackRight: 0b0010,
		CastleDirectionBlackLeft:  0b0001,
	}

	zobristConstantPiece        [2 + 1][6 + 1][64]uint64
	zobristConstantEnPassant    [64]uint64
	zobristConstantCastleRights [16]uint64
	zobristConstantSideWhite    uint64

	// magicBishopAttacks [TotalCells][1]bitmap
	magicBishopMask [TotalCells]bitmap
	// magicBishopNumber  [TotalCells]bitmap
	// magicRookAttacks   [TotalCells][1]bitmap
	magicRookMask [TotalCells]bitmap
	// magicRookNumber    [TotalCells]bitmap

	materialPieceValue = [6 + 1]uint32{
		PiecePawn:   100,
		PieceKnight: 320,
		PieceBishop: 350,
		PieceRook:   500,
		PieceQueen:  900,
	}
)

func init() {
	initMask()
	initZobrist()
	initMagic()
}

func initMask() {
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
}

func initZobrist() {
	r := rand.New(rand.NewSource(7))
	for _, s := range []Side{SideWhite, SideBlack} {
		for _, p := range []Piece{PiecePawn, PieceBishop, PieceKnight, PieceRook, PieceQueen, PieceKing} {
			for pos := position.Pos(0); pos < TotalCells; pos++ {
				zobristConstantPiece[s][p][pos] = r.Uint64()
			}
		}
	}
	for pos := position.Pos(0); pos < TotalCells; pos++ {
		zobristConstantEnPassant[pos] = r.Uint64()
	}
	for pos := position.Pos(0); pos < 16; pos++ {
		zobristConstantCastleRights[pos] = r.Uint64()
	}
	zobristConstantSideWhite = r.Uint64()
}

func initMagic() {
	// Bishop
	for pos := position.Pos(0); pos < TotalCells; pos++ {
		magicBishopMask[pos] = (maskDia[pos] | maskADia[pos]) &^ maskCell[pos]
	}

	// Rook
	for pos := position.Pos(0); pos < TotalCells; pos++ {
		magicRookMask[pos] = (maskRow[pos/Width] | maskCol[pos%Width]) &^ maskCell[pos]
	}

	// TODO: try using magics
}
