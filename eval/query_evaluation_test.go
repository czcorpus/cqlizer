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
	"testing"
)

func TestNewQueryEvaluationSimple(t *testing.T) {
	corpusSize := 1000000.0
	eval, err := NewQueryEvaluation(`[word="test"]`, corpusSize)
	if err != nil {
		t.Fatalf("Failed to parse simple query: %v", err)
	}

	if len(eval.Positions) != 1 {
		t.Errorf("Expected 1 position, got %d", len(eval.Positions))
	}

	if eval.Positions[0].Index != 0 {
		t.Errorf("Expected position index 0, got %d", eval.Positions[0].Index)
	}

	if eval.CorpusSize != corpusSize {
		t.Errorf("Expected corpus size %.0f, got %.0f", corpusSize, eval.CorpusSize)
	}
}

func TestNewQueryEvaluationMultiplePositions(t *testing.T) {
	eval, err := NewQueryEvaluation(`[word="hello"] [lemma="world"]`, 1000000.0)
	if err != nil {
		t.Fatalf("Failed to parse query: %v", err)
	}

	if len(eval.Positions) != 2 {
		t.Errorf("Expected 2 positions, got %d", len(eval.Positions))
	}

	if eval.Positions[0].Index != 0 {
		t.Errorf("Expected position 0 index 0, got %d", eval.Positions[0].Index)
	}

	if eval.Positions[1].Index != 1 {
		t.Errorf("Expected position 1 index 1, got %d", eval.Positions[1].Index)
	}
}

func TestNewQueryEvaluationWithWildcards(t *testing.T) {
	eval, err := NewQueryEvaluation(`[word="test.*"]`, 1000000.0)
	if err != nil {
		t.Fatalf("Failed to parse query with wildcards: %v", err)
	}

	if len(eval.Positions) != 1 {
		t.Errorf("Expected 1 position, got %d", len(eval.Positions))
	}

	pos := eval.Positions[0]
	if pos.Regexp.NumWildcards == 0 {
		t.Errorf("Expected wildcards to be detected, got %d", pos.Regexp.NumWildcards)
	}
}

func TestNewQueryEvaluationWithWithin(t *testing.T) {
	eval, err := NewQueryEvaluation(`[word="test"] within <s/>`, 1000000.0)
	if err != nil {
		t.Fatalf("Failed to parse query with within: %v", err)
	}

	if eval.ContainsWithin != 1 {
		t.Errorf("Expected ContainsWithin=1, got %d", eval.ContainsWithin)
	}
}

func TestNewQueryEvaluationMaxPositions(t *testing.T) {
	// Test that we correctly limit to MaxPositions (4)
	eval, err := NewQueryEvaluation(`[word="a"] [word="b"] [word="c"] [word="d"] [word="e"]`, 1000000.0)
	if err != nil {
		t.Fatalf("Failed to parse query with 5 positions: %v", err)
	}

	if len(eval.Positions) != MaxPositions {
		t.Errorf("Expected %d positions (max), got %d", MaxPositions, len(eval.Positions))
	}
}

func TestNewQueryEvaluationSmallCardAttr(t *testing.T) {
	// Test that small cardinality attributes are detected
	eval, err := NewQueryEvaluation(`[tag="N.*"]`, 1000000.0)
	if err != nil {
		t.Fatalf("Failed to parse query with tag attribute: %v", err)
	}

	if len(eval.Positions) != 1 {
		t.Errorf("Expected 1 position, got %d", len(eval.Positions))
	}

	if eval.Positions[0].HasSmallCardAttr != 1 {
		t.Errorf("Expected HasSmallCardAttr=1 for tag attribute, got %d", eval.Positions[0].HasSmallCardAttr)
	}
}

func TestNewQueryEvaluationEmptyPosition(t *testing.T) {
	// Test that empty position [] is considered a small cardinality attribute
	eval, err := NewQueryEvaluation(`[]`, 1000000.0)
	if err != nil {
		t.Fatalf("Failed to parse empty position query: %v", err)
	}

	if len(eval.Positions) != 1 {
		t.Errorf("Expected 1 position, got %d", len(eval.Positions))
	}

	if eval.Positions[0].HasSmallCardAttr != 1 {
		t.Errorf("Expected HasSmallCardAttr=1 for empty position [], got %d", eval.Positions[0].HasSmallCardAttr)
	}
}

func TestNewQueryEvaluationWordAttr(t *testing.T) {
	// Test that word attribute is NOT a small cardinality attribute
	eval, err := NewQueryEvaluation(`[word="test"]`, 1000000.0)
	if err != nil {
		t.Fatalf("Failed to parse query with word attribute: %v", err)
	}

	if len(eval.Positions) != 1 {
		t.Errorf("Expected 1 position, got %d", len(eval.Positions))
	}

	if eval.Positions[0].HasSmallCardAttr != 0 {
		t.Errorf("Expected HasSmallCardAttr=0 for word attribute, got %d", eval.Positions[0].HasSmallCardAttr)
	}
}
