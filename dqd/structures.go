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
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/czcorpus/cqlizer/cql"
)

// Translation of CQL structures (within, containing).
//
// A CQL structure becomes a DQD unit of the structure's layer. How it is
// connected with the rest of the query depends on the structure's position
// in the layer hierarchy:
//
//   - the segment layer itself (e.g. <s/>): the structure's conditions are
//     added directly to the segment unit
//   - a structure containing segments (e.g. <doc/>):
//     "Document document1" + "Segment@document1 s"
//   - other structures (e.g. a named entity inside a sentence):
//     "NamedEntity@s namedEntity1" + all the tokens are "@namedEntity1"
//
// Supported forms:
//
//   - query within <struct attrs/> (also multiple within clauses)
//   - <struct attrs/> containing query (the result entity is the structure)
//   - <struct attrs/> !containing query (using "!EXISTS")
//
// Negated within (!within) is not supported - DQD would have to express
// "the segment is not part of any such structure" which requires
// a reference from inside a quantification to the outer segment in the
// opposite direction.

const (
	keywordContaining = "containing"
)

// queryParts collects the parts of a DQD query while translating a CQL
// query with structures. The parts are then assembled in the order
// required by DQD (labels must be declared before they are referenced).
type queryParts struct {
	// above are structures containing the segment
	above []*Unit

	segment *Unit

	// below are structures inside the segment
	below []*Unit

	// tokenParts are the translated token sequences (the main query
	// or the queries following "containing")
	tokenParts []QueryItem
}

// structureTag returns the structure in case the sequence consists
// of a single structure tag (e.g. <s/>). Otherwise, nil is returned.
func structureTag(seq *cql.Seq) *cql.Structure {
	if seq.Not != "" || len(seq.Repetition) != 1 || seq.Repetition[0].Variant2 == nil {
		return nil
	}
	return seq.Repetition[0].Variant2.OpenStructTag.Structure
}

// structAttrs creates an attrResolver for attributes of a structure.
func structAttrs(structName string, sp StructProps) attrResolver {
	return func(cqlAttr string) (string, error) {
		dqdAttr, ok := sp.Attrs[cqlAttr]
		if !ok {
			return "", unsupported(fmt.Sprintf(
				"attribute '%s' of structure <%s> (no mapping in corpus props)", cqlAttr, structName))
		}
		return dqdAttr, nil
	}
}

// structLabelPrefix derives a label prefix from a layer name
// (e.g. NamedEntity => namedEntity) so labels are readable.
func structLabelPrefix(layer string) string {
	first, size := utf8.DecodeRuneInString(layer)
	return string(unicode.ToLower(first)) + layer[size:]
}

// addStructure adds a unit for a CQL structure (see the description above)
// and returns its label.
func (t *translator) addStructure(parts *queryParts, st *cql.Structure) (string, error) {
	name := st.AttName.String()
	sp, ok := t.corpusProps.Structures[name]
	if !ok {
		return "", unsupported(fmt.Sprintf("structure <%s> (no mapping in corpus props)", name))
	}
	var constraints []Constraint
	if st.AttValList != nil {
		var err error
		constraints, err = t.translateAttValList(st.AttValList, structAttrs(name, sp))
		if err != nil {
			return "", err
		}
	}
	switch {
	case sp.Layer == t.corpusProps.SegmentLayer:
		parts.segment.Constraints = append(parts.segment.Constraints, constraints...)
		return parts.segment.Label, nil

	case sp.ContainsSegments:
		unit := &Unit{
			Layer:       sp.Layer,
			Label:       t.nextLabel(structLabelPrefix(sp.Layer)),
			Constraints: constraints,
		}
		parts.above = append(parts.above, unit)
		parts.segment.PartOf = append(parts.segment.PartOf, PartOf{Kind: PartOfStream, Label: unit.Label})
		return unit.Label, nil

	default:
		unit := &Unit{
			Layer:       sp.Layer,
			Label:       t.nextLabel(structLabelPrefix(sp.Layer)),
			PartOf:      []PartOf{{Kind: PartOfStream, Label: parts.segment.Label}},
			Constraints: constraints,
		}
		parts.below = append(parts.below, unit)
		return unit.Label, nil
	}
}

// withinContainingSeq returns the sequence following within/containing
// or an error in case it is a different construct.
func withinContainingSeq(wc *cql.WithinOrContaining) (*cql.Seq, error) {
	part := wc.WithinContainingPart
	switch {
	case part.WithinNumber() != nil:
		return nil, unsupported(fmt.Sprintf("%s a number of tokens", wc.Keyword))
	case part.AlignedPart() != nil:
		return nil, unsupported("aligned (parallel) corpora queries")
	}
	if len(part.Sequence().Seq) != 1 {
		return nil, unsupported(fmt.Sprintf("alternative sequences (seq1 | seq2) after %s", wc.Keyword))
	}
	return part.Sequence().Seq[0], nil
}

