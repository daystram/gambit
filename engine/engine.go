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
	Infinity int32 = math.MaxInt32
	MaxPly   uint8 = 255

	DefaultDepth           uint8 = 10
	DefaultTimeoutDuration       = 10 * time.Second

	killerCount    = 2
	scoreCheckmate = Infinity - 2
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
	MaxDepth uint8
	Timeout  time.Duration
	Debug    bool
}

type Engine struct {
	tt      *TranspositionTable
	killers [MaxPly][killerCount]*board.Move

	searchedNodes int
	logger        func(...any)
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
		logger: cfg.Logger,
	}
}

func (e *Engine) Search(ctx context.Context, b *board.Board, cfg *SearchConfig) (*board.Move, error) {
	if cfg.MaxDepth == 0 || cfg.MaxDepth == MaxPly {
		cfg.MaxDepth = DefaultDepth
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeoutDuration
	}

	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

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

	// TODO: Null-move heuristic (may not be necessary for now)
	for d := uint8(1); d < cfg.MaxDepth+1; d++ {
		e.searchedNodes = 0
		e.tt.ResetStats()
		pvl := PVLine{}

		var candidateScore int32
		startTime := time.Now()
		candidateScore, err = e.negamax(ctx, b, nil, &pvl, d, -Infinity, Infinity)
		endTime := time.Now()

		if err != nil {
			break
		}

		bestMove = pvl.GetPV()
		bestScore = candidateScore

		if cfg.Debug {
			e.logger(message.NewPrinter(language.English).
				Sprintf("depth:%d [%s] nodes:%d (%dn/s) t:%s\n    %s",
					d, formatScoreDebug(bestScore, pvl), e.searchedNodes, e.searchedNodes*1e9/int(endTime.Sub(startTime).Nanoseconds()+1), endTime.Sub(startTime), pvl.String(b)))
		} else {
			e.logger(fmt.Sprintf("info depth %d score %s time %d nodes %d nps %d pv %s",
				d, formatScoreUCI(bestScore, pvl), endTime.Sub(startTime).Milliseconds(), e.searchedNodes, e.searchedNodes*1e9/int(endTime.Sub(startTime).Nanoseconds()+1), pvl.StringUCI()))
		}

		if bestScore == scoreCheckmate {
			break
		}

	}
	return bestMove, err
}

// For a given board, regardless turn, we always want to maximize alpha.
// TODO: parallelize
func (e *Engine) negamax(
	ctx context.Context,
	b *board.Board,
	mv *board.Move,
	pvl *PVLine,
	depth uint8,
	alpha, beta int32,
) (int32, error) {
	e.searchedNodes++
	initialAlpha := alpha

	// check if max depth reached or deadline exceeded
	if err := ctx.Err(); depth == 0 || err != nil {
		s := e.evaluate(b, mv)
		return s, err
	}

	// check from TranspositionTable
	typ, ttMove, ttScore, ttDepth, ok := e.tt.Get(b)
	if ok && ttDepth >= depth {
		switch typ {
		case EntryTypeExact:
			pvl.Set(ttMove, PVLine{})
			return ttScore, nil
		case EntryTypeLowerBound:
			alpha = max(alpha, ttScore)
		case EntryTypeUpperBound:
			beta = min(beta, ttScore)
		}
		if alpha >= beta {
			pvl.Set(ttMove, PVLine{})
			return ttScore, nil
		}
	}

	// generate next moves
	mvs := b.GenerateMoves()

	// end early if game has ended
	if len(mvs) == 0 {
		var score int32
		st := b.State()
		turn := b.Turn()
		if (turn == board.SideWhite && st == board.StateCheckmateWhite) ||
			(turn == board.SideBlack && st == board.StateCheckmateBlack) {
			score = -scoreCheckmate
		}
		if st.IsDraw() {
			score = 0
		}
		return score, nil
	}

	// assign score to moves
	e.scoreMoves(b, ttMove, &mvs)

	var bestMove *board.Move
	bestScore := -Infinity
	for i := 0; i < len(mvs); i++ {
		e.sortMoves(&mvs, i)
		mv := mvs[i]

		bb := b.Clone()
		bb.Apply(mv)

		childPVL := PVLine{}
		score, err := e.negamax(ctx, bb, mv, &childPVL, depth-1, -beta, -alpha)
		score = -score // invert score

		if score > bestScore {
			bestMove = mv
			bestScore = score
		}
		if bestScore > alpha {
			alpha = bestScore
			pvl.Set(mv, childPVL)
		}
		if alpha >= beta {
			// set Killer move
			if depth > 1 && !bestMove.IsCapture {
				ply := b.Ply()
				if !bestMove.Equals(e.killers[ply][0]) {
					for i := killerCount - 1; i >= 1; i-- {
						temp := e.killers[ply][i]
						e.killers[ply][i] = temp
					}
					e.killers[ply][0] = bestMove
				}
			}
			break // cut-off
		}
		if err != nil {
			return bestScore, err
		}
	}

	// set TranspositionTable
	switch {
	case bestScore <= initialAlpha:
		typ = EntryTypeUpperBound
	case bestScore >= beta:
		typ = EntryTypeLowerBound
	default:
		typ = EntryTypeExact
	}
	e.tt.Set(typ, b, bestMove, bestScore, depth)

	return bestScore, nil
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
	if s == Infinity {
		return "+inf"
	}
	if s == -Infinity {
		return "-inf"
	}
	if s == scoreCheckmate {
		return fmt.Sprintf("#+%d", pvl.Len()/2+1)
	}
	if s == -scoreCheckmate {
		return fmt.Sprintf("#-%d", pvl.Len()/2+1)
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
		return fmt.Sprintf("mate -%d", pvl.Len()/2+1)
	}
	return fmt.Sprintf("%d", s)
}
