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

func search(fen string, steps, maxDepth, timeout int) error {
	rand.Seed(time.Now().Unix())
	b, _ := board.NewBoard(board.WithFEN(fen))
	e := engine.NewEngine(&engine.EngineConfig{
		HashTableSize: engine.DefaultHashTableSizeMB,
	})
	fmt.Println(b.Draw())
	fmt.Println(b.FEN())
	fmt.Println(b.DebugString())
	initialBoard := b.Clone()
	playingSide := b.Turn()

	searchCfg := &engine.SearchConfig{
		ClockConfig: engine.ClockConfig{
			Movetime: time.Duration(timeout) * time.Millisecond,
			Depth:    uint8(maxDepth),
		},
		Debug: true,
	}

	getMove := func(ctx context.Context, b *board.Board) board.Move {
		if b.Turn() == playingSide {
			mv, err := e.Search(ctx, b, searchCfg)
			if err != nil {
				panic(err)
			}
			return mv
		} else {
			mvs := b.GeneratePseudoLegalMoves()
			return mvs[rand.Intn(len(mvs))]
		}
	}

	ctx := context.Background()
	var history []board.Move
	for step := 1; step <= steps; step++ {
		fmt.Printf("\n=============== Move %d\n", b.FullMoveClock())

		// White's move
		if b.Turn() == board.SideWhite {
			fmt.Printf("\n>>> %s\n", board.SideWhite)
			mv := getMove(ctx, b)
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
			mv := getMove(ctx, b)
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
