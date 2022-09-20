package main

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/daystram/gambit/board"
)

func perft(depth int) error {
	for _, p := range []struct {
		name string
		f    perftFunc
	}{
		// {name: "dfs", f: perftDFS},
		{name: "parallel dfs", f: perftParallelDFS},
	} {
		log.Printf("============ perft(%d): %s\n", depth, p.name)

		var nodes, cap, enp, cas, pro, chk uint64
		b, _, _ := board.NewBoard(board.WithFEN("rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8"))

		start := time.Now()
		p.f(b, depth, &nodes, &cap, &enp, &cas, &pro, &chk)
		end := time.Now()

		log.Println(message.NewPrinter(language.English).
			Sprintf("d=%d nodes=%d rate=%dn/s cap=%d enp=%d cas=%d pro=%d chk=%d (%.3fs elapsed)",
				depth, nodes, int(float64(nodes)/end.Sub(start).Seconds()), cap, enp, cas, pro, chk, end.Sub(start).Seconds()))
	}
	return nil
}

type perftFunc func(b *board.Board, d int, nodes, cap, enp, cas, pro, chk *uint64)

func perftParallelDFS(b *board.Board, d int, nodes, cap, enp, cas, pro, chk *uint64) {
	if d == 0 {
		atomic.AddUint64(nodes, 1)
		fmt.Println("FEN", b.FEN())
		return
	}

	mvs := b.GenerateMoves(b.Turn())

	var wg sync.WaitGroup
	for _, mv := range mvs {
		mv := mv
		wg.Add(1)
		go func() {
			defer wg.Done()
			bb := b.Clone()
			bb.Apply(mv)
			perftParallelDFS(bb, d-1, nodes, cap, enp, cas, pro, chk)
			if mv.IsCapture {
				atomic.AddUint64(cap, 1)
			}
			if mv.IsEnPassant {
				atomic.AddUint64(enp, 1)
			}
			if mv.IsCastle != board.CastleDirectionUnknown {
				atomic.AddUint64(cas, 1)
			}
			if mv.IsPromote != board.PieceUnknown {
				atomic.AddUint64(pro, 1)
			}
			if mv.IsCheck {
				atomic.AddUint64(chk, 1)
			}
		}()
	}
	wg.Wait()
}

func perftDFS(b *board.Board, d int, nodes, cap, enp, cas, pro, chk *uint64) {
	if d == 0 {
		atomic.AddUint64(nodes, 1)
		return
	}

	s := board.SideWhite
	if d%2 == 1 {
		s = board.SideBlack
	}

	mvs := b.GenerateMoves(s)

	for _, mv := range mvs {
		bb := b.Clone()
		bb.Apply(mv)
		perftDFS(bb, d-1, nodes, cap, enp, cas, pro, chk)
		if mv.IsCapture {
			atomic.AddUint64(cap, 1)
		}
		if mv.IsEnPassant {
			atomic.AddUint64(enp, 1)
		}
		if mv.IsCastle != board.CastleDirectionUnknown {
			atomic.AddUint64(cas, 1)
		}
		if mv.IsPromote != board.PieceUnknown {
			atomic.AddUint64(pro, 1)
		}
		if mv.IsCheck {
			atomic.AddUint64(chk, 1)
		}
	}
}
