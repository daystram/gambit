package board

type State uint8

const (
	// StateUnknown is when game state is unknown.
	StateUnknown State = iota

	// StateRunning is when game is on progress.
	StateRunning

	// StateCheckWhite is when White King is in check.
	StateCheckWhite

	// StateCheckBlack is when Black King is in check.
	StateCheckBlack

	// StateCheckmateWhite is when White King is in checkmate.
	StateCheckmateWhite

	// StateCheckmateBlack is when Black King is in checkmate.
	StateCheckmateBlack

	// StateStalemate is when a side cannot move a piece and King is not in check.
	StateStalemate

	// StateFiftyMoveViolated is when the game has gone through 50 moves without any captures or pawn moves.
	StateFiftyMoveViolated

	// TODO: lack of material
)

func (s State) IsRunning() bool {
	switch s {
	case StateRunning, StateCheckWhite, StateCheckBlack:
		return true
	default:
		return false
	}
}

func (s State) IsCheck() bool {
	switch s {
	case StateCheckWhite, StateCheckBlack:
		return true
	default:
		return false
	}
}

func (s State) IsCheckmate() bool {
	switch s {
	case StateCheckmateWhite, StateCheckmateBlack:
		return true
	default:
		return false
	}
}

func (s State) IsDraw() bool {
	switch s {
	case StateStalemate, StateFiftyMoveViolated:
		return true
	default:
		return false
	}
}

func (s State) String() string {
	switch s {
	case StateUnknown:
		return "StateUnknown"
	case StateRunning:
		return "StateRunning"
	case StateCheckWhite:
		return "StateCheckWhite"
	case StateCheckBlack:
		return "StateCheckBlack"
	case StateCheckmateWhite:
		return "StateCheckmateWhite"
	case StateCheckmateBlack:
		return "StateCheckmateBlack"
	case StateStalemate:
		return "StateStalemate"
	case StateFiftyMoveViolated:
		return "StateFiftyMoveViolated"
	default:
		return ""
	}
}
