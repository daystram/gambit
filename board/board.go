package board

import (
	"errors"
	"fmt"
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
	occupied        bitmap
	sides           sideBitmaps
	pieces          pieceBitmaps
	cells           cellList
	materialValue   sideValue
	positionValueMG sideValue
	positionValueEG sideValue
	phase           uint8

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
	sides, pieces, cells, materialValue, positionValueMG, positionValueEG, castleRights, enPassant, halfMoveClock, fullMoveClock, turn, err := parseFEN(cfg.fen)
	if err != nil {
		return nil, SideUnknown, err
	}

	return &Board{
		occupied:        Union(sides[SideBlack], sides[SideWhite]),
		sides:           sides,
		pieces:          pieces,
		cells:           cells,
		materialValue:   materialValue,
		positionValueMG: positionValueMG,
		positionValueEG: positionValueEG,
		phase:           PhaseTotal, // TODO: need to calc phase on non-startpos starting states?
		enPassant:       enPassant,
		castleRights:    castleRights,
		halfMoveClock:   halfMoveClock,
		fullMoveClock:   fullMoveClock,
		ply:             0,
		state:           StateUnknown,
		turn:            turn,
		hash:            0, // TODO: need to calc hash on non-startpos starting states?
	}, turn, nil
}

func (b *Board) IsLegal(mv Move) bool {
	unApply, isLegal := b.Apply(mv)
	unApply()
	return isLegal
}

