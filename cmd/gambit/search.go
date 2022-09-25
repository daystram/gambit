package main

import (
	"context"
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
	e := engine.NewEngine(&engine.EngineConfig{})
	fmt.Println(b.Draw())
	fmt.Println(b.FEN())
	fmt.Println(b.DebugString())
	initialBoard := b.Clone()

	playingSide := b.Turn()
	getMove := func(b *board.Board) *board.Move {
		if b.Turn() == playingSide {
			mv, err := e.Search(context.Background(), b, &engine.SearchConfig{
				MaxDepth: 12,
				Timeout:  30 * time.Second,
				Debug:    true,
			})
			if err != nil {
				panic(err)
			}
			fmt.Println(e.TranspositionStats())
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
			fmt.Printf("\n>>> %s\n", board.SideWhite)
			mv := getMove(b)
			b.Apply(mv)
			history = append(history, mv)
			fmt.Printf("--- %s\n", mv)

			fmt.Println(b.FEN())
			fmt.Println(b.Draw())
			if !b.State().IsRunning() {
				break
			}
			<-time.Tick(2 * time.Millisecond)
		}

		// Black's move
		if b.Turn() == board.SideBlack {
			fmt.Printf("\n>>> %s\n", board.SideBlack)
			mv := getMove(b)
			b.Apply(mv)
			history = append(history, mv)
			fmt.Printf("--- %s\n", mv)

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
	fmt.Println(engine.DumpHistory(initialBoard, history))

	return nil
}
