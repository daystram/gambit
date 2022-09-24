package board

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/daystram/gambit/position"
)

var (
	ErrInvalidFEN = errors.New("invalid fen")
)

type bitmap uint64
type sideBitmaps [3]bitmap
type pieceBitmaps [7]bitmap
type cellList [64]uint8

// Little-endian rank-file (LERF) mapping
type Board struct {
	// grid data
	sides    sideBitmaps
	pieces   pieceBitmaps
	occupied bitmap
	cells    cellList

	// meta
	enPassant     bitmap
	castleRights  CastleRights
	halfMoveClock uint8
	fullMoveClock uint8
	ply           uint8
	state         State
	turn          Side
	hash          uint64
}

type boardConfig struct {
	fen string
}

type BoardOption func(*boardConfig)

func WithFEN(fen string) BoardOption {
	return func(cfg *boardConfig) {
		cfg.fen = fen
	}
}

func NewBoard(opts ...BoardOption) (*Board, Side, error) {
	cfg := &boardConfig{
		fen: DefaultStartingPositionFEN,
	}
	for _, f := range opts {
		f(cfg)
	}
	sides, pieces, pieceList, castleRights, enPassant, halfMoveClock, fullMoveClock, turn, err := parseFEN(cfg.fen)
	if err != nil {
		return nil, SideUnknown, err
	}

	return &Board{
		sides:         sides,
		pieces:        pieces,
		occupied:      Union(sides[SideBlack], sides[SideWhite]),
		cells:         pieceList,
		enPassant:     enPassant,
		castleRights:  castleRights,
		halfMoveClock: halfMoveClock,
		fullMoveClock: fullMoveClock,
		turn:          turn,
	}, turn, nil
}

