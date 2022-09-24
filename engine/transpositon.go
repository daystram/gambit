package engine

import (
	"github.com/daystram/gambit/board"
)

type EntryType uint8

const (
	DefaultHashTableSize = 1 << 24 // number of entries

	EntryTypeUnknown EntryType = iota
	EntryTypeExact
	EntryTypeLowerBound
	EntryTypeUpperBound
)

type TranspositionTable struct {
	table    []*entry
	size     uint64
	maskHash uint64

	// stats
	hits   int
	misses int
	writes int
}

type entry struct {
	typ   EntryType
	mv    *board.Move
	score int32
	depth uint8
	hash  uint64
	age   uint8
}

func NewTranspositionTable(size uint64) *TranspositionTable {
	return &TranspositionTable{
		table:    make([]*entry, size),
		size:     size,
		maskHash: size - 1,
	}
}

func (t *TranspositionTable) Set(typ EntryType, b *board.Board, mv *board.Move, score int32, depth uint8) {
	hash := b.Hash()
	index := hash & t.maskHash
	e := t.table[index]
	age := b.HalfMoveClock()
	if e == nil || e.age < age || e.depth > depth {
		t.writes++
		t.table[index] = &entry{
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

func (t *TranspositionTable) Get(b *board.Board) (EntryType, *board.Move, int32, uint8, bool) {
	hash := b.Hash()
	index := hash & t.maskHash
	e := t.table[index]
	if e == nil || e.hash != hash || e.age < b.HalfMoveClock() {
		t.misses++
		return EntryTypeUnknown, nil, 0, 0, false
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
