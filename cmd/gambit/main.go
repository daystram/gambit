package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"

	"github.com/daystram/gambit/board"
)

var (
	profile = flag.Bool("profile", false, "serve pprof endpoint")

	movegenRun  = flag.Bool("movegen", false, "run movegen mode")
	movegenDraw = flag.Bool("movegen.draw", false, "draw applied moves in movegen mode")

	stepRun = flag.Bool("step", false, "run step mode")

	searchRun      = flag.Bool("search", false, "run search mode")
	searchDepth    = flag.Int("search.depth", 0, "search depth in search mode")
	searchMovetime = flag.Int("search.movetime", 0, "search movetime in milliseconds in search mode")
)

func main() {
	flag.Parse()

	if *profile {
		runProfiler()
	}

	err := realMain(os.Args[1:])
	if err != nil {
		log.Println(err)
		os.Exit(exitErr)
	}
	os.Exit(exitOK)
}

func runProfiler() {
	go func() {
		addr := "localhost:6060"
		log.Printf("starting pprof endpoint: http://%s/debug/pprof\n", addr)
		_ = http.ListenAndServe(addr, nil)
	}()
}

func realMain(args []string) error {
	fen := board.DefaultStartingPositionFEN
	if len(args) > 1 {
		fen = strings.Join(args[1:], " ")
	}
	if *movegenRun {
		return movegen(fen, *movegenDraw)
	}
	if *stepRun {
		return step(fen)
	}
	if *searchRun {
		return search(fen, 50, *searchDepth, *searchMovetime)
	}

	return runUCI()
}
