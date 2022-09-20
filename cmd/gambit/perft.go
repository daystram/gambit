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

func perft(depth int) error {
	for name, pf := range map[string]perftFunc{
		"parallel dfs": pertfParallelDFS,
		"dfs":          pertfDFS,
	} {
		log.Printf("============ perft(%d): %s\n", depth, name)

		var pNodes, pCap, pEnp, pCas, pPro, pChk uint64
		b, _, _ := board.NewBoard()

		pStart := time.Now()
		pf(b, 0, depth, &pNodes, &pCap, &pEnp, &pCas, &pPro, &pChk)
		pEnd := time.Now()

		log.Println(message.NewPrinter(language.English).
			Sprintf("d=%d nodes=%d rate=%dn/s cap=%d enp=%d cas=%d pro=%d chk=%d (%.3fs elapsed)",
				depth, pNodes, int(float64(pNodes)/pEnd.Sub(pStart).Seconds()), pCap, pEnp, pCas, pPro, pChk, pEnd.Sub(pStart).Seconds()))
	}
	return nil
}

type perftFunc func(b *board.Board, d, maxD int, nodes, cap, enp, cas, pro, chk *uint64)

func pertfParallelDFS(b *board.Board, d, maxD int, nodes, cap, enp, cas, pro, chk *uint64) {
	if d == maxD {
		atomic.AddUint64(nodes, 1)
		return
	}

	s := board.SideWhite
	if d%2 == 1 {
		s = board.SideBlack
	}

	var wg sync.WaitGroup
	mvs := b.GenerateMoves(s)
	for _, mv := range mvs {
		mv := mv
		wg.Add(1)
		go func() {
			defer wg.Done()
			bb := b.Clone()
			bb.Apply(mv)
			pertfParallelDFS(bb, d+1, maxD, nodes, cap, enp, cas, pro, chk)
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

func pertfDFS(b *board.Board, d, maxD int, nodes, cap, enp, cas, pro, chk *uint64) {
	if d == maxD {
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
		pertfDFS(bb, d+1, maxD, nodes, cap, enp, cas, pro, chk)
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
