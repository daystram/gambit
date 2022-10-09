package board

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/daystram/gambit/position"
)

func parseFEN(fen string) (sideBitmaps, pieceBitmaps, cellList, sideValue, sideValue, sideValue, CastleRights, bitmap, uint8, uint8, Side, uint64, error) {
	segments := strings.Split(fen, " ")
	if len(segments) != 6 {
		return sideBitmaps{}, pieceBitmaps{}, cellList{}, sideValue{}, sideValue{}, sideValue{}, CastleRights(0), bitmap(0), 0, 0, SideUnknown, 0, fmt.Errorf("%w: incorrect number of segments", ErrInvalidFEN)
	}

	var sides sideBitmaps
	var pieces pieceBitmaps
	var cells cellList
	var materialValue sideValue
	var positionValueMG sideValue
	var positionValueEG sideValue
	var hash uint64
	rows := strings.Split(segments[0], "/")
	if len(rows) != int(Height) {
		return sideBitmaps{}, pieceBitmaps{}, cellList{}, sideValue{}, sideValue{}, sideValue{}, CastleRights(0), bitmap(0), 0, 0, SideUnknown, 0, fmt.Errorf("%w: invalid board configuration", ErrInvalidFEN)
	}
	for y := position.Pos(0); y < Height; y++ {
		ptrX, ptrY := -1, Height-y-1
		for x := position.Pos(0); x < Width; x++ {
			ptrX++
			if ptrX >= len(rows[ptrY]) {
				return sideBitmaps{}, pieceBitmaps{}, cellList{}, sideValue{}, sideValue{}, sideValue{}, CastleRights(0), bitmap(0), 0, 0, SideUnknown, 0, fmt.Errorf("%w: missing cells", ErrInvalidFEN)
			}
			var s Side
			var p Piece
			switch cell := rune(rows[ptrY][ptrX]); cell {
			case 'P':
				s, p = SideWhite, PiecePawn
			case 'B':
				s, p = SideWhite, PieceBishop
			case 'N':
				s, p = SideWhite, PieceKnight
			case 'R':
				s, p = SideWhite, PieceRook
			case 'Q':
				s, p = SideWhite, PieceQueen
			case 'K':
				s, p = SideWhite, PieceKing
			case 'p':
				s, p = SideBlack, PiecePawn
			case 'b':
				s, p = SideBlack, PieceBishop
			case 'n':
				s, p = SideBlack, PieceKnight
			case 'r':
				s, p = SideBlack, PieceRook
			case 'q':
				s, p = SideBlack, PieceQueen
			case 'k':
				s, p = SideBlack, PieceKing
			default:
				if cell != '0' && unicode.IsDigit(cell) {
					skip := position.Pos(cell - '0')
					if skip != 0 && x+skip-1 < Width {
						x += skip - 1
						continue
					}
					return sideBitmaps{}, pieceBitmaps{}, cellList{}, sideValue{}, sideValue{}, sideValue{}, CastleRights(0), bitmap(0), 0, 0, SideUnknown, 0, fmt.Errorf("%w: skip out of bounds", ErrInvalidFEN)
				}
				return sideBitmaps{}, pieceBitmaps{}, cellList{}, sideValue{}, sideValue{}, sideValue{}, CastleRights(0), bitmap(0), 0, 0, SideUnknown, 0, fmt.Errorf("%w: unknown symbol '%s'", ErrInvalidFEN, string(cell))
			}
			pos := y*Width + x
			sides[s] = Set(sides[s], pos, true)
			pieces[p] = Set(pieces[p], pos, true)
			cells[pos] = uint8(s)<<4 + uint8(p)
			materialValue[s] += scoreMaterial[p]
			positionValueMG[s] += scorePositionMG[p][scorePositionMap[s][pos]]
			positionValueEG[s] += scorePositionEG[p][scorePositionMap[s][pos]]
			hash ^= zobristConstantPiece[s][p][pos]
		}
	}

	var turn Side
	switch segments[1] {
	case "w":
		turn = SideWhite
		hash ^= zobristConstantSideWhite
	case "b":
		turn = SideBlack
	default:
		return sideBitmaps{}, pieceBitmaps{}, cellList{}, sideValue{}, sideValue{}, sideValue{}, CastleRights(0), bitmap(0), 0, 0, SideUnknown, 0, fmt.Errorf("%w: invalid turn", ErrInvalidFEN)
	}

	var castleRights CastleRights
	if len(segments[2]) > 4 {
		return sideBitmaps{}, pieceBitmaps{}, cellList{}, sideValue{}, sideValue{}, sideValue{}, CastleRights(0), bitmap(0), 0, 0, SideUnknown, 0, fmt.Errorf("%w: invalid castling rights", ErrInvalidFEN)
	}
crLoop:
	for i, e := range segments[2] {
		switch e {
		case 'K':
			castleRights.Set(CastleDirectionWhiteRight, true)
		case 'k':
			castleRights.Set(CastleDirectionBlackRight, true)
		case 'Q':
			castleRights.Set(CastleDirectionWhiteLeft, true)
		case 'q':
			castleRights.Set(CastleDirectionBlackLeft, true)
		default:
			if i == 0 && e == '-' {
				break crLoop
			}
			return sideBitmaps{}, pieceBitmaps{}, cellList{}, sideValue{}, sideValue{}, sideValue{}, CastleRights(0), bitmap(0), 0, 0, SideUnknown, 0, fmt.Errorf("%w: invalid castling rights", ErrInvalidFEN)
		}
	}
	hash ^= zobristConstantCastleRights[castleRights]

	var enPassant bitmap
	if segments[3] != "-" {
		pos, err := position.NewPosFromNotation(segments[3])
		if err != nil {
			return sideBitmaps{}, pieceBitmaps{}, cellList{}, sideValue{}, sideValue{}, sideValue{}, CastleRights(0), bitmap(0), 0, 0, SideUnknown, 0, fmt.Errorf("%w: %v", fmt.Errorf("%w: invalid enpassant position", ErrInvalidFEN), err)
		}
		enPassant = maskCell[pos]
		if enPassant&(maskRow[2]|maskRow[5]) == 0 {
			return sideBitmaps{}, pieceBitmaps{}, cellList{}, sideValue{}, sideValue{}, sideValue{}, CastleRights(0), bitmap(0), 0, 0, SideUnknown, 0, fmt.Errorf("%w: %v", fmt.Errorf("%w: invalid enpassant position", ErrInvalidFEN), err)
		}
	}
	hash ^= zobristConstantEnPassant[enPassant.LS1B()]

	halfMoveClock, err := strconv.ParseUint(segments[4], 10, 8)
	if err != nil {
		return sideBitmaps{}, pieceBitmaps{}, cellList{}, sideValue{}, sideValue{}, sideValue{}, CastleRights(0), bitmap(0), 0, 0, SideUnknown, 0, fmt.Errorf("%w: invalid half move clock", ErrInvalidFEN)
	}

	fullMoveClock, err := strconv.ParseUint(segments[5], 10, 8)
	if err != nil {
		return sideBitmaps{}, pieceBitmaps{}, cellList{}, sideValue{}, sideValue{}, sideValue{}, CastleRights(0), bitmap(0), 0, 0, SideUnknown, 0, fmt.Errorf("%w: invalid full move clock", ErrInvalidFEN)
	}

	return sides, pieces, cells, materialValue, positionValueMG, positionValueEG, castleRights, enPassant, uint8(halfMoveClock), uint8(fullMoveClock), turn, hash, nil
}

