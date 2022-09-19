package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/daystram/gambit/board"
)

func main() {
	err := realMain(os.Args[1:])
	if err != nil {
		log.Println(err)
		os.Exit(exitErr)
	}
	os.Exit(exitOK)
}

func realMain(args []string) error {
	if len(args) != 0 && args[0] == "movegen" {
		log.Println("============ test movegen")
		b, t, err := board.NewBoard(
			// board.WithFEN("8/8/8/3Pp3/8/8/8/8 w - e6 0 1"),
			// board.WithFEN("8/7R/3k4/3R4/3K4/8/8/2R1R3 b - - 0 1"),
			// board.WithFEN("8/p1p1k3/Pp3n2/1P2P3/4Pp2/5P2/2q3r1/1BK5 w - - 0 1"),
			board.WithFEN("rnbqkbnr/pppppppp/8/8/2B5/1N1QBN2/PPPPPPPP/R3K2R w KQkq - 0 1"),
		)
		if err != nil {
			return err
		}
		fmt.Println("to move:", t)
		fmt.Println(b.Draw())
		fmt.Println(b.State())
		dumpMoves(b.GenerateMoves(t))

		for _, mv := range b.GenerateMoves(t) {
			bb := b.Clone()
			bb.Apply(mv)
			fmt.Println(mv)
			fmt.Println(bb.Draw())
		}
	}

	if len(args) != 0 && args[0] == "stepping" {
		log.Println("============ test stepping")
		var (
			timesGenerateMoves []time.Duration
			timesApply         []time.Duration
			timesState         []time.Duration
		)
		b, _, _ := board.NewBoard()
		rand.Seed(1)
	stepLoop:
		for step := 0; step < 5000; step++ {
			turn := board.SideWhite
			if step%2 == 1 {
				turn = board.SideBlack
			}

			// drop cache
			b = b.Clone()

			t := time.Now()
			mvs := b.GenerateMoves(turn)
			timesGenerateMoves = append(timesGenerateMoves, time.Since(t))
			if len(mvs) == 0 {
				return fmt.Errorf("unexpected move exhaustion: state=%s", b.State())
			}
			mv := mvs[rand.Intn(len(mvs))]

			fmt.Printf("\n===== [#%d] %s: %s\n", step/2+1, mv.Side, mv)
			t = time.Now()
			b.Apply(mv)
			timesApply = append(timesApply, time.Since(t))
			fmt.Println(b.Draw())

			t = time.Now()
			st := b.State()
			timesState = append(timesState, time.Since(t))
			switch {
			case !st.IsRunning():
				fmt.Println(b.State())
				break stepLoop
			case st.IsCheck():
				<-time.Tick(500 * time.Millisecond)
			default:
				<-time.Tick(10 * time.Millisecond)
			}
		}

		avg := func(ds []time.Duration) time.Duration {
			var s time.Duration
			for _, d := range ds {
				s += d
			}
			return time.Duration(s.Seconds() / float64(len(ds)) * float64(time.Second))
		}
		fmt.Println("genmv:", avg(timesGenerateMoves))
		fmt.Println("apply:", avg(timesApply))
		fmt.Println("state:", avg(timesState))
	}

	if len(args) != 0 && args[0] == "search" {
		maxD := 5
		b, _, _ := board.NewBoard()
		startP := time.Now()
		log.Println("============ test search: parallel dfs")
		nodesP, capturesP, epP, castlesP, promotionsP, checksP := pdfs(b, 0, maxD, "")
		endP := time.Now()

		b, _, _ = board.NewBoard()
		startS := time.Now()
		log.Println("============ test search: dfs")
		cntS := dfs(b, 0, maxD, "")
		endS := time.Now()

		fmt.Printf("par d=%d nodes=%d captures=%d eps=%d cas=%d pros=%d chks=%d (%s elapsed)\n", maxD, nodesP, capturesP, epP, castlesP, promotionsP, checksP, endP.Sub(startP))
		fmt.Printf("seq d=%d nodes=%d (%s elapsed)\n", maxD, cntS, endS.Sub(startS))
	}
	return nil
}

func pdfs(b *board.Board, d, maxD int, desc string) (uint64, uint64, uint64, uint64, uint64, uint64) {
	// if d <= 2 {
	// 	fmt.Printf("pdfs depth=%d %s\n", d, desc)
	// }
	if d == maxD {
		return 1, 0, 0, 0, 0, 0
	}

	s := board.SideWhite
	if d%2 == 1 {
		s = board.SideBlack
	}

	var wg sync.WaitGroup
	nodesP, capturesP, epP, castlesP, promotionsP, checksP := uint64(0), uint64(0), uint64(0), uint64(0), uint64(0), uint64(0)
	mvs := b.GenerateMoves(s)
	for i, mv := range mvs {
		i, mv := i, mv
		wg.Add(1)
		go func() {
			defer wg.Done()
			bb := b.Clone()
			bb.Apply(mv)
			nodesPC, capturesPC, epPC, castlesPC, promotionsPC, checksPC := pdfs(bb, d+1, maxD, fmt.Sprintf("%s(%d/%d)", desc, i+1, len(mvs)))
			if mv.IsCapture {
				capturesPC++
			}
			if mv.IsEnPassant {
				epPC++
			}
			if mv.IsCastle != board.CastleDirectionUnknown {
				castlesPC++
			}
			if mv.IsPromote != board.PieceUnknown {
				promotionsPC++
			}
			if mv.IsCheck {
				checksPC++
			}
			atomic.AddUint64(&nodesP, nodesPC)
			atomic.AddUint64(&capturesP, capturesPC)
			atomic.AddUint64(&epP, epPC)
			atomic.AddUint64(&castlesP, castlesPC)
			atomic.AddUint64(&promotionsP, promotionsPC)
			atomic.AddUint64(&checksP, checksPC)
		}()
	}

	wg.Wait()
	return nodesP, capturesP, epP, castlesP, promotionsP, checksP
}

func dfs(b *board.Board, d, maxD int, desc string) uint64 {
	// if d <= 2 {
	// 	fmt.Printf("dfs depth=%d %s\n", d, desc)
	// }
	if d == maxD {
		return 1
	}

	s := board.SideWhite
	if d%2 == 1 {
		s = board.SideBlack
	}

	var cnt uint64
	mvs := b.GenerateMoves(s)
	for i, mv := range mvs {
		bb := b.Clone()
		bb.Apply(mv)
		cnt += dfs(bb, d+1, maxD, fmt.Sprintf("%s(%d/%d)", desc, i+1, len(mvs)))
	}
	return cnt
}

func dumpMoves(mvs []*board.Move) {
	for i, mv := range mvs {
		fmt.Printf("option %d: [%s] %s %s %s => %s (captures=%v) (checks=%v) (enpassants=%v) (promotes=%s)\n",
			i, mv, mv.Side, mv.Piece, mv.From, mv.To, mv.IsCapture, mv.IsCheck, mv.IsEnPassant, mv.IsPromote)
	}
}

func dumpHistor(mvs []*board.Move) {
	for i, mv := range mvs {
		if mv.Side == board.SideWhite {
			fmt.Printf("%d.", i/2+1)
		}
		fmt.Printf("%s ", mv)
	}
	fmt.Println()
}
