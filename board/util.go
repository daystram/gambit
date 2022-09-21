package board

import (
	"fmt"
	"log"
	"math/bits"
	"strings"
	"time"

	"github.com/daystram/gambit/position"
)

func reverse(bm bitmap) bitmap {
	return bitmap(bits.Reverse64(uint64(bm)))
}

func AllBitsSet(bm bitmap) bool {
	return ((bm+1)&bm == 0) && (bm != 0)
}

func ShiftNW(bm bitmap) bitmap {
	return bm << 7
}

func ShiftN(bm bitmap) bitmap {
	return bm << 8
}

func ShiftNE(bm bitmap) bitmap {
	return bm << 9
}

func ShiftE(bm bitmap) bitmap {
	return bm << 1
}

func ShiftSE(bm bitmap) bitmap {
	return bm >> 7
}

func ShiftS(bm bitmap) bitmap {
	return bm >> 8
}

func ShiftSW(bm bitmap) bitmap {
	return bm >> 9
}

func ShiftW(bm bitmap) bitmap {
	return bm >> 1
}

func Union(bms ...bitmap) bitmap {
	var u bitmap
	for _, bm := range bms {
		u |= bm
	}
	return u
}

func Intersect(bms ...bitmap) bitmap {
	var u bitmap
	for _, bm := range bms {
		u &= bm
	}
	return u
}
func HitDiagonals(pos position.Pos, cell, occupied bitmap) bitmap {
	return ScanHit(cell, occupied, maskDia[pos]) | ScanHit(cell, occupied, maskADia[pos])
}

func HitLaterals(pos position.Pos, cell, occupied bitmap) bitmap {
	return ScanHit(cell, occupied, maskCol[pos.X()]) | ScanHit(cell, occupied, maskRow[pos.Y()])
}

// ScanHit uses o^(o-2*r) trick.
func ScanHit(cell, occupied, mask bitmap) bitmap {
	blocker := occupied & mask
	return ((blocker - 2*cell) ^ reverse(reverse(blocker)-2*reverse(cell))) & mask
}

func Set(b bitmap, pos position.Pos, value bool) bitmap {
	if value {
		return b | maskCell[pos]
	}
	return b &^ maskCell[pos]
}

func (b bitmap) Dump(sym ...rune) string {
	builder := strings.Builder{}
	for y := position.Pos(Height); y > 0; y-- {
		_, _ = builder.WriteString(fmt.Sprintf(" %d |", y))
		for x := position.Pos(0); x < Width; x++ {
			if b&maskCell[(y-1)*Height+x] != 0 {
				s := "#"
				if len(sym) == 1 {
					s = string(sym[0])
				}
				_, _ = builder.WriteString(fmt.Sprintf(" %s ", s))
			} else {
				_, _ = builder.WriteString(" . ")
			}
		}
		_, _ = builder.WriteString("\n")
	}
	_, _ = builder.WriteString("    ------------------------\n    ")
	for x := position.Pos(0); x < Width; x++ {
		_, _ = builder.WriteString(fmt.Sprintf(" %s ", x.NotationComponentX()))
	}
	return builder.String()
}

var (
	maskCell = [TotalCells]bitmap{}
	maskCol  = [Width]bitmap{
		0x_01_01_01_01_01_01_01_01,
		0x_02_02_02_02_02_02_02_02,
		0x_04_04_04_04_04_04_04_04,
		0x_08_08_08_08_08_08_08_08,
		0x_10_10_10_10_10_10_10_10,
		0x_20_20_20_20_20_20_20_20,
		0x_40_40_40_40_40_40_40_40,
		0x_80_80_80_80_80_80_80_80,
	}
	maskRow = [Height]bitmap{
		0x_00_00_00_00_00_00_00_FF,
		0x_00_00_00_00_00_00_FF_00,
		0x_00_00_00_00_00_FF_00_00,
		0x_00_00_00_00_FF_00_00_00,
		0x_00_00_00_FF_00_00_00_00,
		0x_00_00_FF_00_00_00_00_00,
		0x_00_FF_00_00_00_00_00_00,
		0x_FF_00_00_00_00_00_00_00,
	}
	maskDia    = [TotalCells]bitmap{}
	maskADia   = [TotalCells]bitmap{}
	maskKnight = [TotalCells]bitmap{}
	maskKing   = [TotalCells]bitmap{}

	maskCastling = [5]bitmap{}
	posCastling  = [5]map[Piece][2]position.Pos{
		CastleDirectionWhiteRight: {
			PieceKing: [2]position.Pos{4, 6},
			PieceRook: [2]position.Pos{7, 5},
		},
		CastleDirectionWhiteLeft: {
			PieceKing: [2]position.Pos{4, 2},
			PieceRook: [2]position.Pos{0, 3}},
		CastleDirectionBlackRight: {
			PieceKing: [2]position.Pos{4 + 7*Width, 6 + 7*Width},
			PieceRook: [2]position.Pos{7 + 7*Width, 5 + 7*Width},
		},
		CastleDirectionBlackLeft: {
			PieceKing: [2]position.Pos{4 + 7*Width, 2 + 7*Width},
			PieceRook: [2]position.Pos{0 + 7*Width, 3 + 7*Width}},
	}
)

