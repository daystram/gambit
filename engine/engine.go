package engine

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/daystram/gambit/board"
	"golang.org/x/exp/constraints"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const (
	ScoreInfinite int16 = math.MaxInt16

	clockTimePVConsistencyDecay       = 0.95 // more reduction with decay towards 0
	clockTimeScoreConsistencyMaxDecay = 0.95
	clockTimeScoreConsistencyWindow   = 0.75
	nullMoveReduction                 = 2
	lateMoveReductionFullMoves        = 4
	lateMoveReductionDepthLimit       = 3

	scoreCheckmate = ScoreInfinite - 1
)

func DefaultLogger(a ...any) {
	fmt.Println(a...)
}

type PVLine struct {
	mvs []board.Move
}

func (pvl *PVLine) GetPV() board.Move {
	if len(pvl.mvs) == 0 {
		return board.Move{}
	}
	return pvl.mvs[0]
}

func (pvl *PVLine) Set(mv board.Move, nextPVL PVLine) {
	if pvl == nil {
		return
	}
	pvl.mvs = append([]board.Move{mv}, nextPVL.mvs...)
}

func (pvl *PVLine) Clear() {
	pvl.mvs = pvl.mvs[:0] // memory not released for GC
}

func (pvl *PVLine) Len() int {
	return len(pvl.mvs)
}

func (pvl *PVLine) StringUCI() string {
	if pvl == nil {
		return ""
	}
	builder := strings.Builder{}
	for i, mv := range pvl.mvs {
		_, _ = builder.WriteString(mv.UCI())
		if i < len(pvl.mvs)-1 {
			_, _ = builder.WriteRune(' ')
		}
	}
	return builder.String()
}

func (pvl *PVLine) String(b *board.Board) string {
	return DumpHistory(b, pvl.mvs)
}

func DumpHistory(b *board.Board, mvs []board.Move) string {
	if b == nil || mvs == nil || len(mvs) < 1 {
		return ""
	}
	builder := strings.Builder{}
	bb := b.Clone()
	fullMoveClock := bb.FullMoveClock()
	if mvs[0].IsTurn == board.SideBlack {
		_, _ = builder.WriteString(fmt.Sprintf("%d... ", fullMoveClock))
	}
	for i, mv := range mvs {
		bb.Apply(mv)
		if mv.IsTurn == board.SideWhite {
			_, _ = builder.WriteString(fmt.Sprintf("%d. %s", fullMoveClock, mv))
		} else {
			_, _ = builder.WriteString(mv.String())
			fullMoveClock++
		}
		if bb.State().IsCheck() {
			_, _ = builder.WriteRune('+')
		}
		if bb.State().IsCheckmate() {
			_, _ = builder.WriteRune('#')
		}
		if bb.State().IsDraw() {
			_, _ = builder.WriteRune('=')
		}
		if i < len(mvs)-1 {
			_, _ = builder.WriteRune(' ')
		}
	}
	return builder.String()
}

type EngineConfig struct {
	HashTableSize uint32
	Logger        func(...any)
}

type SearchConfig struct {
	ClockConfig ClockConfig
	Debug       bool
}

type Engine struct {
	tt           *TranspositionTable
	killers      [MaxDepth][2]board.Move
	boardHistory [1024]uint64
	clock        *Clock

	currentPly  uint16
	currentTurn board.Side
	nodes       uint32
	elapsedTime time.Duration
	logger      func(...any)
}

func NewEngine(cfg *EngineConfig) *Engine {
	if cfg.Logger == nil {
		cfg.Logger = DefaultLogger
	}

	return &Engine{
		tt:     NewTranspositionTable(cfg.HashTableSize),
		clock:  NewClock(),
		logger: cfg.Logger,
	}
}

func (e *Engine) Search(ctx context.Context, b *board.Board, cfg *SearchConfig) (board.Move, error) {
	mv, err := e.search(ctx, b, cfg)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		return board.Move{}, err
	}
	if mv.IsNull() {
		return board.Move{}, errors.New("cannot resolve best move")
	}

	return mv, nil
}

