package main

import (
	"errors"
	"log"
	"os"
	"strconv"

	"github.com/daystram/gambit/board"
)

const (
	usage = "usage: gambit [movegen|step|perft]"
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
	switch args[0] {
	case "movegen":
		fen := board.DefaultStartingPositionFEN
		if len(args) == 2 {
			fen = args[1]
		}
		err := movegen(fen)
		if err != nil {
			return err
		}

	case "step":
		err := step()
		if err != nil {
			return err
		}

	case "perft":
		depth, err := strconv.Atoi(args[1])
		if err != nil {
			return err
		}
		err = perft(depth)
		if err != nil {
			return err
		}

	default:
		return errors.New(usage)
	}
	return nil
}
