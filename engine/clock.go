package engine

import (
	"context"
	"math"
	"time"

	"github.com/daystram/gambit/board"
)

const (
	DefaultMovetime = 10 * time.Second

	MaxMovetime       = 24 * time.Hour
	MaxDepth    uint8 = 255
	MaxNodes          = math.MaxUint32

	minMovetime = 350 * time.Millisecond

	expectedGameMoves         uint8 = 40
	movetimeAccumulationRatio       = 0.8
	movetimeMargin                  = 100 * time.Millisecond
)

type ClockMode uint8

const (
	ClockModeInfinite ClockMode = iota
	ClockModeMovetime
	ClockModeGametime
	ClockModeDepth
	ClockModeNodes
)

type Clock struct {
	mode              ClockMode
	allocatedMovetime time.Duration
	allocatedDepth    uint8
	allocatedNodes    uint32

	done   bool
	stopCh chan struct{}
}

func NewClock() *Clock {
	return &Clock{
		done:   true,
		stopCh: make(chan struct{}),
	}
}

type ClockConfig struct {
	WhiteTime      time.Duration
	BlackTime      time.Duration
	WhiteIncrement time.Duration
	BlackIncrement time.Duration

	Movetime time.Duration

	Depth uint8

	Nodes uint32
}

func (c *Clock) Start(ctx context.Context, turn board.Side, fullMoveClock uint8, cfg *ClockConfig) {
	c.Stop()
	c.allocatedMovetime = MaxMovetime
	c.allocatedDepth = MaxDepth
	c.allocatedNodes = MaxNodes
	c.done = false

	if cfg.Movetime != 0 || cfg.WhiteTime != 0 || cfg.BlackTime != 0 {
		if cfg.Movetime != 0 {
			// movetime constraint
			c.mode = ClockModeMovetime
			c.allocatedMovetime = cfg.Movetime
		} else {
			// game clock constraint
			// TODO: improve heuristics
			c.mode = ClockModeGametime
			phase := max(int64(expectedGameMoves-fullMoveClock), 1)
			if turn == board.SideWhite {
				c.allocatedMovetime = time.Duration(float64(cfg.WhiteTime)/float64(phase)) + time.Duration(float64(cfg.WhiteIncrement)*(1-movetimeAccumulationRatio))
			} else {
				c.allocatedMovetime = time.Duration(float64(cfg.BlackTime)/float64(phase)) + time.Duration(float64(cfg.BlackIncrement)*(1-movetimeAccumulationRatio))
			}
		}
		if c.allocatedMovetime < minMovetime {
			c.allocatedMovetime = minMovetime
		}
	} else if cfg.Depth != 0 {
		c.mode = ClockModeDepth
		c.allocatedDepth = cfg.Depth
		if c.allocatedDepth > MaxDepth {
			c.allocatedDepth = MaxDepth
		}
	} else if cfg.Nodes != 0 {
		c.mode = ClockModeNodes
		c.allocatedNodes = cfg.Nodes
		if c.allocatedNodes > MaxNodes {
			c.allocatedNodes = MaxNodes
		}
	} else {
		c.mode = ClockModeInfinite
	}

	go func() {
		var cancel context.CancelFunc
		if c.allocatedMovetime != 0 {
			ctx, cancel = context.WithTimeout(ctx, c.allocatedMovetime-movetimeMargin)
			defer cancel()
		}
		select {
		case <-ctx.Done():
		case <-c.stopCh:
		}
		c.done = true
	}()
}

func (c *Clock) Stop() {
	if !c.done {
		c.stopCh <- struct{}{}
	}
}

func (c *Clock) DoneByMovetime() bool {
	return c.done
}

func (c *Clock) DoneByDepth(depth uint8) bool {
	return depth > c.allocatedDepth
}

func (c *Clock) DoneByNodes(nodes uint32) bool {
	return nodes > c.allocatedNodes
}

func (c *Clock) Mode() ClockMode {
	return c.mode
}

func (c *Clock) AllocatedMovetime() time.Duration {
	return c.allocatedMovetime
}
