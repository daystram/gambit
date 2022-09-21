package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/daystram/gambit/board"
)

func movegen(fen string) error {
	log.Println("============ movegen")
	b, t, err := board.NewBoard(board.WithFEN(fen))
	if err != nil {
		return err
	}
	fmt.Println("to move:", t)
	fmt.Println(b.Dump())
	fmt.Println(b.Draw())
	fmt.Println(b.State())
	dumpMoves(b.GenerateMoves(t))

	for _, mv := range b.GenerateMoves(t) {
		bb := b.Clone()
		bb.Apply(mv)
		fmt.Println(mv)
		fmt.Println(bb.Draw())
		fmt.Println(bb.FEN())
	}
	return nil
}

func dumpMoves(mvs []*board.Move) {
	for i, mv := range mvs {
		fmt.Printf("option %*d: [%s] %s %s %s => %s (cap=%v) (enp=%v) (cas=%s) (pro=%s) (chk=%v)\n",
			len(strconv.Itoa(len(mvs))), i, mv, mv.IsTurn, mv.Piece, mv.From, mv.To, mv.IsCapture, mv.IsEnPassant, mv.IsCastle, mv.IsPromote, mv.IsCheck)
	}
}
