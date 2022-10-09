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
	table    []entry
	count    uint64
	disabled bool
}

type entry struct {
	typ   EntryType
	mv    board.Move
	score int32
	depth uint8
	hash  uint64
	age   uint16
}

func NewTranspositionTable(sizeMB uint32) *TranspositionTable {
	fmt.Print("Initializing transposition table... ")
	entrySize := uint32(unsafe.Sizeof(entry{}))
	count := sizeMB * 1e6 / entrySize
	tt := TranspositionTable{
		table:    make([]entry, count+1),
		count:    uint64(count),
		disabled: sizeMB == 0,
	}
	fmt.Printf("Done (%.3fMB)\n", float64(count*entrySize)/1e6)
	return &tt
}

func (t *TranspositionTable) Set(b *board.Board, age uint16, typ EntryType, mv board.Move, score int32, depth uint8) {
	hash := b.Hash()
	index := hash % t.count
	e := t.table[index]
	if !t.disabled && (e.typ == EntryTypeUnknown || e.age != age || e.depth <= depth) {
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

func (t *TranspositionTable) Get(b *board.Board, age uint16) (EntryType, board.Move, int32, uint8, bool) {
	hash := b.Hash()
	index := hash % t.count
	e := t.table[index]
	if t.disabled || e.typ == EntryTypeUnknown || e.age != age || e.hash != hash {
		return EntryTypeUnknown, board.Move{}, 0, 0, false
	}
	return e.typ, e.mv, e.score, e.depth, true
}
