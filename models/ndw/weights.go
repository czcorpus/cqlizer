package ndw

const (
	paramsVectorSize = 38
)

type UnlimiteFloats []float64

func (f UnlimiteFloats) Get(i int) float64 {
	if i >= len(f) {
		return f[len(f)-1]
	}
	return f[i]
}

type Params struct {
	Sequence             float64
	Seq                  float64
	GlobPart             float64
	WithinOrContaining   float64
	WithinContainingPart float64
	GlobCond             float64
	Structure            float64
	AttValList           float64
	NumberedPosition     float64
	OnePosition          float64
	Position             float64
	RegExp               float64
	MuPart               float64
	Repetition           float64
	AtomQuery            float64
	RepOpt               float64
	OpenStructTag        float64
	CloseStructTag       float64
	AlignedPart          float64
	AttValAnd            float64
	AttVal               float64
	AttValVariantNeg     float64
	WithinNumber         float64
	RegExpRaw            float64
	RawString            float64
	SimpleString         float64
	RgGrouped            float64
	RgSimple             float64
	RgPosixClass         float64
	RgLook               float64
	RgAlt                float64
	RgRange              float64
	RgRangeSpec          float64
	AnyLetter            float64
	RgOp                 float64
	RgAltVal             float64
	RgAny                float64
	RgQM                 float64
	RgRepeat             float64
}

func (p Params) ToVec() []float64 {
	return []float64{
		p.Sequence,
		p.Seq,
		p.GlobPart,
		p.WithinOrContaining,
		p.WithinContainingPart,
		p.GlobCond,
		p.Structure,
		p.AttValList,
		p.NumberedPosition,
		p.OnePosition,
		p.Position,
		p.RegExp,
		p.MuPart,
		p.Repetition,
		p.AtomQuery,
		p.RepOpt,
		p.OpenStructTag,
		p.CloseStructTag,
		p.AlignedPart,
		p.AttValAnd,
		p.AttVal,
		p.WithinNumber,
		p.RegExpRaw,
		p.RawString,
		p.SimpleString,
		p.RgGrouped,
		p.RgSimple,
		p.RgPosixClass,
		p.RgLook,
		p.RgAlt,
		p.RgRange,
		p.RgRangeSpec,
		p.AnyLetter,
		p.RgOp,
		p.RgAltVal,
		p.RgAny,
		p.RgQM,
		p.RgRepeat,
	}
}

func (p *Params) FromVec(v []float64) {
	p.Sequence = v[0]
	p.Seq = v[1]
	p.GlobPart = v[2]
	p.WithinOrContaining = v[3]
	p.WithinContainingPart = v[4]
	p.GlobCond = v[5]
	p.Structure = v[6]
	p.AttValList = v[7]
	p.NumberedPosition = v[8]
	p.OnePosition = v[9]
	p.Position = v[10]
	p.RegExp = v[11]
	p.MuPart = v[12]
	p.Repetition = v[13]
	p.AtomQuery = v[14]
	p.RepOpt = v[15]
	p.OpenStructTag = v[16]
	p.CloseStructTag = v[17]
	p.AlignedPart = v[18]
	p.AttValAnd = v[19]
	p.AttVal = v[20]
	p.WithinNumber = v[21]
	p.RegExpRaw = v[22]
	p.RawString = v[23]
	p.SimpleString = v[24]
	p.RgGrouped = v[25]
	p.RgSimple = v[26]
	p.RgPosixClass = v[27]
	p.RgLook = v[28]
	p.RgAlt = v[29]
	p.RgRange = v[30]
	p.RgRangeSpec = v[31]
	p.AnyLetter = v[32]
	p.RgOp = v[33]
	p.RgAltVal = v[34]
	p.RgAny = v[35]
	p.RgQM = v[36]
	p.RgRepeat = v[37]
}
