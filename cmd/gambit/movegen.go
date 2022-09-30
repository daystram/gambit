package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/daystram/gambit/board"
)

func movegen(fen string, draw bool) error {
	log.Println("============ movegen")
	b, t, err := board.NewBoard(board.WithFEN(fen))
	if err != nil {
		return err
	}
	fmt.Println("to move:", t)
	fmt.Println(b.Dump())
	fmt.Println(b.Draw())
	fmt.Println(b.State())
	dumpMoves(b)

	if draw {
		for _, mv := range b.GeneratePseudoLegalMoves() {
			if !b.IsLegal(mv) {
				continue
			}
			bb := b.Clone()
			bb.Apply(mv)
			fmt.Println(mv)
			fmt.Println(bb.Draw())
			fmt.Println(bb.FEN())
		}
	}
	return nil
}

func dumpMoves(b *board.Board) {
	mvs := b.GeneratePseudoLegalMoves()
	for i, mv := range mvs {
		if !b.IsLegal(mv) {
			continue
		}
		fmt.Printf("option %*d: [%s] [%s] %s %s %s => %s (cap=%v) (enp=%v) (cas=%s) (pro=%s)\n",
			len(strconv.Itoa(len(mvs))), i+1, mv.UCI(), mv.Algebra(), mv.IsTurn, mv.Piece, mv.From, mv.To, mv.IsCapture, mv.IsEnPassant, mv.IsCastle, mv.IsPromote)
	}
}
