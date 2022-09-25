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
	profile    = flag.Bool("profile", false, "serve pprof endpoint")
	runMovegen = flag.Bool("movegen", false, "run movegen mode")
	runStep    = flag.Bool("step", false, "run step mode")
	runSearch  = flag.Bool("search", false, "run search mode")
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
	if *runMovegen {
		fen := board.DefaultStartingPositionFEN
		if len(args) > 1 {
			fen = strings.Join(args[1:], " ")
		}
		return movegen(fen)
	}
	if *runStep {
		return step()
	}
	if *runSearch {
		return search(50)
	}

	return runUCI()
}
