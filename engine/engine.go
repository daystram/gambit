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
	ScoreInfinite int32 = math.MaxInt32

	movetimeEndEarlyThreshold = 0.75
	nullMoveReduction         = 2

	killerCount    = 2
	scoreCheckmate = ScoreInfinite - 1
)

func DefaultLogger(a ...any) {
	fmt.Println(a...)
}

type PVLine struct {
	mvs []*board.Move
}

func (pvl *PVLine) GetPV() *board.Move {
	if len(pvl.mvs) == 0 {
		return nil
	}
	return pvl.mvs[0]
}

func (pvl *PVLine) Set(mv *board.Move, nextPVL PVLine) {
	if pvl == nil {
		return
	}
	pvl.mvs = append([]*board.Move{mv}, nextPVL.mvs...)
}

func (pvl *PVLine) Len() int {
	return len(pvl.mvs)
}

func (pvl *PVLine) StringUCI() string {
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

func DumpHistory(b *board.Board, mvs []*board.Move) string {
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
	HashTableSize uint64
	Logger        func(...any)
}

type SearchConfig struct {
	ClockConfig ClockConfig
	Debug       bool
}

type Engine struct {
	tt      *TranspositionTable
	killers [MaxDepth][killerCount]*board.Move
	clock   *Clock

	currentPly       uint8
	currentTurn      board.Side
	nodes            uint32
	totalNodes       uint32
	totalElapsedTime time.Duration
	logger           func(...any)
}

func NewEngine(cfg *EngineConfig) *Engine {
	if cfg.HashTableSize == 0 {
		cfg.HashTableSize = DefaultHashTableSize
	}
	if cfg.Logger == nil {
		cfg.Logger = DefaultLogger
	}

	return &Engine{
		tt:     NewTranspositionTable(cfg.HashTableSize),
		clock:  NewClock(),
		logger: cfg.Logger,
	}
}

func (e *Engine) Search(ctx context.Context, b *board.Board, cfg *SearchConfig) (*board.Move, error) {
	mv, err := e.search(ctx, b, cfg)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		return nil, err
	}
	if mv == nil {
		return nil, errors.New("cannot resolve best move")
	}

	return mv, nil
}

func (e *Engine) search(ctx context.Context, b *board.Board, cfg *SearchConfig) (*board.Move, error) {
	var err error
	var bestMove *board.Move
	var bestScore int32
	e.currentPly = b.Ply()
	e.currentTurn = b.Turn()
	e.totalNodes = 0
	e.totalElapsedTime = 0

	e.clock.Start(ctx, b.Turn(), b.FullMoveClock(), &cfg.ClockConfig)

	for d := uint8(1); !e.clock.DoneByDepth(d); d++ {
		e.nodes = 0
		e.tt.ResetStats()
		pvl := PVLine{}

		var candidateScore int32
		startTime := time.Now()
		candidateScore = e.negamax(ctx, b, bestMove, &pvl, d, -ScoreInfinite, ScoreInfinite, true)
		elapsedTime := time.Since(startTime)

		if e.clock.DoneByMovetime() {
			break
		}

		bestMove = pvl.GetPV()
		bestScore = candidateScore

		if cfg.Debug {
			e.logger(message.NewPrinter(language.English).
				Sprintf("depth:%d [%s] nodes:%d (%.0fn/s) t:%s\n    %s",
					d, formatScoreDebug(bestScore, pvl), e.nodes, float64(e.nodes)/((elapsedTime + 1).Seconds()), elapsedTime, pvl.String(b)))
		} else {
			e.logger(fmt.Sprintf("info depth %d score %s time %d nodes %d nps %.0f pv %s",
				d, formatScoreUCI(bestScore, pvl), elapsedTime.Milliseconds(), e.nodes, float64(e.nodes)/((elapsedTime + 1).Seconds()), pvl.StringUCI()))
		}

		e.totalNodes += e.nodes
		e.totalElapsedTime += elapsedTime
		if bestScore == scoreCheckmate || bestScore == -scoreCheckmate ||
			(e.clock.Mode() == ClockModeGametime && e.totalElapsedTime.Seconds() > e.clock.AllocatedMovetime().Seconds()*movetimeEndEarlyThreshold) {
			break
		}
	}

	e.clock.Stop()
	return bestMove, err
}

