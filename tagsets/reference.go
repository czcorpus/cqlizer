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

package tagsets

import (
	"fmt"
	"slices"
	"strings"
)

// udUPOSLabels describe UPOS values (from the Alpha project)
var udUPOSLabels = map[string]string{
	"NOUN": "common noun", "PROPN": "proper noun", "VERB": "verb", "AUX": "auxiliary verb",
	"ADJ": "adjective", "ADV": "adverb", "PRON": "pronoun", "DET": "determiner", "NUM": "numeral",
	"ADP": "adposition", "CCONJ": "coordinating conjunction", "SCONJ": "subordinating conjunction",
	"PART": "particle", "INTJ": "interjection", "PUNCT": "punctuation", "SYM": "symbol", "X": "other",
}

// udDeprelLabels describe universal dependency relations (from the Alpha project)
var udDeprelLabels = map[string]string{
	"nsubj": "nominal subject", "csubj": "clausal subject", "obj": "object", "iobj": "indirect object",
	"ccomp": "clausal complement", "xcomp": "open clausal complement", "obl": "oblique nominal",
	"vocative": "vocative", "expl": "expletive", "dislocated": "dislocated element",
	"advcl": "adverbial clause modifier", "advmod": "adverbial modifier", "discourse": "discourse element",
	"aux": "auxiliary", "cop": "copula", "mark": "marker", "nmod": "nominal modifier",
	"appos": "appositional modifier", "nummod": "numeric modifier", "acl": "clausal modifier of a noun",
	"amod": "adjectival modifier", "det": "determiner", "clf": "classifier", "case": "case marker",
	"conj": "conjunct", "cc": "coordinating conjunction", "fixed": "part of a fixed multiword expression",
	"flat": "part of a flat multiword expression", "compound": "compound", "list": "list item",
	"parataxis": "parataxis", "orphan": "orphan", "goeswith": "part of a split word",
	"reparandum": "overridden disfluency", "punct": "punctuation", "root": "root of the sentence",
	"dep": "unspecified dependent",
}

// Describe returns a Markdown reference of the tagset used by a corpus
// attribute. For positional tagsets, the description can be limited to
// a single position (1-based; 0 = all positions).
func Describe(attr string, position int, cc CorpusContext) (string, error) {
	scheme, err := SchemeFor(attr, cc)
	if err != nil {
		return "", err
	}
	var buff strings.Builder
	switch s := scheme.(type) {
	case *positionalScheme:
		if err := describePositional(&buff, attr, s.ts, position); err != nil {
			return "", err
		}
	case *udUPOS:
		describeUPOS(&buff, attr)
	case *udFeats:
		describeFeats(&buff, attr, cc)
	case *udDeprel:
		describeDeprel(&buff, attr, s)
	case *udSingleFeat:
		fmt.Fprintf(&buff, "# Attribute `%s`: values of the UD feature %s\n\n", attr, s.name)
		fmt.Fprintf(&buff, "Values: %s\n", strings.Join(s.set.values, ", "))
	case nil:
		return "", fmt.Errorf("attribute %s has no known tagset", attr)
	default:
		return "", fmt.Errorf("no description available for %s", scheme.ID())
	}
	return buff.String(), nil
}

func describePositional(buff *strings.Builder, attr string, ts *positionalTagset, position int) error {
	if position < 0 || position > ts.numPos {
		return fmt.Errorf("invalid position %d, the %s tagset has %d positions", position, ts.ident, ts.numPos)
	}
	fmt.Fprintf(buff, "# Attribute `%s`: positional tagset %s\n\n", attr, ts.ident)
	fmt.Fprintf(buff,
		"Each tag has exactly %d characters, each position encodes one category (`-` = not applicable). "+
			"A pattern must match the whole tag: use `.` for each position without a condition and end "+
			"the pattern with `.*` if the remaining positions are not specified (e.g. `%s`). "+
			"Do not use `.*` in the middle of a pattern, as it makes the positions of the following characters arbitrary. "+
			"Only values occurring in the corpus data are listed.\n\n",
		ts.numPos, ts.exampleTag())
	if position == 0 {
		fmt.Fprintf(buff, "Positions: ")
		labels := make([]string, ts.numPos)
		for i, p := range ts.positions {
			labels[i] = fmt.Sprintf("%d. %s", i+1, p.label)
		}
		buff.WriteString(strings.Join(labels, ", "))
		buff.WriteString("\n\n")
	}
	for i, p := range ts.positions {
		if position > 0 && i != position-1 {
			continue
		}
		if len(p.used) == 1 && p.used[0] == '-' {
			fmt.Fprintf(buff, "## Position %d: %s\n\nalways `-` (unused)\n\n", i+1, p.label)
			continue
		}
		fmt.Fprintf(buff, "## Position %d: %s\n\n", i+1, p.label)
		var withPOS map[rune][]rune
		if i == 1 {
			withPOS = ts.cooccurrence(0, 1)
		}
		for _, v := range p.used {
			fmt.Fprintf(buff, "- `%c`", v)
			if d := p.describe(v); d != "" {
				fmt.Fprintf(buff, " %s", d)
			}
			if pos := withPOS[v]; len(pos) > 0 {
				fmt.Fprintf(buff, " (with position 1: %s)", string(pos))
			}
			buff.WriteString("\n")
		}
		buff.WriteString("\n")
	}
	return nil
}

