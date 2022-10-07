package engine

import (
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
	count uint64

	// stats
	hits   int
	misses int
	writes int
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
	count := sizeMB * 1e6 / uint64(unsafe.Sizeof(entry{}))
	return &TranspositionTable{
		table: make([]entry, count),
		count: count,
	}
}

func (t *TranspositionTable) Set(typ EntryType, b *board.Board, mv board.Move, score int32, depth, age uint8) {
	hash := b.Hash()
	index := hash % t.count
	e := t.table[index]
	if e.typ == EntryTypeUnknown || e.age != age || e.depth <= depth {
		t.writes++
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
	index := hash % t.count
	e := t.table[index]
	if e.typ == EntryTypeUnknown || e.hash != hash || e.age != age {
		t.misses++
		return EntryTypeUnknown, board.Move{}, 0, 0, false
	}
	t.hits++
	return e.typ, e.mv, e.score, e.depth, true
}

func (t *TranspositionTable) ResetStats() {
	t.hits = 0
	t.misses = 0
	t.writes = 0
}

func (t *TranspositionTable) Stats() (int, int, int) {
	return t.hits, t.misses, t.writes
}