func (b *Board) GeneratePseudoLegalMoves() []Move {
	mvs := make([]Move, 0, 64)
	theirSide := b.turn.Opposite()
	sideMask := b.sides[b.turn]
	nonSelfMask := ^sideMask

	kingPos := b.GetBitmap(b.turn, PieceKing).LS1B()
	checkerCount, attackedMask := b.GetCellAttackers(theirSide, kingPos, 2)

	if checkerCount == 2 {
		b.generateMoveKing(&mvs, kingPos, (^attackedMask|b.sides[theirSide])&nonSelfMask)
		return mvs
	}

	if checkerCount == 1 {
		b.generateMovePawn(&mvs, sideMask&b.pieces[PiecePawn], attackedMask)
		b.generateMoveKnight(&mvs, sideMask&b.pieces[PieceKnight], attackedMask)
		b.generateMoveBishop(&mvs, sideMask&b.pieces[PieceBishop], attackedMask)
		b.generateMoveRook(&mvs, sideMask&b.pieces[PieceRook], attackedMask)
		b.generateMoveQueen(&mvs, sideMask&b.pieces[PieceQueen], attackedMask)
		b.generateMoveKing(&mvs, kingPos, (^attackedMask|b.sides[theirSide])&nonSelfMask)
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

func (b *Board) GetCellAttackers(attackerSide Side, pos position.Pos, limit uint8) (uint8, bitmap) {
	var count uint8
	var attackBM bitmap
	attackerSideMask := b.sides[attackerSide]
	posMask := maskCell[pos]

	// find lateral attacker pieces
	m := magicRook[pos]
	attackerLaterals := m.Attacks[m.GetIndex(b.occupied)] & attackerSideMask & (b.pieces[PieceRook] | b.pieces[PieceQueen])
	countLateral := attackerLaterals.BitCount()
	count += countLateral
	attackBM |= attackerLaterals
	if count >= limit {
		return count, attackBM
	}

	// find diagonal attacker pieces
	m = magicBishop[pos]
	attackerDiagonals := m.Attacks[m.GetIndex(b.occupied)] & attackerSideMask & (b.pieces[PieceBishop] | b.pieces[PieceQueen])
	count += attackerDiagonals.BitCount()
	attackBM |= attackerDiagonals
	if count >= limit {
		return count, attackBM
	}

	// fill rays
	for attackerLaterals != 0 {
		attackerPos := attackerLaterals.LS1B()
		attackerLaterals &= attackerLaterals - 1

		m1, m2 := magicRook[pos], magicRook[attackerPos]
		attackBM |= m1.Attacks[m1.GetIndex(maskCell[attackerPos])] & m2.Attacks[m2.GetIndex(posMask)]
	}
	for attackerDiagonals != 0 {
		attackerPos := attackerDiagonals.LS1B()
		attackerDiagonals &= attackerDiagonals - 1

		m1, m2 := magicBishop[pos], magicBishop[attackerPos]
		attackBM |= m1.Attacks[m1.GetIndex(maskCell[attackerPos])] & m2.Attacks[m2.GetIndex(posMask)]
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
		if attackerPawns := (ShiftSW(posMask&^maskRow[0]&^maskCol[0]) | ShiftSE(posMask&^maskRow[0]&^maskCol[7])) & attackerSideMask & b.pieces[PiecePawn]; attackerPawns != 0 {
			count += attackerPawns.BitCount()
			attackBM |= attackerPawns
			if count >= limit {
				return count, attackBM
			}
		}
	} else {
		if attackerPawns := (ShiftNW(posMask&^maskRow[7]&^maskCol[0]) | ShiftNE(posMask&^maskRow[7]&^maskCol[7])) & attackerSideMask & b.pieces[PiecePawn]; attackerPawns != 0 {
			count += attackerPawns.BitCount()
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

func (b *Board) IsKingChecked(s Side) bool {
	c, _ := b.GetCellAttackers(s.Opposite(), b.GetBitmap(s, PieceKing).LS1B(), 1)
	return c != 0
}

func (b *Board) generateMovePawn(mvs *[]Move, fromMask, allowedToMask bitmap) {
	for fromMask != 0 {
		fromPos := fromMask.LS1B()
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
			toPos := candidateToBM.LS1B()
			toCell := maskCell[toPos]
			candidateToBM &= candidateToBM - 1

			isEnPassant := toCell == b.enPassant
			isCapture := toCell&b.occupied != 0 || isEnPassant
			if toCell&(maskRow[0]|maskRow[7]) == 0 {
				*mvs = append(*mvs, Move{
					From:        fromPos,
					To:          toPos,
					Piece:       PiecePawn,
					IsTurn:      b.turn,
					IsCapture:   isCapture,
					IsEnPassant: isEnPassant,
				})
			} else {
				for _, prom := range PawnPromoteCandidates {
					*mvs = append(*mvs, Move{
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

func (b *Board) generateMoveKnight(mvs *[]Move, fromMask, allowedToMask bitmap) {
	for fromMask != 0 {
		fromPos := fromMask.LS1B()
		fromMask &= fromMask - 1

		candidateToBM := maskKnight[fromPos] & allowedToMask

		for candidateToBM != 0 {
			toPos := candidateToBM.LS1B()
			toCell := maskCell[toPos]
			candidateToBM &= candidateToBM - 1

			isCapture := toCell&b.occupied != 0
			*mvs = append(*mvs, Move{
				From:      fromPos,
				To:        toPos,
				Piece:     PieceKnight,
				IsTurn:    b.turn,
				IsCapture: isCapture,
			})
		}
	}
}

func (b *Board) generateMoveBishop(mvs *[]Move, fromMask, allowedToMask bitmap) {
	for fromMask != 0 {
		fromPos := fromMask.LS1B()
		fromMask &= fromMask - 1

		m := magicBishop[fromPos]
		candidateToBM := m.Attacks[m.GetIndex(b.occupied)] & allowedToMask

		for candidateToBM != 0 {
			toPos := candidateToBM.LS1B()
			toCell := maskCell[toPos]
			candidateToBM &= candidateToBM - 1

			isCapture := toCell&b.occupied != 0
			*mvs = append(*mvs, Move{
				From:      fromPos,
				To:        toPos,
				Piece:     PieceBishop,
				IsTurn:    b.turn,
				IsCapture: isCapture,
			})
		}
	}
}

func (b *Board) generateMoveRook(mvs *[]Move, fromMask, allowedToMask bitmap) {
	for fromMask != 0 {
		fromPos := fromMask.LS1B()
		fromMask &= fromMask - 1

		m := magicRook[fromPos]
		candidateToBM := m.Attacks[m.GetIndex(b.occupied)] & allowedToMask

		for candidateToBM != 0 {
			toPos := candidateToBM.LS1B()
			toCell := maskCell[toPos]
			candidateToBM &= candidateToBM - 1

			isCapture := toCell&b.occupied != 0
			*mvs = append(*mvs, Move{
				From:      fromPos,
				To:        toPos,
				Piece:     PieceRook,
				IsTurn:    b.turn,
				IsCapture: isCapture,
			})
		}
	}
}

func (b *Board) generateMoveQueen(mvs *[]Move, fromMask, allowedToMask bitmap) {
	for fromMask != 0 {
		fromPos := fromMask.LS1B()
		fromMask &= fromMask - 1

		m1, m2 := magicBishop[fromPos], magicRook[fromPos]
		candidateToBM := (m1.Attacks[m1.GetIndex(b.occupied)] | m2.Attacks[m2.GetIndex(b.occupied)]) & allowedToMask

		for candidateToBM != 0 {
			toPos := candidateToBM.LS1B()
			toCell := maskCell[toPos]
			candidateToBM &= candidateToBM - 1

			isCapture := toCell&b.occupied != 0
			*mvs = append(*mvs, Move{
				From:      fromPos,
				To:        toPos,
				Piece:     PieceQueen,
				IsTurn:    b.turn,
				IsCapture: isCapture,
			})
		}
	}
}

func (b *Board) generateMoveKing(mvs *[]Move, fromPos position.Pos, allowedToMask bitmap) {
	candidateToBM := maskKing[fromPos] & allowedToMask

	for candidateToBM != 0 {
		toPos := candidateToBM.LS1B()
		toCell := maskCell[toPos]
		candidateToBM &= candidateToBM - 1

		isCapture := toCell&b.occupied != 0
		*mvs = append(*mvs, Move{
			From:      fromPos,
			To:        toPos,
			Piece:     PieceKing,
			IsTurn:    b.turn,
			IsCapture: isCapture,
		})
	}
}

func (b *Board) generateCastling(mvs *[]Move) {
	ourSide, theirSide := b.turn, b.turn.Opposite()
	if b.castleRights.IsSideAllowed(ourSide) {
		if ourSide == SideWhite {
			if b.castleRights.IsAllowed(CastleDirectionWhiteLeft) &&
				b.occupied&maskCastling[CastleDirectionWhiteLeft] == 0 {
				if c, _ := b.GetCellAttackers(theirSide, position.C1, 1); c == 0 {
					if c, _ = b.GetCellAttackers(theirSide, position.D1, 1); c == 0 {
						jump := posCastling[CastleDirectionWhiteLeft][PieceKing]
						*mvs = append(*mvs, Move{
							From:     jump[0],
							To:       jump[1],
							Piece:    PieceKing,
							IsTurn:   ourSide,
							IsCastle: CastleDirectionWhiteLeft,
						})
					}
				}
			}
			if b.castleRights.IsAllowed(CastleDirectionWhiteRight) &&
				b.occupied&maskCastling[CastleDirectionWhiteRight] == 0 {
				if c, _ := b.GetCellAttackers(theirSide, position.F1, 1); c == 0 {
					if c, _ = b.GetCellAttackers(theirSide, position.G1, 1); c == 0 {
						jump := posCastling[CastleDirectionWhiteRight][PieceKing]
						*mvs = append(*mvs, Move{
							From:     jump[0],
							To:       jump[1],
							Piece:    PieceKing,
							IsTurn:   ourSide,
							IsCastle: CastleDirectionWhiteRight,
						})
					}
				}
			}
		} else {
			if b.castleRights.IsAllowed(CastleDirectionBlackLeft) &&
				b.occupied&maskCastling[CastleDirectionBlackLeft] == 0 {
				if c, _ := b.GetCellAttackers(theirSide, position.C8, 1); c == 0 {
					if c, _ = b.GetCellAttackers(theirSide, position.D8, 1); c == 0 {
						jump := posCastling[CastleDirectionBlackLeft][PieceKing]
						*mvs = append(*mvs, Move{
							From:     jump[0],
							To:       jump[1],
							Piece:    PieceKing,
							IsTurn:   ourSide,
							IsCastle: CastleDirectionBlackLeft,
						})
					}
				}
			}
			if b.castleRights.IsAllowed(CastleDirectionBlackRight) &&
				b.occupied&maskCastling[CastleDirectionBlackRight] == 0 {
				if c, _ := b.GetCellAttackers(theirSide, position.F8, 1); c == 0 {
					if c, _ = b.GetCellAttackers(theirSide, position.G8, 1); c == 0 {
						jump := posCastling[CastleDirectionBlackRight][PieceKing]
						*mvs = append(*mvs, Move{
							From:     jump[0],
							To:       jump[1],
							Piece:    PieceKing,
							IsTurn:   ourSide,
							IsCastle: CastleDirectionBlackRight,
						})
					}
				}
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

func (b *Board) NewMoveFromUCI(notation string) (Move, error) {
	if len(notation) < 4 || len(notation) > 5 {
		return Move{}, ErrInvalidMove
	}

	var err error
	mv := Move{}

	mv.From, err = position.NewPosFromNotation(notation[0:2])
	if err != nil {
		return Move{}, err
	}
	mv.To, err = position.NewPosFromNotation(notation[2:4])
	if err != nil {
		return Move{}, err
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
	prevHash := b.hash

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
		b.enPassant = prevEnPassant

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

		// revert state cache
		b.state = prevState

		// revert hash
		b.hash = prevHash
	}
}

func (b *Board) Apply(mv Move) (UnApplyFunc, bool) {
	ourTurn, theirTurn := b.turn, b.turn.Opposite()
	fromPos, toPos, capturedPos := mv.From, mv.To, mv.To
	fromPiece, toPiece := mv.Piece, mv.Piece
	_, capturedPiece := b.GetSideAndPieces(mv.To)
	isCapture, isCastle := mv.IsCapture, mv.IsCastle
	prevHash := b.hash

	if isCastle != CastleDirectionUnknown {
		// perform castling
		hopsKing := posCastling[isCastle][PieceKing]
		hopsRook := posCastling[isCastle][PieceRook]

		b.flip(ourTurn, PieceKing, hopsKing[0])
		b.flip(ourTurn, PieceKing, hopsKing[1])
		b.cells[hopsKing[1]] = b.cells[hopsKing[0]]
		b.cells[hopsKing[0]] = 0
		b.positionValueMG[ourTurn] -= scorePositionMG[PieceKing][scorePositionMap[ourTurn][hopsKing[0]]]
		b.positionValueMG[ourTurn] += scorePositionMG[PieceKing][scorePositionMap[ourTurn][hopsKing[1]]]
		b.positionValueEG[ourTurn] -= scorePositionEG[PieceKing][scorePositionMap[ourTurn][hopsKing[0]]]
		b.positionValueEG[ourTurn] += scorePositionEG[PieceKing][scorePositionMap[ourTurn][hopsKing[1]]]

		b.flip(ourTurn, PieceRook, hopsRook[0])
		b.flip(ourTurn, PieceRook, hopsRook[1])
		b.cells[hopsRook[1]] = b.cells[hopsRook[0]]
		b.cells[hopsRook[0]] = 0
		b.positionValueMG[ourTurn] -= scorePositionMG[PieceRook][scorePositionMap[ourTurn][hopsRook[0]]]
		b.positionValueMG[ourTurn] += scorePositionMG[PieceRook][scorePositionMap[ourTurn][hopsRook[1]]]
		b.positionValueEG[ourTurn] -= scorePositionEG[PieceRook][scorePositionMap[ourTurn][hopsRook[0]]]
		b.positionValueEG[ourTurn] += scorePositionEG[PieceRook][scorePositionMap[ourTurn][hopsRook[1]]]
	} else {
		// remove moving piece at fromPos
		b.flip(ourTurn, fromPiece, fromPos)
		b.cells[fromPos] = 0
		b.materialValue[ourTurn] -= scoreMaterial[fromPiece]
		b.positionValueMG[ourTurn] -= scorePositionMG[fromPiece][scorePositionMap[ourTurn][fromPos]]
		b.positionValueEG[ourTurn] -= scorePositionEG[fromPiece][scorePositionMap[ourTurn][fromPos]]

		// remove captured piece at capturedPos
		if isCapture {
			if mv.IsEnPassant {
				capturedPiece = PiecePawn
				capturedPos = toPos - Width // pos of opponent Pawn to remove by enPassant
				if ourTurn == SideBlack {
					capturedPos = toPos + Width
				}
			}
			b.flip(theirTurn, capturedPiece, capturedPos)
			b.cells[capturedPos] = 0
			b.materialValue[theirTurn] -= scoreMaterial[capturedPiece]
			b.positionValueMG[theirTurn] -= scorePositionMG[capturedPiece][scorePositionMap[theirTurn][capturedPos]]
			b.positionValueEG[theirTurn] -= scorePositionEG[capturedPiece][scorePositionMap[theirTurn][capturedPos]]
			b.phase -= phaseConstant[capturedPiece]
		}

		// place moving piece at toPos
		if mv.IsPromote != PieceUnknown {
			toPiece = mv.IsPromote
		}
		b.flip(ourTurn, toPiece, toPos)
		b.setSideAndPieces(toPos, ourTurn, toPiece)
		b.materialValue[ourTurn] += scoreMaterial[toPiece]
		b.positionValueMG[ourTurn] += scorePositionMG[toPiece][scorePositionMap[ourTurn][toPos]]
		b.positionValueEG[ourTurn] += scorePositionEG[toPiece][scorePositionMap[ourTurn][toPos]]
	}

	// update enPassant
	prevEnPassant := b.enPassant
	b.hash ^= zobristConstantEnPassant[b.enPassant.LS1B()]
	b.enPassant = bitmap(0)
	if fromPiece == PiecePawn {
		if ourTurn == SideWhite && toPos-fromPos == 16 {
			b.enPassant = maskCell[toPos-Width]
		} else if ourTurn == SideBlack && fromPos-toPos == 16 {
			b.enPassant = maskCell[toPos+Width]
		}
	}
	b.hash ^= zobristConstantEnPassant[b.enPassant.LS1B()]

	// update castleRights
	prevCastleRights := b.castleRights
	b.hash ^= zobristConstantCastleRights[b.castleRights]
	if fromPiece == PieceKing {
		if ourTurn == SideWhite {
			b.castleRights.Set(CastleDirectionWhiteRight, false)
			b.castleRights.Set(CastleDirectionWhiteLeft, false)
		} else {
			b.castleRights.Set(CastleDirectionBlackRight, false)
			b.castleRights.Set(CastleDirectionBlackLeft, false)
		}
	}
	if fromPiece == PieceRook {
		if maskCell[fromPos]&maskCol[position.FileH] != 0 {
			if ourTurn == SideWhite {
				b.castleRights.Set(CastleDirectionWhiteRight, false)
			} else {
				b.castleRights.Set(CastleDirectionBlackRight, false)
			}
		}
		if maskCell[fromPos]&maskCol[position.FileA] != 0 {
			if ourTurn == SideWhite {
				b.castleRights.Set(CastleDirectionWhiteLeft, false)
			} else {
				b.castleRights.Set(CastleDirectionBlackLeft, false)
			}
		}
	}
	// remove castling rights when Rook is captured
	if capturedPiece == PieceRook {
		if theirTurn == SideWhite {
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
	prevHalfMoveClock := b.halfMoveClock
	if fromPiece == PiecePawn || isCapture {
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

	// update turn
	b.turn = theirTurn
	b.hash ^= zobristConstantSideWhite

	// reset state cache
	prevState := b.state
	b.state = StateUnknown

	return func() {
		if isCastle != CastleDirectionUnknown {
			// unperform castling
			hopsKing := posCastling[isCastle][PieceKing]
			hopsRook := posCastling[isCastle][PieceRook]

			b.flip(ourTurn, PieceKing, hopsKing[1])
			b.flip(ourTurn, PieceKing, hopsKing[0])
			b.cells[hopsKing[0]] = b.cells[hopsKing[1]]
			b.cells[hopsKing[1]] = 0
			b.positionValueMG[ourTurn] -= scorePositionMG[PieceKing][scorePositionMap[ourTurn][hopsKing[1]]]
			b.positionValueMG[ourTurn] += scorePositionMG[PieceKing][scorePositionMap[ourTurn][hopsKing[0]]]
			b.positionValueEG[ourTurn] -= scorePositionEG[PieceKing][scorePositionMap[ourTurn][hopsKing[1]]]
			b.positionValueEG[ourTurn] += scorePositionEG[PieceKing][scorePositionMap[ourTurn][hopsKing[0]]]

			b.flip(ourTurn, PieceRook, hopsRook[1])
			b.flip(ourTurn, PieceRook, hopsRook[0])
			b.cells[hopsRook[0]] = b.cells[hopsRook[1]]
			b.cells[hopsRook[1]] = 0
			b.positionValueMG[ourTurn] -= scorePositionMG[PieceRook][scorePositionMap[ourTurn][hopsRook[1]]]
			b.positionValueMG[ourTurn] += scorePositionMG[PieceRook][scorePositionMap[ourTurn][hopsRook[0]]]
			b.positionValueEG[ourTurn] -= scorePositionEG[PieceRook][scorePositionMap[ourTurn][hopsRook[1]]]
			b.positionValueEG[ourTurn] += scorePositionEG[PieceRook][scorePositionMap[ourTurn][hopsRook[0]]]
		} else {
			// remove moving piece at toPos
			b.flip(ourTurn, toPiece, toPos)
			b.cells[toPos] = 0
			b.materialValue[ourTurn] -= scoreMaterial[toPiece]
			b.positionValueMG[ourTurn] -= scorePositionMG[toPiece][scorePositionMap[ourTurn][toPos]]
			b.positionValueEG[ourTurn] -= scorePositionEG[toPiece][scorePositionMap[ourTurn][toPos]]

			// place captured piece at capturedPos
			if isCapture {
				b.flip(theirTurn, capturedPiece, capturedPos)
				b.setSideAndPieces(capturedPos, theirTurn, capturedPiece)
				b.materialValue[theirTurn] += scoreMaterial[capturedPiece]
				b.positionValueMG[theirTurn] += scorePositionMG[capturedPiece][scorePositionMap[theirTurn][capturedPos]]
				b.positionValueEG[theirTurn] += scorePositionEG[capturedPiece][scorePositionMap[theirTurn][capturedPos]]
				b.phase += phaseConstant[capturedPiece]
			}

			// place moving piece at fromPos
			b.flip(ourTurn, fromPiece, fromPos)
			b.setSideAndPieces(fromPos, ourTurn, fromPiece)
			b.materialValue[ourTurn] += scoreMaterial[fromPiece]
			b.positionValueMG[ourTurn] += scorePositionMG[fromPiece][scorePositionMap[ourTurn][fromPos]]
			b.positionValueEG[ourTurn] += scorePositionEG[fromPiece][scorePositionMap[ourTurn][fromPos]]
		}

		// revert enPassant
		b.enPassant = prevEnPassant

		// revert castleRights
		b.castleRights = prevCastleRights

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

		// revert cache
		b.state = prevState

		// revert hash
		b.hash = prevHash
	}, !b.IsKingChecked(ourTurn)
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
	if b.IsKingChecked(b.turn) {
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

func (b *Board) GetPositionValue() (int32, int32, int32, int32) {
	return b.positionValueMG[SideWhite], b.positionValueMG[SideBlack],
		b.positionValueEG[SideWhite], b.positionValueEG[SideBlack]
}

func (b *Board) Phase() uint8 {
	return b.phase
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
		occupied:        b.occupied,
		sides:           b.sides,
		pieces:          b.pieces,
		cells:           b.cells,
		materialValue:   b.materialValue,
		positionValueMG: b.positionValueMG,
		positionValueEG: b.positionValueEG,
		phase:           b.phase,
		enPassant:       b.enPassant,
		castleRights:    b.castleRights,
		halfMoveClock:   b.halfMoveClock,
		fullMoveClock:   b.fullMoveClock,
		ply:             b.ply,
		state:           b.state,
		turn:            b.turn,
		hash:            b.hash,
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
