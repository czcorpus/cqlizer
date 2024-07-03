package feats

var (
	ParamsSize = 12
	CurrWinner = []float64{0.043, 0.030, 0.672, 0.990, 0.229, 0.042, 0.857, 0.612, 0.694, 0.912, 0.033, 0.774}
)

type UnlimiteFloats []float64

func (f UnlimiteFloats) Get(i int) float64 {
	if i >= len(f) {
		return f[len(f)-1]
	}
	return f[i]
}

type Params struct {
	ChunkLenScore           float64 // 0
	WildcardedChunkLenScore float64 // 1
	PureWildcardScore       float64 // 2
	GlobCondPenalty         float64 // 3
	PositionInfRepScore     float64 // 4
	PositionFewRepScore     float64 // 5
	SmallOpenStructTagProb  float64 // 6
	BigOpenStructScore      float64 // 7
	AnyPositionScore        float64 // 8
	NegationPenalty         float64 // 9
	SmallValSetPenalty      float64
	MeetScore               float64
}

func (p Params) ToVec() []float64 {
	return []float64{
		p.ChunkLenScore,
		p.WildcardedChunkLenScore,
		p.PureWildcardScore,
		p.GlobCondPenalty,
		p.PositionInfRepScore,
		p.PositionFewRepScore,
		p.SmallOpenStructTagProb,
		p.BigOpenStructScore,
		p.AnyPositionScore,
		p.NegationPenalty,
		p.SmallValSetPenalty,
		p.MeetScore,
	}
}

func (p *Params) FromVec(v []float64) {
	p.ChunkLenScore = v[0]
	p.WildcardedChunkLenScore = v[1]
	p.PureWildcardScore = v[2]
	p.GlobCondPenalty = v[3]
	p.PositionInfRepScore = v[4]
	p.PositionFewRepScore = v[5]
	p.SmallOpenStructTagProb = v[6]
	p.BigOpenStructScore = v[7]
	p.AnyPositionScore = v[8]
	p.NegationPenalty = v[9]
	p.SmallValSetPenalty = v[10]
	p.MeetScore = v[11]

}

func NewDefaultParams() Params {
	return Params{
		ChunkLenScore:           0.4,
		WildcardedChunkLenScore: 0.8,
		PureWildcardScore:       1.0,
		GlobCondPenalty:         0.1,
		PositionInfRepScore:     2,
		PositionFewRepScore:     1.5,
		SmallOpenStructTagProb:  0.2,
		BigOpenStructScore:      0.9,
		AnyPositionScore:        1.0,
		NegationPenalty:         5,
		SmallValSetPenalty:      5,
		MeetScore:               0.1,
	}
}
