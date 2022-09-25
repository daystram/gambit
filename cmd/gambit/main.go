package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/daystram/gambit/board"
)

const (
	usage = "usage: gambit [uci|movegen|step|perft|search]"
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
	if len(args) == 0 {
		return errors.New(usage)
	}
	var err error
	switch args[0] {
	case "uci":
		return runUCI()

	case "test":
		err = test()
		if err != nil {
			return err
		}

	case "movegen":
		fen := board.DefaultStartingPositionFEN
		if len(args) > 1 {
			fen = strings.Join(args[1:], " ")
		}
		err = movegen(fen)
		if err != nil {
			return err
		}

	case "step":
		err = step()
		if err != nil {
			return err
		}

	case "search":
		steps := 50
		if len(args) > 1 {
			steps, err = strconv.Atoi(args[1])
			if err != nil {
				return err
			}
		}
		err = search(steps)
		if err != nil {
			return err
		}

	default:
		return errors.New(usage)
	}
	return nil
}

func test() error {
	b, _, _ := board.NewBoard(
		// board.WithFEN("4R3/1q2q2b/8/5pn1/b3K1Pr/3k1p2/3nrN2/8 w - - 0 1"),
		// board.WithFEN("7K/P7/8/8/6p1/2Rb4/PpP1P1P1/8 w - b3 0 1"),
		// board.WithFEN("k7/8/8/4pPp1/3K4/8/8/8 w - e6 0 1"),
		// board.WithFEN("k7/N2N4/8/4q3/8/5N2/1K1Q4/8 w - g6 0 1"),
		// board.WithFEN("k7/8/8/8/5N2/3Q4/8/K7 w - - 0 1"),
		// board.WithFEN("k7/8/5q2/2B5/7B/2K5/7B/8 w - - 0 1"),
		// board.WithFEN("k7/8/4Rq2/8/8/5R2/8/K7 w - - 0 1"),
		// board.WithFEN("k7/8/5q2/8/7Q/8/8/K7 w - - 0 1"),
		// board.WithFEN("3rr3/8/8/8/8/2PKn3/3p4/8 w - - 0 1"),
		// board.WithFEN("k7/8/3r2R1/6q1/8/2R5/2nK1B2/4r3 w - - 0 1"),
		// board.WithFEN("7k/2r5/8/8/8/2K1B1r1/8/8 w - - 0 1"),
		board.WithFEN("rnbqkbnr/pppppppp/7b/8/3P4/1NBQ4/PPP1PPPP/R3K1NR w KQkq - 0 1"),
	)

	fmt.Println(b.Draw())
	mvs := b.GenerateMoves()

	dumpMoves(mvs)
	for i, mv := range mvs {
		bb := b.Clone()
		bb.Apply(mv)
		fmt.Println("================ option:", i, mv)
		fmt.Println(bb.Draw())
		fmt.Println(bb.FEN())
	}
	return nil
}
