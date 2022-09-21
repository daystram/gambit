package main

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/daystram/gambit/board"
)

func perft(depth int, fen string) error {
	for _, p := range []struct {
		name string
		f    perftFunc
	}{
		// {name: "dfs", f: runPerft},
		{name: "parallel dfs", f: runPerftParallel},
	} {
		log.Printf("============ perft(%d): %s\n", depth, p.name)

		var nodes, cap, enp, cas, pro, chk uint64
		b, _, err := board.NewBoard(
			board.WithFEN(fen),
		)
		if err != nil {
			return err
		}

		start := time.Now()
		p.f(b, depth, true, true, &nodes, &cap, &enp, &cas, &pro, &chk)
		end := time.Now()

		log.Println(message.NewPrinter(language.English).
			Sprintf("d=%d nodes=%d rate=%dn/s cap=%d enp=%d cas=%d pro=%d chk=%d (%.3fs elapsed)",
				depth, nodes, int(float64(nodes)/end.Sub(start).Seconds()), cap, enp, cas, pro, chk, end.Sub(start).Seconds()))
	}
	return nil
}

type perftFunc func(b *board.Board, d int, root, debug bool, nodes, cap, enp, cas, pro, chk *uint64) uint64

func runPerft(b *board.Board, d int, root, debug bool, nodes, cap, enp, cas, pro, chk *uint64) uint64 {
	if d == 0 {
		*nodes++
		return 1
	}

	var sum uint64
	for _, mv := range b.GenerateMoves(b.Turn()) {
		var child uint64
		bb := b.Clone()
		bb.Apply(mv)
		if d != 2 {
			child = runPerft(bb, d-1, false, debug, nodes, cap, enp, cas, pro, chk)
		} else {
			leafMoves := bb.GenerateMoves(b.Turn().Opposite())
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
		if debug && root {
			log.Printf("%s: %d\n", mv.UCI(), child)
		}
		sum += child
	}
	return sum
}

func runPerftParallel(b *board.Board, d int, root, debug bool, nodes, cap, enp, cas, pro, chk *uint64) uint64 {
	if d == 0 {
		atomic.AddUint64(nodes, 1)
		return 1
	}

	var sum uint64
	var wg sync.WaitGroup
	for _, mv := range b.GenerateMoves(b.Turn()) {
		mv := mv
		wg.Add(1)
		go func() {
			defer wg.Done()
			var child uint64
			bb := b.Clone()
			bb.Apply(mv)
			if d != 2 {
				child = runPerftParallel(bb, d-1, false, debug, nodes, cap, enp, cas, pro, chk)
			} else {
				leafMoves := bb.GenerateMoves(b.Turn().Opposite())
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
			if debug && root {
				log.Printf("%s: %d\n", mv.UCI(), child)
			}
			atomic.AddUint64(&sum, child)
		}()
	}
	wg.Wait()
	return sum
}
