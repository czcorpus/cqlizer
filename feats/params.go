package feats

var (
	ParamsSize = 10
	//CurrWinner = []float64{0, 0, 0.4932345219573573, 0.5088117936652335, 0, 0, 0, 0.7864255665946478, 0.7832822436954627, 0}
	//CurrWinner = []float64{0, 0, 0.3336088436408728, 0, 0.7877769627559942, 0.1277525739217169, 0, 0.6076132879683428, 0, 0}
	//CurrWinner = []float64{0, 0, 0.5509906009477218, 0, 0.3983578228033612, 0, 0, 0.6987855542703999, 0, 0.7}
	CurrWinner = []float64{0, 0, 0.5215146474304725, 0.0666949865281623, 0, 0, 0, 0.8407653639126037, 0, 0.1}
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
	OpenStructTagProb       float64 // 6
	AnyPositionScore        float64 // 7
	NegationPenalty         float64 // 8
	SmallValSetPenalty      float64
}

func (p Params) ToVec() []float64 {
	return []float64{
		p.ChunkLenScore,
		p.WildcardedChunkLenScore,
		p.PureWildcardScore,
		p.GlobCondPenalty,
		p.PositionInfRepScore,
		p.PositionFewRepScore,
		p.OpenStructTagProb,
		p.AnyPositionScore,
		p.NegationPenalty,
		p.SmallValSetPenalty,
	}
}

func (p *Params) FromVec(v []float64) {
	p.ChunkLenScore = v[0]
	p.WildcardedChunkLenScore = v[1]
	p.PureWildcardScore = v[2]
	p.GlobCondPenalty = v[3]
	p.PositionInfRepScore = v[4]
	p.PositionFewRepScore = v[5]
	p.OpenStructTagProb = v[6]
	p.AnyPositionScore = v[7]
	p.NegationPenalty = v[8]
	p.SmallValSetPenalty = v[9]

}

func NewDefaultParams() Params {
	return Params{
		ChunkLenScore:           0.4,
		WildcardedChunkLenScore: 0.8,
		PureWildcardScore:       1.0,
		GlobCondPenalty:         0.1,
		PositionInfRepScore:     2,
		PositionFewRepScore:     1.5,
		OpenStructTagProb:       0.2,
		AnyPositionScore:        1.0,
		NegationPenalty:         5,
		SmallValSetPenalty:      5,
	}
}