func parseFEN(fen string) (sideBitmaps, pieceBitmaps, [64]uint8, CastleRights, bitmap, uint8, uint8, Side, error) {
	segments := strings.Split(fen, " ")
	if len(segments) != 6 {
		return sideBitmaps{}, pieceBitmaps{}, cellList{}, CastleRights(0), bitmap(0), 0, 0, SideUnknown, fmt.Errorf("%w: incorrect number of segments", ErrInvalidFEN)
	}

	var sides sideBitmaps
	var pieces pieceBitmaps
	var pieceList [64]uint8
	rows := strings.Split(segments[0], "/")
	if len(rows) != int(Height) {
		return sideBitmaps{}, pieceBitmaps{}, cellList{}, CastleRights(0), bitmap(0), 0, 0, SideUnknown, fmt.Errorf("%w: invalid board configuration", ErrInvalidFEN)
	}
	for y := position.Pos(0); y < Height; y++ {
		ptrX, ptrY := -1, Height-y-1
		for x := position.Pos(0); x < Width; x++ {
			ptrX++
			if ptrX >= len(rows[ptrY]) {
				return sideBitmaps{}, pieceBitmaps{}, cellList{}, CastleRights(0), bitmap(0), 0, 0, SideUnknown, fmt.Errorf("%w: missing cells", ErrInvalidFEN)
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
					return sideBitmaps{}, pieceBitmaps{}, cellList{}, CastleRights(0), bitmap(0), 0, 0, SideUnknown, fmt.Errorf("%w: skip out of bounds", ErrInvalidFEN)
				}
				return sideBitmaps{}, pieceBitmaps{}, cellList{}, CastleRights(0), bitmap(0), 0, 0, SideUnknown, fmt.Errorf("%w: unknown symbol '%s'", ErrInvalidFEN, string(cell))
			}
			pos := y*Width + x
			sides[s] = Set(sides[s], pos, true)
			pieces[p] = Set(pieces[p], pos, true)
			pieceList[pos] = uint8(s)<<4 + uint8(p)
		}
	}

	var turn Side
	switch segments[1] {
	case "w":
		turn = SideWhite
	case "b":
		turn = SideBlack
	default:
		return sideBitmaps{}, pieceBitmaps{}, cellList{}, CastleRights(0), bitmap(0), 0, 0, SideUnknown, fmt.Errorf("%w: invalid turn", ErrInvalidFEN)
	}

	var castleRights CastleRights
	if len(segments[2]) > 4 {
		return sideBitmaps{}, pieceBitmaps{}, cellList{}, CastleRights(0), bitmap(0), 0, 0, SideUnknown, fmt.Errorf("%w: invalid castling rights", ErrInvalidFEN)
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
			return sideBitmaps{}, pieceBitmaps{}, cellList{}, CastleRights(0), bitmap(0), 0, 0, SideUnknown, fmt.Errorf("%w: invalid castling rights", ErrInvalidFEN)
		}
	}

	var enPassant bitmap
	if segments[3] != "-" {
		pos, err := position.NewPosFromNotation(segments[3])
		if err != nil {
			return sideBitmaps{}, pieceBitmaps{}, cellList{}, CastleRights(0), bitmap(0), 0, 0, SideUnknown, fmt.Errorf("%w: %v", fmt.Errorf("%w: invalid enpassant position", ErrInvalidFEN), err)
		}
		enPassant = maskCell[pos]
		if enPassant&(maskRow[2]|maskRow[5]) == 0 {
			return sideBitmaps{}, pieceBitmaps{}, cellList{}, CastleRights(0), bitmap(0), 0, 0, SideUnknown, fmt.Errorf("%w: %v", fmt.Errorf("%w: invalid enpassant position", ErrInvalidFEN), err)
		}
	}

	halfMoveClock, err := strconv.ParseUint(segments[4], 10, 8)
	if err != nil {
		return sideBitmaps{}, pieceBitmaps{}, cellList{}, CastleRights(0), bitmap(0), 0, 0, SideUnknown, fmt.Errorf("%w: invalid half move clock", ErrInvalidFEN)
	}

	fullMoveClock, err := strconv.ParseUint(segments[5], 10, 8)
	if err != nil {
		return sideBitmaps{}, pieceBitmaps{}, cellList{}, CastleRights(0), bitmap(0), 0, 0, SideUnknown, fmt.Errorf("%w: invalid full move clock", ErrInvalidFEN)
	}

	return sides, pieces, pieceList, castleRights, enPassant, uint8(halfMoveClock), uint8(fullMoveClock), turn, nil
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

func (b *Board) GenerateMoves() []*Move {
	mvs := make([]*Move, 0, 64)
	opponentSide := b.turn.Opposite()
	sideMask := b.sides[b.turn]
	nonSelfMask := ^sideMask

	kingPos := b.GetBitmap(b.turn, PieceKing).LS1B()
	checkerCount, attackedMask := b.GetCellAttackers(opponentSide, kingPos, 0, 2)

	if checkerCount == 2 {
		b.generateMoveKing(&mvs, kingPos, ^attackedMask&^sideMask)
		return mvs
	}

	if checkerCount == 1 {
		b.generateMovePawn(&mvs, sideMask&b.pieces[PiecePawn], attackedMask)
		b.generateMoveKnight(&mvs, sideMask&b.pieces[PieceKnight], attackedMask)
		b.generateMoveBishop(&mvs, sideMask&b.pieces[PieceBishop], attackedMask)
		b.generateMoveRook(&mvs, sideMask&b.pieces[PieceRook], attackedMask)
		b.generateMoveQueen(&mvs, sideMask&b.pieces[PieceQueen], attackedMask)
		b.generateMoveKing(&mvs, kingPos, (^attackedMask|b.sides[opponentSide])&nonSelfMask)
		return mvs
	}

	b.generateMovePawn(&mvs, sideMask&b.pieces[PiecePawn], nonSelfMask)
	b.generateMoveKnight(&mvs, sideMask&b.pieces[PieceKnight], nonSelfMask)
	b.generateMoveBishop(&mvs, sideMask&b.pieces[PieceBishop], nonSelfMask)
	b.generateMoveRook(&mvs, sideMask&b.pieces[PieceRook], nonSelfMask)
	b.generateMoveQueen(&mvs, sideMask&b.pieces[PieceQueen], nonSelfMask)
	b.generateMoveKing(&mvs, kingPos, nonSelfMask)
	b.generateCastling(&mvs)

	// TODO: try strictly legal generator and remove this
	i := b.filterInvalidMoves(&mvs, kingPos)
	return mvs[:i]
}