func (b *Board) FEN() string {
	builder := strings.Builder{}
	var skip uint8
	for y := position.Pos(Height) - 1; y >= 0; y-- {
		for x := position.Pos(0); x < Width; x++ {
			for skip = 0; x < Width && maskCell[y*Width+x]&b.occupied == 0; x++ {
				skip++
			}
			if skip != 0 {
				_, _ = builder.WriteRune(rune(skip + '0'))
			}
			if x < Width {
				for p, pBM := range b.pieces[1:] {
					if maskCell[y*Width+x]&pBM != 0 {
						s := SideWhite
						if maskCell[y*Width+x]&b.sides[SideBlack] != 0 {
							s = SideBlack
						}
						_, _ = builder.WriteString(Piece(p + 1).SymbolFEN(s))
						break
					}
				}
			}
		}
		if y > 0 {
			_, _ = builder.WriteRune('/')
		}
	}

	if b.turn == SideWhite {
		_, _ = builder.WriteString(" w ")
	} else {
		_, _ = builder.WriteString(" b ")
	}

	if b.castleRights == 0 {
		_, _ = builder.WriteRune('-')
	} else {
		if b.castleRights.IsAllowed(CastleDirectionWhiteRight) {
			_, _ = builder.WriteRune('K')
		}
		if b.castleRights.IsAllowed(CastleDirectionWhiteLeft) {
			_, _ = builder.WriteRune('Q')
		}
		if b.castleRights.IsAllowed(CastleDirectionBlackRight) {
			_, _ = builder.WriteRune('k')
		}
		if b.castleRights.IsAllowed(CastleDirectionBlackLeft) {
			_, _ = builder.WriteRune('q')
		}
	}
	_, _ = builder.WriteRune(' ')

	if b.enPassant == 0 {
		_, _ = builder.WriteRune('-')
	} else {
		_, _ = builder.WriteString(b.enPassant.LS1B().Notation())
	}

	_, _ = builder.WriteString(fmt.Sprintf(" %d %d", b.halfMoveClock, b.fullMoveClock))

	return builder.String()
}
