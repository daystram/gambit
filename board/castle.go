package board

type CastleDirection uint8

const (
	CastleDirectionUnknown CastleDirection = iota
	CastleDirectionWhiteRight
	CastleDirectionWhiteLeft
	CastleDirectionBlackRight
	CastleDirectionBlackLeft
)

func (d CastleDirection) String() string {
	switch d {
	case CastleDirectionWhiteRight:
		return "White 0-0"
	case CastleDirectionWhiteLeft:
		return "White 0-0-0"
	case CastleDirectionBlackRight:
		return "Black 0-0"
	case CastleDirectionBlackLeft:
		return "Black 0-0-0"
	default:
		return ""
	}
}

func (d CastleDirection) IsWhite() bool {
	return d == CastleDirectionWhiteRight || d == CastleDirectionWhiteLeft
}

func (d CastleDirection) IsRight() bool {
	return d == CastleDirectionWhiteRight || d == CastleDirectionBlackRight
}

type CastleRights uint8

func (c *CastleRights) Set(d CastleDirection, allow bool) {
	if allow {
		*c |= maskCastleRights[d]
	} else {
		*c &^= maskCastleRights[d]
	}
}

func (c *CastleRights) IsAllowed(d CastleDirection) bool {
	return *c&maskCastleRights[d] != 0
}

func (c *CastleRights) IsSideAllowed(s Side) bool {
	if s == SideWhite {
		return *c&(maskCastleRights[CastleDirectionWhiteLeft]|maskCastleRights[CastleDirectionWhiteRight]) != 0
	}
	return *c&(maskCastleRights[CastleDirectionBlackLeft]|maskCastleRights[CastleDirectionBlackRight]) != 0
}
