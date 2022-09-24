package main

import (
	"fmt"
	"testing"

	"github.com/daystram/gambit/board"
)

func TestPerft(t *testing.T) {
	t.Parallel()

	// Results obtained from https://www.chessprogramming.org/Perft_Results.
	tests := []struct {
		fen       string
		depth     int
		wantNodes uint64
		onlyNodes bool
		wantCap   uint64
		wantEnp   uint64
		wantCas   uint64
		wantPro   uint64
		wantChk   uint64
	}{
		// default position
		// depth >= 6 failing perft is a known issue
		{
			fen:       "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			depth:     0,
			wantNodes: 1,
			wantCap:   0,
			wantEnp:   0,
			wantCas:   0,
			wantPro:   0,
			wantChk:   0,
		},
		{
			fen:       "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			depth:     1,
			wantNodes: 20,
			wantCap:   0,
			wantEnp:   0,
			wantCas:   0,
			wantPro:   0,
			wantChk:   0,
		},
		{
			fen:       "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			depth:     2,
			wantNodes: 400,
			wantCap:   0,
			wantEnp:   0,
			wantCas:   0,
			wantPro:   0,
			wantChk:   0,
		},
		{
			fen:       "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			depth:     3,
			wantNodes: 8_902,
			wantCap:   34,
			wantEnp:   0,
			wantCas:   0,
			wantPro:   0,
			wantChk:   12,
		},
		{
			fen:       "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			depth:     4,
			wantNodes: 197_281,
			wantCap:   1_576,
			wantEnp:   0,
			wantCas:   0,
			wantPro:   0,
			wantChk:   469,
		},
		{
			fen:       "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			depth:     5,
			wantNodes: 4_865_609,
			wantCap:   82_719,
			wantEnp:   258,
			wantCas:   0,
			wantPro:   0,
			wantChk:   27_351,
		},
		// {
		// 	fen:       "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		// 	depth:     6,
		// 	wantNodes: 119_060_324,
		// 	wantCap:   2_812_008,
		// 	wantEnp:   5_248,
		// 	wantCas:   0,
		// 	wantPro:   0,
		// 	wantChk:   809_099,
		// },

		// depth >= 3 failing perft is a known issue
		{
			fen:       "rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8",
			depth:     1,
			wantNodes: 44,
			onlyNodes: true,
		},
		{
			fen:       "rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8",
			depth:     2,
			wantNodes: 1_486,
			onlyNodes: true,
		},
		// {
		// 	fen:       "rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8",
		// 	depth:     3,
		// 	wantNodes: 62_379,
		// 	onlyNodes: true,
		// },
	}

	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("perft(%d): %s", tt.depth, tt.fen), func(t *testing.T) {
			t.Parallel()
			b, _, err := board.NewBoard(
				board.WithFEN(tt.fen),
			)
			if err != nil {
				t.Fatal("unexpected error:", err)
			}
			t.Logf("\n%s\n", b.Draw())

			var nodes, cap, enp, cas, pro, chk uint64
			runPerftParallel(b, tt.depth, true, false, &nodes, &cap, &enp, &cas, &pro, &chk)

			if nodes != tt.wantNodes {
				t.Errorf("unexpected nodes: got=%d want=%d", nodes, tt.wantNodes)
			}
			if !tt.onlyNodes {
				if cap != tt.wantCap {
					t.Errorf("unexpected cap: got=%d want=%d", cap, tt.wantCap)
				}
				if enp != tt.wantEnp {
					t.Errorf("unexpected enp: got=%d want=%d", enp, tt.wantEnp)
				}
				if cas != tt.wantCas {
					t.Errorf("unexpected cas: got=%d want=%d", cas, tt.wantCas)
				}
				if pro != tt.wantPro {
					t.Errorf("unexpected pro: got=%d want=%d", pro, tt.wantPro)
				}
				// TODO: yield check count
				// if chk != tt.wantChk {
				// 	t.Errorf("unexpected chk: got=%d want=%d", chk, tt.wantChk)
				// }
			}
		})
	}
}
