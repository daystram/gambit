package engine

import (
	"github.com/daystram/gambit/board"
	"github.com/daystram/gambit/position"
)

var (
	// PST table taken from https://www.chessprogramming.org/Simplified_Evaluation_Function
	// TODO: tapered/transitioning PST
	// TODO: create tuner
	scorePiecePosition = [6 + 1][64]int32{
		board.PiecePawn: {
			0, 0, 0, 0, 0, 0, 0, 0,
			50, 50, 50, 50, 50, 50, 50, 50,
			10, 10, 20, 30, 30, 20, 10, 10,
			5, 5, 10, 25, 25, 10, 5, 5,
			0, 0, 0, 20, 20, 0, 0, 0,
			5, -5, -10, 0, 0, -10, -5, 5,
			5, 10, 10, -20, -20, 10, 10, 5,
			0, 0, 0, 0, 0, 0, 0, 0,
		},
		board.PieceKnight: {
			-50, -40, -30, -30, -30, -30, -40, -50,
			-40, -20, 0, 0, 0, 0, -20, -40,
			-30, 0, 10, 15, 15, 10, 0, -30,
			-30, 5, 15, 20, 20, 15, 5, -30,
			-30, 0, 15, 20, 20, 15, 0, -30,
			-30, 5, 10, 15, 15, 10, 5, -30,
			-40, -20, 0, 5, 5, 0, -20, -40,
			-50, -40, -30, -30, -30, -30, -40, -50,
		},
		board.PieceBishop: {
			-20, -10, -10, -10, -10, -10, -10, -20,
			-10, 0, 0, 0, 0, 0, 0, -10,
			-10, 0, 5, 10, 10, 5, 0, -10,
			-10, 5, 5, 10, 10, 5, 5, -10,
			-10, 0, 10, 10, 10, 10, 0, -10,
			-10, 10, 10, 10, 10, 10, 10, -10,
			-10, 5, 0, 0, 0, 0, 5, -10,
			-20, -10, -10, -10, -10, -10, -10, -20,
		},
		board.PieceRook: {
			0, 0, 0, 0, 0, 0, 0, 0,
			5, 10, 10, 10, 10, 10, 10, 5,
			-5, 0, 0, 0, 0, 0, 0, -5,
			-5, 0, 0, 0, 0, 0, 0, -5,
			-5, 0, 0, 0, 0, 0, 0, -5,
			-5, 0, 0, 0, 0, 0, 0, -5,
			-5, 0, 0, 0, 0, 0, 0, -5,
			0, 0, 0, 5, 5, 0, 0, 0,
		},
		board.PieceQueen: {
			-20, -10, -10, -5, -5, -10, -10, -20,
			-10, 0, 0, 0, 0, 0, 0, -10,
			-10, 0, 5, 5, 5, 5, 0, -10,
			-5, 0, 5, 5, 5, 5, 0, -5,
			0, 0, 5, 5, 5, 5, 0, -5,
			-10, 5, 5, 5, 5, 5, 0, -10,
			-10, 0, 5, 0, 0, 0, 0, -10,
			-20, -10, -10, -5, -5, -10, -10, -20,
		},
		board.PieceKing: {
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
	scorePiecePositionMap = [2 + 1][64]position.Pos{
		board.SideWhite: {
			position.A8, position.B8, position.C8, position.D8, position.E8, position.F8, position.G8, position.H8,
			position.A7, position.B7, position.C7, position.D7, position.E7, position.F7, position.G7, position.H7,
			position.A6, position.B6, position.C6, position.D6, position.E6, position.F6, position.G6, position.H6,
			position.A5, position.B5, position.C5, position.D5, position.E5, position.F5, position.G5, position.H5,
			position.A4, position.B4, position.C4, position.D4, position.E4, position.F4, position.G4, position.H4,
			position.A3, position.B3, position.C3, position.D3, position.E3, position.F3, position.G3, position.H3,
			position.A2, position.B2, position.C2, position.D2, position.E2, position.F2, position.G2, position.H2,
			position.A1, position.B1, position.C1, position.D1, position.E1, position.F1, position.G1, position.H1,
		},
		board.SideBlack: { // horizontal flip of White
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

	scoreTempoBonus int32 = 30

	offsetPV     uint8 = 255
	offsetMVVLVA uint8 = offsetPV - 64
	scoreMVVLVA        = [6 + 1][6 + 1]uint8{
		//                     P   N   B   R   Q
		board.PiecePawn:   {0, 15, 25, 35, 45, 55},
		board.PieceKnight: {0, 14, 24, 34, 44, 54},
		board.PieceBishop: {0, 13, 23, 33, 43, 53},
		board.PieceRook:   {0, 12, 22, 32, 42, 52},
		board.PieceQueen:  {0, 11, 21, 31, 41, 51},
		board.PieceKing:   {0, 10, 20, 30, 40, 50},
	}
	scoreKiller uint8 = 10
)

func (e *Engine) scoreMoves(b *board.Board, pv *board.Move, mvs *[]*board.Move) {
	for i, mv := range *mvs {
		var score uint8
		if mv.Equals(pv) {
			score = offsetPV
		} else if mv.IsCapture {
			capturedPiece, _ := b.GetSideAndPieces(mv.To)
			score = offsetMVVLVA + scoreMVVLVA[mv.Piece][capturedPiece]
		} else {
			for i, killer := range e.killers[b.Ply()] {
				if mv.Equals(killer) {
					score = offsetMVVLVA - uint8(i+1)*scoreKiller
					break
				}
			}
		}
		(*mvs)[i].Score = score
	}
}

func (e *Engine) sortMoves(mvs *[]*board.Move, index int) {
	bestIndex, bestScore := index, uint8(0)
	for i := index; i < len(*mvs); i++ {
		mv := (*mvs)[i]
		if mv.Score > bestScore {
			bestIndex = i
			bestScore = mv.Score
		}
	}
	temp := (*mvs)[index]
	(*mvs)[index] = (*mvs)[bestIndex]
	(*mvs)[bestIndex] = temp
}

// evaluate returns the score evaluated from the given board.
// The score is positive relative to the currently playing side.
func (e *Engine) evaluate(b *board.Board) int32 {
	ourTurn := b.Turn()

	// TODO: check game state here?

	// Material heuristic
	var totalScorePiece int32
	scorePieceWhite, scorePieceBlack := b.GetMaterialBalance()
	if ourTurn == board.SideWhite {
		totalScorePiece = int32(scorePieceWhite) - int32(scorePieceBlack)
	} else {
		totalScorePiece = int32(scorePieceBlack) - int32(scorePieceWhite)
	}

	// PST heuristic
	var totalScorePiecePosition int32
	for _, p := range []board.Piece{board.PiecePawn,
		board.PieceKnight,
		board.PieceBishop,
		board.PieceRook,
		board.PieceQueen,
		board.PieceKing,
	} {
		var pos position.Pos
		bm := b.GetBitmap(ourTurn, p)
		for bm != 0 {
			if bm&1 == 1 {
				totalScorePiecePosition += scorePiecePosition[p][scorePiecePositionMap[ourTurn][p]]
			}
			pos++
			bm >>= 1
		}
	}

	// Tempo bonus to reduce early game oscillation due to leaf parity
	var totalTempoBonus int32
	if ourTurn == e.currentTurn {
		totalTempoBonus = scoreTempoBonus
	}

	return totalScorePiece + totalScorePiecePosition + totalTempoBonus
}
