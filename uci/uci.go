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
		hashTableSize: engine.DefaultHashTableSize,
		parallelPerft: true,
	}
)

type options struct {
	debug         bool
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
		args := strings.Fields(strings.TrimSpace(cmd))
		if len(args) == 0 {
			continue
		}

		switch args[0] {
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
	i.println(fmt.Sprintf("option name Debug type check default %v", defaultOptions.debug))
	i.println(fmt.Sprintf("option name Hash type spin default %d min 0 max 16777216", defaultOptions.hashTableSize))
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
	case "hash":
		value, err := strconv.ParseUint(valueStr, 10, 64)
		if err != nil || value > 1<<24 {
			return
		}
		i.options.hashTableSize = value
	case "parallelperft":
		value, err := strconv.ParseBool(valueStr)
		if err != nil {
			return
		}
		i.options.parallelPerft = value
	}
}

func (i *Interface) commandPosition(_ context.Context, args []string) {
	if i.engineRunning || len(args) == 0 {
		return
	}

	var fen string
	switch args[0] {
	case "fen":
		if len(args) < 7 {
			panic("bad fen")
		}
		fen = strings.Join(args[1:7], " ")
		args = args[7:]
	case "startpos":
		fen = board.DefaultStartingPositionFEN
		args = args[1:]
	default:
		return
	}

	b, _, err := board.NewBoard(board.WithFEN(fen))
	if err != nil {
		return
	}

	if len(args) > 0 && args[0] == "moves" {
		for _, notation := range args[1:] {
			mv, err := b.NewMoveFromUCI(notation)
			if err != nil {
				return
			}
			b.Apply(mv)
		}
	}

	i.board = b
}

func (i *Interface) commandDraw(_ context.Context) {
	i.println(i.board.Draw())
	i.println(i.board.FEN())
}

func (i *Interface) commandGo(ctx context.Context, args []string) {
	cfg := &engine.SearchConfig{
		MaxDepth: engine.DefaultDepth,
		Timeout:  engine.DefaultTimeoutDuration,
	}
	if len(args) > 0 {
		switch args[0] {
		case "infinite":
			cfg.MaxDepth = engine.MaxDepth
			cfg.Timeout = engine.MaxTimeoutDuration

		case "depth":
			if len(args) != 2 {
				return
			}
			depth, err := strconv.ParseUint(args[1], 10, 8)
			if err != nil {
				return
			}
			cfg.MaxDepth = uint8(depth)

		case "movetime":
			if len(args) != 2 {
				return
			}
			timeout, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return
			}
			cfg.Timeout = time.Duration(timeout * uint64(time.Millisecond))

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

		bestMove, err := i.engine.Search(engineCtx, i.board, cfg)
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
