package board

import "github.com/daystram/gambit/position"

type Move struct {
	From, To position.Pos
	Piece    Piece

	IsTurn      Side
	IsCapture   bool
	IsCheck     bool
	IsCastle    CastleDirection
	IsEnPassant bool
	IsPromote   Piece
}

func (m Move) String() string {
	return m.Algebra()
}

func (m Move) Algebra() string {
	if m.IsCastle != CastleDirectionUnknown {
		if m.IsCastle.IsRight() {
			return "0-0"
		}
		return "0-0-0"
	}
	nt := m.Piece.SymbolAlgebra(SideWhite) // SideWhite because it returns capital symbols
	if m.IsCapture {
		if m.Piece == PiecePawn {
			nt += m.From.X().NotationComponentX()
		} else {
			nt += m.From.Notation()
		}
		nt += "x"
	}
	nt += m.To.Notation()
	if m.IsPromote != PieceUnknown {
		nt += m.IsPromote.SymbolAlgebra(SideWhite)
	}
	if m.IsCheck {
		nt += "+"
	}
	if m.IsEnPassant {
		nt += " e.p."
	}
	return nt
}

func (m Move) UCI() string {
	return m.From.Notation() + m.To.Notation() + m.IsPromote.SymbolAlgebra(SideBlack)
}
