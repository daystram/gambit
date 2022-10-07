package board

import "github.com/daystram/gambit/position"

type Move struct {
	From, To position.Pos
	Piece    Piece

	IsTurn      Side
	IsCapture   bool
	IsCheck     bool // not populated on move generation
	IsCastle    CastleDirection
	IsEnPassant bool
	IsPromote   Piece

	Score uint8 // used for move ordering
}

func (mv Move) IsNull() bool {
	return mv.Piece == PieceUnknown
}

func (mv Move) Equals(other Move) bool {
	return mv.From == other.From &&
		mv.To == other.To &&
		mv.IsTurn == other.IsTurn &&
		mv.IsCapture == other.IsCapture &&
		mv.IsCastle == other.IsCastle &&
		mv.IsEnPassant == other.IsEnPassant &&
		mv.IsPromote == other.IsPromote
}

func (mv Move) String() string {
	return mv.Algebra()
}

func (mv Move) Algebra() string {
	if mv.IsCastle != CastleDirectionUnknown {
		if mv.IsCastle.IsRight() {
			return "0-0"
		}
		return "0-0-0"
	}
	nt := mv.Piece.SymbolAlgebra(SideWhite) // SideWhite because it returns capital symbols
	if mv.IsCapture {
		if mv.Piece == PiecePawn {
			nt += mv.From.X().NotationComponentX()
		} else {
			nt += mv.From.Notation()
		}
		nt += "x"
	}
	nt += mv.To.Notation()
	if mv.IsPromote != PieceUnknown {
		nt += mv.IsPromote.SymbolAlgebra(SideWhite)
	}
	if mv.IsCheck {
		nt += "+"
	}
	if mv.IsEnPassant {
		nt += " e.p."
	}
	return nt
}

func (mv Move) UCI() string {
	return mv.From.Notation() + mv.To.Notation() + mv.IsPromote.SymbolAlgebra(SideBlack)
}
