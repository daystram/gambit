package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/daystram/gambit/board"
)

func step() error {
	log.Println("============ step")
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

		t1 := time.Now()
		mvs := b.GenerateMoves(turn)
		t2 := time.Now()
		timesGenerateMoves = append(timesGenerateMoves, t2.Sub(t1))
		if len(mvs) == 0 {
			return fmt.Errorf("unexpected move exhaustion: state=%s", b.State())
		}
		mv := mvs[rand.Intn(len(mvs))]

		t1 = time.Now()
		b.Apply(mv)
		t2 = time.Now()
		timesApply = append(timesApply, t2.Sub(t1))

		t1 = time.Now()
		st := b.State()
		t2 = time.Now()
		timesState = append(timesState, t2.Sub(t1))

		fmt.Printf("\n===== [#%d] %s: %s\n", step/2+1, mv.Side, mv)
		fmt.Println(b.Draw())
		fmt.Println(b.DebugString())
		switch {
		case !st.IsRunning():
			break stepLoop
		case st.IsCheck():
			<-time.Tick(100 * time.Millisecond)
			fallthrough
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

	fmt.Println()
	fmt.Println(b.State())
	fmt.Println("genmv:", avg(timesGenerateMoves))
	fmt.Println("apply:", avg(timesApply))
	fmt.Println("state:", avg(timesState))
	return nil
}

func dumpHistory(mvs []*board.Move) {
	for i, mv := range mvs {
		if mv.Side == board.SideWhite {
			fmt.Printf("%d.", i/2+1)
		}
		fmt.Printf("%s ", mv)
	}
	fmt.Println()
}