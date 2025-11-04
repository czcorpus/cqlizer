package eval

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
)

const NumFeatures = 36

type CostProvider interface {
	Cost(model ModelParams) float64
}

// -----------------------------------

type ModelParams struct {
	WildcardPrefix0 float64
	Wildcards0      float64
	RangeOp0        float64

	// SmallCardAttr0 if 1, it means that we search by an attribute which
	// has only a few possible values and thus the resulting set will be large.
	// This typically applies for attributes/searches like [tag="..."], [pos="..."]
	// and we also consider the special `[]` query (= any word) as part of that.
	SmallCardAttr0 float64
	ConcreteChars0 float64
	AvgCharProb0   float64
	NumPosAlts0    float64

	WildcardPrefix1 float64
	Wildcards1      float64
	RangeOp1        float64
	SmallCardAttr1  float64
	ConcreteChars1  float64
	AvgCharProb1    float64
	NumPosAlts1     float64

	WildcardPrefix2 float64
	Wildcards2      float64
	RangeOp2        float64
	SmallCardAttr2  float64
	ConcreteChars2  float64
	AvgCharProb2    float64
	NumPosAlts2     float64

	WildcardPrefix3 float64
	Wildcards3      float64
	RangeOp3        float64
	SmallCardAttr3  float64
	ConcreteChars3  float64
	AvgCharProb3    float64
	NumPosAlts3     float64

	GlobCond    float64
	Meet        float64
	Union       float64
	Within      float64
	Containing  float64
	CorpusSize  float64 // Impact of corpus size on query time
	AlignedPart float64
	Bias        float64
}

func (p ModelParams) ToSlice() []float64 {
	return []float64{
		p.WildcardPrefix0,
		p.Wildcards0,
		p.RangeOp0,
		p.SmallCardAttr0,
		p.ConcreteChars0,
		p.AvgCharProb0,
		p.NumPosAlts0,
		p.WildcardPrefix1,
		p.Wildcards1,
		p.RangeOp1,
		p.SmallCardAttr1,
		p.ConcreteChars1,
		p.AvgCharProb1,
		p.NumPosAlts1,
		p.WildcardPrefix2,
		p.Wildcards2,
		p.RangeOp2,
		p.SmallCardAttr2,
		p.ConcreteChars2,
		p.AvgCharProb2,
		p.NumPosAlts2,
		p.WildcardPrefix3,
		p.Wildcards3,
		p.RangeOp3,
		p.SmallCardAttr3,
		p.ConcreteChars3,
		p.AvgCharProb3,
		p.NumPosAlts3,
		p.GlobCond,
		p.Meet,
		p.Union,
		p.Within,
		p.Containing,
		p.CorpusSize,
		p.AlignedPart,
		p.Bias,
	}
}

func SliceToModelParams(slice []float64) ModelParams {
	if len(slice) != NumFeatures {
		panic(fmt.Sprintf("slice must have %d elements", NumFeatures))
	}
	return ModelParams{
		WildcardPrefix0: slice[0],
		Wildcards0:      slice[1],
		RangeOp0:        slice[2],
		SmallCardAttr0:  slice[3],
		ConcreteChars0:  slice[4],
		AvgCharProb0:    slice[5],
		NumPosAlts0:     slice[6],
		WildcardPrefix1: slice[7],
		Wildcards1:      slice[8],
		RangeOp1:        slice[9],
		SmallCardAttr1:  slice[10],
		ConcreteChars1:  slice[11],
		AvgCharProb1:    slice[12],
		NumPosAlts1:     slice[13],
		WildcardPrefix2: slice[14],
		Wildcards2:      slice[15],
		RangeOp2:        slice[16],
		SmallCardAttr2:  slice[17],
		ConcreteChars2:  slice[18],
		AvgCharProb2:    slice[19],
		NumPosAlts2:     slice[20],
		WildcardPrefix3: slice[21],
		Wildcards3:      slice[22],
		RangeOp3:        slice[23],
		SmallCardAttr3:  slice[24],
		ConcreteChars3:  slice[25],
		AvgCharProb3:    slice[26],
		NumPosAlts3:     slice[27],
		GlobCond:        slice[28],
		Meet:            slice[29],
		Union:           slice[30],
		Within:          slice[31],
		Containing:      slice[32],
		CorpusSize:      slice[33],
		AlignedPart:     slice[34],
		Bias:            slice[35],
	}
}

// SaveToFile saves the model parameters to a JSON file
func (p ModelParams) SaveToFile(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty-print with 2-space indentation
	if err := encoder.Encode(p); err != nil {
		return fmt.Errorf("failed to encode model: %w", err)
	}

	return nil
}

