package position

import (
	"errors"
	"testing"
)

func TestNewPosFromNotation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		notation string
		want     Pos
		wantErr  error
	}{
		{
			name:     "ok 1",
			notation: "e4",
			want:     Pos(28),
			wantErr:  nil,
		},
		{
			name:     "ok 2",
			notation: "h8",
			want:     Pos(63),
			wantErr:  nil,
		},
		{
			name:     "ok 3",
			notation: "a1",
			want:     Pos(0),
			wantErr:  nil,
		},
		{
			name:     "bad 1",
			notation: "",
			wantErr:  ErrInvalidNotation,
		},
		{
			name:     "bad 2",
			notation: "a",
			wantErr:  ErrInvalidNotation,
		},
		{
			name:     "bad 3",
			notation: "4",
			wantErr:  ErrInvalidNotation,
		},
		{
			name:     "bad 4",
			notation: "m4",
			wantErr:  ErrInvalidNotation,
		},
		{
			name:     "bad 5",
			notation: "e9",
			wantErr:  ErrInvalidNotation,
		},
		{
			name:     "bad 6",
			notation: "e0",
			wantErr:  ErrInvalidNotation,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewPosFromNotation(tt.notation)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("unexpected error: got=%v want=%v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("unexpected result: got=%v want=%v", got, tt.want)
			}
		})
	}
}
