package board

import (
	"errors"
	"fmt"
	"math/bits"
	"strings"

	"github.com/daystram/gambit/position"
)

var (
	ErrInvalidFEN  = errors.New("invalid fen")
	ErrInvalidMove = errors.New("invalid move")
)

type bitmap uint64
type sideBitmaps [3]bitmap
type pieceBitmaps [7]bitmap
type cellList [64]uint8
type sideValue [3]int32

// Little-endian rank-file (LERF) mapping
type Board struct {
	// grid data
	sides         sideBitmaps
	pieces        pieceBitmaps
	occupied      bitmap
	cells         cellList
	materialValue sideValue
	positionValue sideValue

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
	sides, pieces, cells, materialValue, positionValue, castleRights, enPassant, halfMoveClock, fullMoveClock, turn, err := parseFEN(cfg.fen)
	if err != nil {
		return nil, SideUnknown, err
	}

	return &Board{
		sides:         sides,
		pieces:        pieces,
		occupied:      Union(sides[SideBlack], sides[SideWhite]),
		cells:         cells,
		materialValue: materialValue,
		positionValue: positionValue,
		enPassant:     enPassant,
		castleRights:  castleRights,
		halfMoveClock: halfMoveClock,
		fullMoveClock: fullMoveClock,
		turn:          turn,
	}, turn, nil
}

func (b *Board) IsLegal(mv *Move) bool {
	var c int
	ourSide, oppSide := b.turn, b.turn.Opposite()

	// check if move leaves our King in check
	bb := b.Clone()
	bb.Apply(mv)
	if c, _ = bb.GetCellAttackers(oppSide, bb.GetBitmap(ourSide, PieceKing).LS1B(), 1); c != 0 {
		return false
	}

	// check if cells between castling is attacked
	switch mv.IsCastle {
	case CastleDirectionWhiteLeft:
		if c, _ = b.GetCellAttackers(oppSide, position.C1, 1); c != 0 {
			return false
		}
		if c, _ = b.GetCellAttackers(oppSide, position.D1, 1); c != 0 {
			return false
		}
	case CastleDirectionWhiteRight:
		if c, _ = b.GetCellAttackers(oppSide, position.F1, 1); c != 0 {
			return false
		}
		if c, _ = b.GetCellAttackers(oppSide, position.G1, 1); c != 0 {
			return false
		}
	case CastleDirectionBlackLeft:
		if c, _ = b.GetCellAttackers(oppSide, position.C8, 1); c != 0 {
			return false
		}
		if c, _ = b.GetCellAttackers(oppSide, position.D8, 1); c != 0 {
			return false
		}
	case CastleDirectionBlackRight:
		if c, _ = b.GetCellAttackers(oppSide, position.F8, 1); c != 0 {
			return false
		}
		if c, _ = b.GetCellAttackers(oppSide, position.G8, 1); c != 0 {
			return false
		}
	}

	return true
}

