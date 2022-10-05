package board

type PseudoRand struct {
	s uint64
}

func NewPseudoRand() *PseudoRand {
	return &PseudoRand{}
}

func (r *PseudoRand) Seed(seed uint64) {
	r.s = seed
}

func (r *PseudoRand) SparseUint64() uint64 {
	//nolint:staticcheck // SA4000 intentional
	return r.Uint64() & r.Uint64() & r.Uint64()
}

func (r *PseudoRand) Uint64() uint64 {
	r.s ^= r.s >> 12
	r.s ^= r.s << 25
	r.s ^= r.s >> 27
	return r.s * 2685821657736338717
}
