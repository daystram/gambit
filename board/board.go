package board

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/daystram/gambit/position"
)

const (
	Width      = position.MaxComponentScalar
	Height     = position.MaxComponentScalar
	TotalCells = Width * Height

	DefaultStartingPositionFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
)

var (
	ErrInvalidFEN = errors.New("invalid fen")
)

type Move struct {
	From, To position.Pos
	Piece    Piece

	IsSide      Side
	IsCapture   bool
	IsCheck     bool
	IsCastle    CastleDirection
	IsEnPassant bool
	IsPromote   Piece
}

func (m Move) String() string {
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

type bitmap uint64

type CastleDirection uint8

const (
	CastleDirectionUnknown CastleDirection = iota
	CastleDirectionWhiteRight
	CastleDirectionWhiteLeft
	CastleDirectionBlackRight
	CastleDirectionBlackLeft
)

func (d CastleDirection) String() string {
	switch d {
	case CastleDirectionWhiteRight:
		return "White 0-0"
	case CastleDirectionWhiteLeft:
		return "White 0-0-0"
	case CastleDirectionBlackRight:
		return "Black 0-0"
	case CastleDirectionBlackLeft:
		return "Black 0-0-0"
	default:
		return ""
	}
}

func (d CastleDirection) IsWhite() bool {
	return d == CastleDirectionWhiteRight || d == CastleDirectionWhiteLeft
}

func (d CastleDirection) IsRight() bool {
	return d == CastleDirectionWhiteRight || d == CastleDirectionBlackRight
}

type CastleRights uint8

var (
	maskCastleRights = [5]CastleRights{
		0,
		0b1000, // CastleDirectionWhiteOO
		0b0100, // CastleDirectionWhiteOOO
		0b0010, // CastleDirectionBlackOO
		0b0001, // CastleDirectionBlackOOO
	}
)

func (c *CastleRights) Set(d CastleDirection, allow bool) {
	if allow {
		*c |= maskCastleRights[d]
	} else {
		*c &^= maskCastleRights[d]
	}
}

func (c *CastleRights) IsAllowed(d CastleDirection) bool {
	return *c&maskCastleRights[d] != 0
}

func (c *CastleRights) IsSideAllowed(s Side) bool {
	if s == SideWhite {
		return *c&(maskCastleRights[CastleDirectionWhiteLeft]|maskCastleRights[CastleDirectionWhiteRight]) != 0
	}
	return *c&(maskCastleRights[CastleDirectionBlackLeft]|maskCastleRights[CastleDirectionBlackRight]) != 0
}

// Little-endian rank-file (LERF) mapping
type Board struct {
	// grid data
	sides    map[Side]bitmap
	pieces   map[Piece]bitmap
	occupied bitmap

	// meta
	enPassantPos  position.Pos
	castleRights  CastleRights
	halfMoveClock uint64
	fullMoveClock uint64
	state         State
	turn          Side

	// cache
	cacheMoves map[Side]map[Piece][]*Move
}

// ======================================================= DEBUG

func (b *Board) DumpEnPassant() string {
	if b.enPassantPos == flagNoEnpassant {
		return bitmap(0).Dump()
	}
	return maskCell[b.enPassantPos].Dump()
}

func (b *Board) DumpOccupied() string {
	return b.occupied.Dump()
}

// ======================================================= DEBUG

func initCacheMoves() map[Side]map[Piece][]*Move {
	c := make(map[Side]map[Piece][]*Move, 2)
	c[SideWhite] = make(map[Piece][]*Move, 8)
	c[SideBlack] = make(map[Piece][]*Move, 8)
	return c
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
	sides, pieces, castleRights, enPassantPos, halfMoveClock, fullMoveClock, turn, err := parseFEN(cfg.fen)
	if err != nil {
		return nil, SideUnknown, err
	}

	// sides := map[Side]bitmap{
	// 	SideBlack: 0x_FF_FF_00_00_00_00_00_00,
	// 	SideWhite: 0x_00_00_00_00_00_00_FF_FF,
	// }
	// pieces := map[Piece]bitmap{
	// 	PiecePawn:   0x_00_FF_00_00_00_00_FF_00,
	// 	PieceBishop: 0x_24_00_00_00_00_00_00_24,
	// 	PieceKnight: 0x_42_00_00_00_00_00_00_42,
	// 	PieceRook:   0x_81_00_00_00_00_00_00_81,
	// 	PieceQueen:  0x_08_00_00_00_00_00_00_08,
	// 	PieceKing:   0x_10_00_00_00_00_00_00_10,
	// }

	return &Board{
		sides:         sides,
		pieces:        pieces,
		occupied:      Union(sides[SideBlack], sides[SideWhite]),
		enPassantPos:  enPassantPos,
		castleRights:  castleRights,
		halfMoveClock: halfMoveClock,
		fullMoveClock: fullMoveClock,
		turn:          turn,
		cacheMoves:    initCacheMoves(),
	}, turn, nil
}

func parseFEN(fen string) (map[Side]bitmap, map[Piece]bitmap, CastleRights, position.Pos, uint64, uint64, Side, error) {
	segments := strings.Split(fen, " ")
	if len(segments) != 6 {
		return nil, nil, CastleRights(0), position.Pos(0), 0, 0, SideUnknown, fmt.Errorf("%w: incorrect number of segments", ErrInvalidFEN)
	}

	sides := make(map[Side]bitmap, 2)
	pieces := make(map[Piece]bitmap, 6)
	rows := strings.Split(segments[0], "/")
	if len(rows) != int(Height) {
		return nil, nil, CastleRights(0), position.Pos(0), 0, 0, SideUnknown, fmt.Errorf("%w: invalid board configuration", ErrInvalidFEN)
	}
	for y := position.Pos(0); y < Height; y++ {
		ptrX, ptrY := -1, Height-y-1
		for x := position.Pos(0); x < Width; x++ {
			ptrX++
			if ptrX >= len(rows[ptrY]) {
				return nil, nil, CastleRights(0), position.Pos(0), 0, 0, SideUnknown, fmt.Errorf("%w: missing cells", ErrInvalidFEN)
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
					return nil, nil, CastleRights(0), position.Pos(0), 0, 0, SideUnknown, fmt.Errorf("%w: skip out of bounds", ErrInvalidFEN)
				}
				return nil, nil, CastleRights(0), position.Pos(0), 0, 0, SideUnknown, fmt.Errorf("%w: unknown symbol '%s'", ErrInvalidFEN, string(cell))
			}
			pos := y*Width + x
			sides[s] = Set(sides[s], pos, true)
			pieces[p] = Set(pieces[p], pos, true)
		}
	}

	var turn Side
	switch segments[1] {
	case "w":
		turn = SideWhite
	case "b":
		turn = SideBlack
	default:
		return nil, nil, CastleRights(0), position.Pos(0), 0, 0, SideUnknown, fmt.Errorf("%w: invalid turn", ErrInvalidFEN)
	}

	var castleRights CastleRights
	if len(segments[2]) > 4 {
		return nil, nil, CastleRights(0), position.Pos(0), 0, 0, SideUnknown, fmt.Errorf("%w: invalid castling rights", ErrInvalidFEN)
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
			return nil, nil, CastleRights(0), position.Pos(0), 0, 0, SideUnknown, fmt.Errorf("%w: invalid castling rights", ErrInvalidFEN)
		}
	}

	enPassantPos := flagNoEnpassant
	if segments[3] != "-" {
		var err error
		enPassantPos, err = position.NewPosFromNotation(segments[3])
		if err != nil {
			return nil, nil, CastleRights(0), position.Pos(0), 0, 0, SideUnknown, fmt.Errorf("%w: %v", fmt.Errorf("%w: invalid enpassant position", ErrInvalidFEN), err)
		}
	}

	halfMoveClock, err := strconv.ParseUint(segments[4], 10, 64)
	if err != nil {
		return nil, nil, CastleRights(0), position.Pos(0), 0, 0, SideUnknown, fmt.Errorf("%w: invalid half move clock", ErrInvalidFEN)
	}

	fullMoveClock, err := strconv.ParseUint(segments[5], 10, 64)
	if err != nil {
		return nil, nil, CastleRights(0), position.Pos(0), 0, 0, SideUnknown, fmt.Errorf("%w: invalid full move clock", ErrInvalidFEN)
	}

	return sides, pieces, castleRights, enPassantPos, halfMoveClock, fullMoveClock, turn, nil
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
				for p, pBM := range b.pieces {
					if maskCell[y*Width+x]&pBM != 0 {
						s := SideWhite
						if maskCell[y*Width+x]&b.sides[SideBlack] != 0 {
							s = SideBlack
						}
						_, _ = builder.WriteString(p.SymbolFEN(s))
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

	if b.enPassantPos == flagNoEnpassant {
		_, _ = builder.WriteRune('-')
	} else {
		_, _ = builder.WriteString(b.enPassantPos.Notation())
	}

	_, _ = builder.WriteString(fmt.Sprintf(" %d %d", b.halfMoveClock, b.fullMoveClock))

	return builder.String()
}

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

func (b *Board) GenerateMoves(s Side) []*Move {
	var mvs []*Move
	for p := range b.pieces {
		mvs = append(mvs, b.GenerateMovesForPiece(s, p)...)
	}
	return mvs
}

func (b *Board) GenerateMovesForPiece(s Side, p Piece) []*Move {
	if mvs, ok := b.cacheMoves[s][p]; ok {
		return mvs
	}

	// TRY: if in check, only allow moves to stop it. currently only filter in-post (probably good enough)
	var mvs []*Move
	fromBM := b.getBitmap(s, p)
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
							IsSide:    s,
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
						IsSide: s,
						Piece:  p,
						From:   fromPos,
						To:     toPos,
					},
				)
			}
			for _, mv := range candidateMoves {
				// flag enpassant
				mv.IsEnPassant = p == PiecePawn && mv.To == b.enPassantPos

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
				mvs = append(mvs, mv)
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
				mvs = append(mvs, &Move{
					IsSide:   s,
					Piece:    p,
					IsCastle: d,
				})
			}
		}
	}

	b.cacheMoves[s][p] = mvs
	return mvs
}