func (b *Board) generateMovePawn(mvs *[]*Move, fromMask, allowedToMask bitmap) {
	for fromPos := position.Pos(0); fromPos < TotalCells; fromPos++ {
		fromCell := maskCell[fromPos] & fromMask
		if fromCell == 0 {
			continue
		}

		var candidateToBM bitmap
		var candidateEnPassantTargetBM bitmap
		if b.turn == SideWhite {
			moveN1 := ShiftN(fromCell&^maskRow[7]) &^ b.occupied
			moveN2 := ShiftN(moveN1&maskRow[2]) &^ b.occupied
			captureNW := ShiftNW(fromCell&^maskRow[7]&^maskCol[0]) & (b.sides[SideBlack] | b.enPassant)
			captureNE := ShiftNE(fromCell&^maskRow[7]&^maskCol[7]) & (b.sides[SideBlack] | b.enPassant)
			candidateToBM = moveN1 | moveN2 | captureNW | captureNE
			candidateEnPassantTargetBM = ShiftS(b.enPassant)
		} else {
			moveS1 := ShiftS(fromCell) &^ b.occupied
			moveS2 := ShiftS(moveS1&maskRow[5]) &^ b.occupied
			captureSW := ShiftSW(fromCell&^maskRow[0]&^maskCol[0]) & (b.sides[SideWhite] | b.enPassant)
			captureSE := ShiftSE(fromCell&^maskRow[0]&^maskCol[7]) & (b.sides[SideWhite] | b.enPassant)
			candidateToBM = moveS1 | moveS2 | captureSW | captureSE
			candidateEnPassantTargetBM = ShiftN(b.enPassant)
		}
		if allowedToMask&candidateEnPassantTargetBM != 0 {
			// enPasssant may be stopping a King check
			candidateToBM &= allowedToMask | b.enPassant
		} else {
			candidateToBM &= allowedToMask
		}

		for toPos := position.Pos(0); toPos < TotalCells; toPos++ {
			toCell := maskCell[toPos] & candidateToBM
			if toCell == 0 {
				continue
			}

			isEnPassant := toCell == b.enPassant
			isCapture := toCell&b.occupied != 0 || isEnPassant
			if toCell&(maskRow[0]|maskRow[7]) == 0 {
				*mvs = append(*mvs, &Move{
					From:        fromPos,
					To:          toPos,
					Piece:       PiecePawn,
					IsTurn:      b.turn,
					IsCapture:   isCapture,
					IsEnPassant: isEnPassant,
				})
			} else {
				for _, prom := range PawnPromoteCandidates {
					*mvs = append(*mvs, &Move{
						From:      fromPos,
						To:        toPos,
						Piece:     PiecePawn,
						IsTurn:    b.turn,
						IsCapture: isCapture,
						IsPromote: prom,
					})
				}
			}
		}
	}
}

func (b *Board) generateMoveKnight(mvs *[]*Move, fromMask, allowedToMask bitmap) {
	for fromPos := position.Pos(0); fromPos < TotalCells; fromPos++ {
		fromCell := maskCell[fromPos] & fromMask
		if fromCell == 0 {
			continue
		}

		var candidateToBM bitmap
		candidateToBM |= maskKnight[fromPos]
		candidateToBM &= allowedToMask

		for toPos := position.Pos(0); toPos < TotalCells; toPos++ {
			toCell := maskCell[toPos] & candidateToBM
			if toCell == 0 {
				continue
			}

			isCapture := toCell&b.occupied != 0
			*mvs = append(*mvs, &Move{
				From:      fromPos,
				To:        toPos,
				Piece:     PieceKnight,
				IsTurn:    b.turn,
				IsCapture: isCapture,
			})
		}
	}
}

