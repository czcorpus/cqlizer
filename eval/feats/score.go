// Copyright 2025 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2025 Department of Linguistics,
// Faculty of Arts, Charles University
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package feats

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
)

const NumFeatures = 50

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
	PosRepetition0 float64
	CharClasses0   float64
	HasNegation0   float64

	WildcardPrefix1 float64
	Wildcards1      float64
	RangeOp1        float64
	SmallCardAttr1  float64
	ConcreteChars1  float64
	AvgCharProb1    float64
	NumPosAlts1     float64
	PosRepetition1  float64
	CharClasses1    float64
	HasNegation1    float64

	WildcardPrefix2 float64
	Wildcards2      float64
	RangeOp2        float64
	SmallCardAttr2  float64
	ConcreteChars2  float64
	AvgCharProb2    float64
	NumPosAlts2     float64
	PosRepetition2  float64
	CharClasses2    float64
	HasNegation2    float64

	WildcardPrefix3 float64
	Wildcards3      float64
	RangeOp3        float64
	SmallCardAttr3  float64
	ConcreteChars3  float64
	AvgCharProb3    float64
	NumPosAlts3     float64
	PosRepetition3  float64
	CharClasses3    float64
	HasNegation3    float64

	GlobCond           float64
	Meet               float64
	Union              float64
	Within             float64
	AdhocSubcorpus     float64
	Containing         float64
	CorpusSize         float64 // Impact of corpus size on query time
	NamedSubcorpusSize float64
	AlignedPart        float64
	Bias               float64
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
		p.PosRepetition0,
		p.CharClasses0,
		p.HasNegation0,
		p.WildcardPrefix1,
		p.Wildcards1,
		p.RangeOp1,
		p.SmallCardAttr1,
		p.ConcreteChars1,
		p.AvgCharProb1,
		p.NumPosAlts1,
		p.PosRepetition1,
		p.CharClasses1,
		p.HasNegation1,
		p.WildcardPrefix2,
		p.Wildcards2,
		p.RangeOp2,
		p.SmallCardAttr2,
		p.ConcreteChars2,
		p.AvgCharProb2,
		p.NumPosAlts2,
		p.PosRepetition2,
		p.CharClasses2,
		p.HasNegation2,
		p.WildcardPrefix3,
		p.Wildcards3,
		p.RangeOp3,
		p.SmallCardAttr3,
		p.ConcreteChars3,
		p.AvgCharProb3,
		p.NumPosAlts3,
		p.PosRepetition3,
		p.CharClasses3,
		p.HasNegation3,
		p.GlobCond,
		p.Meet,
		p.Union,
		p.Within,
		p.AdhocSubcorpus,
		p.Containing,
		p.CorpusSize,
		p.NamedSubcorpusSize,
		p.AlignedPart,
		p.Bias,
	}
}