// cooccurrence returns, for each value at position `b`, values found
// at position `a` in the same tags
func (ts *positionalTagset) cooccurrence(a, b int) map[rune][]rune {
	ans := make(map[rune][]rune)
	for _, tag := range ts.tags {
		va, vb := rune(tag[a]), rune(tag[b])
		if !slices.Contains(ans[vb], va) {
			ans[vb] = append(ans[vb], va)
		}
	}
	for k := range ans {
		slices.Sort(ans[k])
	}
	return ans
}

// exampleTag returns an example pattern (nouns in accusative if possible)
func (ts *positionalTagset) exampleTag() string {
	for i, p := range ts.positions {
		if p.label == lblCase {
			return "N" + strings.Repeat(".", i-1) + "4.*"
		}
	}
	for _, tag := range ts.tags {
		if tag[0] != '-' {
			return tag[:2] + ".*"
		}
	}
	return ts.tags[0]
}

func describeUPOS(buff *strings.Builder, attr string) {
	fmt.Fprintf(buff, "# Attribute `%s`: Universal Dependencies part of speech (UPOS)\n\n", attr)
	for _, v := range udUPOSValues {
		fmt.Fprintf(buff, "- `%s` %s\n", v, udUPOSLabels[v])
	}
}

func describeFeats(buff *strings.Builder, attr string, cc CorpusContext) {
	fmt.Fprintf(buff, "# Attribute `%s`: Universal Dependencies features\n\n", attr)
	buff.WriteString("A value contains `Name=Value` pairs separated by `|` and sorted alphabetically " +
		"by name (case-insensitive), e.g. `Case=Acc|Gender=Fem|Number=Plur`. Multiple values of " +
		"a feature are separated by commas (`Gender=Fem,Neut`). Patterns must match the whole value, so " +
		"a literal `|` must be escaped (`\\|`) and patterns usually start and end with `.*`. ")
	if cc.MultiValueSeps[attr] == "|" {
		fmt.Fprintf(buff, "In this corpus, `%s` is a multivalue attribute, so a pattern also matches "+
			"individual features: use separate conditions like `%s=\"Case=Acc\" & %s=\"Number=Plur\"`, "+
			"which do not depend on the order of features.", attr, attr, attr)
	} else {
		fmt.Fprintf(buff, "Features required in one pattern must follow the alphabetical order, "+
			"e.g. `%s=\".*Case=Acc.*Number=Plur.*\"`.", attr)
	}
	buff.WriteString("\n\n")
	data := udData()
	feats, ok := data.feats[cc.Lang]
	if ok {
		fmt.Fprintf(buff, "Features used in UD for language `%s`:\n\n", cc.Lang)
	} else {
		feats = data.allFeats
		buff.WriteString("Features used in UD (all languages):\n\n")
	}
	for _, name := range sortedKeys(feats) {
		fmt.Fprintf(buff, "- %s: %s\n", name, listValues(feats[name]))
	}
}

func describeDeprel(buff *strings.Builder, attr string, s *udDeprel) {
	fmt.Fprintf(buff, "# Attribute `%s`: Universal Dependencies relations\n\n", attr)
	buff.WriteString("Relations with subtypes have the form `base:subtype` (e.g. `nsubj:pass`); " +
		"to match a relation including its subtypes, use e.g. `nsubj(:.*)?`.\n\n")
	values := s.set.values
	if s.set.lang != "" {
		fmt.Fprintf(buff, "Relations used in UD for language `%s`:\n\n", s.set.lang)

	} else {
		// subtypes of all languages would be too many
		buff.WriteString("Universal relations (subtypes are language-specific):\n\n")
		values = sortedKeys(udDeprelLabels)
	}
	for _, v := range values {
		base, _, _ := strings.Cut(v, ":")
		if d := udDeprelLabels[base]; d != "" && base == v {
			fmt.Fprintf(buff, "- `%s` %s\n", v, d)
		} else {
			fmt.Fprintf(buff, "- `%s`\n", v)
		}
	}
}
