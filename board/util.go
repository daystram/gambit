package board

import (
	"fmt"
	"math/bits"
	"strings"

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
func HitDiagonals(pos position.Pos, occupied bitmap) bitmap {
	return ScanHit(maskCell[pos], occupied, maskDia[pos]) | ScanHit(maskCell[pos], occupied, maskADia[pos])
}

func HitLaterals(pos position.Pos, occupied bitmap) bitmap {
	return ScanHit(maskCell[pos], occupied, maskCol[pos.X()]) | ScanHit(maskCell[pos], occupied, maskRow[pos.Y()])
}

// ScanHit uses o^(o-2*r) trick.
func ScanHit(cell, occupied, mask bitmap) bitmap {
	blocker := occupied & mask
	return ((blocker - 2*cell) ^ reverse(reverse(blocker)-2*reverse(cell))) & mask
}

type Magic struct {
	Attacks *[4096]bitmap
	Magic   bitmap
	Mask    bitmap
	Shift   uint8
}

func (m *Magic) GetIndex(occupancy bitmap) uint16 {
	return uint16(((occupancy & m.Mask) * m.Magic) >> m.Shift)
}

func (bm *bitmap) Set(pos position.Pos) {
	*bm |= maskCell[pos]
}

func (bm *bitmap) Unset(pos position.Pos) {
	*bm &^= maskCell[pos]
}

func (bm bitmap) LS1B() position.Pos {
	return position.Pos(bits.TrailingZeros64(uint64(bm)))
}

func (bm bitmap) BitCount() uint8 {
	return uint8(bits.OnesCount64(uint64(bm)))
}

func (bm bitmap) Dump(sym ...rune) string {
	builder := strings.Builder{}
	for y := position.Pos(Height); y > 0; y-- {
		_, _ = builder.WriteString(fmt.Sprintf(" %d |", y))
		for x := position.Pos(0); x < Width; x++ {
			if bm&maskCell[(y-1)*Height+x] != 0 {
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

func min(a, b position.Pos) position.Pos {
	if a < b {
		return a
	}
	return b
}