func (b *Board) generateMoveBishop(mvs *[]*Move, fromMask, allowedToMask bitmap) {
	for fromPos := position.Pos(0); fromPos < TotalCells; fromPos++ {
		fromCell := maskCell[fromPos] & fromMask
		if fromCell == 0 {
			continue
		}

		var candidateToBM bitmap
		candidateToBM |= HitDiagonals(fromPos, b.occupied)
		candidateToBM &= allowedToMask

		for toPos := position.Pos(0); toPos < TotalCells; toPos++ {
			toCell := maskCell[toPos] & candidateToBM
			if toCell == 0 {
				continue
			}

			isCapture := toCell&b.occupied != 0
			*mvs = append(*mvs, &Move{
				From:      fromPos,
				To:        toPos,
				Piece:     PieceBishop,
				IsTurn:    b.turn,
				IsCapture: isCapture,
			})
		}
	}
}

func (b *Board) generateMoveRook(mvs *[]*Move, fromMask, allowedToMask bitmap) {
	for fromPos := position.Pos(0); fromPos < TotalCells; fromPos++ {
		fromCell := maskCell[fromPos] & fromMask
		if fromCell == 0 {
			continue
		}

		var candidateToBM bitmap
		candidateToBM |= HitLaterals(fromPos, b.occupied)
		candidateToBM &= allowedToMask

		for toPos := position.Pos(0); toPos < TotalCells; toPos++ {
			toCell := maskCell[toPos] & candidateToBM
			if toCell == 0 {
				continue
			}

			isCapture := toCell&b.occupied != 0
			*mvs = append(*mvs, &Move{
				From:      fromPos,
				To:        toPos,
				Piece:     PieceRook,
				IsTurn:    b.turn,
				IsCapture: isCapture,
			})
		}
	}
}

func (b *Board) generateMoveQueen(mvs *[]*Move, fromMask, allowedToMask bitmap) {
	for fromPos := position.Pos(0); fromPos < TotalCells; fromPos++ {
		fromCell := maskCell[fromPos] & fromMask
		if fromCell == 0 {
			continue
		}

		var candidateToBM bitmap
		candidateToBM |= (HitDiagonals(fromPos, b.occupied) | HitLaterals(fromPos, b.occupied))
		candidateToBM &= allowedToMask

		for toPos := position.Pos(0); toPos < TotalCells; toPos++ {
			toCell := maskCell[toPos] & candidateToBM
			if toCell == 0 {
				continue
			}

			isCapture := toCell&b.occupied != 0
			*mvs = append(*mvs, &Move{
				From:      fromPos,
				To:        toPos,
				Piece:     PieceQueen,
				IsTurn:    b.turn,
				IsCapture: isCapture,
			})
		}
	}
}