// addWithin translates "within <struct/>".
func (t *translator) addWithin(parts *queryParts, wc *cql.WithinOrContaining) error {
	if wc.IsNegated() {
		return unsupported("negated within (!within)")
	}
	seq, err := withinContainingSeq(wc)
	if err != nil {
		return err
	}
	st := structureTag(seq)
	if st == nil {
		return unsupported("within a query (only within <structure/> is supported)")
	}
	_, err = t.addStructure(parts, st)
	return err
}

// addContaining translates "containing query" (the structure the query
// is contained in is translated separately).
func (t *translator) addContaining(parts *queryParts, wc *cql.WithinOrContaining) error {
	seq, err := withinContainingSeq(wc)
	if err != nil {
		return err
	}
	if structureTag(seq) != nil {
		return unsupported("containing a structure")
	}
	item, _, err := t.translateSeq(seq)
	if err != nil {
		return err
	}
	if wc.IsNegated() {
		item = QueryItem{Quantification: &Quantification{
			Negated:  true,
			Unit:     item.Unit,
			Sequence: item.Sequence,
		}}
	}
	parts.tokenParts = append(parts.tokenParts, item)
	return nil
}

// translateStructureQuery translates "<struct/> containing query
// [within/containing ...]". The structure becomes the result entity.
func (t *translator) translateStructureQuery(
	parts *queryParts,
	target *cql.Structure,
	clauses []*cql.WithinOrContaining,
) ([]string, error) {
	label, err := t.addStructure(parts, target)
	if err != nil {
		return nil, err
	}
	var numContaining int
	for _, wc := range clauses {
		if strings.EqualFold(wc.Keyword.String(), keywordContaining) {
			err = t.addContaining(parts, wc)
			numContaining++

		} else {
			err = t.addWithin(parts, wc)
		}
		if err != nil {
			return nil, err
		}
	}
	if numContaining == 0 {
		return nil, unsupported("a structure alone as a query (only <structure/> containing ... is supported)")
	}
	return []string{label}, nil
}

// translateTokenQuery translates "query [within <struct/> ...]".
func (t *translator) translateTokenQuery(
	parts *queryParts,
	seq *cql.Seq,
	clauses []*cql.WithinOrContaining,
) ([]string, error) {
	item, entities, err := t.translateSeq(seq)
	if err != nil {
		return nil, err
	}
	parts.tokenParts = append(parts.tokenParts, item)
	for _, wc := range clauses {
		if strings.EqualFold(wc.Keyword.String(), keywordContaining) {
			return nil, unsupported("containing after a token query (only <structure/> containing ... is supported)")
		}
		if err := t.addWithin(parts, wc); err != nil {
			return nil, err
		}
	}
	return entities, nil
}

// assemble creates the final list of query items. Tokens are made part
// of all the structures below the segment.
func (parts *queryParts) assemble() []QueryItem {
	for _, st := range parts.below {
		for _, item := range parts.tokenParts {
			forEachUnit(item, func(u *Unit) {
				u.PartOf = append(u.PartOf, PartOf{Kind: PartOfStream, Label: st.Label})
			})
		}
	}
	var ans []QueryItem
	for _, u := range parts.above {
		ans = append(ans, QueryItem{Unit: u})
	}
	ans = append(ans, QueryItem{Unit: parts.segment})
	for _, u := range parts.below {
		ans = append(ans, QueryItem{Unit: u})
	}
	return append(ans, parts.tokenParts...)
}

// forEachUnit calls fn for each unit within a query item (recursively).
func forEachUnit(item QueryItem, fn func(u *Unit)) {
	switch {
	case item.Unit != nil:
		fn(item.Unit)
	case item.Sequence != nil:
		forEachSequenceUnit(item.Sequence, fn)
	case item.Quantification != nil:
		forEachUnit(QueryItem{Unit: item.Quantification.Unit, Sequence: item.Quantification.Sequence}, fn)
	}
}

func forEachSequenceUnit(seq *Sequence, fn func(u *Unit)) {
	for _, m := range seq.Members {
		switch {
		case m.Unit != nil:
			fn(m.Unit)
		case m.Sequence != nil:
			forEachSequenceUnit(m.Sequence, fn)
		}
	}
}