func (e *Engine) search(ctx context.Context, b *board.Board, cfg *SearchConfig) (board.Move, error) {
	var err error
	var bestMove, prevMove board.Move
	var bestScore, prevScore int16
	var pvl PVLine
	e.currentPly = b.Ply()
	e.currentTurn = b.Turn()
	e.nodes = 0
	e.elapsedTime = 0
	timeDecay := float64(1)

	e.clock.Start(ctx, b.Turn(), b.FullMoveClock(), &cfg.ClockConfig)

	for d := uint8(1); !e.clock.DoneByDepth(d); d++ {
		startTime := time.Now()
		candidateScore := e.negamax(b, board.Move{}, &pvl, d, 0, -ScoreInfinite, ScoreInfinite)
		e.elapsedTime += time.Since(startTime)

		if e.clock.DoneByMovetime() {
			break
		}

		bestMove = pvl.GetPV()
		bestScore = candidateScore

		if cfg.Debug {
			e.logger(message.NewPrinter(language.English).
				Sprintf("depth:%d [%s] nodes:%d (%.0fn/s) t:%s\n    %s",
					d, formatScoreDebug(bestScore, pvl), e.nodes, float64(e.nodes)/((e.elapsedTime + 1).Seconds()), e.elapsedTime, pvl.String(b)))
		} else {
			e.logger(fmt.Sprintf("info depth %d score %s time %d nodes %d nps %.0f pv %s",
				d, formatScoreUCI(bestScore, pvl), e.elapsedTime.Milliseconds(), e.nodes, float64(e.nodes)/((e.elapsedTime + 1).Seconds()), pvl.StringUCI()))
		}

		if bestScore == scoreCheckmate || bestScore == -scoreCheckmate {
			break
		}
		if d > 1 && e.clock.Mode() == ClockModeGametime {
			if prevMove.Equals(bestMove) {
				timeDecay *= clockTimePVConsistencyDecay // carry decay from previous iteration
			} else {
				timeDecay = 1 // reset decay factor
			}
			timeDecay *= min(max(
				float64(abs(prevScore-bestScore))/float64(max(abs(prevScore), 1))/clockTimeScoreConsistencyWindow,
				clockTimeScoreConsistencyMaxDecay,
			), 1)
			// TODO: measure decay by complexity
			if e.elapsedTime.Seconds() > e.clock.allocatedMovetime.Seconds()*timeDecay {
				break
			}
		}
		pvl.Clear()
		prevMove = bestMove
		prevScore = bestScore
	}

	e.clock.Stop()
	return bestMove, err
}

// For a given board, regardless turn, we always want to maximize alpha.
// TODO: parallelize
func (e *Engine) negamax(
	b *board.Board,
	prevMove board.Move,
	pvl *PVLine,
	depth, dist uint8,
	alpha, beta int16,
) int16 {
	e.nodes++

	// check if movetime exceeded
	if e.clock.DoneByMovetime() {
		return 0
	}

	// check if leaf reached
	if depth == 0 {
		return e.quiescence(b, pvl, alpha, beta)
	}

	// check if repeated
	if e.isBoardRepeated(b, dist) {
		return 0
	}

	isRoot := dist == 0

	// check from TranspositionTable
	ttType, ttMove, ttScore, ttDepth, ok := e.tt.Get(b, e.currentPly)
	if !isRoot && ok && ttDepth >= depth {
		switch ttType {
		case EntryTypeExact:
			return ttScore
		case EntryTypeLowerBound:
			if ttScore <= alpha {
				return alpha
			}
		case EntryTypeUpperBound:
			if ttScore >= beta {
				return beta
			}
		}
	}

	isCheck := b.IsKingChecked(b.Turn())
	isPVNode := beta-alpha > 1

	// null move pruning
	if !isCheck && !isRoot && depth >= 3 {
		unApply := b.ApplyNull()
		e.boardHistory[dist] = b.Hash()
		score := -e.negamax(b, board.Move{}, nil, depth-(nullMoveReduction+1), dist+(nullMoveReduction+1), -beta, -(beta - 1))
		unApply()

		if score >= beta {
			return beta
		}
		if e.clock.DoneByMovetime() {
			return 0
		}
	}

	// generate next moves
	mvs := b.GeneratePseudoLegalMoves()

	// assign score to moves
	e.scoreMoves(b, ttMove, &mvs)

	var moveCount int8
	var bestMove board.Move
	var childPVL PVLine
	bestScore := -ScoreInfinite
	ttType = EntryTypeLowerBound
	for i := 0; i < len(mvs); i++ {
		e.sortMoves(&mvs, i)
		mv := mvs[i]

		unApply, ok := b.Apply(mv)
		if !ok {
			unApply()
			continue
		}
		moveCount++
		e.boardHistory[dist] = b.Hash()
		var score int16
		if moveCount == 1 {
			score = -e.negamax(b, mv, &childPVL, depth-1, dist+1, -beta, -alpha)
		} else {
			// late move reduction
			if !isPVNode && !isCheck && !prevMove.IsCapture && prevMove.IsPromote == board.PieceUnknown &&
				moveCount >= lateMoveReductionFullMoves && depth >= lateMoveReductionDepthLimit {
				reduction := uint8(1)
				if moveCount > 6 {
					reduction = depth / 3
				}
				score = -e.negamax(b, mv, &childPVL, depth-(reduction+1), dist+1, -(alpha + 1), -alpha)
				if score > alpha {
					// re-search at full depth
					score = -e.negamax(b, mv, &childPVL, depth-1, dist+1, -beta, -alpha)
				}
			} else {
				score = -e.negamax(b, mv, &childPVL, depth-1, dist+1, -beta, -alpha)
			}
		}
		unApply()

		if score > bestScore {
			bestMove = mv
			bestScore = score
		}
		if score >= beta {
			// set Killer move
			if depth > 0 && !bestMove.IsCapture {
				ply := b.Ply()
				if !bestMove.Equals(e.killers[ply][0]) {
					e.killers[ply][1] = e.killers[ply][0]
					e.killers[ply][0] = bestMove
				}
			}
			ttType = EntryTypeUpperBound
			break // fail-hard cutoff
		}
		if score > alpha {
			alpha = score
			pvl.Set(mv, childPVL)
			ttType = EntryTypeExact
		}

		if e.clock.DoneByMovetime() {
			break
		}
		childPVL.Clear()
	}

	// no moves were explored, game has terminated
	if moveCount == 0 {
		if isCheck {
			// game is Checkmate
			return -scoreCheckmate
		}
		// game is Stalemate
		return 0
	}

	// set TranspositionTable
	e.tt.Set(b, e.currentPly, ttType, bestMove, bestScore, depth)

	return bestScore
}