func (b *Board) generateCastling(mvs *[]*Move) {
	opponentSide := b.turn.Opposite()
	if b.castleRights.IsSideAllowed(b.turn) {
		if b.turn == SideWhite {
			if b.castleRights.IsAllowed(CastleDirectionWhiteLeft) &&
				b.occupied&maskCastling[CastleDirectionWhiteLeft] == 0 {
				attackerB1, _ := b.GetCellAttackers(opponentSide, 1, 0, 1)
				attackerC1, _ := b.GetCellAttackers(opponentSide, 2, 0, 1)
				attackerD1, _ := b.GetCellAttackers(opponentSide, 3, 0, 1)
				if attackerB1+attackerC1+attackerD1 == 0 {
					*mvs = append(*mvs, &Move{
						IsTurn:   b.turn,
						Piece:    PieceKing,
						IsCastle: CastleDirectionWhiteLeft,
					})
				}
			}
			if b.castleRights.IsAllowed(CastleDirectionWhiteRight) &&
				b.occupied&maskCastling[CastleDirectionWhiteRight] == 0 {
				attackerF1, _ := b.GetCellAttackers(opponentSide, 5, 0, 1)
				attackerG1, _ := b.GetCellAttackers(opponentSide, 6, 0, 1)
				if attackerF1+attackerG1 == 0 {
					*mvs = append(*mvs, &Move{
						IsTurn:   b.turn,
						Piece:    PieceKing,
						IsCastle: CastleDirectionWhiteRight,
					})
				}
			}
		} else {
			if b.castleRights.IsAllowed(CastleDirectionBlackLeft) &&
				b.occupied&maskCastling[CastleDirectionBlackLeft] == 0 {
				attackerB1, _ := b.GetCellAttackers(opponentSide, 57, 0, 1)
				attackerC1, _ := b.GetCellAttackers(opponentSide, 58, 0, 1)
				attackerD1, _ := b.GetCellAttackers(opponentSide, 59, 0, 1)
				if attackerB1+attackerC1+attackerD1 == 0 {
					*mvs = append(*mvs, &Move{
						IsTurn:   b.turn,
						Piece:    PieceKing,
						IsCastle: CastleDirectionBlackLeft,
					})
				}
			}
			if b.castleRights.IsAllowed(CastleDirectionBlackRight) &&
				b.occupied&maskCastling[CastleDirectionBlackRight] == 0 {
				attackerF8, _ := b.GetCellAttackers(opponentSide, 61, 0, 1)
				attackerG8, _ := b.GetCellAttackers(opponentSide, 62, 0, 1)
				if attackerF8+attackerG8 == 0 {
					*mvs = append(*mvs, &Move{
						IsTurn:   b.turn,
						Piece:    PieceKing,
						IsCastle: CastleDirectionBlackRight,
					})
				}
			}
		}
	}
}

func (b *Board) filterInvalidMoves(mvs *[]*Move, kingPos position.Pos) int {
	opponentSide := b.turn.Opposite()
	i := 0
	for _, mv := range *mvs {
		bb := b.Clone()
		bb.Apply(mv)
		if c, _ := bb.GetCellAttackers(opponentSide, kingPos, 0, 1); c == 0 {
			(*mvs)[i] = mv
			i++
		}
	}
	return i
}

func (b *Board) generateMoveKing(mvs *[]*Move, fromPos position.Pos, allowedToMask bitmap) {
	var candidateToBM bitmap
	candidateToBM |= maskKing[fromPos]
	candidateToBM &= allowedToMask

	for toPos := position.Pos(0); toPos < TotalCells; toPos++ {
		toCell := maskCell[toPos] & candidateToBM
		if toCell == 0 {
			continue
		}

		attackerCount, _ := b.GetCellAttackers(b.turn.Opposite(), toPos, fromPos, 1)
		if attackerCount != 0 {
			continue
		}

		isCapture := toCell&b.occupied != 0
		*mvs = append(*mvs, &Move{
			From:      fromPos,
			To:        toPos,
			Piece:     PieceKing,
			IsTurn:    b.turn,
			IsCapture: isCapture,
		})
	}
}

