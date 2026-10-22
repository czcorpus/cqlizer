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

// UnsupportedFeature marks a CQL construct we're unable to
// translate to DQD.
type UnsupportedFeature struct {
	Feature string
}

func (err *UnsupportedFeature) Error() string {
	return fmt.Sprintf("Unsupported feature %s", err.Feature)
}

// CorpusProps contains the corpus-schema-specific information CQL2DQD
// needs but that a CQL query itself never carries - DQD's Layer/attribute
// names are per-corpus (e.g. CQL "word" might be DQD "Token.form" in one
// corpus and "Token.text" in another), so they can't be derived from the
// query alone.
//
// TODO: stub - fields below are a first guess at the shape; fill in once
// a real corpus config source is wired up.
type CorpusProps struct {
	// PosAttrLayer is the DQD layer CQL positional attributes belong to
	// (almost always "Token").
	PosAttrLayer string

	// PosAttrs maps a CQL positional attribute name (e.g. "word") to its
	// DQD attribute name (e.g. "form").
	PosAttrs map[string]string

	// Structures maps a CQL structure name (e.g. "text") to its DQD
	// layer name (e.g. "Text"). This can't be a simple capitalize() -
	// DQD layer names are corpus-defined, not derived from CQL names.
	Structures map[string]string

	// StructAttrs maps a CQL structure attribute name to its DQD
	// attribute name, keyed by CQL structure name.
	StructAttrs map[string]map[string]string
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
	// "=>" in DQD source).
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
	t := &translator{corpusProps: cp, viewOpts: vo}
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
	labelSeq    int
}

// nextLabel allocates a fresh DQD label (e.g. "t1", "t2", ...) - DQD
// requires explicit labels that CQL positions don't carry.
func (t *translator) nextLabel(prefix string) string {
	t.labelSeq++
	return fmt.Sprintf("%s%d", prefix, t.labelSeq)
}

// translateQuery is the entry point of the AST walk, mapping a
// cql.Query onto the IR in ir.go. Stubbed for now - the planned mapping:
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
	return nil, &UnsupportedFeature{Feature: "query translation (not implemented yet)"}
}