func (e *Engine) quiescence(b *board.Board, pvl *PVLine, alpha, beta int16) int16 {
	e.nodes++

	if e.clock.DoneByMovetime() {
		return 0
	}

	eval := e.Evaluate(b)
	if b.Ply() >= uint16(MaxDepth) {
		return eval
	}
	isCheck := b.IsKingChecked(b.Turn())
	if !isCheck && eval >= beta {
		return beta // cutoff, but full search if in check
	}
	if alpha < eval {
		alpha = eval
	}

	mvs := b.GeneratePseudoLegalMoves()

	e.scoreMoves(b, board.Move{}, &mvs)

	var childPVL PVLine
	bestScore := eval
	for i := 0; i < len(mvs); i++ {
		e.sortMoves(&mvs, i)
		mv := mvs[i]
		if !isCheck && !mv.IsCapture {
			continue
		}

		unApply, ok := b.Apply(mv)
		if !ok {
			unApply()
			continue
		}
		score := -e.quiescence(b, &childPVL, -beta, -alpha)
		unApply()

		if score > bestScore {
			bestScore = score
		}
		if score >= beta {
			break // fail-hard cutoff
		}
		if score > alpha {
			alpha = score
			pvl.Set(mv, childPVL)
		}

		if e.clock.DoneByMovetime() {
			break
		}
		childPVL.Clear()
	}

	return bestScore
}

func (e *Engine) isBoardRepeated(b *board.Board, dist uint8) bool {
	count := 0
	for ply := uint8(0); ply < dist; ply++ {
		if e.boardHistory[ply] == b.Hash() {
			if count++; count >= 2 {
				return true // TODO: try strict repetition check on first match?
			}
		}
	}
	return false
}

func max[T constraints.Ordered](x1, x2 T) T {
	if x1 > x2 {
		return x1
	}
	return x2
}

func min[T constraints.Ordered](x1, x2 T) T {
	if x1 < x2 {
		return x1
	}
	return x2
}

func abs[T constraints.Signed](x T) T {
	if x < 0 {
		return x * -1
	}
	return x
}

func formatScoreDebug(s int16, pvl PVLine) string {
	if s == ScoreInfinite {
		return "+inf"
	}
	if s == -ScoreInfinite {
		return "-inf"
	}
	if s == scoreCheckmate {
		return fmt.Sprintf("#+%d", pvl.Len()/2+1)
	}
	if s == -scoreCheckmate {
		return fmt.Sprintf("#-%d", pvl.Len()/2)
	}
	if s > 0 {
		return fmt.Sprintf("+%.2f", float64(s)/100)
	}
	if s < 0 {
		return fmt.Sprintf("%.2f", float64(s)/100)
	}
	return "0"
}

func formatScoreUCI(s int16, pvl PVLine) string {
	if s == scoreCheckmate {
		return fmt.Sprintf("mate %d", pvl.Len()/2+1)
	}
	if s == -scoreCheckmate {
		return fmt.Sprintf("mate -%d", pvl.Len()/2)
	}
	return fmt.Sprintf("cp %d", s)
}