func (b *Board) GetCellAttackers(attackerSide Side, pos, xrayPos position.Pos, limit int) (int, bitmap) {
	var count int
	var attackBM bitmap
	attackerSideMask := b.sides[attackerSide]
	posMask := maskCell[pos]

	// find lateral attacker pieces
	candidateRay := HitLaterals(pos, magicRookMask[pos]&(b.occupied&^maskCell[xrayPos]))
	attackerLaterals := candidateRay & attackerSideMask & (b.pieces[PieceRook] | b.pieces[PieceQueen])
	countLateral := attackerLaterals.BitCount()
	count += countLateral
	attackBM |= attackerLaterals
	if count >= limit {
		return count, attackBM
	}

	// find diagonal attacker pieces
	candidateRay = HitDiagonals(pos, magicBishopMask[pos]&(b.occupied&^maskCell[xrayPos]))
	attackerDiagonals := candidateRay & attackerSideMask & (b.pieces[PieceBishop] | b.pieces[PieceQueen])
	countDiagonal := attackerDiagonals.BitCount()
	count += countDiagonal
	attackBM |= attackerDiagonals
	if count >= limit {
		return count, attackBM
	}

	// fill rays
	for attackerPos := position.Pos(0); countLateral != 0 && attackerPos < TotalCells; attackerPos++ {
		if attackerLaterals&maskCell[attackerPos] != 0 {
			attackBM |= HitLaterals(pos, maskCell[attackerPos]) & HitLaterals(attackerPos, posMask)
		}
	}
	for attackerPos := position.Pos(0); countDiagonal != 0 && attackerPos < TotalCells; attackerPos++ {
		if attackerDiagonals&maskCell[attackerPos] != 0 {
			attackBM |= HitDiagonals(pos, maskCell[attackerPos]) & HitDiagonals(attackerPos, posMask)
		}
	}

	// find Knight attacks
	if attackerKnights := maskKnight[pos] & attackerSideMask & b.pieces[PieceKnight]; attackerKnights != 0 {
		count += attackerKnights.BitCount()
		attackBM |= attackerKnights
		if count >= limit {
			return count, attackBM
		}
	}

	// find Pawn attacks
	if attackerSide == SideWhite {
		if attackerPawns := (ShiftSW(posMask) | ShiftSE(posMask)) & attackerSideMask & b.pieces[PiecePawn]; attackerPawns != 0 {
			count += attackerPawns.BitCount()
			attackBM |= attackerPawns
			if count >= limit {
				return count, attackBM
			}
		}
	} else {
		if attackerPawns := (ShiftNW(posMask) | ShiftNE(posMask)) & attackerSideMask & b.pieces[PiecePawn]; attackerPawns != 0 {
			count += attackerPawns.BitCount()
			attackBM |= attackerPawns
			if count >= limit {
				return count, attackBM
			}
		}
	}

	// find King attacks
	if attackersKing := maskKing[pos] & attackerSideMask & b.pieces[PieceKing]; attackersKing != 0 {
		attackBM |= attackersKing
	}

	return count, attackBM
}

func (b *Board) flip(s Side, p Piece, pos position.Pos) {
	b.sides[s] ^= maskCell[pos]
	b.pieces[p] ^= maskCell[pos]
	b.occupied ^= maskCell[pos]
	b.hash ^= zobristConstantGrid[s][p][pos]
}

