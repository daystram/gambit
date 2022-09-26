package engine

import (
	"github.com/daystram/gambit/board"
	"github.com/daystram/gambit/position"
)

var (
	scorePiecePosition = [2 + 2][6 + 1][64]int32{
		board.SideWhite: {
			board.PiecePawn: {
				0, 0, 0, 0, 0, 0, 0, 0,
				10, 0, 0, -100, -100, 0, 0, 10,
				0, 10, 10, 20, 20, 10, 10, 0,
				0, 0, 20, 40, 40, 20, 0, 0,
				0, 0, 20, 60, 60, 20, 0, 0,
				30, 30, 50, 50, 50, 50, 30, 30,
				60, 60, 60, 60, 60, 60, 60, 60,
				70, 70, 70, 70, 70, 70, 70, 70,
			},
		},
		board.SideBlack: {},
	}

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

// +ve for our side, -ve for opponent
func (e *Engine) evaluate(b *board.Board) int32 {
	ourTurn := b.Turn()

	// TODO: check game state here?

	var totalScorePiece int32
	scorePieceWhite, scorePieceBlack := b.GetMaterialBalance()
	if ourTurn == board.SideWhite {
		totalScorePiece = int32(scorePieceWhite) - int32(scorePieceBlack)
	} else {
		totalScorePiece = int32(scorePieceBlack) - int32(scorePieceWhite)
	}

	// TODO: PST heuristic
	var totalScorePiecePosition int32
	for _, p := range []board.Piece{board.PiecePawn} {
		var pos position.Pos
		bm := b.GetBitmap(ourTurn, p)
		for bm != 0 {
			if bm&1 == 1 {
				totalScorePiecePosition += scorePiecePosition[ourTurn][p][pos]
			}
			pos++
			bm >>= 1
		}
	}

	return totalScorePiece + totalScorePiecePosition
}
