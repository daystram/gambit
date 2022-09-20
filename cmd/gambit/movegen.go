package main

import (
	"fmt"
	"log"

	"github.com/daystram/gambit/board"
)

func movegen(fen string) error {
	log.Println("============ movegen")
	b, t, err := board.NewBoard(board.WithFEN(fen))
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
	return nil
}

func dumpMoves(mvs []*board.Move) {
	for i, mv := range mvs {
		fmt.Printf("option %d: [%s] %s %s %s => %s (captures=%v) (checks=%v) (enpassants=%v) (promotes=%s)\n",
			i, mv, mv.Side, mv.Piece, mv.From, mv.To, mv.IsCapture, mv.IsCheck, mv.IsEnPassant, mv.IsPromote)
	}
}
