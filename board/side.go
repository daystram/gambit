package board

type Side uint8

const (
	SideUnknown Side = iota
	SideWhite
	SideBlack
)

func (s Side) String() string {
	switch s {
	case SideWhite:
		return "White"
	case SideBlack:
		return "Black"
	default:
		return ""
	}
}

func (s Side) Opposite() Side {
	switch s {
	case SideWhite:
		return SideBlack
	case SideBlack:
		return SideWhite
	default:
		return SideUnknown
	}
}
