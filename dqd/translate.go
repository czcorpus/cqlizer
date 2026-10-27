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

// Package dqd translates CQL queries into DQD (Data Query DSL).
// The actual version of the DQD grammar used for this package can
// be found at
// https://github.com/liri-uzh/lcp/blob/1fc30d0da68c67706c26a46a4b5b127f76cceac1/parser/dqd_grammar.lark
package dqd

import (
	"fmt"

	"github.com/czcorpus/cqlizer/cql"
)

const (
	defaultResultLabel = "res"
	segmentLabel       = "s"
	tokenLabelPrefix   = "t"
)

// UnsupportedFeature marks a CQL construct we're unable to
// translate to DQD.
type UnsupportedFeature struct {
	Feature string
}

func (err *UnsupportedFeature) Error() string {
	return fmt.Sprintf("Unsupported feature %s", err.Feature)
}

func unsupported(feature string) error {
	return &UnsupportedFeature{Feature: feature}
}

// ViewOptions controls the "results" block CQL2DQD generates - DQD
// requires at least one result, but CQL carries no information about
// what a caller wants to see, so this can't be derived from the query
// either.
//
// TODO: stub - currently only enough to name a single "plain" result;
// extend once analysis/collocation output is supported.
type ViewOptions struct {
	// ResultLabel names the generated result entry (the label before
	// "=>" in DQD source). If empty, "res" is used.
	ResultLabel string
}

// CQL2DQD translates a CQL query into its DQD textual representation.
//
// Returns an error rather than just a string (unlike the originally
// sketched signature) so that CQL parse failures and
// ErrUnsupportedFeature are visible to the caller instead of silently
// producing empty or partial output.
func CQL2DQD(cp CorpusProps, vo ViewOptions, query string) (string, error) {
	q, err := cql.ParseCQL("query", query)
	if err != nil {
		return "", fmt.Errorf("failed to parse CQL query: %w", err)
	}
	t := &translator{corpusProps: cp, viewOpts: vo, labelSeq: make(map[string]int)}
	dq, err := t.translateQuery(q)
	if err != nil {
		return "", err
	}
	return dq.String(), nil
}

// translator holds the state needed while walking a cql.Query and
// building the equivalent dqd.Query.
type translator struct {
	corpusProps CorpusProps
	viewOpts    ViewOptions

	// labelSeq holds the last used number for each label prefix
	labelSeq map[string]int
}

// nextLabel allocates a fresh DQD label (e.g. "t1", "t2", ...) - DQD
// requires explicit labels that CQL positions don't carry.
func (t *translator) nextLabel(prefix string) string {
	t.labelSeq[prefix]++
	return fmt.Sprintf("%s%d", prefix, t.labelSeq[prefix])
}

// translateQuery is the entry point of the AST walk, mapping a
// cql.Query onto the IR in ir.go.
//
// Currently supported: a single position with at most one attribute
// condition. The planned mapping for the rest:
//
//   - cql.Sequence            -> Sequence, one Unit per bracket position
//   - AttValAnd ("&"-joined)  -> plain Unit.Constraints entries (DQD
//     ANDs same-unit constraints implicitly, no LogicalExpression needed)
//   - AttValList ("|"-joined) -> LogicalExpression{Op: LogicalOr}
//   - cql.WithinOrContaining  -> a Unit for the structure + PartOf on
//     each enclosed Token
//   - cql.MuPart (union/meet) -> ErrUnsupportedFeature (no DQD primitive)
//   - cql.GlobCond's freq(...) form -> ErrUnsupportedFeature (DQD only
//     has frequency inside results_analysis, not as a query constraint)
func (t *translator) translateQuery(q *cql.Query) (*Query, error) {
	if q.GlobPart != nil {
		return nil, unsupported("global conditions (& 1.attr = 2.attr)")
	}
	if len(q.WithinOrContaining) > 0 {
		return nil, unsupported("within/containing")
	}
	if len(q.Sequence.Seq) != 1 {
		return nil, unsupported("alternative sequences (seq1 | seq2)")
	}
	seq := q.Sequence.Seq[0]
	if seq.Not != "" {
		return nil, unsupported("negated sequence")
	}
	if len(seq.Repetition) != 1 {
		return nil, unsupported("sequence of multiple positions")
	}

	segment := &Unit{Layer: t.corpusProps.SegmentLayer, Label: segmentLabel}
	token, err := t.translateRepetition(seq.Repetition[0])
	if err != nil {
		return nil, err
	}
	resultLabel := t.viewOpts.ResultLabel
	if resultLabel == "" {
		resultLabel = defaultResultLabel
	}
	return &Query{
		Items: []QueryItem{{Unit: segment}, {Unit: token}},
		Results: []Result{{
			Label: resultLabel,
			Plain: &PlainResult{
				Context:  []string{segment.Label},
				Entities: []string{token.Label},
			},
		}},
	}, nil
}

