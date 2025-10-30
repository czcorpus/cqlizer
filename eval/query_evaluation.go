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

package eval

import (
	"math"
	"strings"

	"github.com/czcorpus/cqlizer/cql"
)

// NewQueryEvaluation creates a QueryEvaluation from a CQL query string and corpus size
func NewQueryEvaluation(cqlQuery string, corpusSize, procTime float64) (QueryEvaluation, error) {
	query, err := cql.ParseCQL("", cqlQuery)
	if err != nil {
		return QueryEvaluation{}, err
	}

	eval := QueryEvaluation{
		ProcTime:   procTime,
		Positions:  make([]Position, 0, MaxPositions),
		CorpusSize: math.Log(corpusSize),
	}

	// Extract features from the parsed query
	extractFeaturesFromQuery(query, &eval)
	return eval, nil
}

// extractFeaturesFromQuery walks the AST and extracts relevant features
func extractFeaturesFromQuery(query *cql.Query, eval *QueryEvaluation) {
	// First pass: collect all OnePosition nodes in order and extract their features
	if query.Sequence != nil {
		positionIndex := 0
		query.Sequence.ForEachElement(query.Sequence, func(parent, v cql.ASTNode) {
			if onePos, ok := v.(*cql.OnePosition); ok && positionIndex < MaxPositions {
				pos := extractPositionFeatures(onePos)
				pos.Index = positionIndex
				eval.Positions = append(eval.Positions, pos)
				positionIndex++
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

		case *cql.MeetOp:
			eval.ContainsMeet = 1

		case *cql.UnionOp:
			eval.ContainsUnion = 1

		case *cql.AlignedPart:
			eval.AlignedPart = 1
		}
	})
}

// extractPositionFeatures analyzes a position to extract all features including regexp and attribute info
func extractPositionFeatures(pos *cql.OnePosition) Position {
	regexp := Regexp{
		StartsWithWildCard: 0,
		NumConcreteChars:   0,
		NumWildcards:       0,
		HasRange:           0,
	}
	position := Position{
		HasSmallCardAttr: 0,
	}

	// Check if this is an empty position query []
	isEmpty := true
	numAlternatives := 0
	// Traverse the position to find regexp patterns and attribute info
	// Using DFS-like approach to maintain proper parent-child context
	pos.ForEachElement(pos, func(parent, v cql.ASTNode) {
		switch typedNode := v.(type) {
		case *cql.RegExp:
			isEmpty = false
			analyzeRegExp(typedNode, &regexp)

		case *cql.RgSimple:
			isEmpty = false
			// Use the built-in method to count wildcards
			regexp.NumWildcards += typedNode.NumWildcards()

		case *cql.RawString:
			isEmpty = false
			// Simple string - count characters
			text := typedNode.Text()
			if len(text) > 2 {
				regexp.NumConcreteChars = float64(len(text) - 2) // -2 for quotes
			}

		case *cql.AttVal:
			isEmpty = false
			// Check if this is a small cardinality attribute
			if isSmallCardinalityAttr(typedNode) {
				position.HasSmallCardAttr = 1
			}
			if !typedNode.IsRecursive() {
				numAlternatives++
			}
		}
	})

	// Empty query [] is considered a small cardinality attribute search
	if isEmpty {
		position.NumAlternatives = 1
		position.HasSmallCardAttr = 1

	} else if numAlternatives > 0 {
		position.NumAlternatives = numAlternatives

	} else {
		position.NumAlternatives = 1 // AUTO-FIX
		// TODO - this should be solved within the AST,
		// it is caused by direct regexp queries: "foo"
	}
	regexp.NumConcreteChars /= float64(position.NumAlternatives)
	position.Regexp = regexp

	//fmt.Printf("POSION >>> %#v\n", position)
	return position
}

// analyzeRegExp examines a RegExp node to extract features
func analyzeRegExp(re *cql.RegExp, regexp *Regexp) {
	if len(re.RegExpRaw) == 0 {
		return
	}

	// Check if starts with wildcard
	firstRaw := re.RegExpRaw[0]
	if startsWithWildcard(firstRaw) {
		regexp.StartsWithWildCard = 1
	}

	// Count concrete chars and check for ranges
	concreteChars := 0
	hasRange := false

	for _, raw := range re.RegExpRaw {
		raw.ForEachElement(raw, func(parent, v cql.ASTNode) {
			switch typedNode := v.(type) {
			case *cql.RgRange:
				hasRange = true

			case *cql.RgChar:
				if typedNode.IsConstant() {
					concreteChars++
				}
			}
		})
	}

	if concreteChars > 0 {
		regexp.NumConcreteChars += float64(concreteChars)
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