// For a given board, regardless turn, we always want to maximize alpha.
// TODO: parallelize
func (e *Engine) negamax(
	ctx context.Context,
	b *board.Board,
	lastPV *board.Move,
	pvl *PVLine,
	depth uint8,
	alpha, beta int32,
	isRoot bool,
) int32 {
	e.nodes++
	initialAlpha := alpha

	// check if max depth reached or movetime exceeded
	if depth == 0 || e.clock.DoneByMovetime() {
		return e.Evaluate(b)
	}

	// check from TranspositionTable
	typ, ttMove, ttScore, ttDepth, ok := e.tt.Get(b, e.currentPly)
	if !isRoot && !ok && ttDepth >= depth {
		switch typ {
		case EntryTypeExact:
			return ttScore
		case EntryTypeLowerBound:
			alpha = max(alpha, ttScore)
		case EntryTypeUpperBound:
			beta = min(beta, ttScore)
		}
		if alpha >= beta {
			return ttScore
		}
	}

	isCheck := b.IsKingChecked()

	// null move pruning
	if !isRoot && depth >= 3 && !isCheck {
		unApply := b.ApplyNull()
		score := -e.negamax(ctx, b, nil, nil, depth-nullMoveReduction-1, -beta, -alpha, false)
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
	e.scoreMoves(b, lastPV, ttMove, &mvs)

	var moveCount int8
	var bestMove *board.Move
	var bestChildPVL PVLine
	bestScore := -ScoreInfinite
	for i := 0; i < len(mvs); i++ {
		e.sortMoves(&mvs, i)
		mv := mvs[i]
		if !b.IsLegal(mv) {
			continue
		}
		moveCount++

		var childPVL PVLine
		unApply := b.Apply(mv)
		score := -e.negamax(ctx, b, nil, &childPVL, depth-1, -beta, -alpha, false)
		unApply()

		if score > bestScore || bestMove == nil {
			bestMove = mv
			bestChildPVL = childPVL
			bestScore = score
		}
		if score >= beta {
			// set Killer move
			if depth > 0 && !bestMove.IsCapture {
				ply := b.Ply()
				if !bestMove.Equals(e.killers[ply][0]) {
					for i := killerCount - 1; i >= 1; i-- {
						e.killers[ply][i] = e.killers[ply][i-1]
					}
					e.killers[ply][0] = bestMove
				}
			}
			break // fail-hard cutoff
		}
		if score > alpha {
			alpha = score
		}

		if e.clock.DoneByMovetime() {
			break
		}
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
	switch {
	case bestScore <= initialAlpha:
		typ = EntryTypeLowerBound
	case bestScore >= beta:
		typ = EntryTypeUpperBound
	default:
		typ = EntryTypeExact
	}
	e.tt.Set(typ, b, bestMove, bestScore, depth, e.currentPly)

	pvl.Set(bestMove, bestChildPVL)
	return bestScore
}

func (e *Engine) TranspositionStats() string {
	h, m, w := e.tt.Stats()
	return fmt.Sprintf("hits=%d misses=%d writes=%d", h, m, w)
}

func min[T constraints.Ordered](x1, x2 T) T {
	if x1 < x2 {
		return x1
	}
	return x2
}

func max[T constraints.Ordered](x1, x2 T) T {
	if x1 > x2 {
		return x1
	}
	return x2
}

func formatScoreDebug(s int32, pvl PVLine) string {
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

func formatScoreUCI(s int32, pvl PVLine) string {
	if s == scoreCheckmate {
		return fmt.Sprintf("mate %d", pvl.Len()/2+1)
	}
	if s == -scoreCheckmate {
		return fmt.Sprintf("mate -%d", pvl.Len()/2)
	}
	return fmt.Sprintf("cp %d", s)
}
