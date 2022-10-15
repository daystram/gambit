package board

import (
	"fmt"
	"math/rand"
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
	magicBishop [TotalCells]Magic
	magicRook   [TotalCells]Magic

	scoreMaterial = [6 + 1]int16{
		PiecePawn:   100,
		PieceKnight: 320,
		PieceBishop: 350,
		PieceRook:   500,
		PieceQueen:  900,
	}

	phaseConstant = [6 + 1]int8{
		PiecePawn:   0,
		PieceKnight: 1,
		PieceBishop: 1,
		PieceRook:   2,
		PieceQueen:  4,
	}
	PhaseTotal = 16*phaseConstant[PiecePawn] +
		4*phaseConstant[PieceKnight] +
		4*phaseConstant[PieceBishop] +
		4*phaseConstant[PieceRook] +
		2*phaseConstant[PieceQueen]

	// PST midgame table taken from http://www.talkchess.com/forum3/viewtopic.php?f=2&t=68311&start=19
	// TODO: create tuner
	scorePositionMG = [6 + 1][TotalCells]int16{
		PiecePawn: {
			0, 0, 0, 0, 0, 0, 0, 0,
			98, 134, 61, 95, 68, 126, 34, -11,
			-6, 7, 26, 31, 65, 56, 25, -20,
			-14, 13, 6, 21, 23, 12, 17, -23,
			-27, -2, -5, 12, 17, 6, 10, -25,
			-26, -4, -4, -10, 3, 3, 33, -12,
			-35, -1, -20, -23, -15, 24, 38, -22,
			0, 0, 0, 0, 0, 0, 0, 0,
		},
		PieceKnight: {
			-167, -89, -34, -49, 61, -97, -15, -107,
			-73, -41, 72, 36, 23, 62, 7, -17,
			-47, 60, 37, 65, 84, 129, 73, 44,
			-9, 17, 19, 53, 37, 69, 18, 22,
			-13, 4, 16, 13, 28, 19, 21, -8,
			-23, -9, 12, 10, 19, 17, 25, -16,
			-29, -53, -12, -3, -1, 18, -14, -19,
			-105, -21, -58, -33, -17, -28, -19, -23,
		},
		PieceBishop: {
			-29, 4, -82, -37, -25, -42, 7, -8,
			-26, 16, -18, -13, 30, 59, 18, -47,
			-16, 37, 43, 40, 35, 50, 37, -2,
			-4, 5, 19, 50, 37, 37, 7, -2,
			-6, 13, 13, 26, 34, 12, 10, 4,
			0, 15, 15, 15, 14, 27, 18, 10,
			4, 15, 16, 0, 7, 21, 33, 1,
			-33, -3, -14, -21, -13, -12, -39, -21,
		},
		PieceRook: {
			32, 42, 32, 51, 63, 9, 31, 43,
			27, 32, 58, 62, 80, 67, 26, 44,
			-5, 19, 26, 36, 17, 45, 61, 16,
			-24, -11, 7, 26, 24, 35, -8, -20,
			-36, -26, -12, -1, 9, -7, 6, -23,
			-45, -25, -16, -17, 3, 0, -5, -33,
			-44, -16, -20, -9, -1, 11, -6, -71,
			-19, -13, 1, 17, 16, 7, -37, -26,
		},
		PieceQueen: {
			-28, 0, 29, 12, 59, 44, 43, 45,
			-24, -39, -5, 1, -16, 57, 28, 54,
			-13, -17, 7, 8, 29, 56, 47, 57,
			-27, -27, -16, -16, -1, 17, -2, 1,
			-9, -26, -9, -10, -2, -4, 3, -3,
			-14, 2, -11, -2, -5, 2, 14, 5,
			-35, -8, 11, 2, 8, 15, -3, 1,
			-1, -18, -9, 10, -15, -25, -31, -50,
		},
		PieceKing: {
			-65, 23, 16, -15, -56, -34, 2, 13,
			29, -1, -20, -7, -8, -4, -38, -29,
			-9, 24, 2, -16, -20, 6, 22, -22,
			-17, -20, -12, -27, -30, -25, -14, -36,
			-49, -1, -27, -39, -46, -44, -33, -51,
			-14, -14, -22, -46, -44, -30, -15, -27,
			1, 7, -8, -64, -43, -16, 9, 8,
			-15, 36, 12, -54, 8, -28, 24, 14,
		},
	}

	// PST endgame table taken from http://www.talkchess.com/forum3/viewtopic.php?f=2&t=68311&start=19
	scorePositionEG = [6 + 1][TotalCells]int16{
		PiecePawn: {
			0, 0, 0, 0, 0, 0, 0, 0,
			178, 173, 158, 134, 147, 132, 165, 187,
			94, 100, 85, 67, 56, 53, 82, 84,
			32, 24, 13, 5, -2, 4, 17, 17,
			13, 9, -3, -7, -7, -8, 3, -1,
			4, 7, -6, 1, 0, -5, -1, -8,
			13, 8, 8, 10, 13, 0, 2, -7,
			0, 0, 0, 0, 0, 0, 0, 0,
		},
		PieceKnight: {
			-58, -38, -13, -28, -31, -27, -63, -99,
			-25, -8, -25, -2, -9, -25, -24, -52,
			-24, -20, 10, 9, -1, -9, -19, -41,
			-17, 3, 22, 22, 22, 11, 8, -18,
			-18, -6, 16, 25, 16, 17, 4, -18,
			-23, -3, -1, 15, 10, -3, -20, -22,
			-42, -20, -10, -5, -2, -20, -23, -44,
			-29, -51, -23, -15, -22, -18, -50, -64,
		},
		PieceBishop: {
			-14, -21, -11, -8, -7, -9, -17, -24,
			-8, -4, 7, -12, -3, -13, -4, -14,
			2, -8, 0, -1, -2, 6, 0, 4,
			-3, 9, 12, 9, 14, 10, 3, 2,
			-6, 3, 13, 19, 7, 10, -3, -9,
			-12, -3, 8, 10, 13, 3, -7, -15,
			-14, -18, -7, -1, 4, -9, -15, -27,
			-23, -9, -23, -5, -9, -16, -5, -17,
		},
		PieceRook: {
			13, 10, 18, 15, 12, 12, 8, 5,
			11, 13, 13, 11, -3, 3, 8, 3,
			7, 7, 7, 5, 4, -3, -5, -3,
			4, 3, 13, 1, 2, 1, -1, 2,
			3, 5, 8, 4, -5, -6, -8, -11,
			-4, 0, -5, -1, -7, -12, -8, -16,
			-6, -6, 0, 2, -9, -9, -11, -3,
			-9, 2, 3, -1, -5, -13, 4, -20,
		},
		PieceQueen: {
			-9, 22, 22, 27, 27, 19, 10, 20,
			-17, 20, 32, 41, 58, 25, 30, 0,
			-20, 6, 9, 49, 47, 35, 19, 9,
			3, 22, 24, 45, 57, 40, 57, 36,
			-18, 28, 19, 47, 31, 34, 39, 23,
			-16, -27, 15, 6, 9, 17, 10, 5,
			-22, -23, -30, -16, -16, -23, -36, -32,
			-33, -28, -22, -43, -5, -32, -20, -41,
		},
		PieceKing: {
			-74, -35, -18, -18, -11, 15, 4, -17,
			-12, 17, 14, 17, 17, 38, 23, 11,
			10, 17, 23, 15, 20, 45, 44, 13,
			-8, 22, 24, 27, 26, 33, 26, 3,
			-18, -4, 21, 24, 27, 23, 9, -11,
			-19, -3, 11, 21, 23, 16, 7, -9,
			-27, -11, 4, 13, 14, 4, -5, -17,
			-53, -34, -21, -11, -28, -14, -24, -43,
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
	start := time.Now()
	fmt.Print("Initializing lookup boards... ")
	initMask()
	initZobrist()
	initMagic(PieceBishop)
	initMagic(PieceRook)
	fmt.Printf("Done (%.3fs)\n", time.Since(start).Seconds())
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
	var magics *[TotalCells]Magic
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
		m := Magic{}
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
			m.Magic = 0
			// ensure sparse magic, as seen on https://github.com/official-stockfish/Stockfish
			for ((m.Magic * m.Mask) >> 56).BitCount() < 6 {
				m.Magic = bitmap(r.SparseUint64())
			}
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