func (b *Board) isKingChecked(s Side) bool {
	return b.getBitmap(s, PieceKing)&b.genAttackArea(s.Opposite()) != 0
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
			attackBM |= b.genValidDestination(pos, s, p)
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
		var maskEnPassant bitmap
		if b.enPassantPos != flagNoEnpassant {
			maskEnPassant = maskCell[b.enPassantPos]
		}
		if s == SideWhite {
			moveN1 := ShiftN(cell&^maskRow[7]) &^ b.occupied
			moveN2 := ShiftN(moveN1&maskRow[2]) &^ b.occupied
			captureNW := ShiftNW(cell&^maskRow[7]&^maskCol[0]) & (b.sides[SideBlack] | maskEnPassant)
			captureNE := ShiftNE(cell&^maskRow[7]&^maskCol[7]) & (b.sides[SideBlack] | maskEnPassant)
			return moveN1 | moveN2 | captureNW | captureNE
		}
		moveS1 := ShiftS(cell) &^ b.occupied
		moveS2 := ShiftS(moveS1&maskRow[5]) &^ b.occupied
		captureSW := ShiftSW(cell&^maskRow[0]&^maskCol[0]) & (b.sides[SideWhite] | maskEnPassant)
		captureSE := ShiftSE(cell&^maskRow[0]&^maskCol[7]) & (b.sides[SideWhite] | maskEnPassant)
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

func (b *Board) set(s Side, p Piece, pos position.Pos, value bool) {
	b.sides[s] = Set(b.sides[s], pos, value)
	b.pieces[p] = Set(b.pieces[p], pos, value)
	b.occupied = Set(b.occupied, pos, value)
}

// TODO: return sucess bool
func (b *Board) Apply(mv *Move) {
	if mv.IsCastle != CastleDirectionUnknown {
		hopsKing := posCastling[mv.IsCastle][PieceKing]
		hopsRook := posCastling[mv.IsCastle][PieceRook]
		b.set(b.turn, PieceKing, hopsKing[0], false)
		b.set(b.turn, PieceRook, hopsRook[0], false)
		b.set(b.turn, PieceKing, hopsKing[1], true)
		b.set(b.turn, PieceRook, hopsRook[1], true)
	} else {
		// remove from
		b.set(b.turn, mv.Piece, mv.From, false)

		// place to
		if mv.IsCapture {
			// remove captured piece
			if mv.IsEnPassant {
				var targetPawnPos position.Pos // pos of opponent Pawn to remove by enPassant
				switch b.turn {
				case SideWhite:
					targetPawnPos = mv.To - Width
				case SideBlack:
					targetPawnPos = mv.To + Width
				}
				b.set(b.turn.Opposite(), PiecePawn, targetPawnPos, false)
			} else {
				for p := range b.pieces {
					b.set(b.turn.Opposite(), p, mv.To, false)
				}
			}
		}
		if mv.IsPromote == PieceUnknown {
			b.set(b.turn, mv.Piece, mv.To, true)
		} else {
			b.set(b.turn, mv.IsPromote, mv.To, true)
		}
	}

	// update enPassantPos
	b.enPassantPos = flagNoEnpassant
	if mv.Piece == PiecePawn {
		if b.turn == SideWhite && maskCell[mv.From]&maskRow[1] != 0 && maskCell[mv.To]&maskRow[3] != 0 {
			b.enPassantPos = mv.To - Width
		} else if b.turn == SideBlack && maskCell[mv.From]&maskRow[6] != 0 && maskCell[mv.To]&maskRow[4] != 0 {
			b.enPassantPos = mv.To + Width
		}
	}

	// update castlingRights
	if mv.Piece == PieceKing {
		if b.turn == SideWhite {
			b.castleRights.Set(CastleDirectionWhiteRight, false)
			b.castleRights.Set(CastleDirectionWhiteLeft, false)
		} else {
			b.castleRights.Set(CastleDirectionBlackRight, false)
			b.castleRights.Set(CastleDirectionBlackLeft, false)
		}
	}
	if mv.Piece == PieceRook {
		if maskCell[mv.From]&maskCol[7] != 0 {
			if b.turn == SideWhite {
				b.castleRights.Set(CastleDirectionWhiteRight, false)
			} else {
				b.castleRights.Set(CastleDirectionBlackRight, false)
			}
		}
		if maskCell[mv.From]&maskCol[0] != 0 {
			if b.turn == SideWhite {
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
	if b.turn == SideBlack {
		b.fullMoveClock++
	}

	// set update turn
	b.turn = b.turn.Opposite()

	// flush cache
	b.cacheMoves = initCacheMoves()
}

func (b *Board) getBitmap(s Side, p Piece) bitmap {
	return b.sides[s] & b.pieces[p]
}

func (b *Board) Dump() string {
	builder := strings.Builder{}
	for y := position.Pos(Height) - 1; y >= 0; y-- {
		_, _ = builder.WriteString("   +---+---+---+---+---+---+---+---+\n")
		_, _ = builder.WriteString(fmt.Sprintf(" %d |", y+1))
		for x := position.Pos(0); x < Width; x++ {
			s, p := b.getSideAndPiecesByPos((y*Width + x))
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
			s, p := b.getSideAndPiecesByPos((y*Width + x))
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

func (b *Board) getSideAndPiecesByPos(i position.Pos) (Side, Piece) {
	s := SideUnknown
	for side, sideMap := range b.sides {
		if sideMap&maskCell[i] != 0 {
			s = side
			break
		}
	}
	p := PieceUnknown
	for piece, pieceMap := range b.pieces {
		if pieceMap&maskCell[i] != 0 {
			p = piece
			break
		}
	}
	return s, p
}

func (b *Board) Clone() *Board {
	sides := make(map[Side]bitmap, 2)
	for s, m := range b.sides {
		sides[s] = m
	}
	pieces := make(map[Piece]bitmap, 6)
	for p, m := range b.pieces {
		pieces[p] = m
	}
	return &Board{
		sides:         sides,
		pieces:        pieces,
		occupied:      b.occupied,
		enPassantPos:  b.enPassantPos,
		castleRights:  b.castleRights,
		halfMoveClock: b.halfMoveClock,
		fullMoveClock: b.fullMoveClock,
		state:         b.state,
		turn:          b.turn,
		cacheMoves:    initCacheMoves(),
	}
}