// translateRepetition translates a single CQL position (with its
// optional repetition operator) into a Token unit being part of the
// segment.
func (t *translator) translateRepetition(rep *cql.Repetition) (*Unit, error) {
	if rep.Variant1 == nil {
		return nil, unsupported("structure tags (<s>, </s>) inside a sequence")
	}
	if rep.Variant1.RepOpt != nil {
		return nil, unsupported("repetition (*, +, ?, {n,m})")
	}
	pos := rep.Variant1.AtomQuery.Position()
	if pos == nil {
		return nil, unsupported("parenthesized sub-query")
	}
	onePos := pos.OnePosition()
	if onePos == nil {
		return nil, unsupported("numbered position (1:[...])")
	}
	unit := &Unit{
		Layer:  t.corpusProps.PosAttrLayer,
		Label:  t.nextLabel(tokenLabelPrefix),
		PartOf: []PartOf{{Kind: PartOfStream, Label: segmentLabel}},
	}
	switch {
	case onePos.Variant1 != nil:
		// [attr="value" ...] or empty []
		if onePos.Variant1.AttValList == nil {
			break
		}
		c, err := t.translateAttValList(onePos.Variant1.AttValList)
		if err != nil {
			return nil, err
		}
		unit.Constraints = append(unit.Constraints, c)

	case onePos.Variant2 != nil:
		// bare "value" applied to the default attribute
		c, err := t.attrComparison(t.corpusProps.DefaultAttr, false, onePos.Variant2.RegExp)
		if err != nil {
			return nil, err
		}
		unit.Constraints = append(unit.Constraints, c)

	default:
		return nil, unsupported("this type of position (meet/union, ~ query)")
	}
	return unit, nil
}

func (t *translator) translateAttValList(avl *cql.AttValList) (Constraint, error) {
	if len(avl.AttValAnd) != 1 || len(avl.AttValAnd[0].AttVal) != 1 {
		return Constraint{}, unsupported("boolean operators (&, |) inside a position")
	}
	return t.translateAttVal(avl.AttValAnd[0].AttVal[0])
}

func (t *translator) translateAttVal(av *cql.AttVal) (Constraint, error) {
	switch {
	case av.Variant1 != nil:
		// attr == "literal"
		value := ""
		if av.Variant1.RawString.SimpleString != nil {
			value = av.Variant1.RawString.SimpleString.String()
		}
		if hasEscapes(value) {
			return Constraint{}, unsupported("escaped characters in a value")
		}
		return t.comparison(av.Variant1.AttName.String(), av.Variant1.Not, value)

	case av.Variant2 != nil:
		// attr = "regexp"
		if av.Variant2.Op.String() != "=" {
			return Constraint{}, unsupported(fmt.Sprintf("operator %s", av.Variant2.Op))
		}
		return t.attrComparison(av.Variant2.AttName.String(), av.Variant2.Not, av.Variant2.RegExp)
	}
	return Constraint{}, unsupported("this type of attribute condition")
}

// attrComparison creates a comparison of a positional attribute with
// a CQL regexp value.
//
// TODO: only regexps without any special characters (i.e. plain strings)
// are supported for now.
func (t *translator) attrComparison(cqlAttr string, not bool, rg *cql.RegExp) (Constraint, error) {
	value, ok := plainRegExpValue(rg)
	if !ok {
		return Constraint{}, unsupported("regular expressions")
	}
	return t.comparison(cqlAttr, not, value)
}

func (t *translator) comparison(cqlAttr string, not bool, value string) (Constraint, error) {
	dqdAttr, ok := t.corpusProps.PosAttrs[cqlAttr]
	if !ok {
		return Constraint{}, unsupported(
			fmt.Sprintf("positional attribute '%s' (no mapping in corpus props)", cqlAttr))
	}
	return Constraint{Comparison: &Comparison{
		Left:       Operand{Reference: &Reference{Value: dqdAttr}},
		Not:        not,
		Comparator: "=",
		Right:      Operand{String: &StringLit{Value: value}},
	}}, nil
}
