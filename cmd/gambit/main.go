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
	searchMaxDepth = flag.Int("search.maxdepth", 0, "search max depth in search mode")
	searchTimeout  = flag.Int("search.timeout", 0, "search timeout in seconds in search mode")
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
		return search(50, *searchMaxDepth, *searchTimeout)
	}

	return runUCI()
}
