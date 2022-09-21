package board

type Piece uint8

const (
	PieceUnknown Piece = iota
	PiecePawn
	PieceBishop
	PieceKnight
	PieceRook
	PieceQueen
	PieceKing
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

func (p Piece) SymbolAlgebra(s Side) string {
	if p == PiecePawn {
		return ""
	}
	return p.SymbolFEN(s)
}

func (p Piece) SymbolFEN(s Side) string {
	var sym rune
	switch p {
	case PiecePawn:
		sym = 'P'
	case PieceBishop:
		sym = 'B'
	case PieceKnight:
		sym = 'N'
	case PieceRook:
		sym = 'R'
	case PieceQueen:
		sym = 'Q'
	case PieceKing:
		sym = 'K'
	default:
		return ""
	}
	if s == SideBlack {
		sym |= 0x20 // lowercase is +32 uppercase
	}
	return string(sym)
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
