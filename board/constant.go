package board

import (
	"fmt"
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
	maskCell   [TotalCells + 1]bitmap
	maskDia    [TotalCells + 1]bitmap
	maskADia   [TotalCells + 1]bitmap
	maskKnight [TotalCells + 1]bitmap
	maskKing   [TotalCells + 1]bitmap

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

	zobristConstantPiece        [2 + 1][6 + 1][TotalCells + 1]uint64
	zobristConstantEnPassant    [TotalCells + 1]uint64
	zobristConstantCastleRights [16]uint64
	zobristConstantSideWhite    uint64

	// Seeds taken from https://github.com/official-stockfish/Stockfish.
	magicSeeds  = []uint64{8977, 44560, 54343, 38998, 5731, 95205, 104912, 17020}
	magicBishop [TotalCells]*Magic
	magicRook   [TotalCells]*Magic

	scoreMaterial = [6 + 1]int32{
		PiecePawn:   100,
		PieceKnight: 320,
		PieceBishop: 350,
		PieceRook:   500,
		PieceQueen:  900,
	}
	// PST table taken from https://www.chessprogramming.org/Simplified_Evaluation_Function
	// TODO: tapered/transitioning PST
	// TODO: create tuner
	scorePosition = [6 + 1][TotalCells]int32{
		PiecePawn: {
			0, 0, 0, 0, 0, 0, 0, 0,
			50, 50, 50, 50, 50, 50, 50, 50,
			10, 10, 20, 30, 30, 20, 10, 10,
			5, 5, 10, 25, 25, 10, 5, 5,
			0, 0, 0, 20, 20, 0, 0, 0,
			5, -5, -10, 0, 0, -10, -5, 5,
			5, 10, 10, -20, -20, 10, 10, 5,
			0, 0, 0, 0, 0, 0, 0, 0,
		},
		PieceKnight: {
			-50, -40, -30, -30, -30, -30, -40, -50,
			-40, -20, 0, 0, 0, 0, -20, -40,
			-30, 0, 10, 15, 15, 10, 0, -30,
			-30, 5, 15, 20, 20, 15, 5, -30,
			-30, 0, 15, 20, 20, 15, 0, -30,
			-30, 5, 10, 15, 15, 10, 5, -30,
			-40, -20, 0, 5, 5, 0, -20, -40,
			-50, -40, -30, -30, -30, -30, -40, -50,
		},
		PieceBishop: {
			-20, -10, -10, -10, -10, -10, -10, -20,
			-10, 0, 0, 0, 0, 0, 0, -10,
			-10, 0, 5, 10, 10, 5, 0, -10,
			-10, 5, 5, 10, 10, 5, 5, -10,
			-10, 0, 10, 10, 10, 10, 0, -10,
			-10, 10, 10, 10, 10, 10, 10, -10,
			-10, 5, 0, 0, 0, 0, 5, -10,
			-20, -10, -10, -10, -10, -10, -10, -20,
		},
		PieceRook: {
			0, 0, 0, 0, 0, 0, 0, 0,
			5, 10, 10, 10, 10, 10, 10, 5,
			-5, 0, 0, 0, 0, 0, 0, -5,
			-5, 0, 0, 0, 0, 0, 0, -5,
			-5, 0, 0, 0, 0, 0, 0, -5,
			-5, 0, 0, 0, 0, 0, 0, -5,
			-5, 0, 0, 0, 0, 0, 0, -5,
			0, 0, 0, 5, 5, 0, 0, 0,
		},
		PieceQueen: {
			-20, -10, -10, -5, -5, -10, -10, -20,
			-10, 0, 0, 0, 0, 0, 0, -10,
			-10, 0, 5, 5, 5, 5, 0, -10,
			-5, 0, 5, 5, 5, 5, 0, -5,
			0, 0, 5, 5, 5, 5, 0, -5,
			-10, 5, 5, 5, 5, 5, 0, -10,
			-10, 0, 5, 0, 0, 0, 0, -10,
			-20, -10, -10, -5, -5, -10, -10, -20,
		},
		PieceKing: {
			-30, -40, -40, -50, -50, -40, -40, -30,
			-30, -40, -40, -50, -50, -40, -40, -30,
			-30, -40, -40, -50, -50, -40, -40, -30,
			-30, -40, -40, -50, -50, -40, -40, -30,
			-20, -30, -30, -40, -40, -30, -30, -20,
			-10, -20, -20, -20, -20, -20, -20, -10,
			20, 20, 0, 0, 0, 0, 20, 20,
			20, 30, 10, 0, 0, 10, 30, 20,
		},
	}
	scorePositionMap = [2 + 1][TotalCells]position.Pos{
		SideWhite: {
			position.A8, position.B8, position.C8, position.D8, position.E8, position.F8, position.G8, position.H8,
			position.A7, position.B7, position.C7, position.D7, position.E7, position.F7, position.G7, position.H7,
			position.A6, position.B6, position.C6, position.D6, position.E6, position.F6, position.G6, position.H6,
			position.A5, position.B5, position.C5, position.D5, position.E5, position.F5, position.G5, position.H5,
			position.A4, position.B4, position.C4, position.D4, position.E4, position.F4, position.G4, position.H4,
			position.A3, position.B3, position.C3, position.D3, position.E3, position.F3, position.G3, position.H3,
			position.A2, position.B2, position.C2, position.D2, position.E2, position.F2, position.G2, position.H2,
			position.A1, position.B1, position.C1, position.D1, position.E1, position.F1, position.G1, position.H1,
		},
		SideBlack: { // horizontal flip of White
			position.A1, position.B1, position.C1, position.D1, position.E1, position.F1, position.G1, position.H1,
			position.A2, position.B2, position.C2, position.D2, position.E2, position.F2, position.G2, position.H2,
			position.A3, position.B3, position.C3, position.D3, position.E3, position.F3, position.G3, position.H3,
			position.A4, position.B4, position.C4, position.D4, position.E4, position.F4, position.G4, position.H4,
			position.A5, position.B5, position.C5, position.D5, position.E5, position.F5, position.G5, position.H5,
			position.A6, position.B6, position.C6, position.D6, position.E6, position.F6, position.G6, position.H6,
			position.A7, position.B7, position.C7, position.D7, position.E7, position.F7, position.G7, position.H7,
			position.A8, position.B8, position.C8, position.D8, position.E8, position.F8, position.G8, position.H8,
		},
	}
)