func SliceToModelParams(slice []float64) ModelParams {
	if len(slice) != NumFeatures {
		panic(fmt.Sprintf("slice must have %d elements", NumFeatures))
	}
	return ModelParams{
		WildcardPrefix0:    slice[0],
		Wildcards0:         slice[1],
		RangeOp0:           slice[2],
		SmallCardAttr0:     slice[3],
		ConcreteChars0:     slice[4],
		AvgCharProb0:       slice[5],
		NumPosAlts0:        slice[6],
		PosRepetition0:     slice[7],
		CharClasses0:       slice[8],
		HasNegation0:       slice[9],
		WildcardPrefix1:    slice[10],
		Wildcards1:         slice[11],
		RangeOp1:           slice[12],
		SmallCardAttr1:     slice[13],
		ConcreteChars1:     slice[14],
		AvgCharProb1:       slice[15],
		NumPosAlts1:        slice[16],
		PosRepetition1:     slice[17],
		CharClasses1:       slice[18],
		HasNegation1:       slice[19],
		WildcardPrefix2:    slice[20],
		Wildcards2:         slice[21],
		RangeOp2:           slice[22],
		SmallCardAttr2:     slice[23],
		ConcreteChars2:     slice[24],
		AvgCharProb2:       slice[25],
		NumPosAlts2:        slice[26],
		PosRepetition2:     slice[27],
		CharClasses2:       slice[28],
		HasNegation2:       slice[29],
		WildcardPrefix3:    slice[30],
		Wildcards3:         slice[31],
		RangeOp3:           slice[32],
		SmallCardAttr3:     slice[33],
		ConcreteChars3:     slice[34],
		AvgCharProb3:       slice[35],
		NumPosAlts3:        slice[36],
		PosRepetition3:     slice[37],
		CharClasses3:       slice[38],
		HasNegation3:       slice[39],
		GlobCond:           slice[40],
		Meet:               slice[41],
		Union:              slice[42],
		Within:             slice[43],
		AdhocSubcorpus:     slice[44],
		Containing:         slice[45],
		CorpusSize:         slice[46],
		NamedSubcorpusSize: slice[47],
		AlignedPart:        slice[48],
		Bias:               slice[49],
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
	CharClasses        float64 `msgpack:"charClasses"`
}

// -----------------------------------

type Position struct {
	Index            int     `msgpack:"index"`
	Regexp           Regexp  `msgpack:"regexp"`
	HasSmallCardAttr int     `msgpack:"hasSmallCardAttr"` // 1 if searching by attribute with small cardinality (tag, pos, etc.) or empty query []
	NumAlternatives  int     `msgpack:"numAlternatives"`  // at least 1, solves situations like [lemma="foo" | word="fooish"]
	PosRepetition    float64 `msgpack:"posRepetition"`    // stuff like [word="foo"]+
	HasNegation      int     `msgpack:"hasNegation"`
}

// -----------------------------------

type QueryEvaluation struct {
	ProcTime float64 `msgpack:"procTime"`

	OrigQuery          string     `msgpack:"q"`
	Positions          []Position `msgpack:"positions"`
	NumGlobConditions  int        `msgpack:"numGlobConditions"`
	ContainsMeet       int        `msgpack:"containsMeet"`
	ContainsUnion      int        `msgpack:"containsUnion"`
	ContainsWithin     int        `msgpack:"containsWithin"`
	AdhocSubcorpus     float64    `msgpack:"adhocSubcorpus"`
	ContainsContaining int        `msgpack:"containsContaining"`
	CorpusSize         float64    `msgpack:"corpusSize"` // Size of the corpus being searched (e.g., number of tokens)
	NamedSubcorpusSize float64    `msgpack:"namedSubcorpusSize"`
	AlignedPart        int        `msgpack:"alignedPart"`
}

func (eval QueryEvaluation) UniqKey() string {
	return fmt.Sprintf("%s-%.5f", eval.OrigQuery, eval.CorpusSize)
}

func (eval QueryEvaluation) Show() string {
	var ans strings.Builder
	for i, pos := range eval.Positions {
		ans.WriteString(fmt.Sprintf("position %d:\n", i))
		ans.WriteString(fmt.Sprintf("    HasSmallCardAttr: %d\n", pos.HasSmallCardAttr))
		ans.WriteString(fmt.Sprintf("    NumAlternatives: %d\n", pos.NumAlternatives))
		ans.WriteString(fmt.Sprintf("    PosRepetition: %.2f\n", pos.PosRepetition))
		ans.WriteString(fmt.Sprintf("    HasNegation: %d\n", pos.HasNegation))
		ans.WriteString("        regexp:    \n")
		ans.WriteString(fmt.Sprintf("            StartsWithWildCard: %d\n", pos.Regexp.StartsWithWildCard))
		ans.WriteString(fmt.Sprintf("            NumConcreteChars: %.2f\n", pos.Regexp.NumConcreteChars))
		ans.WriteString(fmt.Sprintf("            AvgCharProb: %.2f\n", pos.Regexp.AvgCharProb))
		ans.WriteString(fmt.Sprintf("            WildcardScore: %.2f\n", pos.Regexp.WildcardScore))
		ans.WriteString(fmt.Sprintf("            HasRange: %d\n", pos.Regexp.HasRange))
		ans.WriteString(fmt.Sprintf("            CharClasses: %.2f\n", pos.Regexp.CharClasses))
	}
	ans.WriteString(fmt.Sprintf("NumGlobConditions: %d\n", eval.NumGlobConditions))
	ans.WriteString(fmt.Sprintf("ContainsMeet: %d\n", eval.ContainsMeet))
	ans.WriteString(fmt.Sprintf("ContainsUnion: %d\n", eval.ContainsUnion))
	ans.WriteString(fmt.Sprintf("ContainsWithin: %d\n", eval.ContainsWithin))
	ans.WriteString(fmt.Sprintf("AdhocSubcorpus: %.2f\n", eval.AdhocSubcorpus))
	ans.WriteString(fmt.Sprintf("ContainsContaining: %d\n", eval.ContainsContaining))
	ans.WriteString(fmt.Sprintf("CorpusSize: %0.2f\n", eval.CorpusSize))
	ans.WriteString(fmt.Sprintf("NamedSubcorpusSize: %0.2f\n", eval.NamedSubcorpusSize))
	ans.WriteString(fmt.Sprintf("AlignedPart: %d\n", eval.AlignedPart))

	return ans.String()
}

func (eval QueryEvaluation) Cost(model ModelParams) float64 {
	var total float64

	// Compute position-specific costs
	for i := 0; i < len(eval.Positions) && i < MaxPositions; i++ {
		pos := eval.Positions[i]
		// Get position-specific parameters
		var wildcardPrefix, wildcards, rangeOp, smallCardAttr, concreteChars,
			avgCharProb, numPosAlts, posRepetition, charClasses, hasNegation float64
		switch i {
		case 0:
			wildcardPrefix = model.WildcardPrefix0
			wildcards = model.Wildcards0
			rangeOp = model.RangeOp0
			smallCardAttr = model.SmallCardAttr0
			concreteChars = model.ConcreteChars0
			avgCharProb = model.AvgCharProb0
			numPosAlts = model.NumPosAlts0
			posRepetition = model.PosRepetition0
			charClasses = model.CharClasses0
			hasNegation = model.HasNegation0
		case 1:
			wildcardPrefix = model.WildcardPrefix1
			wildcards = model.Wildcards1
			rangeOp = model.RangeOp1
			smallCardAttr = model.SmallCardAttr1
			concreteChars = model.ConcreteChars1
			avgCharProb = model.AvgCharProb1
			numPosAlts = model.NumPosAlts1
			posRepetition = model.PosRepetition1
			charClasses = model.CharClasses1
			hasNegation = model.HasNegation1
		case 2:
			wildcardPrefix = model.WildcardPrefix2
			wildcards = model.Wildcards2
			rangeOp = model.RangeOp2
			smallCardAttr = model.SmallCardAttr2
			concreteChars = model.ConcreteChars2
			avgCharProb = model.AvgCharProb2
			numPosAlts = model.NumPosAlts2
			posRepetition = model.PosRepetition2
			charClasses = model.CharClasses2
			hasNegation = model.HasNegation2
		case 3:
			wildcardPrefix = model.WildcardPrefix3
			wildcards = model.Wildcards3
			rangeOp = model.RangeOp3
			smallCardAttr = model.SmallCardAttr3
			concreteChars = model.ConcreteChars3
			avgCharProb = model.AvgCharProb3
			numPosAlts = model.NumPosAlts3
			posRepetition = model.PosRepetition3
			charClasses = model.CharClasses3
			hasNegation = model.HasNegation3
		}

		// Calculate position cost
		positionCost := (wildcardPrefix*float64(pos.Regexp.StartsWithWildCard) +
			wildcards*float64(pos.Regexp.WildcardScore) +
			rangeOp*float64(pos.Regexp.HasRange) +
			smallCardAttr*float64(pos.HasSmallCardAttr)) +
			concreteChars*float64(pos.Regexp.NumConcreteChars) +
			avgCharProb*float64(pos.Regexp.AvgCharProb) +
			numPosAlts*float64(pos.NumAlternatives) +
			posRepetition*pos.PosRepetition +
			charClasses*pos.Regexp.CharClasses +
			hasNegation*float64(pos.HasNegation)

		total += positionCost
	}

	// Add global costs
	total += model.GlobCond * float64(eval.NumGlobConditions)
	total += model.Meet * float64(eval.ContainsMeet)
	total += model.Union * float64(eval.ContainsUnion)
	total += model.Within * float64(eval.ContainsWithin)
	total += model.AdhocSubcorpus * float64(eval.AdhocSubcorpus)
	total += model.Containing * float64(eval.ContainsContaining)
	total += model.CorpusSize * math.Log(eval.CorpusSize)
	total += model.NamedSubcorpusSize * math.Log(eval.NamedSubcorpusSize)
	total += model.Bias

	return total
}
