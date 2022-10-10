package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/daystram/gambit/board"
)

func movegen(fen string, draw bool) error {
	log.Println("============ movegen")
	b, err := board.NewBoard(board.WithFEN(fen))
	if err != nil {
		return err
	}
	fmt.Println("to move:", b.Turn())
	fmt.Println(b.Dump())
	fmt.Println(b.Draw())
	fmt.Println(b.State())
	dumpMoves(b)

	if draw {
		for _, mv := range b.GeneratePseudoLegalMoves() {
			unApply, ok := b.Apply(mv)
			if !ok {
				unApply()
				continue
			}
			fmt.Println(mv)
			fmt.Println(b.Draw())
			fmt.Println(b.FEN())
			unApply()
		}
	}
	return nil
}

func dumpMoves(b *board.Board) {
	mvs := b.GeneratePseudoLegalMoves()
	i := 0
	for _, mv := range mvs {
		if !b.IsLegal(mv) {
			continue
		}
		i++
		fmt.Printf("option %*d: [%s] [%s] %s %s %s => %s (cap=%v) (enp=%v) (cas=%s) (pro=%s)\n",
			len(strconv.Itoa(len(mvs))), i, mv.UCI(), mv.Algebra(), mv.IsTurn, mv.Piece, mv.From, mv.To, mv.IsCapture, mv.IsEnPassant, mv.IsCastle, mv.IsPromote)
	}
}