// TODO: return undo func?
func (b *Board) Apply(mv *Move) {
	ourTurn := b.turn
	oppTurn := ourTurn.Opposite()
	if mv.IsCastle != CastleDirectionUnknown {
		// perform castling
		hopsKing := posCastling[mv.IsCastle][PieceKing]
		hopsRook := posCastling[mv.IsCastle][PieceRook]

		b.flip(ourTurn, PieceKing, hopsKing[0])
		b.flip(ourTurn, PieceKing, hopsKing[1])
		b.cells[hopsKing[1]] = b.cells[hopsKing[0]]
		b.cells[hopsKing[0]] = 0

		b.flip(ourTurn, PieceRook, hopsRook[0])
		b.flip(ourTurn, PieceRook, hopsRook[1])
		b.cells[hopsRook[1]] = b.cells[hopsRook[0]]
		b.cells[hopsRook[0]] = 0
	} else {
		// remove moving piece at mv.From
		b.flip(ourTurn, mv.Piece, mv.From)
		b.cells[mv.From] = 0

		// remove captured piece at mv.To
		if mv.IsCapture {
			var capturedPiece Piece
			var targetPos position.Pos
			if mv.IsEnPassant {
				capturedPiece = PiecePawn
				targetPos = mv.To - Width // pos of opponent Pawn to remove by enPassant
				if ourTurn == SideBlack {
					targetPos = mv.To + Width
				}
			} else {
				mask := maskCell[mv.To]
				for piece, pieceBM := range b.pieces {
					if pieceBM&mask != 0 {
						capturedPiece = Piece(piece)
						break
					}
				}
				targetPos = mv.To
			}
			b.flip(oppTurn, capturedPiece, targetPos)
			b.cells[targetPos] = 0
		}

		// place moving piece at mv.To
		targetPiece := mv.Piece
		if mv.IsPromote != PieceUnknown {
			targetPiece = mv.IsPromote
		}
		b.setSideAndPieces(mv.To, ourTurn, targetPiece)
		b.flip(ourTurn, targetPiece, mv.To)
	}

	// update enPassantPos
	b.enPassant = bitmap(0)
	if mv.Piece == PiecePawn {
		if ourTurn == SideWhite && maskCell[mv.From]&maskRow[1] != 0 && maskCell[mv.To]&maskRow[3] != 0 {
			b.enPassant = maskCell[mv.To-Width]
		} else if ourTurn == SideBlack && maskCell[mv.From]&maskRow[6] != 0 && maskCell[mv.To]&maskRow[4] != 0 {
			b.enPassant = maskCell[mv.To+Width]
		}
	}
	b.hash ^= uint64(b.enPassant)

	// update castlingRights
	// TODO: hash castlingRights
	if mv.Piece == PieceKing {
		if ourTurn == SideWhite {
			b.castleRights.Set(CastleDirectionWhiteRight, false)
			b.castleRights.Set(CastleDirectionWhiteLeft, false)
		} else {
			b.castleRights.Set(CastleDirectionBlackRight, false)
			b.castleRights.Set(CastleDirectionBlackLeft, false)
		}
	}
	// TODO: remove castling rights whhen Rook is captured
	if mv.Piece == PieceRook {
		if maskCell[mv.From]&maskCol[7] != 0 {
			if ourTurn == SideWhite {
				b.castleRights.Set(CastleDirectionWhiteRight, false)
			} else {
				b.castleRights.Set(CastleDirectionBlackRight, false)
			}
		}
		if maskCell[mv.From]&maskCol[0] != 0 {
			if ourTurn == SideWhite {
				b.castleRights.Set(CastleDirectionWhiteLeft, false)
			} else {
				b.castleRights.Set(CastleDirectionBlackLeft, false)
			}
		}

	}

	// update half move clock
	if mv.Piece == PiecePawn || mv.IsCapture {
		b.halfMoveClock = 0
	} else {
		b.halfMoveClock++
	}

	// update full move clock
	if ourTurn == SideBlack {
		b.fullMoveClock++
	}

	// update ply
	b.ply++

	// set update turn
	b.turn = oppTurn
	b.hash ^= zobristConstantSideWhite
}

func (b *Board) GetBitmap(s Side, p Piece) bitmap {
	return b.sides[s] & b.pieces[p]
}

func (b *Board) GetSideAndPieces(pos position.Pos) (Side, Piece) {
	l := b.cells[pos]
	return Side(l >> 4), Piece(l & 0x0F)
}

func (b *Board) setSideAndPieces(pos position.Pos, s Side, p Piece) {
	b.cells[pos] = uint8(s)<<4 + uint8(p)
}

