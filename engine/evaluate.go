package engine

import (
	"github.com/daystram/gambit/board"
)

var (
	scoreBishopPair int16 = 50
	scoreTempoBonus int16 = 20

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

func (e *Engine) scoreMoves(b *board.Board, pvMove board.Move, mvs *[]board.Move) {
	for i, mv := range *mvs {
		var score uint8
		if mv.Equals(pvMove) {
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

func (e *Engine) sortMoves(mvs *[]board.Move, index int) {
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

// Evaluate returns the score evaluated from the given board.
// The score is positive relative to the currently playing side.
func (e *Engine) Evaluate(b *board.Board) int16 {
	ourTurn := b.Turn()
	theirTurn := ourTurn.Opposite()

	var (
		materialMG, materialEG     int16 // Material heuristic
		positionMG, positionEG     int16 // PST heuristic
		bishopPairMG, bishopPairEG int16 // Bishop pair
		tempoMG, tempoEG           int16 // Tempo bonus to reduce early game oscillation due to leaf parity
	)

	materialWhiteMG, materialBlackMG := b.GetMaterialValue()
	materialWhiteEG, materialBlackEG := materialWhiteMG, materialBlackMG // TODO: tapering material value
	positionWhiteMG, positionBlackMG, positionWhiteEG, positionBlackEG := b.GetPositionValue()
	if ourTurn == board.SideWhite {
		materialMG, materialEG = materialWhiteMG-materialBlackMG, materialWhiteEG-materialBlackEG
		positionMG, positionEG = positionWhiteMG-positionBlackMG, positionWhiteEG-positionBlackEG
	} else {
		materialMG, materialEG = materialBlackMG-materialWhiteMG, materialBlackEG-materialWhiteEG
		positionMG, positionEG = positionBlackMG-positionWhiteMG, positionBlackEG-positionWhiteEG
	}

	if b.GetBitmap(ourTurn, board.PieceBishop).BitCount() >= 2 { // TODO: score for different color pair only?
		bishopPairMG += scoreBishopPair
		bishopPairEG += scoreBishopPair
	}
	if b.GetBitmap(theirTurn, board.PieceBishop).BitCount() >= 2 {
		bishopPairMG -= scoreBishopPair
		bishopPairEG -= scoreBishopPair
	}

	if ourTurn == e.currentTurn {
		tempoMG = scoreTempoBonus
	}

	scoreMG, scoreEG := materialMG+positionMG+bishopPairMG+tempoMG, materialEG+positionEG+bishopPairEG+tempoEG
	phaseMG := int16(max(b.Phase(), 0))
	phaseEG := int16(board.PhaseTotal) - phaseMG
	return (scoreMG*phaseMG + scoreEG*phaseEG) / int16(board.PhaseTotal)
}