func init() {
	fmt.Print("Initializing lookup boards... ")
	initMask()
	initZobrist()
	initMagic(PieceBishop)
	initMagic(PieceRook)
	fmt.Println("Done")
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
		CastleDirectionWhiteRight: maskCell[position.F1] | maskCell[position.G1],
		CastleDirectionWhiteLeft:  maskCell[position.B1] | maskCell[position.C1] | maskCell[position.D1],
		CastleDirectionBlackRight: maskCell[position.F8] | maskCell[position.G8],
		CastleDirectionBlackLeft:  maskCell[position.B8] | maskCell[position.C8] | maskCell[position.D8],
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

func initMagic(p Piece) {
	var magics *[TotalCells]*Magic
	var genMask func(position.Pos) bitmap
	var genMovesBM func(position.Pos, bitmap) bitmap
	switch p {
	case PieceBishop:
		magics = &magicBishop
		genMask = func(pos position.Pos) bitmap {
			edge := maskCol[position.FileA] | maskCol[position.FileH] | maskRow[position.Rank1] | maskRow[position.Rank8]
			return HitDiagonals(pos, 0) &^ edge
		}
		genMovesBM = HitDiagonals
	case PieceRook:
		magics = &magicRook
		genMask = func(pos position.Pos) bitmap {
			var edge bitmap
			if pos.X() != position.FileA {
				edge |= maskCol[position.FileA]
			}
			if pos.X() != position.FileH {
				edge |= maskCol[position.FileH]
			}
			if pos.Y() != position.Rank1 {
				edge |= maskRow[position.Rank1]
			}
			if pos.Y() != position.Rank8 {
				edge |= maskRow[position.Rank8]
			}
			return HitLaterals(pos, 0) &^ edge
		}
		genMovesBM = HitLaterals
	default:
		return
	}

	r := NewPseudoRand()
	for pos := position.A1; pos <= position.H8; pos++ {
		m := &Magic{}
		m.Mask = genMask(pos) &^ maskCell[pos]
		m.Shift = 64 - m.Mask.BitCount()

		var size int
		var blocker bitmap
		var blockers, attacks [4096]bitmap
		for size, blocker = 0, bitmap(0); size == 0 || blocker != 0; size++ {
			blockers[size] = blocker
			attacks[size] = genMovesBM(pos, blocker)
			blocker = (blocker - m.Mask) & m.Mask
		}
		r.Seed(magicSeeds[pos.Y()])
		for i := 0; i < size; {
			m.Magic = bitmap(r.SparseUint64())
			m.Attacks = &[4096]bitmap{}
			for i = 0; i < size; i++ {
				idx := m.GetIndex(blockers[i])
				if (*m.Attacks)[idx] != 0 && (*m.Attacks)[idx] != attacks[i] {
					break
				}
				(*m.Attacks)[idx] = attacks[i]
			}
		}
		(*magics)[pos] = m
	}
}
