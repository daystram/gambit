package bench

import (
	"fmt"
	"testing"

	"github.com/daystram/gambit/board"
)

func TestPerft(t *testing.T) {
	t.Parallel()

	// Results obtained from https://www.chessprogramming.org/Perft_Results.
	tests := map[string][]struct {
		depth     int
		wantNodes uint64
		onlyNodes bool
		wantCap   uint64
		wantEnp   uint64
		wantCas   uint64
		wantPro   uint64
		wantChk   uint64
	}{
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1": {
			{
				depth:     0,
				wantNodes: 1,
				wantCap:   0,
				wantEnp:   0,
				wantCas:   0,
				wantPro:   0,
				wantChk:   0,
			},
			{
				depth:     1,
				wantNodes: 20,
				wantCap:   0,
				wantEnp:   0,
				wantCas:   0,
				wantPro:   0,
				wantChk:   0,
			},
			{
				depth:     2,
				wantNodes: 400,
				wantCap:   0,
				wantEnp:   0,
				wantCas:   0,
				wantPro:   0,
				wantChk:   0,
			},
			{
				depth:     3,
				wantNodes: 8_902,
				wantCap:   34,
				wantEnp:   0,
				wantCas:   0,
				wantPro:   0,
				wantChk:   12,
			},
			{
				depth:     4,
				wantNodes: 197_281,
				wantCap:   1_576,
				wantEnp:   0,
				wantCas:   0,
				wantPro:   0,
				wantChk:   469,
			},
			{
				depth:     5,
				wantNodes: 4_865_609,
				wantCap:   82_719,
				wantEnp:   258,
				wantCas:   0,
				wantPro:   0,
				wantChk:   27_351,
			},
			{
				depth:     6,
				wantNodes: 119_060_324,
				wantCap:   2_812_008,
				wantEnp:   5_248,
				wantCas:   0,
				wantPro:   0,
				wantChk:   809_099,
			},
		},
		"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1": {
			{
				depth:     2,
				wantNodes: 2039,
				wantCap:   351,
				wantEnp:   1,
				wantCas:   91,
				wantPro:   0,
				wantChk:   3,
			},
			{
				depth:     3,
				wantNodes: 97862,
				wantCap:   17102,
				wantEnp:   45,
				wantCas:   3162,
				wantPro:   0,
				wantChk:   993,
			},
		},
		"rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8": {
			{
				depth:     1,
				wantNodes: 44,
				onlyNodes: true,
			},
			{
				depth:     2,
				wantNodes: 1_486,
				onlyNodes: true,
			},
			{
				depth:     3,
				wantNodes: 62_379,
				onlyNodes: true,
			},
		},
	}

	for fen, constraints := range tests {
		for _, tt := range constraints {
			tt := tt
			t.Run(fmt.Sprintf("perft(%d): %s", tt.depth, fen), func(t *testing.T) {
				// t.Parallel()
				b, _, err := board.NewBoard(
					board.WithFEN(fen),
				)
				if err != nil {
					t.Fatal("unexpected error:", err)
				}

				var nodes, cap, enp, cas, pro, chk uint64
				runPerft(b, tt.depth, true, false, nil, &nodes, &cap, &enp, &cas, &pro, &chk)

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
				}
			})
		}
	}
}
