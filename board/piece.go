package board

type Piece rune

const (
	PieceUnknown Piece = 0
	PiecePawn    Piece = 'P'
	PieceBishop  Piece = 'B'
	PieceKnight  Piece = 'N'
	PieceRook    Piece = 'R'
	PieceQueen   Piece = 'Q'
	PieceKing    Piece = 'K'
)

// PawnPromoteCandidates represents the candidates for pawn promotion.
var PawnPromoteCandidates = []Piece{PieceBishop, PieceKnight, PieceRook, PieceQueen}

func (p Piece) String() string {
	return p.Name()
}

func (p Piece) Name() string {
	switch p {
	case PiecePawn:
		return "Pawn"
	case PieceBishop:
		return "Bishop"
	case PieceKnight:
		return "Knight"
	case PieceRook:
		return "Rook"
	case PieceQueen:
		return "Queen"
	case PieceKing:
		return "King"
	default:
		return ""
	}
}

func (p Piece) SymbolFEN(s Side) string {
	switch s {
	case SideWhite:
		return string(p)
	case SideBlack:
		return string(p ^ 0x20) // lowercase is +32 uppercase
	default:
		return ""
	}
}

func (p Piece) SymbolAlgebra(s Side) string {
	if p == PiecePawn {
		return ""
	}
	switch s {
	case SideWhite:
		return string(p)
	case SideBlack:
		return string(p ^ 0x20) // lowercase is +32 uppercase
	default:
		return ""
	}
}

func (p Piece) SymbolUnicode(s Side, invert bool) string {
	if invert {
		s = s.Opposite()
	}
	switch s {
	case SideWhite:
		switch p {
		case PiecePawn:
			return "♙"
		case PieceBishop:
			return "♗"
		case PieceKnight:
			return "♘"
		case PieceRook:
			return "♖"
		case PieceQueen:
			return "♕"
		case PieceKing:
			return "♔"
		default:
			return ""
		}
	case SideBlack:
		switch p {
		case PiecePawn:
			return "♟"
		case PieceBishop:
			return "♝"
		case PieceKnight:
			return "♞"
		case PieceRook:
			return "♜"
		case PieceQueen:
			return "♛"
		case PieceKing:
			return "♚"
		default:
			return ""
		}
	default:
		return ""
	}
}
