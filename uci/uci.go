package uci

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/daystram/gambit/bench"
	"github.com/daystram/gambit/board"
	"github.com/daystram/gambit/engine"
)

var (
	EngineName   = "Gambit"
	EngineAuthor = "Danny August Ramaputra"

	defaultOptions = options{
		debug:         false,
		timeout:       engine.DefaultTimeoutDuration,
		hashTableSize: engine.DefaultHashTableSize,
		parallelPerft: true,
	}
)

type options struct {
	debug         bool
	timeout       time.Duration
	hashTableSize uint64
	parallelPerft bool
}

type Interface struct {
	board   *board.Board
	engine  *engine.Engine
	options options

	engineRunning bool
	engineCancel  context.CancelFunc
}

func NewInterface() *Interface {
	return &Interface{
		options: defaultOptions,
	}
}

func (i *Interface) Run() error {
	ctx := context.Background()
	i.reset(ctx)

	reader := bufio.NewReader(os.Stdin)
	for {
		cmd, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		cmd = strings.TrimSpace(cmd)

		switch args := strings.Split(cmd, " "); args[0] {
		case "uci":
			i.commandUCI(ctx)
		case "ucinewgame":
			i.reset(ctx)
		case "isready":
			i.commandReady(ctx)
		case "setoption":
			i.commandSetOption(ctx, args[1:])
		case "position":
			i.commandPosition(ctx, args[1:])
		case "d":
			i.commandDraw(ctx)
		case "go":
			i.commandGo(ctx, args[1:])
		case "stop":
			i.commandStop(ctx)
		case "quit":
			return nil
		}
	}
}

func (i *Interface) commandUCI(_ context.Context) {
	i.println(fmt.Sprintf("id name %s", EngineName))
	i.println(fmt.Sprintf("id author %s", EngineAuthor))
	i.println(fmt.Sprintf("option Debug type check default %v", defaultOptions.debug))
	i.println(fmt.Sprintf("option Timeout type spin default %d min 100 max 3600000", defaultOptions.timeout.Milliseconds()))
	i.println(fmt.Sprintf("option Hash type spin default %d min 0 max 16777216", defaultOptions.hashTableSize))
	i.println("uciok")
}

func (i *Interface) commandReady(_ context.Context) {
	if i.board != nil && i.engine != nil {
		i.println("readyok")
	}
}

func (i *Interface) commandSetOption(_ context.Context, args []string) {
	// TODO: support comma separated names
	if len(args) < 4 || args[0] != "name" || args[2] != "value" {
		return
	}
	switch name, valueStr := strings.ToLower(args[1]), args[3]; name {
	case "debug":
		value, err := strconv.ParseBool(valueStr)
		if err != nil {
			return
		}
		i.options.debug = value
	case "timeout":
		value, err := strconv.ParseUint(valueStr, 10, 64)
		if err != nil || value < 100 || value > 3600000 {
			return
		}
		i.options.timeout = time.Duration(value * uint64(time.Millisecond))
	case "hash":
		value, err := strconv.ParseUint(valueStr, 10, 64)
		if err != nil || value > 1<<24 {
			return
		}
		i.options.hashTableSize = value
	}
}

func (i *Interface) commandPosition(_ context.Context, args []string) {
	if i.engineRunning || len(args) == 0 {
		return
	}

	var fen string
	switch args[0] {
	case "fen":
		fen = strings.Join(args[1:], " ")
	case "startpos":
		fen = board.DefaultStartingPositionFEN
	default:
		return
	}

	b, _, err := board.NewBoard(board.WithFEN(fen))
	if err != nil {
		return
	}
	i.board = b
}

func (i *Interface) commandDraw(_ context.Context) {
	i.println(i.board.Draw())
}

func (i *Interface) commandGo(ctx context.Context, args []string) {
	if len(args) > 0 {
		switch mode := args[0]; mode {
		case "perft":
			if len(args) != 2 {
				return
			}
			depth, err := strconv.Atoi(args[1])
			if err != nil {
				return
			}

			out := make(chan string, 64)
			go func() {
				for s := range out {
					i.println(s)
				}
			}()
			defer close(out)

			_ = bench.Perft(depth, i.board.FEN(), i.options.parallelPerft, true, out)
			return

		// TODO: search args, e.g. depth, nodes
		default:
			return
		}
	}

	go func() {
		engineCtx, engineCancel := context.WithCancel(ctx)
		i.engineCancel = engineCancel
		i.engineRunning = true
		defer engineCancel()

		bestMove, err := i.engine.Search(engineCtx, i.board, &engine.SearchConfig{
			MaxDepth: 15,
			Timeout:  i.options.timeout,
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			panic(err)
		}

		i.println(fmt.Sprintf("bestmove %s", bestMove.UCI()))
		i.engineRunning = false
	}()
}

func (i *Interface) commandStop(ctx context.Context) {
	if i.engineRunning {
		i.engineCancel()
	}
}

func (i *Interface) reset(ctx context.Context) {
	i.commandStop(ctx)
	i.commandPosition(ctx, []string{"startpos"})
	i.engine = engine.NewEngine(&engine.EngineConfig{
		HashTableSize: i.options.hashTableSize,
		Logger:        i.println,
	})
}

func (i *Interface) println(a ...any) {
	fmt.Fprintln(os.Stdout, a...)
}
