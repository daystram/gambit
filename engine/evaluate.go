package engine

import (
	"github.com/daystram/gambit/board"
	"github.com/daystram/gambit/position"
)

var (
	scorePiece = [2 + 2][6 + 1][64]int32{
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

	scoresMVVLVA = [6 + 1][6 + 1]int32{
		//                     P   N   B   R   Q
		board.PiecePawn:   {0, 15, 25, 35, 45, 55},
		board.PieceKnight: {0, 14, 24, 34, 44, 54},
		board.PieceBishop: {0, 13, 23, 33, 43, 53},
		board.PieceRook:   {0, 12, 22, 32, 42, 52},
		board.PieceQueen:  {0, 11, 21, 31, 41, 51},
		board.PieceKing:   {0, 10, 20, 30, 40, 50},
	}
)

// TODO: Killer heuristic
func (e *Engine) scoreMoves(b *board.Board, pv *board.Move, mvs *[]*board.Move) {
	for i, mv := range *mvs {
		score := int32(0)
		if mv.Equals(pv) {
			score += 100
		}
		if mv.IsCapture {
			capturedPiece, _ := b.GetSideAndPieces(mv.To)
			score += scoresMVVLVA[mv.Piece][capturedPiece]
		}
		(*mvs)[i].Score = score
	}
}

func (e *Engine) sortMoves(mvs *[]*board.Move, index int) {
	bestIndex, bestScore := index, int32(0)
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
func (e *Engine) evaluate(b *board.Board, mv *board.Move) int32 {
	ourTurn := b.Turn()
	oppTurn := ourTurn.Opposite()
	ourPBM := b.GetBitmap(ourTurn, board.PiecePawn)
	ourBBM := b.GetBitmap(ourTurn, board.PieceBishop)
	ourNBM := b.GetBitmap(ourTurn, board.PieceKnight)
	ourRBM := b.GetBitmap(ourTurn, board.PieceRook)
	ourQBM := b.GetBitmap(ourTurn, board.PieceQueen)
	oppPBM := b.GetBitmap(oppTurn, board.PiecePawn)
	oppBBM := b.GetBitmap(oppTurn, board.PieceBishop)
	oppNBM := b.GetBitmap(oppTurn, board.PieceKnight)
	oppRBM := b.GetBitmap(oppTurn, board.PieceRook)
	oppQBM := b.GetBitmap(oppTurn, board.PieceQueen)

	var scorePieceCount int32
	scorePieceCount += 100 * int32(ourPBM.BitCount()-oppPBM.BitCount())
	scorePieceCount += 350 * int32(ourBBM.BitCount()-oppBBM.BitCount())
	scorePieceCount += 300 * int32(ourNBM.BitCount()-oppNBM.BitCount())
	scorePieceCount += 500 * int32(ourRBM.BitCount()-oppRBM.BitCount())
	scorePieceCount += 900 * int32(ourQBM.BitCount()-oppQBM.BitCount())

	// TODO: support other pieces
	var scorePiecePosition int32
	for _, p := range []board.Piece{board.PiecePawn} {
		var pos position.Pos
		bm := b.GetBitmap(ourTurn, p)
		for bm != 0 {
			if bm&1 == 1 {
				scorePiecePosition += scorePiece[ourTurn][p][pos]
			}
			pos++
			bm >>= 1
		}
	}

	return scorePieceCount + scorePiecePosition
}
