package engine

import (
	"fmt"
	"unsafe"

	"github.com/daystram/gambit/board"
)

type EntryType uint8

const (
	DefaultHashTableSizeMB = 64 // 64 MB
)

const (
	EntryTypeUnknown EntryType = iota
	EntryTypeExact
	EntryTypeLowerBound
	EntryTypeUpperBound
)

type TranspositionTable struct {
	table []entry
	mask  uint64
}

type entry struct {
	typ   EntryType
	mv    board.Move
	score int32
	depth uint8
	hash  uint64
	age   uint8
}

func NewTranspositionTable(sizeMB uint64) *TranspositionTable {
	fmt.Print("Initializing transposition table... ")
	entrySize := uint64(unsafe.Sizeof(entry{}))
	count := sizeMB * 1e6 / entrySize

	allocCount := uint64(1)
	for allocCount < count {
		allocCount <<= 1
	}

	tt := TranspositionTable{
		table: make([]entry, allocCount),
		mask:  allocCount - 1,
	}

	fmt.Printf("Done (%.3fMB)\n", float64(allocCount*entrySize)/1e6)
	return &tt
}

func (t *TranspositionTable) Set(b *board.Board, age uint8, typ EntryType, mv board.Move, score int32, depth uint8) {
	hash := b.Hash()
	index := hash & t.mask
	e := t.table[index]
	if e.typ == EntryTypeUnknown || e.age != age || e.depth <= depth {
		t.table[index] = entry{
			typ:   typ,
			mv:    mv,
			score: score,
			depth: depth,
			hash:  hash,
			age:   age,
		}
		return
	}
}

func (t *TranspositionTable) Get(b *board.Board, age uint8) (EntryType, board.Move, int32, uint8, bool) {
	hash := b.Hash()
	index := hash & t.mask
	e := t.table[index]
	if e.typ == EntryTypeUnknown || e.age != age || e.hash != hash {
		return EntryTypeUnknown, board.Move{}, 0, 0, false
	}
	return e.typ, e.mv, e.score, e.depth, true
}
