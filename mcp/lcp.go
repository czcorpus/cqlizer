// Copyright 2026 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2026 Department of Linguistics,
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

package mcp

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/czcorpus/cqlizer/cql"
	"github.com/czcorpus/cqlizer/dqd"
)

// Support for LCP corpora - CQL queries written for them are translated to
// DQD (see the dqd package), so the corpora are described exactly the way
// the translation maps CQL attributes and structures to DQD ones.

// maxListedValues limits the number of values listed in a validation
// message (all the values are available via get_corpus_structure)
const maxListedValues = 30

// rgSpecialChars are characters making a CQL value a regular expression
const rgSpecialChars = `.[]()*+?{}|^$\`

// lcpTagsetToolsMsg is returned by the tagset tools for LCP corpora
const lcpTagsetToolsMsg = "%s is an LCP corpus - tagset tools are not available for it. " +
	"Use the attribute values listed by get_corpus_structure instead."

// lcpAttrValues returns all the possible values of an attribute (if known).
func lcpAttrValues(li *dqd.LayerInfo, la *dqd.LayerAttr) []string {
	if la == nil {
		return nil
	}
	if len(la.Values) > 0 {
		return la.Values
	}
	if la.IsGlobal {
		return li.GlobalAttrValues[la.Name]
	}
	return nil
}

// lcpLayerAttr finds a (regular or meta) attribute of a layer
func lcpLayerAttr(li *dqd.LayerInfo, layer, attr string) *dqd.LayerAttr {
	ld, ok := li.Layers[layer]
	if !ok {
		return nil
	}
	if la, ok := ld.Attributes[attr]; ok {
		return la
	}
	return ld.Meta[attr]
}

// lcpCorpusInfo describes an LCP corpus in terms of CQL
func lcpCorpusInfo(corpusID string, li *dqd.LayerInfo) (*corpusInfo, error) {
	cp, err := dqd.CorpusPropsFromLayerInfo(li)
	if err != nil {
		return nil, fmt.Errorf("invalid layer info of LCP corpus %s: %w", corpusID, err)
	}
	ans := &corpusInfo{
		Name: corpusID,
		Notes: []string{
			"This is an LCP corpus - CQL queries are translated to DQD, so only the attributes " +
				"and structures listed here can be used (with exactly these names).",
			fmt.Sprintf(
				"A match cannot cross the boundary of a <%s> structure (the segment layer of the corpus).",
				cp.SegmentLayer),
			"Structures can be used via `within` (e.g. `[...] within <Name attr=\"value\"/>`) or " +
				"`<Name/> containing [...]`. Negated `!within` is not supported.",
			"Use cql_validate to check a query - it also checks the query can be translated.",
		},
		PositionalAttributes: make([]attr, 0, len(cp.PosAttrs)),
		Structures:           make([]structure, 0, len(cp.Structures)),
	}
	for _, name := range slices.Sorted(maps.Keys(cp.PosAttrs)) {
		dqdName := cp.PosAttrs[name]
		la := lcpLayerAttr(li, cp.PosAttrLayer, dqdName)
		a := attr{Name: name, Values: lcpAttrValues(li, la)}
		if dqdName != name {
			a.Label = fmt.Sprintf("an alias of `%s`", dqdName)

		} else if la != nil {
			a.Label = la.DisplayName
		}
		ans.PositionalAttributes = append(ans.PositionalAttributes, a)
	}
	for _, name := range slices.Sorted(maps.Keys(cp.Structures)) {
		sp := cp.Structures[name]
		st := structure{Name: name, Attributes: make([]attr, 0, len(sp.Attrs))}
		switch {
		case sp.Layer == cp.SegmentLayer:
			st.Note = "the segment"
		case sp.ContainsSegments:
			st.Note = "contains segments"
		default:
			st.Note = "inside segments"
		}
		for _, aname := range slices.Sorted(maps.Keys(sp.Attrs)) {
			la := lcpLayerAttr(li, sp.Layer, sp.Attrs[aname])
			st.Attributes = append(st.Attributes, attr{Name: aname, Values: lcpAttrValues(li, la)})
		}
		ans.Structures = append(ans.Structures, st)
	}
	return ans, nil
}

// isPlainValue tells whether a CQL value matches just itself
func isPlainValue(v string) bool {
	return !strings.ContainsAny(v, rgSpecialChars)
}

// validateLCPQuery checks that a CQL query can be translated to DQD
// for an LCP corpus and that plain values of attributes with known
// values are valid. It returns a list of issues (empty if the query is OK).
func validateLCPQuery(corpusID string, li *dqd.LayerInfo, query string) ([]string, error) {
	cp, err := dqd.CorpusPropsFromLayerInfo(li)
	if err != nil {
		return nil, fmt.Errorf("invalid layer info of LCP corpus %s: %w", corpusID, err)
	}
	q, err := cql.ParseCQL("", query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}
	// unknown names are reported the same way as for Manatee corpora
	// (the translation would report them too, but in its own terms)
	for _, prop := range q.ExtractProps() {
		switch {
		case prop.IsPosattr() && prop.Name != "":
			if _, ok := cp.PosAttrs[prop.Name]; !ok {
				return []string{fmt.Sprintf("unsupported positional attribute `%s`", prop.Name)}, nil
			}
		case prop.IsStructure():
			if _, ok := cp.Structures[prop.Structure]; !ok {
				return []string{fmt.Sprintf("unsupported structure <%s>", prop.Structure)}, nil
			}
		case prop.IsStructAttr():
			if _, ok := cp.Structures[prop.Structure].Attrs[prop.Name]; !ok {
				return []string{fmt.Sprintf("unsupported structural attribute <%s %s=\"...\">", prop.Structure, prop.Name)}, nil
			}
		}
	}
	if _, err := dqd.CQL2DQD(cp, dqd.ViewOptions{}, query); err != nil {
		var uf *dqd.UnsupportedFeature
		if errors.As(err, &uf) {
			return []string{fmt.Sprintf("The query cannot be used with LCP corpus %s: %s", corpusID, uf.Feature)}, nil
		}
		return nil, err
	}
	var issues []string
	for _, prop := range q.ExtractProps() {
		if prop.Name == "" || !isPlainValue(prop.Value) {
			continue
		}
		var la *dqd.LayerAttr
		var attrDesc string
		switch {
		case prop.IsPosattr():
			la = lcpLayerAttr(li, cp.PosAttrLayer, cp.PosAttrs[prop.Name])
			attrDesc = fmt.Sprintf("attribute `%s`", prop.Name)
		case prop.IsStructAttr():
			sp := cp.Structures[prop.Structure]
			la = lcpLayerAttr(li, sp.Layer, sp.Attrs[prop.Name])
			attrDesc = fmt.Sprintf("attribute `%s` of structure <%s>", prop.Name, prop.Structure)
		}
		values := lcpAttrValues(li, la)
		if len(values) == 0 || slices.Contains(values, prop.Value) {
			continue
		}
		issue := fmt.Sprintf("`%s` is not a valid value of %s", prop.Value, attrDesc)
		if len(values) <= maxListedValues {
			issue += fmt.Sprintf(" (possible values: %s)", formatValues(values))

		} else {
			issue += " (see get_corpus_structure for possible values)"
		}
		issues = append(issues, issue)
	}
	return issues, nil
}