// LoadModelFromFile loads model parameters from a JSON file
func LoadModelFromFile(filePath string) (ModelParams, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return ModelParams{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var params ModelParams
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&params); err != nil {
		return ModelParams{}, fmt.Errorf("failed to decode model: %w", err)
	}
	return params, nil
}

// -----------------------------------

type Regexp struct {
	StartsWithWildCard int     `msgpack:"startsWithWildCard"`
	NumConcreteChars   float64 `msgpack:"numConcreteChars"`
	AvgCharProb        float64 `msgpack:"avgCharProb"`
	WildcardScore      float64 `msgpack:"wildcardScore"`
	HasRange           int     `msgpack:"hasRange"`
}

// -----------------------------------

type Position struct {
	Index            int    `msgpack:"index"`
	Regexp           Regexp `msgpack:"regexp"`
	HasSmallCardAttr int    `msgpack:"hasSmallCardAttr"` // 1 if searching by attribute with small cardinality (tag, pos, etc.) or empty query []
	NumAlternatives  int    `msgpack:"numAlternatives"`  // at least 1, solves situations like [lemma="foo" | word="fooish"]
}

// -----------------------------------

type QueryEvaluation struct {
	ProcTime float64 `msgpack:"procTime"`

	Positions          []Position `msgpack:"positions"`
	NumGlobConditions  int        `msgpack:"numGlobConditions"`
	ContainsMeet       int        `msgpack:"containsMeet"`
	ContainsUnion      int        `msgpack:"containsUnion"`
	ContainsWithin     int        `msgpack:"containsWithin"`
	ContainsContaining int        `msgpack:"containsContaining"`
	CorpusSize         float64    `msgpack:"corpusSize"` // Size of the corpus being searched (e.g., number of tokens)
	AlignedPart        int        `msgpack:"alignedPart"`
}

func (eval QueryEvaluation) Cost(model ModelParams) float64 {
	var total float64

	// Compute position-specific costs
	for i := 0; i < len(eval.Positions) && i < MaxPositions; i++ {
		pos := eval.Positions[i]
		// Get position-specific parameters
		var wildcardPrefix, wildcards, rangeOp, smallCardAttr, concreteChars, avgCharProb, numPosAlts float64
		switch i {
		case 0:
			wildcardPrefix = model.WildcardPrefix0
			wildcards = model.Wildcards0
			rangeOp = model.RangeOp0
			smallCardAttr = model.SmallCardAttr0
			concreteChars = model.ConcreteChars0
			avgCharProb = model.AvgCharProb0
			numPosAlts = model.NumPosAlts0
		case 1:
			wildcardPrefix = model.WildcardPrefix1
			wildcards = model.Wildcards1
			rangeOp = model.RangeOp1
			smallCardAttr = model.SmallCardAttr1
			concreteChars = model.ConcreteChars1
			avgCharProb = model.AvgCharProb1
			numPosAlts = model.NumPosAlts1
		case 2:
			wildcardPrefix = model.WildcardPrefix2
			wildcards = model.Wildcards2
			rangeOp = model.RangeOp2
			smallCardAttr = model.SmallCardAttr2
			concreteChars = model.ConcreteChars2
			avgCharProb = model.AvgCharProb2
			numPosAlts = model.NumPosAlts2
		case 3:
			wildcardPrefix = model.WildcardPrefix3
			wildcards = model.Wildcards3
			rangeOp = model.RangeOp3
			smallCardAttr = model.SmallCardAttr3
			concreteChars = model.ConcreteChars3
			avgCharProb = model.AvgCharProb3
			numPosAlts = model.NumPosAlts3
		}

		// Calculate position cost
		positionCost := (wildcardPrefix*float64(pos.Regexp.StartsWithWildCard) +
			wildcards*float64(pos.Regexp.WildcardScore) +
			rangeOp*float64(pos.Regexp.HasRange) +
			smallCardAttr*float64(pos.HasSmallCardAttr)) +
			concreteChars*float64(pos.Regexp.NumConcreteChars) +
			avgCharProb*float64(pos.Regexp.AvgCharProb) +
			numPosAlts*float64(pos.NumAlternatives)

		total += positionCost
	}

	// Add global costs
	total += model.GlobCond * float64(eval.NumGlobConditions)
	total += model.Meet * float64(eval.ContainsMeet)
	total += model.Union * float64(eval.ContainsUnion)
	total += model.Within * float64(eval.ContainsWithin)
	total += model.Containing * float64(eval.ContainsContaining)
	total += model.CorpusSize * math.Log(eval.CorpusSize)
	total += model.Bias

	return total
}
