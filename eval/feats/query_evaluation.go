// Copyright 2024 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2024 Institute of the Czech National Corpus,
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
	"math"
	"strings"

	"github.com/czcorpus/cqlizer/cql"
)

const (
	MaxPositions = 4
)

type CorpusProps struct {
	Size int    `json:"size"`
	Lang string `json:"lang"`
}

// NewQueryEvaluation creates a QueryEvaluation from a CQL query string and corpus size
func NewQueryEvaluation(cqlQuery string, corpusSize, procTime float64, charProbs charProbabilityProvider) (QueryEvaluation, error) {
	query, err := cql.ParseCQL("", cqlQuery)
	if err != nil {
		return QueryEvaluation{}, err
	}

	eval := QueryEvaluation{
		OrigQuery:  cqlQuery,
		ProcTime:   procTime,
		Positions:  make([]Position, 0, MaxPositions),
		CorpusSize: math.Log(corpusSize),
	}

	// Extract features from the parsed query
	extractFeaturesFromQuery(query, &eval, charProbs)
	return eval, nil
}

// extractFeaturesFromQuery walks the AST and extracts relevant features
func extractFeaturesFromQuery(query *cql.Query, eval *QueryEvaluation, charProbs charProbabilityProvider) {
	// First pass: collect all OnePosition nodes in order and extract their features
	if query.Sequence != nil {
		positionIndex := 0
		query.Sequence.ForEachElement(query.Sequence, func(parent, v cql.ASTNode) {
			switch typedNode := v.(type) {
			case *cql.Repetition:
				if positionIndex < MaxPositions {
					var pos Position
					if typedNode.IsAnyPosition() {
						pos.NumAlternatives = 1
						pos.Regexp.StartsWithWildCard = 1
						pos.Regexp.WildcardScore = 500 // TODO is this equivalent score to [attr=".*"]
					}
					pos.PosRepetition = typedNode.RepetitionScore()
					typedNode.ForEachElement(typedNode, func(parent, v2 cql.ASTNode) {
						switch typedNode2 := v2.(type) {
						case *cql.OnePosition:
							extractPositionFeatures(typedNode2, charProbs, &pos)
							pos.Index = positionIndex
							eval.Positions = append(eval.Positions, pos)
							positionIndex++
						}
					})
				}
			}
		})
	}

	// Second pass: extract global features from entire query
	query.ForEachElement(func(parent, v cql.ASTNode) {
		switch typedNode := v.(type) {
		case *cql.GlobPart:
			eval.NumGlobConditions++

		case *cql.WithinOrContaining:
			if typedNode.NumWithinParts() > 0 {
				eval.ContainsWithin = 1
			}
			if typedNode.NumContainingParts() > 0 {
				eval.ContainsContaining = 1
			}

			typedNode.ForEachElement(typedNode, func(parent, v2 cql.ASTNode) {
				switch typedNode2 := v2.(type) {
				case *cql.Repetition:
					eval.AdhocSubcorpus += typedNode2.SubcorpusDefScore()
				}
			})

		case *cql.MeetOp:
			eval.ContainsMeet = 1

		case *cql.UnionOp:
			eval.ContainsUnion = 1

		case *cql.AlignedPart:
			eval.AlignedPart = 1
		}
	})
}

func textToProbs(v string, probsMap charProbabilityProvider) float64 {
	var ansProb float64 = 0
	var size int
	for _, c := range v {
		ansProb += probsMap.CharProbability(c)
		size++
	}
	return ansProb / float64(size)
}

// extractPositionFeatures analyzes a position to extract all features including regexp and attribute info
func extractPositionFeatures(pos *cql.OnePosition, charProbs charProbabilityProvider, outPos *Position) {

	// Check if this is an empty position query []
	numAlternatives := 0
	// Traverse the position to find regexp patterns and attribute info
	// Using DFS-like approach to maintain proper parent-child context
	pos.ForEachElement(pos, func(parent, v cql.ASTNode) {
		switch typedNode := v.(type) {
		case *cql.RegExp:
			analyzeRegExp(typedNode, &outPos.Regexp, charProbs)

		case *cql.RgSimple:
			// Use the built-in method to count wildcards
			outPos.Regexp.WildcardScore += typedNode.WildcardScore()

		case *cql.RawString:
			// Simple string - count characters
			text := typedNode.Text()
			if len(text) > 2 {
				text = strings.Trim(text, `"`)
				outPos.Regexp.NumConcreteChars = float64(len(text) - 2) // -2 for quotes
				outPos.Regexp.AvgCharProb = textToProbs(text, charProbs)
			}

		case *cql.RgAlt:
			outPos.Regexp.CharClasses = typedNode.Score()

		case *cql.AttVal:
			// Check if this is a small cardinality attribute
			if isSmallCardinalityAttr(typedNode) {
				outPos.HasSmallCardAttr = 500
			}
			if !typedNode.IsRecursive() {
				numAlternatives++
			}
			if typedNode.IsNegation() {
				outPos.HasNegation = 1
			}
		}
	})

	if numAlternatives > 0 {
		outPos.NumAlternatives = numAlternatives

	} else {
		outPos.NumAlternatives = 1 // AUTO-FIX
		// TODO - this should be solved within the AST,
		// it is caused by direct regexp queries: "foo"
	}
	outPos.Regexp.NumConcreteChars /= float64(outPos.NumAlternatives)
}

