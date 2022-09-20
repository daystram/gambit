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
	Side     Side
	Piece    Piece

	IsCapture   bool
	IsCheck     bool
	IsCastle    CastleDirection
	IsEnPassant bool
	IsPromote   Piece
}

func (m Move) String() string {
	switch m.IsCastle {
	case CastleDirectionKing:
		return "0-0"
	case CastleDirectionQueen:
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
	CastleDirectionQueen
	CastleDirectionKing
)

type CastleRights uint8

var (
	maskCastleRights = map[Side]map[CastleDirection]CastleRights{
		SideWhite: {
			CastleDirectionKing:  0b1000,
			CastleDirectionQueen: 0b0100,
		},
		SideBlack: {
			CastleDirectionKing:  0b0010,
			CastleDirectionQueen: 0b0001,
		},
	}
)

func (c *CastleRights) Set(s Side, d CastleDirection, allow bool) {
	if allow {
		*c |= maskCastleRights[s][d]
	} else {
		*c &^= maskCastleRights[s][d]
	}
}

func (c *CastleRights) IsAllowed(s Side, d CastleDirection) bool {
	return *c&maskCastleRights[s][d] != 0
}

// Little-endian rank-file (LERF) mapping
type Board struct {
	sides        map[Side]bitmap
	pieces       map[Piece]bitmap
	occupied     bitmap
	enPassantPos position.Pos

	// meta
	castleRights  CastleRights
	halfMoveClock uint64
	fullMoveClock uint64
	state         State

	// cache
	cacheMoves map[Side]map[Piece][]*Move
}

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
			castleRights.Set(SideWhite, CastleDirectionKing, true)
		case 'k':
			castleRights.Set(SideBlack, CastleDirectionKing, true)
		case 'Q':
			castleRights.Set(SideWhite, CastleDirectionQueen, true)
		case 'q':
			castleRights.Set(SideBlack, CastleDirectionQueen, true)
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
							Side:      s,
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
						Side:  s,
						Piece: p,
						From:  fromPos,
						To:    toPos,
					},
				)
			}
			for _, mv := range candidateMoves {
				// flag enpassant
				mv.IsEnPassant = mv.To == b.enPassantPos

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
	if p == PieceKing && (b.castleRights.IsAllowed(s, CastleDirectionKing) || b.castleRights.IsAllowed(s, CastleDirectionQueen)) {
		oppositeAttackBM := b.genAttackArea(s.Opposite())
		for _, d := range []CastleDirection{CastleDirectionKing, CastleDirectionQueen} {
			if b.castleRights.IsAllowed(s, d) &&
				maskCastling[s][d]&oppositeAttackBM == 0 &&
				maskCastling[s][d]&b.occupied == 0 {
				mvs = append(mvs, &Move{
					Side:     s,
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
		hopsKing := posCastling[mv.Side][mv.IsCastle][PieceKing]
		hopsRook := posCastling[mv.Side][mv.IsCastle][PieceRook]
		b.set(mv.Side, PieceKing, hopsKing[0], false)
		b.set(mv.Side, PieceRook, hopsRook[0], false)
		b.set(mv.Side, PieceKing, hopsKing[1], true)
		b.set(mv.Side, PieceRook, hopsRook[1], true)
	} else {
		// remove from
		b.set(mv.Side, mv.Piece, mv.From, false)

		// place to
		if mv.IsCapture {
			// remove captured piece
			if mv.IsEnPassant {
				var targetPawnPos position.Pos
				switch mv.Side {
				case SideWhite:
					targetPawnPos = mv.To - Width
				case SideBlack:
					targetPawnPos = mv.To + Width
				}
				b.set(mv.Side.Opposite(), PiecePawn, targetPawnPos, false)
			} else {
				for p := range b.pieces {
					b.set(mv.Side.Opposite(), p, mv.To, false)
				}
			}
		}
		if mv.IsPromote == PieceUnknown {
			b.set(mv.Side, mv.Piece, mv.To, true)
		} else {
			b.set(mv.Side, mv.IsPromote, mv.To, true)
		}
	}

	// update enPassantPos
	if mv.Piece == PiecePawn {
		if mv.Side == SideWhite && maskCell[mv.From]&maskRow[1] != 0 && maskCell[mv.To]&maskRow[3] != 0 {
			b.enPassantPos = mv.To - Width
		} else if mv.Side == SideBlack && maskCell[mv.From]&maskRow[6] != 0 && maskCell[mv.To]&maskRow[4] != 0 {
			b.enPassantPos = mv.To + Width
		}
	} else {
		b.enPassantPos = flagNoEnpassant
	}

	// update castlingRights
	if mv.Piece == PieceKing {
		b.castleRights.Set(mv.Side, CastleDirectionKing, false)
		b.castleRights.Set(mv.Side, CastleDirectionQueen, false)
	}
	if mv.Piece == PieceRook {
		if maskCell[mv.From]&maskCol[7] != 0 {
			b.castleRights.Set(mv.Side, CastleDirectionKing, false)
		}
		if maskCell[mv.From]&maskCol[0] != 0 {
			b.castleRights.Set(mv.Side, CastleDirectionQueen, false)
		}

	}

	// update half move clock
	if mv.Piece == PiecePawn || mv.IsCapture {
		b.halfMoveClock = 0
	} else {
		b.halfMoveClock++
	}

	// update full move clock
	if mv.Side == SideBlack {
		b.fullMoveClock++
	}

	// flush cache
	b.cacheMoves = initCacheMoves()
}

func (b *Board) getBitmap(s Side, p Piece) bitmap {
	return b.sides[s] & b.pieces[p]
}

func (b *Board) Dump() string {
	builder := strings.Builder{}
	for y := position.Pos(Height); y > 0; y-- {
		_, _ = builder.WriteString(fmt.Sprintf(" %d |", y))
		for x := position.Pos(0); x < Width; x++ {
			s, p := b.getSideAndPiecesByPos(((y-1)*Width + x))
			sym := p.SymbolFEN(s)
			if s == SideUnknown {
				sym = "."
			}
			_, _ = builder.WriteString(fmt.Sprintf(" %s ", sym))
		}
		_, _ = builder.WriteString("\n")
	}
	_, _ = builder.WriteString("    ------------------------\n    ")
	for x := position.Pos(0); x < Width; x++ {
		_, _ = builder.WriteString(fmt.Sprintf(" %s ", x.NotationComponentX()))
	}
	return builder.String()
}

func (b *Board) Draw() string {
	builder := strings.Builder{}
	for y := position.Pos(Height); y > 0; y-- {
		_, _ = builder.WriteString(fmt.Sprintf("\033[1m %d \033[0m", y))
		for x := position.Pos(0); x < Width; x++ {
			s, p := b.getSideAndPiecesByPos(((y-1)*Width + x))
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
		cacheMoves:    initCacheMoves(),
	}
}