func (b *Board) GeneratePseudoLegalMoves() []*Move {
	mvs := make([]*Move, 0, 64)
	opponentSide := b.turn.Opposite()
	sideMask := b.sides[b.turn]
	nonSelfMask := ^sideMask

	kingPos := b.GetBitmap(b.turn, PieceKing).LS1B()
	checkerCount, attackedMask := b.GetCellAttackers(opponentSide, kingPos, 2)

	if checkerCount == 2 {
		b.generateMoveKing(&mvs, kingPos, (^attackedMask|b.sides[opponentSide])&nonSelfMask)
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
	return mvs
}

func (b *Board) GetCellAttackers(attackerSide Side, pos position.Pos, limit int) (int, bitmap) {
	var count int
	var attackBM bitmap
	attackerSideMask := b.sides[attackerSide]
	posMask := maskCell[pos]

	// find lateral attacker pieces
	candidateRay := HitLaterals(pos, magicRookMask[pos]&(b.occupied))
	attackerLaterals := candidateRay & attackerSideMask & (b.pieces[PieceRook] | b.pieces[PieceQueen])
	countLateral := bits.OnesCount64(uint64(attackerLaterals))
	count += countLateral
	attackBM |= attackerLaterals
	if count >= limit {
		return count, attackBM
	}

	// find diagonal attacker pieces
	candidateRay = HitDiagonals(pos, magicBishopMask[pos]&(b.occupied))
	attackerDiagonals := candidateRay & attackerSideMask & (b.pieces[PieceBishop] | b.pieces[PieceQueen])
	countDiagonal := bits.OnesCount64(uint64(attackerDiagonals))
	count += countDiagonal
	attackBM |= attackerDiagonals
	if count >= limit {
		return count, attackBM
	}

	// fill rays
	for attackerLaterals != 0 {
		attackerPos := position.Pos(bits.TrailingZeros64(uint64(attackerLaterals)))
		attackerLaterals &= attackerLaterals - 1

		attackBM |= HitLaterals(pos, maskCell[attackerPos]) & HitLaterals(attackerPos, posMask)
	}
	for attackerDiagonals != 0 {
		attackerPos := position.Pos(bits.TrailingZeros64(uint64(attackerDiagonals)))
		attackerDiagonals &= attackerDiagonals - 1

		attackBM |= HitDiagonals(pos, maskCell[attackerPos]) & HitDiagonals(attackerPos, posMask)
	}

	// find Knight attacks
	if attackerKnights := maskKnight[pos] & attackerSideMask & b.pieces[PieceKnight]; attackerKnights != 0 {
		count += bits.OnesCount64(uint64(attackerKnights))
		attackBM |= attackerKnights
		if count >= limit {
			return count, attackBM
		}
	}

	// find Pawn attacks
	if attackerSide == SideWhite {
		if attackerPawns := (ShiftSW(posMask&^maskRow[0]&^maskCol[0]) | ShiftSE(posMask&^maskRow[0]&^maskCol[7])) & attackerSideMask & b.pieces[PiecePawn]; attackerPawns != 0 {
			count += bits.OnesCount64(uint64(attackerPawns))
			attackBM |= attackerPawns
			if count >= limit {
				return count, attackBM
			}
		}
	} else {
		if attackerPawns := (ShiftNW(posMask&^maskRow[7]&^maskCol[0]) | ShiftNE(posMask&^maskRow[7]&^maskCol[7])) & attackerSideMask & b.pieces[PiecePawn]; attackerPawns != 0 {
			count += bits.OnesCount64(uint64(attackerPawns))
			attackBM |= attackerPawns
			if count >= limit {
				return count, attackBM
			}
		}
	}

	// find King attacks
	if attackersKing := maskKing[pos] & attackerSideMask & b.pieces[PieceKing]; attackersKing != 0 {
		count++
		attackBM |= attackersKing
	}

	return count, attackBM
}

func (b *Board) IsKingChecked() bool {
	c, _ := b.GetCellAttackers(b.turn.Opposite(), b.GetBitmap(b.turn, PieceKing).LS1B(), 1)
	return c != 0
}

func (b *Board) generateMovePawn(mvs *[]*Move, fromMask, allowedToMask bitmap) {
	for fromMask != 0 {
		fromPos := position.Pos(bits.TrailingZeros64(uint64(fromMask)))
		fromCell := maskCell[fromPos] & fromMask
		fromMask &= fromMask - 1

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

		for candidateToBM != 0 {
			toPos := position.Pos(bits.TrailingZeros64(uint64(candidateToBM)))
			toCell := maskCell[toPos] & candidateToBM
			candidateToBM &= candidateToBM - 1

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
	for fromMask != 0 {
		fromPos := position.Pos(bits.TrailingZeros64(uint64(fromMask)))
		fromMask &= fromMask - 1

		candidateToBM := maskKnight[fromPos] & allowedToMask

		for candidateToBM != 0 {
			toPos := position.Pos(bits.TrailingZeros64(uint64(candidateToBM)))
			toCell := maskCell[toPos] & candidateToBM
			candidateToBM &= candidateToBM - 1

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
	for fromMask != 0 {
		fromPos := position.Pos(bits.TrailingZeros64(uint64(fromMask)))
		fromMask &= fromMask - 1

		candidateToBM := HitDiagonals(fromPos, b.occupied) & allowedToMask

		for candidateToBM != 0 {
			toPos := position.Pos(bits.TrailingZeros64(uint64(candidateToBM)))
			toCell := maskCell[toPos] & candidateToBM
			candidateToBM &= candidateToBM - 1

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
	for fromMask != 0 {
		fromPos := position.Pos(bits.TrailingZeros64(uint64(fromMask)))
		fromMask &= fromMask - 1

		candidateToBM := HitLaterals(fromPos, b.occupied) & allowedToMask

		for candidateToBM != 0 {
			toPos := position.Pos(bits.TrailingZeros64(uint64(candidateToBM)))
			toCell := maskCell[toPos] & candidateToBM
			candidateToBM &= candidateToBM - 1

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
	for fromMask != 0 {
		fromPos := position.Pos(bits.TrailingZeros64(uint64(fromMask)))
		fromMask &= fromMask - 1

		candidateToBM := (HitDiagonals(fromPos, b.occupied) | HitLaterals(fromPos, b.occupied)) & allowedToMask

		for candidateToBM != 0 {
			toPos := position.Pos(bits.TrailingZeros64(uint64(candidateToBM)))
			toCell := maskCell[toPos] & candidateToBM
			candidateToBM &= candidateToBM - 1

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

func (b *Board) generateMoveKing(mvs *[]*Move, fromPos position.Pos, allowedToMask bitmap) {
	candidateToBM := maskKing[fromPos] & allowedToMask

	for candidateToBM != 0 {
		toPos := position.Pos(bits.TrailingZeros64(uint64(candidateToBM)))
		toCell := maskCell[toPos] & candidateToBM
		candidateToBM &= candidateToBM - 1

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

func (b *Board) generateCastling(mvs *[]*Move) {
	if b.castleRights.IsSideAllowed(b.turn) {
		if b.turn == SideWhite {
			if b.castleRights.IsAllowed(CastleDirectionWhiteLeft) &&
				b.occupied&maskCastling[CastleDirectionWhiteLeft] == 0 {
				jump := posCastling[CastleDirectionWhiteLeft][PieceKing]
				*mvs = append(*mvs, &Move{
					From:     jump[0],
					To:       jump[1],
					Piece:    PieceKing,
					IsTurn:   b.turn,
					IsCastle: CastleDirectionWhiteLeft,
				})
			}
			if b.castleRights.IsAllowed(CastleDirectionWhiteRight) &&
				b.occupied&maskCastling[CastleDirectionWhiteRight] == 0 {
				jump := posCastling[CastleDirectionWhiteRight][PieceKing]
				*mvs = append(*mvs, &Move{
					From:     jump[0],
					To:       jump[1],
					Piece:    PieceKing,
					IsTurn:   b.turn,
					IsCastle: CastleDirectionWhiteRight,
				})
			}
		} else {
			if b.castleRights.IsAllowed(CastleDirectionBlackLeft) &&
				b.occupied&maskCastling[CastleDirectionBlackLeft] == 0 {
				jump := posCastling[CastleDirectionBlackLeft][PieceKing]
				*mvs = append(*mvs, &Move{
					From:     jump[0],
					To:       jump[1],
					Piece:    PieceKing,
					IsTurn:   b.turn,
					IsCastle: CastleDirectionBlackLeft,
				})
			}
			if b.castleRights.IsAllowed(CastleDirectionBlackRight) &&
				b.occupied&maskCastling[CastleDirectionBlackRight] == 0 {
				jump := posCastling[CastleDirectionBlackRight][PieceKing]
				*mvs = append(*mvs, &Move{
					From:     jump[0],
					To:       jump[1],
					Piece:    PieceKing,
					IsTurn:   b.turn,
					IsCastle: CastleDirectionBlackRight,
				})
			}
		}
	}
}

func (b *Board) flip(s Side, p Piece, pos position.Pos) {
	b.sides[s] ^= maskCell[pos]
	b.pieces[p] ^= maskCell[pos]
	b.occupied ^= maskCell[pos]
	b.hash ^= zobristConstantPiece[s][p][pos]
}

func (b *Board) NewMoveFromUCI(notation string) (*Move, error) {
	if len(notation) < 4 || len(notation) > 5 {
		return nil, ErrInvalidMove
	}

	var err error
	mv := &Move{}

	mv.From, err = position.NewPosFromNotation(notation[0:2])
	if err != nil {
		return nil, err
	}
	mv.To, err = position.NewPosFromNotation(notation[2:4])
	if err != nil {
		return nil, err
	}
	mv.IsTurn, mv.Piece = b.GetSideAndPieces(mv.From)
	mv.IsEnPassant = mv.Piece == PiecePawn && maskCell[mv.To] == b.enPassant
	mv.IsCapture = b.occupied&maskCell[mv.To] != 0 || mv.IsEnPassant
	if mv.Piece == PieceKing {
		switch notation {
		case "e1g1":
			mv.IsCastle = CastleDirectionWhiteRight
		case "e1c1":
			mv.IsCastle = CastleDirectionWhiteLeft
		case "e8g8":
			mv.IsCastle = CastleDirectionBlackRight
		case "e8c8":
			mv.IsCastle = CastleDirectionBlackLeft
		}
	}
	if len(notation) == 5 {
		switch notation[4] {
		case 'n':
			mv.IsPromote = PieceKnight
		case 'b':
			mv.IsPromote = PieceBishop
		case 'r':
			mv.IsPromote = PieceRook
		case 'q':
			mv.IsPromote = PieceQueen
		}
	}
	return mv, nil
}

type UnApplyFunc func()

func (b *Board) ApplyNull() UnApplyFunc {
	ourTurn, oppTurn := b.turn, b.turn.Opposite()

	// disable enpassant
	prevEnPassant := b.enPassant
	b.hash ^= zobristConstantEnPassant[b.enPassant.LS1B()]
	b.enPassant = bitmap(0)
	b.hash ^= zobristConstantEnPassant[b.enPassant.LS1B()]

	// reset half move clock
	prevHalfMoveClock := b.halfMoveClock
	b.halfMoveClock = 0

	// update full move clock
	if ourTurn == SideBlack {
		b.fullMoveClock++
	}

	// update ply
	b.ply++

	// update turn
	b.turn = oppTurn
	b.hash ^= zobristConstantSideWhite

	// reset state cache
	prevState := b.state
	b.state = StateUnknown

	return func() {
		// revert enpassant
		b.hash ^= zobristConstantEnPassant[b.enPassant.LS1B()]
		b.enPassant = prevEnPassant
		b.hash ^= zobristConstantEnPassant[b.enPassant.LS1B()]

		// revert half move clock
		b.halfMoveClock = prevHalfMoveClock

		// revert full move clock
		if ourTurn == SideBlack {
			b.fullMoveClock--
		}

		// revert ply
		b.ply--

		// revert turn
		b.turn = ourTurn
		b.hash ^= zobristConstantSideWhite

		// revert state cache
		b.state = prevState
	}
}

// TODO: return undo func?
func (b *Board) Apply(mv *Move) {
	ourTurn := b.turn
	oppTurn := ourTurn.Opposite()
	_, capturedPiece := b.GetSideAndPieces(mv.To)
	capturedPos := mv.To

	if mv.IsCastle != CastleDirectionUnknown {
		// perform castling
		hopsKing := posCastling[mv.IsCastle][PieceKing]
		hopsRook := posCastling[mv.IsCastle][PieceRook]

		b.flip(ourTurn, PieceKing, hopsKing[0])
		b.flip(ourTurn, PieceKing, hopsKing[1])
		b.cells[hopsKing[1]] = b.cells[hopsKing[0]]
		b.cells[hopsKing[0]] = 0
		b.positionValue[ourTurn] -= scorePosition[PieceKing][scorePositionMap[ourTurn][hopsKing[0]]]
		b.positionValue[ourTurn] += scorePosition[PieceKing][scorePositionMap[ourTurn][hopsKing[1]]]

		b.flip(ourTurn, PieceRook, hopsRook[0])
		b.flip(ourTurn, PieceRook, hopsRook[1])
		b.cells[hopsRook[1]] = b.cells[hopsRook[0]]
		b.cells[hopsRook[0]] = 0
		b.positionValue[ourTurn] -= scorePosition[PieceRook][scorePositionMap[ourTurn][hopsRook[0]]]
		b.positionValue[ourTurn] += scorePosition[PieceRook][scorePositionMap[ourTurn][hopsRook[1]]]
	} else {
		// remove moving piece at mv.From
		b.flip(ourTurn, mv.Piece, mv.From)
		b.cells[mv.From] = 0
		b.materialValue[ourTurn] -= scoreMaterial[mv.Piece]
		b.positionValue[ourTurn] -= scorePosition[mv.Piece][scorePositionMap[ourTurn][mv.From]]

		// remove captured piece at mv.To
		if mv.IsCapture {
			if mv.IsEnPassant {
				capturedPiece = PiecePawn
				capturedPos = mv.To - Width // pos of opponent Pawn to remove by enPassant
				if ourTurn == SideBlack {
					capturedPos = mv.To + Width
				}
			}
			b.flip(oppTurn, capturedPiece, capturedPos)
			b.cells[capturedPos] = 0
			b.materialValue[oppTurn] -= scoreMaterial[capturedPiece]
			b.positionValue[oppTurn] -= scorePosition[capturedPiece][scorePositionMap[oppTurn][capturedPos]]
		}

		// place moving piece at mv.To
		movingPiece := mv.Piece
		if mv.IsPromote != PieceUnknown {
			movingPiece = mv.IsPromote
		}
		b.flip(ourTurn, movingPiece, mv.To)
		b.setSideAndPieces(mv.To, ourTurn, movingPiece)
		b.materialValue[ourTurn] += scoreMaterial[movingPiece]
		b.positionValue[ourTurn] += scorePosition[movingPiece][scorePositionMap[ourTurn][mv.To]]
	}

	// update enPassant
	b.hash ^= zobristConstantEnPassant[b.enPassant.LS1B()]
	b.enPassant = bitmap(0)
	if mv.Piece == PiecePawn {
		if ourTurn == SideWhite && maskCell[mv.From]&maskRow[1] != 0 && maskCell[mv.To]&maskRow[3] != 0 {
			b.enPassant = maskCell[mv.To-Width]
		} else if ourTurn == SideBlack && maskCell[mv.From]&maskRow[6] != 0 && maskCell[mv.To]&maskRow[4] != 0 {
			b.enPassant = maskCell[mv.To+Width]
		}
	}
	b.hash ^= zobristConstantEnPassant[b.enPassant.LS1B()]

	// update castleRights
	b.hash ^= zobristConstantCastleRights[b.castleRights]
	if mv.Piece == PieceKing {
		if ourTurn == SideWhite {
			b.castleRights.Set(CastleDirectionWhiteRight, false)
			b.castleRights.Set(CastleDirectionWhiteLeft, false)
		} else {
			b.castleRights.Set(CastleDirectionBlackRight, false)
			b.castleRights.Set(CastleDirectionBlackLeft, false)
		}
	}
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
	// remove castling rights when Rook is captured
	if capturedPiece == PieceRook {
		if oppTurn == SideWhite {
			if capturedPos == position.H1 {
				b.castleRights.Set(CastleDirectionWhiteRight, false)
			}
			if capturedPos == position.A1 {
				b.castleRights.Set(CastleDirectionWhiteLeft, false)
			}
		} else {
			if capturedPos == position.H8 {
				b.castleRights.Set(CastleDirectionBlackRight, false)
			}
			if capturedPos == position.A8 {
				b.castleRights.Set(CastleDirectionBlackLeft, false)
			}
		}
	}
	b.hash ^= zobristConstantCastleRights[b.castleRights]

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

	// reset state cache
	b.state = StateUnknown
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

	var legalMoves int
	for _, mv := range b.GeneratePseudoLegalMoves() {
		if b.IsLegal(mv) {
			legalMoves++
		}
	}
	if b.IsKingChecked() {
		if legalMoves == 0 {
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
	if legalMoves == 0 {
		return StateStalemate
	}

	// checkmate takes precedence over the 50 move rule
	if b.halfMoveClock >= 100 {
		return StateFiftyMoveViolated
	}

	return StateRunning
}

func (b *Board) GetMaterialValue() (int32, int32) {
	return b.materialValue[SideWhite], b.materialValue[SideBlack]
}

func (b *Board) GetPositionValue() (int32, int32) {
	return b.positionValue[SideWhite], b.positionValue[SideBlack]
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
		materialValue: b.materialValue,
		positionValue: b.positionValue,
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
