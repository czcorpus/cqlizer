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

package dqd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// test helpers for building operands

func ref(v string) Operand { return Operand{Reference: &Reference{Value: v}} }
func attr(label, name string) Operand {
	return Operand{Attribute: &Attribute{Label: label, Name: name}}
}
func str(v string) Operand { return Operand{String: &StringLit{Value: v}} }
func num(v string) Operand { return Operand{Math: &Math{Number: v}} }

func cmp(left Operand, comparator string, right Operand) Constraint {
	return Constraint{Comparison: &Comparison{Left: left, Comparator: comparator, Right: right}}
}

func plainResult(context string, entities ...string) []Result {
	return []Result{{
		Label: "res",
		Plain: &PlainResult{Context: []string{context}, Entities: entities},
	}}
}

type renderCase struct {
	name     string
	query    Query
	expected string
}

var renderCases = []renderCase{
	{
		name: "single unit",
		query: Query{
			Items: []QueryItem{
				{Unit: &Unit{Layer: "Segment", Label: "s"}},
				{Unit: &Unit{
					Layer:       "Token",
					Label:       "t1",
					PartOf:      []PartOf{{Kind: PartOfStream, Label: "s"}},
					Constraints: []Constraint{cmp(ref("lemma"), "=", str("pes"))},
				}},
			},
			Results: plainResult("s", "t1"),
		},
		expected: `Segment s
Token@s t1
    lemma = "pes"

res => plain
    context
        s
    entities
        t1
`,
	},
	{
		name: "sequence with logic and repetition",
		query: Query{
			Items: []QueryItem{
				{Unit: &Unit{Layer: "Segment", Label: "s"}},
				{Sequence: &Sequence{
					Label: "seq",
					Members: []Member{
						{Unit: &Unit{
							Layer:  "Token",
							Label:  "t1",
							PartOf: []PartOf{{Kind: PartOfStream, Label: "s"}},
							Constraints: []Constraint{
								cmp(ref("upos"), "=", str("ADJ")),
								{Comparison: &Comparison{
									Left: ref("form"), Not: true, Comparator: "=",
									Right: Operand{Regex: &Regex{Pattern: "^x.*"}},
								}},
							},
						}},
						{Sequence: &Sequence{
							Repetition: &Repetition{Min: "0", Max: "*"},
							Members: []Member{
								{Unit: &Unit{
									Layer:  "Token",
									PartOf: []PartOf{{Kind: PartOfStream, Label: "s"}},
								}},
							},
						}},
						{Unit: &Unit{
							Layer:  "Token",
							Label:  "t2",
							PartOf: []PartOf{{Kind: PartOfStream, Label: "s"}},
							Constraints: []Constraint{
								{LogicalExpression: &LogicalExpression{
									Op: LogicalOr,
									Args: []Constraint{
										cmp(ref("lemma"), "=", Operand{Regex: &Regex{Pattern: "pes.*", CaseInsensitive: true}}),
										{LogicalExpression: &LogicalExpression{
											Op:   LogicalNot,
											Args: []Constraint{cmp(ref("upos"), "=", str("NOUN"))},
										}},
									},
								}},
							},
						}},
					},
				}},
			},
			Results: plainResult("s", "t1", "t2"),
		},
		expected: `Segment s
sequence seq
    Token@s t1
        upos = "ADJ"
        form != /^x.*/
    sequence 0..*
        Token@s
    Token@s t2
        OR
            lemma = /pes.*/i
            !
                upos = "NOUN"

res => plain
    context
        s
    entities
        t1
        t2
`,
	},
	{
		name: "quantification, set and cross-unit comparisons",
		query: Query{
			Items: []QueryItem{
				{Unit: &Unit{
					Layer:  "Segment",
					Label:  "s",
					PartOf: []PartOf{{Kind: PartOfStream, Label: "d"}, {Kind: PartOfTime, Label: "x"}},
				}},
				{Set: &Set{
					Label:  "st",
					PartOf: []PartOf{{Kind: PartOfStream, Label: "s"}},
					Members: []Member{
						{Unit: &Unit{Layer: "Token", Label: "t1"}},
						{Unit: &Unit{Layer: "Token", Label: "t2"}},
					},
				}},
				{Quantification: &Quantification{
					Negated: true,
					Unit: &Unit{
						Layer:       "Token",
						Label:       "t3",
						PartOf:      []PartOf{{Kind: PartOfStream, Label: "s"}},
						Constraints: []Constraint{cmp(ref("upos"), "=", str("PUNCT"))},
					},
				}},
				{Constraint: &Constraint{
					Comparison: &Comparison{Left: attr("t1", "lemma"), Comparator: "=", Right: attr("t2", "lemma")},
				}},
				{Constraint: &Constraint{
					Comparison: &Comparison{
						Left: Operand{Function: &Function{
							Name: "length",
							Args: []Operand{ref("s")},
						}},
						Comparator: ">",
						Right: Operand{Math: &Math{Operation: &Operation{
							Left: num("2"), Operator: "+", Right: num("3"),
						}}},
					},
				}},
			},
			Results: plainResult("s", "*"),
		},
		expected: `Segment@d%x s
set@s st
    Token t1
    Token t2
!EXISTS
    Token@s t3
        upos = "PUNCT"
t1.lemma = t2.lemma
length(s) > (2 + 3)

res => plain
    context
        s
    entities
        *
`,
	},
	{
		name: "analysis and collocation results",
		query: Query{
			Items: []QueryItem{
				{Unit: &Unit{Layer: "Segment", Label: "s"}},
				{Unit: &Unit{Layer: "Token", Label: "t", PartOf: []PartOf{{Kind: PartOfStream, Label: "s"}}}},
			},
			Results: []Result{
				{
					Label: "freq",
					Analysis: &AnalysisResult{
						Attributes: []Operand{attr("t", "lemma")},
						Functions:  []string{"frequency"},
						Filter:     []Comparison{{Left: ref("frequency"), Comparator: ">", Right: num("5")}},
					},
				},
				{
					Label:       "coll1",
					Collocation: &CollocationResult{Space: "s", Attribute: "t.lemma"},
				},
				{
					Label: "coll2",
					Collocation: &CollocationResult{
						Center:    "t",
						Window:    &Window{Left: "-5", Right: "5"},
						Attribute: "lemma",
					},
				},
			},
		},
		expected: `Segment s
Token@s t

freq => analysis
    attributes
        t.lemma
    functions
        frequency
    filter
        frequency > 5

coll1 => collocation
    space
        s
    attribute
        t.lemma

coll2 => collocation
    center
        t
    window
        -5..5
    attribute
        lemma
`,
	},
}

func TestRender(t *testing.T) {
	for _, tc := range renderCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.query.String())
		})
	}
}

func TestRenderPanicsOnInvalidIR(t *testing.T) {
	q := Query{Items: []QueryItem{{Unit: &Unit{
		Layer:       "Token",
		Constraints: []Constraint{{}},
	}}}}
	assert.PanicsWithValue(t, "dqd: invalid IR - empty Constraint", func() { _ = q.String() })

	q = Query{Items: []QueryItem{{Constraint: &Constraint{
		LogicalExpression: &LogicalExpression{Op: LogicalNot},
	}}}}
	assert.Panics(t, func() { _ = q.String() })
}

// TestRenderedSyntax validates rendered queries against the reference
// DQD grammar (see checkDQDSyntax).
func TestRenderedSyntax(t *testing.T) {
	skipIfNoDQDCheck(t)
	for _, tc := range renderCases {
		t.Run(tc.name, func(t *testing.T) {
			checkDQDSyntax(t, tc.query.String())
		})
	}
}