func (b *Board) Dump() string {
	builder := strings.Builder{}
	for y := position.Pos(Height) - 1; y >= 0; y-- {
		_, _ = builder.WriteString("   +---+---+---+---+---+---+---+---+\n")
		_, _ = builder.WriteString(fmt.Sprintf(" %d |", y+1))
		for x := position.Pos(0); x < Width; x++ {
			s, p := b.GetSideAndPieces((y*Width + x))
			sym := p.SymbolFEN(s)
			if s == SideUnknown {
				sym = " "
			}
			_, _ = builder.WriteString(fmt.Sprintf(" %s |", sym))
		}
		_, _ = builder.WriteString("\n")
	}
	_, _ = builder.WriteString("   +---+---+---+---+---+---+---+---+\n   ")
	for x := position.Pos(0); x < Width; x++ {
		_, _ = builder.WriteString(fmt.Sprintf("  %s ", x.NotationComponentX()))
	}
	return builder.String()
}

func (b *Board) Draw() string {
	builder := strings.Builder{}
	for y := position.Pos(Height) - 1; y >= 0; y-- {
		_, _ = builder.WriteString(fmt.Sprintf("\033[1m %d \033[0m", y+1))
		for x := position.Pos(0); x < Width; x++ {
			s, p := b.GetSideAndPieces((y*Width + x))
			sym := p.SymbolUnicode(s, false)
			if p == PieceUnknown {
				sym = " "
			}
			var cell string
			if x%2^y%2 == 0 {
				cell = "\033[38;5;233;48;5;77m" + cell
			} else {
				cell = "\033[38;5;233;48;5;194m" + cell
			}
			cell += fmt.Sprintf(" %s ", sym) + "\033[0m"
			builder.WriteString(cell)
		}
		_, _ = builder.WriteString("\n")
	}
	_, _ = builder.WriteString("   ")
	for x := position.Pos(0); x < Width; x++ {
		_, _ = builder.WriteString(fmt.Sprintf("\033[1m %s \033[0m", x.NotationComponentX()))
	}
	return builder.String()
}

func (b *Board) DebugString() string {
	return fmt.Sprintf("turn: %5s\nenp : %5s\ncast:  %04b\nhalf: %5d\nfull: %5d\nply : %5d\nstat: %s", b.turn, b.enPassant.LS1B().Notation(), b.castleRights, b.halfMoveClock, b.fullMoveClock, b.ply, b.State())
}

func (b *Board) State() State {
	if b.state != StateUnknown {
		return b.state
	}

	mvs := b.GenerateMoves()
	if c, _ := b.GetCellAttackers(b.turn.Opposite(), b.GetBitmap(b.turn, PieceKing).LS1B(), 0, 1); c != 0 {
		if len(mvs) == 0 {
			if b.turn == SideWhite {
				return StateCheckmateWhite
			}
			return StateCheckmateBlack
		}
		if b.turn == SideWhite {
			return StateCheckWhite
		}
		return StateCheckBlack
	}
	if len(mvs) == 0 {
		return StateStalemate
	}

	// checkmate takes precedence over the 50 move rule
	if b.halfMoveClock >= 100 {
		return StateFiftyMoveViolated
	}
	return StateRunning
}

func (b *Board) Turn() Side {
	return b.turn
}

func (b *Board) Ply() uint8 {
	return b.ply
}

func (b *Board) HalfMoveClock() uint8 {
	return b.halfMoveClock
}

func (b *Board) FullMoveClock() uint8 {
	return b.fullMoveClock
}

func (b *Board) Clone() *Board {
	return &Board{
		sides:         b.sides,
		pieces:        b.pieces,
		occupied:      b.occupied,
		cells:         b.cells,
		enPassant:     b.enPassant,
		castleRights:  b.castleRights,
		halfMoveClock: b.halfMoveClock,
		fullMoveClock: b.fullMoveClock,
		ply:           b.ply,
		state:         b.state,
		turn:          b.turn,
		hash:          b.hash,
	}
}

func (b *Board) Hash() uint64 {
	return b.hash
}

// ======================================================= DEBUG

func (b *Board) DumpEnPassant() string {
	return b.enPassant.Dump()
}

func (b *Board) DumpOccupied() string {
	return b.occupied.Dump()
}

// ======================================================= DEBUG
