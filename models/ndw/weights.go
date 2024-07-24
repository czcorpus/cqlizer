package ndw

const (
	paramsVectorSize = 32
)

type UnlimiteFloats []float64

func (f UnlimiteFloats) Get(i int) float64 {
	if i >= len(f) {
		return f[len(f)-1]
	}
	return f[i]
}

type Params struct {
	WithinContainingPart float64
	GlobCond             float64
	Structure            float64
	AttValList           float64
	NumberedPosition     float64
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
		p.WithinContainingPart,
		p.GlobCond,
		p.Structure,
		p.AttValList,
		p.NumberedPosition,
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
	p.WithinContainingPart = v[0]
	p.GlobCond = v[1]
	p.Structure = v[2]
	p.AttValList = v[3]
	p.NumberedPosition = v[4]
	p.RegExp = v[5]
	p.MuPart = v[6]
	p.Repetition = v[7]
	p.AtomQuery = v[8]
	p.RepOpt = v[9]
	p.OpenStructTag = v[10]
	p.CloseStructTag = v[11]
	p.AlignedPart = v[12]
	p.AttValAnd = v[13]
	p.AttVal = v[14]
	p.WithinNumber = v[15]
	p.RegExpRaw = v[16]
	p.RawString = v[17]
	p.SimpleString = v[18]
	p.RgGrouped = v[19]
	p.RgSimple = v[20]
	p.RgPosixClass = v[21]
	p.RgLook = v[22]
	p.RgAlt = v[23]
	p.RgRange = v[24]
	p.RgRangeSpec = v[25]
	p.AnyLetter = v[26]
	p.RgOp = v[27]
	p.RgAltVal = v[28]
	p.RgAny = v[29]
	p.RgQM = v[30]
	p.RgRepeat = v[31]
}
