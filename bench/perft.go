package bench

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/daystram/gambit/board"
)

func Perft(depth int, fen string, parallel, verbose bool, out chan string) error {
	var nodes, cap, enp, cas, pro, chk uint64
	b, _, err := board.NewBoard(
		board.WithFEN(fen),
	)
	if err != nil {
		return err
	}

	var run perftFunc
	if parallel {
		run = runPerftParallel
	} else {
		run = runPerft
	}

	start := time.Now()
	run(b, depth, true, true, out, &nodes, &cap, &enp, &cas, &pro, &chk)
	end := time.Now()

	out <- message.NewPrinter(language.English).
		Sprintf("d=%d nodes=%d rate=%dn/s cap=%d enp=%d cas=%d pro=%d chk=%d (%.3fs elapsed)",
			depth, nodes, int(float64(nodes)/end.Sub(start).Seconds()), cap, enp, cas, pro, chk, end.Sub(start).Seconds())

	return nil
}

type perftFunc func(b *board.Board, d int, root, verbose bool, out chan string, nodes, cap, enp, cas, pro, chk *uint64) uint64

func runPerft(b *board.Board, d int, root, verbose bool, out chan string, nodes, cap, enp, cas, pro, chk *uint64) uint64 {
	if d == 0 {
		*nodes++
		return 1
	}

	var sum uint64
	for _, mv := range b.GenerateMoves() {
		var child uint64
		bb := b.Clone()
		bb.Apply(mv)
		if d != 2 {
			child = runPerft(bb, d-1, false, verbose, out, nodes, cap, enp, cas, pro, chk)
		} else {
			leafMoves := bb.GenerateMoves()
			child = uint64(len(leafMoves))
			*nodes += child
			for _, leaf := range leafMoves {
				if leaf.IsCapture {
					*cap++
				}
				if leaf.IsEnPassant {
					*enp++
				}
				if leaf.IsCastle != board.CastleDirectionUnknown {
					*cas++
				}
				if leaf.IsPromote != board.PieceUnknown {
					*pro++
				}
				if leaf.IsCheck {
					*chk++
				}
			}
		}
		if verbose && root {
			out <- fmt.Sprintf("%s: %d", mv.UCI(), child)
		}
		sum += child
	}
	return sum
}

func runPerftParallel(b *board.Board, d int, root, verbose bool, out chan string, nodes, cap, enp, cas, pro, chk *uint64) uint64 {
	if d == 0 {
		atomic.AddUint64(nodes, 1)
		return 1
	}

	var sum uint64
	var wg sync.WaitGroup
	for _, mv := range b.GenerateMoves() {
		mv := mv
		wg.Add(1)
		go func() {
			defer wg.Done()
			var child uint64
			bb := b.Clone()
			bb.Apply(mv)
			if d != 2 {
				child = runPerftParallel(bb, d-1, false, verbose, out, nodes, cap, enp, cas, pro, chk)
			} else {
				leafMoves := bb.GenerateMoves()
				child = uint64(len(leafMoves))
				atomic.AddUint64(nodes, child)
				for _, leaf := range leafMoves {
					if leaf.IsCapture {
						atomic.AddUint64(cap, 1)
					}
					if leaf.IsEnPassant {
						atomic.AddUint64(enp, 1)
					}
					if leaf.IsCastle != board.CastleDirectionUnknown {
						atomic.AddUint64(cas, 1)
					}
					if leaf.IsPromote != board.PieceUnknown {
						atomic.AddUint64(pro, 1)
					}
					if leaf.IsCheck {
						atomic.AddUint64(chk, 1)
					}
				}
			}
			if verbose && root {
				out <- fmt.Sprintf("%s: %d", mv.UCI(), child)
			}
			atomic.AddUint64(&sum, child)
		}()
	}
	wg.Wait()
	return sum
}
