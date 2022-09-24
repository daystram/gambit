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

func (b *Board) GenerateMoves(s Side) []*Move {
	var mvs []*Move
	for p := range b.pieces {
		if Piece(p) == PieceUnknown {
			continue
		}
		// TODO: tailor move generation based on king check/pinning state
		b.GenerateMovesForPiece(&mvs, s, Piece(p))
	}
	return mvs
}

func (b *Board) GenerateMovesForPiece(mvs *[]*Move, s Side, p Piece) {
	// TRY: if in check, only allow moves to stop it. currently only filter in-post (probably good enough)
	fromBM := b.GetBitmap(s, p)
	for fromPos := position.Pos(0); fromPos < TotalCells; fromPos++ {
		// skip if cell is empty
		if maskCell[fromPos]&fromBM == 0 {
			continue
		}

		// get destination bitmap
		toBM := b.genValidDestination(fromPos, s, p) &^ fromBM // exclude self piece
		if toBM == 0 {
			continue
		}

		for toPos := position.Pos(0); toPos < TotalCells; toPos++ {
			// skip if cell is empty
			if maskCell[toPos]&toBM == 0 {
				continue
			}

			var candidateMoves []*Move
			// see if promotion is expected
			if p == PiecePawn && (maskRow[0]|maskRow[7])&maskCell[toPos] != 0 {
				for _, prom := range PawnPromoteCandidates {
					candidateMoves = append(candidateMoves,
						&Move{
							IsTurn:    s,
							Piece:     p,
							From:      fromPos,
							To:        toPos,
							IsPromote: prom,
						},
					)
				}
			} else {
				candidateMoves = append(candidateMoves,
					&Move{
						IsTurn: s,
						Piece:  p,
						From:   fromPos,
						To:     toPos,
					},
				)
			}
			for _, mv := range candidateMoves {
				// flag enpassant
				mv.IsEnPassant = p == PiecePawn && maskCell[mv.To] == b.enPassant

				// flag capture
				mv.IsCapture = maskCell[mv.To]&toBM&b.occupied != 0 || mv.IsEnPassant

				// get representation of next board if move was to be applied
				// this has to be done after the essential (IsCheck is not essential) flags are set
				// because board.Apply() needs them for correct incremental update
				bb := b.Clone()
				bb.Apply(mv)

				// filter moves that leaves our King in check
				if bb.isKingChecked(s) {
					continue
				}

				// flag their King check
				// TRY: do lazily?
				mv.IsCheck = bb.isKingChecked(s.Opposite())

				*mvs = append(*mvs, mv)
			}
		}
	}

	// generate castling moves
	if p == PieceKing && b.castleRights.IsSideAllowed(s) {
		oppositeAttackBM := b.genAttackArea(s.Opposite())

		ds := []CastleDirection{
			CastleDirectionWhiteRight,
			CastleDirectionWhiteLeft,
		}
		if s == SideBlack {
			ds = []CastleDirection{
				CastleDirectionBlackRight,
				CastleDirectionBlackLeft,
			}
		}
		for _, d := range ds {
			if b.castleRights.IsAllowed(d) &&
				maskCastling[d]&oppositeAttackBM == 0 &&
				maskCastling[d]&b.occupied == 0 {
				*mvs = append(*mvs, &Move{
					IsTurn:   s,
					Piece:    p,
					IsCastle: d,
				})
			}
		}
	}
}

func (b *Board) isKingChecked(s Side) bool {
	return b.GetBitmap(s, PieceKing)&b.genAttackArea(s.Opposite()) != 0
}

// genAttackArea returns the attack area bitmap for the given side.
func (b *Board) genAttackArea(s Side) bitmap {
	// TODO: cache?
	attackBM := bitmap(0)
	sideBM := b.sides[s]
	for p, pieceBM := range b.pieces {
		for pos := position.Pos(0); pos < TotalCells; pos++ {
			if maskCell[pos]&pieceBM&sideBM == 0 {
				continue
			}
			attackBM |= b.genValidDestination(pos, s, Piece(p))
		}
	}
	return attackBM
}

// genValidDestination generates the bitmap for the next valid positions.
// This generate function is not strictly legal (e.g., king may be left in check).
func (b *Board) genValidDestination(from position.Pos, s Side, p Piece) bitmap {
	switch p {
	case PiecePawn:
		cell := maskCell[from] & b.sides[s]
		if s == SideWhite {
			moveN1 := ShiftN(cell&^maskRow[7]) &^ b.occupied
			moveN2 := ShiftN(moveN1&maskRow[2]) &^ b.occupied
			captureNW := ShiftNW(cell&^maskRow[7]&^maskCol[0]) & (b.sides[SideBlack] | b.enPassant)
			captureNE := ShiftNE(cell&^maskRow[7]&^maskCol[7]) & (b.sides[SideBlack] | b.enPassant)
			return moveN1 | moveN2 | captureNW | captureNE
		}
		moveS1 := ShiftS(cell) &^ b.occupied
		moveS2 := ShiftS(moveS1&maskRow[5]) &^ b.occupied
		captureSW := ShiftSW(cell&^maskRow[0]&^maskCol[0]) & (b.sides[SideWhite] | b.enPassant)
		captureSE := ShiftSE(cell&^maskRow[0]&^maskCol[7]) & (b.sides[SideWhite] | b.enPassant)
		return moveS1 | moveS2 | captureSW | captureSE
	case PieceBishop:
		return HitDiagonals(from, maskCell[from], b.occupied) &^ b.sides[s]
	case PieceKnight:
		return maskKnight[from] &^ b.sides[s]
	case PieceRook:
		return HitLaterals(from, maskCell[from], b.occupied) &^ b.sides[s]
	case PieceQueen:
		return (HitDiagonals(from, maskCell[from], b.occupied) | HitLaterals(from, maskCell[from], b.occupied)) &^ b.sides[s]
	case PieceKing:
		return maskKing[from] &^ b.sides[s]
	default:
		return 0
	}
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
	return fmt.Sprintf("cast: %04b\nhalf: %4d\nfull: %4d\nstat: %s", b.castleRights, b.halfMoveClock, b.fullMoveClock, b.State())
}

// func (b *Board) GetSideAndPieces(i position.Pos) (Side, Piece) {
// 	var s Side
// 	for side, sideMap := range b.sides {
// 		if sideMap&maskCell[i] != 0 {
// 			s = Side(side)
// 			break
// 		}
// 	}
// 	p := PieceUnknown
// 	for piece, pieceMap := range b.pieces {
// 		if pieceMap&maskCell[i] != 0 {
// 			p = Piece(piece)
// 			break
// 		}
// 	}
// 	return s, p
// }

func (b *Board) State() State {
	if b.state != StateUnknown {
		return b.state
	}

	whiteMoves := b.GenerateMoves(SideWhite)
	if b.isKingChecked(SideWhite) {
		if len(whiteMoves) == 0 {
			return StateCheckmateWhite
		}
		return StateCheckWhite
	}
	blackMoves := b.GenerateMoves(SideBlack)
	if b.isKingChecked(SideBlack) {
		if len(blackMoves) == 0 {
			return StateCheckmateBlack
		}
		return StateCheckBlack
	}
	if len(whiteMoves) == 0 || len(blackMoves) == 0 {
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
