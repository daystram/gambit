package main

import (
	"errors"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/daystram/gambit/board"
)

const (
	usage = "usage: gambit [movegen|step|perft|search]"
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

	case "perft":
		depth, fen := 5, board.DefaultStartingPositionFEN
		if len(args) > 1 {
			depth, err = strconv.Atoi(args[1])
			if err != nil {
				return err
			}
			if len(args) > 2 {
				fen = strings.Join(args[2:], " ")
			}
		}
		err = perft(depth, fen)
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
