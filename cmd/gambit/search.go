package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/daystram/gambit/board"
	"github.com/daystram/gambit/engine"
)

func search(steps int) error {
	rand.Seed(time.Now().Unix())
	b, _, _ := board.NewBoard()
	e := engine.NewEngine(&engine.EngineConfig{
		MaxDepth: 10,
		Timeout:  60 * time.Second,
	})
	fmt.Println(b.Draw())
	fmt.Println(b.FEN())
	fmt.Println(b.DebugString())

	playingSide := b.Turn()
	getMove := func(b *board.Board) *board.Move {
		if b.Turn() == playingSide {
			mv, err := e.Search(b)
			if err != nil {
				panic(err)
			}
			return mv
		} else {
			mvs := b.GenerateMoves()
			return mvs[rand.Intn(len(mvs))]
		}
	}

	var history []*board.Move
	for step := 1; step <= steps; step++ {
		fmt.Printf("\n=============== Move %d\n", b.FullMoveClock())

		// White's move
		if b.Turn() == board.SideWhite {
			mv := getMove(b)
			fmt.Println(e.TranspositionStats())
			b.Apply(mv)
			if b.State() == board.StateCheckBlack {
				mv.IsCheck = true
			}
			history = append(history, mv)

			fmt.Printf("\n>>> %s: %s\n", mv.IsTurn, mv)
			fmt.Println(b.FEN())
			fmt.Println(b.Draw())
			if !b.State().IsRunning() {
				break
			}
			<-time.Tick(2 * time.Millisecond)
		}

		// Black's move
		if b.Turn() == board.SideBlack {
			mv := getMove(b)
			b.Apply(mv)
			if b.State() == board.StateCheckWhite {
				mv.IsCheck = true
			}
			history = append(history, mv)

			fmt.Printf("\n>>> %s: %s\n", mv.IsTurn, mv)
			fmt.Println(b.FEN())
			fmt.Println(b.Draw())
			if !b.State().IsRunning() {
				break
			}
			<-time.Tick(2 * time.Second)
		}
	}
	log.Println("=============== game ended:", b.State())
	fmt.Println(b.FEN())
	dumpHistory(history)

	return nil
}

func dumpHistory(mvs []*board.Move) {
	for i, mv := range mvs {
		if mv.IsTurn == board.SideWhite {
			fmt.Printf("%d.", i/2+1)
		}
		fmt.Printf("%s ", mv)
	}
	fmt.Println()
}
