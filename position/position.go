package position

import (
	"errors"
)

const (
	// MaxComponentScalar is the maximum component scalar the position system supports.
	MaxComponentScalar Pos = 8
)

var (
	// ErrInvalidNotation represents an invalid notation error.
	ErrInvalidNotation = errors.New("invalid notation")
)

type Pos int8

func NewPosFromNotation(n string) (Pos, error) {
	x, y, err := notationToXY(n)
	if err != nil {
		return 0, err
	}
	return MaxComponentScalar*y + x, nil
}

func (p Pos) String() string {
	return p.Notation()
}

func (p Pos) Notation() string {
	if p < 0 || p >= MaxComponentScalar*MaxComponentScalar {
		return ""
	}
	return string(rune('a'+p.X())) + string(rune('1'+p.Y()))
}

func (p Pos) X() Pos {
	return p % MaxComponentScalar
}

func (p Pos) Y() Pos {
	return p / MaxComponentScalar
}

func notationToXY(n string) (Pos, Pos, error) {
	if len(n) != 2 {
		return 0, 0, ErrInvalidNotation
	}
	pX, err := notationToX(n[0])
	if err != nil {
		return 0, 0, err
	}
	pY, err := notationToY(n[1])
	if err != nil {
		return 0, 0, err
	}
	return pX, pY, nil
}

func notationToX(x byte) (Pos, error) {
	pX := Pos(x - 'a')
	if pX < 0 || MaxComponentScalar <= pX {
		return 0, ErrInvalidNotation
	}
	return pX, nil
}

func notationToY(y byte) (Pos, error) {
	pY := Pos(y-'0') - 1
	if pY < 0 || MaxComponentScalar <= pY {
		return 0, ErrInvalidNotation
	}
	return pY, nil
}

func (p Pos) NotationComponentX() string {
	if p < 0 || MaxComponentScalar < p {
		return ""
	}
	return string(rune('a' + p))
}

func (p Pos) NotationComponentY() string {
	if p < 0 || MaxComponentScalar < p {
		return ""
	}
	return string(rune('0' + p + 1))
}