func min(a, b position.Pos) position.Pos {
	if a < b {
		return a
	}
	return b
}

func init() {
	start := time.Now()
	for i := position.Pos(0); i < TotalCells; i++ {
		maskCell[i] = 1 << i
	}

	for i := position.Pos(0); i < TotalCells; i++ {
		mask := bitmap(0)
		x, y := i%Width, i/Width
		x, y = x-min(x, y), y-min(x, y)
		for x < Width && y < Height {
			mask |= bitmap(1 << (y*Width + x))
			x++
			y++
		}
		maskDia[i] = mask
	}

	for i := position.Pos(0); i < TotalCells; i++ {
		mask := bitmap(0)
		x, y := i%Width, i/Width
		x, y = x-min(x, Height-y-1), y+min(x, Height-y-1)
		for x < Width && y >= 0 {
			mask |= bitmap(1 << (y*Width + x))
			x++
			y--
		}
		maskADia[i] = mask
	}

	for i := position.Pos(0); i < TotalCells; i++ {
		cell := maskCell[i]
		mask := bitmap(0)
		mask |= ShiftN(ShiftN(ShiftE(cell &^ maskRow[7] &^ maskRow[6] &^ maskCol[7])))
		mask |= ShiftN(ShiftN(ShiftW(cell &^ maskRow[7] &^ maskRow[6] &^ maskCol[0])))
		mask |= ShiftS(ShiftS(ShiftE(cell &^ maskRow[0] &^ maskRow[1] &^ maskCol[7])))
		mask |= ShiftS(ShiftS(ShiftW(cell &^ maskRow[0] &^ maskRow[1] &^ maskCol[0])))
		mask |= ShiftE(ShiftE(ShiftN(cell &^ maskCol[7] &^ maskCol[6] &^ maskRow[7])))
		mask |= ShiftE(ShiftE(ShiftS(cell &^ maskCol[7] &^ maskCol[6] &^ maskRow[0])))
		mask |= ShiftW(ShiftW(ShiftN(cell &^ maskCol[0] &^ maskCol[1] &^ maskRow[7])))
		mask |= ShiftW(ShiftW(ShiftS(cell &^ maskCol[0] &^ maskCol[1] &^ maskRow[0])))
		maskKnight[i] = mask
	}

	for i := position.Pos(0); i < TotalCells; i++ {
		cell := maskCell[i]
		mask := bitmap(0)
		mask |= ShiftN(cell &^ maskRow[7])
		mask |= ShiftNE(cell &^ maskRow[7] &^ maskCol[7])
		mask |= ShiftE(cell &^ maskCol[7])
		mask |= ShiftSE(cell &^ maskRow[0] &^ maskCol[7])
		mask |= ShiftS(cell &^ maskRow[0])
		mask |= ShiftSW(cell &^ maskRow[0] &^ maskCol[0])
		mask |= ShiftW(cell &^ maskCol[0])
		mask |= ShiftNW(cell &^ maskRow[7] &^ maskCol[0])
		maskKing[i] = mask
	}

	maskCastling = [5]bitmap{
		CastleDirectionWhiteRight: maskRow[0] & (maskCol[5] | maskCol[6]),
		CastleDirectionWhiteLeft:  maskRow[0] & (maskCol[2] | maskCol[3]),
		CastleDirectionBlackRight: maskRow[7] & (maskCol[5] | maskCol[6]),
		CastleDirectionBlackLeft:  maskRow[7] & (maskCol[2] | maskCol[3]),
	}

	log.Printf("init lookup: %s elapsed\n", time.Since(start))
}