// analyzeRegExp examines a RegExp node to extract features
func analyzeRegExp(re *cql.RegExp, regexp *Regexp, charProbs charProbabilityProvider) {
	if len(re.RegExpRaw) == 0 {
		return
	}

	// Check if starts with wildcard
	firstRaw := re.RegExpRaw[0]
	if startsWithWildcard(firstRaw) {
		regexp.StartsWithWildCard = 1
	}

	// Count concrete chars and check for ranges
	var concreteChars int
	var avgCharProb float64
	hasRange := false

	for _, raw := range re.RegExpRaw {
		raw.ForEachElement(raw, func(parent, v cql.ASTNode) {
			switch typedNode := v.(type) {
			case *cql.RgRange:
				hasRange = true

			case *cql.RgChar:
				if typedNode.IsConstant() {
					concreteChars++
					avgCharProb += textToProbs(typedNode.Text(), charProbs)
				}
			}
		})
	}

	if concreteChars > 0 {
		regexp.NumConcreteChars += float64(concreteChars)
		avgCharProb /= float64(concreteChars)
		regexp.AvgCharProb += avgCharProb
	}
	if hasRange {
		regexp.HasRange = 1
	}
}

// startsWithWildcard checks if a RegExpRaw starts with a wildcard operator
func startsWithWildcard(raw *cql.RegExpRaw) bool {
	if len(raw.Values) == 0 {
		return false
	}
	return strings.HasPrefix(raw.Text(), ".+") || strings.HasPrefix(raw.Text(), ".*")
}

// isSmallCardinalityAttr checks if an attribute has small cardinality
// These are attributes like tag, pos, etc. that have few possible values
func isSmallCardinalityAttr(attVal *cql.AttVal) bool {
	var attrName string

	// Extract attribute name from either variant
	if attVal.Variant1 != nil && attVal.Variant1.AttName != "" {
		attrName = strings.ToLower(attVal.Variant1.AttName.String())

	} else if attVal.Variant2 != nil && attVal.Variant2.AttName != "" {
		attrName = strings.ToLower(attVal.Variant2.AttName.String())

	} else {
		return false
	}

	// List of known small cardinality attributes
	// tag, pos are typical linguistic attributes with limited value sets
	smallCardAttrs := []string{"tag", "pos", "postag", "xpos", "upos", "deprel"}

	for _, attr := range smallCardAttrs {
		if attrName == attr {
			return true
		}
	}

	return false
}

// ExtractFeatures converts QueryEvaluation to feature vector (same as Huber)
func ExtractFeatures(eval QueryEvaluation) []float64 {
	features := make([]float64, NumFeatures)
	idx := 0

	// Extract features for up to 4 positions
	for i := 0; i < MaxPositions; i++ {
		if i < len(eval.Positions) {
			pos := eval.Positions[i]
			// Position-specific features (normalized by concrete chars)
			features[idx] = float64(pos.Regexp.StartsWithWildCard)
			features[idx+1] = pos.Regexp.WildcardScore
			features[idx+2] = float64(pos.Regexp.HasRange)
			features[idx+3] = float64(pos.HasSmallCardAttr)
			features[idx+4] = float64(pos.Regexp.NumConcreteChars)
			features[idx+5] = pos.Regexp.AvgCharProb
			features[idx+6] = float64(pos.NumAlternatives)
			features[idx+7] = pos.PosRepetition
			features[idx+8] = pos.Regexp.CharClasses
			features[idx+9] = float64(pos.HasNegation)
		}
		// If position doesn't exist, features remain 0
		idx += 10
	}

	// Global features
	features[40] = float64(eval.NumGlobConditions)
	features[41] = float64(eval.ContainsMeet)
	features[42] = float64(eval.ContainsUnion)
	features[43] = float64(eval.ContainsWithin)
	features[44] = eval.AdhocSubcorpus
	features[45] = float64(eval.ContainsContaining)
	features[46] = math.Log(eval.CorpusSize)
	features[47] = float64(eval.AlignedPart)
	features[48] = 1.0 // Bias term

	return features
}
