package board

import "testing"

func TestFEN(t *testing.T) {
	t.Parallel()
	tests := []struct {
		fen     string
		wantErr bool
	}{
		{fen: DefaultStartingPositionFEN, wantErr: false},
		{fen: "r3k2r/1bppqppp/p1n2n2/2b1p3/B3P3/2NP1N2/1PP2PPP/R1BQ1RK1 b kq - 2 10", wantErr: false},
		{fen: "r4rk1/1bpp1ppp/p2q4/2bPp3/8/1BPP1Q2/1P3PPP/R1B2RK1 b - - 2 15", wantErr: false},
		{fen: "8/5kBp/3p3P/5pb1/8/5P2/4R2K/3r4 b - - 8 52", wantErr: false},
		{fen: "rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 0 2", wantErr: false},
		{fen: "r4rk1/5ppp/p2p4/1bb1p3/BP6/2PP4/5PPP/R1B1R1K1 b - b3 0 20", wantErr: false},
		{fen: "8/7R/5B2/5P1k/p6p/P6P/6P1/7K b - - 2 58", wantErr: false},
		{fen: "r7/p4k2/4p2p/2B4N/4Pn2/2P2P2/PP2r1qP/R5K1 w - - 6 39", wantErr: false},
		{fen: "5k2/R7/4NN1p/p7/5P2/8/P1P3PP/3B2K1 b - - 7 30", wantErr: false},
		{fen: "3r1b1r/5pp1/7p/3P3k/3B2Q1/7N/P3BPK1/1R6 b - - 0 34", wantErr: false},
		{fen: "8/5k2/4N3/8/8/3K4/8/8 w - - 0 71", wantErr: false},
		{fen: "6k1/1p3p2/1P6/p6p/Pq5P/K4n2/3r4/8 w - - 4 56", wantErr: false},
		{fen: "1r3b1r/6pp/8/1p1pN3/3P1PQk/2P5/P7/qN3RK1 b - - 5 26", wantErr: false},
		{fen: "R4k1r/1pNQ3p/4ppp1/8/3Pb1q1/5N2/5PPP/4KB1R b K - 5 22", wantErr: false},
		{fen: "8/7Q/p7/3p4/5K1k/8/p3R3/8 b - - 9 79", wantErr: false},
		{fen: "8/5r2/p4nk1/PP3p2/3P1P1p/5PqK/8/8 w - - 1 50", wantErr: false},
		{fen: "3k4/1p4r1/p7/3p2qK/P5P1/8/2P2P1P/3RR3 w - - 9 42", wantErr: false},
		{fen: "3k2Q1/7R/6K1/5P2/1pP5/1P6/8/8 b - - 36 77", wantErr: false},
		{fen: "2R4Q/1kR5/5R2/4p3/3nP3/5P2/6P1/6K1 b - - 2 68", wantErr: false},
		{fen: "1n2k2r/4pp1p/6p1/8/3b3P/8/5q2/r1K5 w k - 2 31", wantErr: false},
		{fen: "8/8/5B2/p1p2bQk/1p6/8/1P1K4/8 b - - 5 61", wantErr: false},
		{fen: "6k1/1p3pbp/8/p1p1p2p/P5qK/8/1P1r1p1R/8 w - - 0 35", wantErr: false},
		{fen: "8/7k/3p3b/B1pPp3/PPN1PpP1/3P2q1/2QRK3/4r3 w - - 5 41", wantErr: false},
		{fen: "r1b2k2/pp2R2R/4p1P1/8/4prP1/8/PP4n1/2K5 b - - 11 31", wantErr: false},
		{fen: "1rb1B2Q/pp3k2/3Q4/3p3p/1P6/8/P1P2PPP/R1B1K2R b KQ - 1 22", wantErr: false},
		{fen: "8/7k/P5p1/2p1p3/8/2p2r1q/2Pr1p2/1R5K w - - 2 54", wantErr: false},
		{fen: "8/8/4pB2/3pPkQ1/b7/1p6/3N1P1K/8 b - - 1 59", wantErr: false},
		{fen: "7Q/8/8/8/5p2/6p1/6P1/3kr1K1 w - - 0 69", wantErr: false},
		{fen: "8/6qK/4P3/7P/6r1/8/8/3k4 w - - 3 66", wantErr: false},
		{fen: "8/1pQk4/8/5P2/5BP1/3P2P1/1P3P2/4R1K1 b - - 4 36", wantErr: false},
		{fen: "2rBk2r/p1p1Q3/5Pp1/1p1N4/4p3/8/PP3PPP/R5K1 b - - 1 24", wantErr: false},
		{fen: "7k/5pp1/p7/5P2/8/8/2q5/K5q1 w - - 4 62", wantErr: false},
		{fen: "8/r7/8/kn6/8/8/3Q4/1K6 b - - 15 89", wantErr: false},
		{fen: "3Q4/8/k2N4/R5B1/8/8/PP2K1PP/R7 b - - 4 46", wantErr: false},
		{fen: "4B3/3P2k1/3r2P1/8/8/8/5K2/8 b - - 32 74", wantErr: false},
		{fen: "r1b2rk1/ppp2pp1/2n2b2/8/3P3p/3p1q2/6q1/1R5K w - - 0 31", wantErr: false},
		{fen: "6Q1/4k2R/3p1p2/2nB1K2/3N3P/8/5b2/8 b - - 0 71", wantErr: false},
		{fen: "6k1/1p6/n1p2p1p/3p2p1/3P4/4P1q1/PPQN4/3RR1K1 w - - 8 34", wantErr: false},
		{fen: "8/3r1kb1/3Pp1pp/1N6/Q1P3q1/3RB3/P4P1K/5R2 b - - 8 48", wantErr: false},
		{fen: "4Q1k1/p7/1pp4q/4P2p/8/8/P7/K4b2 b - - 11 46", wantErr: false},
		{fen: "8/8/R7/Qk6/3K4/5p2/5p2/8 b - - 10 81", wantErr: false},
		{fen: "7r/1p3RQk/3Np3/7p/3P1Bp1/8/PP3PPP/4R1K1 b - - 0 26", wantErr: false},
		{fen: "r4rk1/ppb2ppp/8/8/B3p3/P7/1B3P1q/5RK1 w - - 4 29", wantErr: false},
		{fen: "4k1R1/7Q/4p3/8/1n3p2/3B4/1P3PP1/6K1 b - - 4 41", wantErr: false},
		{fen: "2r2rk1/5p1p/2P5/1p4n1/1n4P1/7K/5q2/8 w - - 1 45", wantErr: false},
		{fen: "r1b5/1p5p/7R/1p5k/7P/5P1R/PP1Bn2K/8 b - - 2 37", wantErr: false},
		{fen: "5Q2/1p3p2/4Rkpp/3P4/p5P1/5N2/Pb2BR1P/6K1 b - - 1 28", wantErr: false},
		{fen: "7R/6Q1/4N3/7k/4P3/2P5/5PPP/5BK1 b - - 4 54", wantErr: false},
		{fen: "2rr2k1/p1p1pp1p/2p3p1/8/PP3q1K/8/6q1/8 w - - 2 35", wantErr: false},
		{fen: "8/3Rn3/5Q2/p5kp/2B1P3/2P3bP/PP3R2/7K b - - 1 38", wantErr: false},
		{fen: "", wantErr: true},
		{fen: "invalid fen", wantErr: true},
		{fen: "8/3Rn3/5Q2/p5kp/2B1P3/2P3bP/PP3R2/7K badside - - 1 38", wantErr: true},
		{fen: "8/3Rn3/5Q2/p5kp/2B1P3/2P3bP/PP3R2/7K b badcastlingrights - 1 38", wantErr: true},
		{fen: "8/3Rn3/5Q2/p5kp/2B1P3/2P3bP/PP3R2/7K b badcastlingrights - -100 -100", wantErr: true},
		{fen: "8/3Rn3/badboard/p5kp/2B1P3/2P3bP/PP3R2/7K b - - 1 38", wantErr: true},
		{fen: "8/8/8/8/8/8/8/8 w - - 1 0", wantErr: true},
		{fen: "7k/8/8/8/8/1/8/7K w - - 1 0", wantErr: true},
		{fen: "7k/8/8/8/8//8/7K w - - 1 0", wantErr: true},
		{fen: "7k/8/8/8/8/8/7K w - - 1 0", wantErr: true},
		{fen: "7k/8/8/8/8/8/8/7K w - - 1 0 extrasegment", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.fen, func(t *testing.T) {
			t.Parallel()

			b, err := NewBoard(WithFEN(tt.fen))
			if tt.wantErr {
				if err == nil {
					t.Error("error expected: got=nil")
				}
				return
			}
			if err != nil {
				t.Fatal("unexpected error:", err)
			}

			if gotFEN := b.FEN(); gotFEN != tt.fen {
				t.Errorf("unexpected FEN: got=%s want=%s", gotFEN, tt.fen)
			}
		})
	}
}
