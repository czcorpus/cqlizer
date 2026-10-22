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
	"embed"
	"encoding/xml"
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"
)

// kontextTagsetsXML contains descriptions of positional tagsets
// in the format used by KonText
//
//go:embed data/kontext_tagsets.xml
var kontextTagsetsXML []byte

// cqlizerTagsetsXML contains descriptions of tagsets not covered
// by KonText (same format)
//
//go:embed data/cqlizer_tagsets.xml
var cqlizerTagsetsXML []byte

// positionalTagsFS contains lists of all the tags found in large
// reference corpora. The structure is `data/positional/{tagset}/{corpus}`,
// one tag per line. Multiple tags of a single token may be joined by `|`.
// To add a tagset, just add a respective directory.
//
//go:embed data/positional
var positionalTagsFS embed.FS

// ------------------- KonText XML -------------------

type ktDesc struct {
	Lang string `xml:"lang,attr"`
	Text string `xml:",chardata"`
}

type ktValue struct {
	Ident string   `xml:"ident,attr"`
	Descs []ktDesc `xml:"desc"`
}

type ktPosition struct {
	Index  int       `xml:"index,attr"`
	Labels []ktDesc  `xml:"label>desc"`
	Values []ktValue `xml:"value"`
}

type ktTagset struct {
	Ident string `xml:"ident,attr"`

	// Extends refers to a tagset whose descriptions are inherited
	// (a CQLizer extension of the KonText format)
	Extends string `xml:"extends,attr"`

	Positions []ktPosition `xml:"position"`
}

// descChain returns a tagset definition along with all the definitions
// it extends, starting from the most general one
func descChain(ident string, defs map[string]*ktTagset) []*ktTagset {
	var ans []*ktTagset
	seen := make(map[string]bool)
	for def := defs[ident]; def != nil && !seen[def.Ident]; def = defs[def.Extends] {
		seen[def.Ident] = true
		ans = append([]*ktTagset{def}, ans...)
	}
	return ans
}

type ktDocument struct {
	Tagsets []ktTagset `xml:"tagsets>tagset"`
}

func descByLang(descs []ktDesc, lang string) string {
	for _, d := range descs {
		if d.Lang == lang {
			return strings.Join(strings.Fields(d.Text), " ")
		}
	}
	return ""
}

// ------------------- tagset model -------------------

// tagPosition describes a single position of a positional tag
type tagPosition struct {
	label string

	// values contains descriptions of values (incl. ones not present in data)
	values map[rune]string

	// used contains values found in reference data (sorted)
	used []rune
}

func (tp tagPosition) describe(v rune) string {
	if d, ok := tp.values[v]; ok {
		return d
	}
	if v == '-' {
		return "not applicable"
	}
	return ""
}

// positionalTagset is a tagset where each tag is a fixed-length
// string and each character position encodes a single category
type positionalTagset struct {
	ident     string
	numPos    int
	positions []tagPosition

	// tags are all distinct single tags found in reference data
	tags []string

	// sources are reference corpora the tags come from
	sources []string
}

func loadPositionalTags(ident string) (tags []string, sources []string, err error) {
	dir := path.Join("data/positional", ident)
	entries, err := positionalTagsFS.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	uniq := make(map[string]bool)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := positionalTagsFS.ReadFile(path.Join(dir, e.Name()))
		if err != nil {
			return nil, nil, err
		}
		for _, line := range strings.Split(string(data), "\n") {
			for _, tag := range strings.Split(strings.TrimSpace(line), "|") {
				if tag != "" {
					uniq[tag] = true
				}
			}
		}
		sources = append(sources, e.Name())
	}
	return sortedKeys(uniq), sources, nil
}

// newPositionalTagset combines reference tags with descriptions
// from the XML definitions (applied in the order of the chain, so more
// specific definitions override general ones). The number of positions is determined from
// the reference data as it is more reliable (e.g. cs_cnc2000_spk is
// declared as 16-positional while ORTOFON data have 15 positions).
func newPositionalTagset(ident string, xmlChain []*ktTagset) (*positionalTagset, error) {
	tags, sources, err := loadPositionalTags(ident)
	if err != nil {
		return nil, fmt.Errorf("failed to load tags of %s: %w", ident, err)
	}
	if len(tags) == 0 {
		return nil, fmt.Errorf("no tags found for tagset %s", ident)
	}
	ans := &positionalTagset{
		ident:   ident,
		numPos:  len([]rune(tags[0])),
		tags:    tags,
		sources: sources,
	}
	used := make([]map[rune]bool, ans.numPos)
	for i := range used {
		used[i] = make(map[rune]bool)
	}
	for _, tag := range tags {
		rtag := []rune(tag)
		if len(rtag) != ans.numPos {
			return nil, fmt.Errorf("tagset %s: tag %s has %d positions, expected %d", ident, tag, len(rtag), ans.numPos)
		}
		if len(tag) != len(rtag) {
			// validation indexes tags by bytes
			return nil, fmt.Errorf("tagset %s: tag %s contains non-ASCII characters", ident, tag)
		}
		for i, r := range rtag {
			used[i][r] = true
		}
	}
	ans.positions = make([]tagPosition, ans.numPos)
	for i := range ans.positions {
		pos := &ans.positions[i]
		pos.label = fmt.Sprintf("position %d", i+1)
		pos.values = make(map[rune]string)
		for r := range used[i] {
			pos.used = append(pos.used, r)
		}
		sort.Slice(pos.used, func(a, b int) bool { return pos.used[a] < pos.used[b] })
	}
	for _, xmlDef := range xmlChain {
		for _, xp := range xmlDef.Positions {
			if xp.Index < 0 || xp.Index >= ans.numPos {
				continue
			}
			pos := &ans.positions[xp.Index]
			if label := descByLang(xp.Labels, "en"); label != "" {
				pos.label = label
			}
			for _, xv := range xp.Values {
				rv := []rune(xv.Ident)
				if len(rv) == 1 {
					pos.values[rv[0]] = descByLang(xv.Descs, "en")
				}
			}
		}
	}
	return ans, nil
}

var positionalTagsets = sync.OnceValue(func() map[string]*positionalTagset {
	xmlDefs := make(map[string]*ktTagset)
	for _, src := range [][]byte{kontextTagsetsXML, cqlizerTagsetsXML} {
		var doc ktDocument
		if err := xml.Unmarshal(src, &doc); err != nil {
			panic("failed to parse embedded tagsets XML: " + err.Error())
		}
		for i, ts := range doc.Tagsets {
			xmlDefs[ts.Ident] = &doc.Tagsets[i]
		}
	}
	entries, err := positionalTagsFS.ReadDir("data/positional")
	if err != nil {
		panic("failed to read embedded positional tags: " + err.Error())
	}
	ans := make(map[string]*positionalTagset)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		ident := e.Name()
		ts, err := newPositionalTagset(ident, descChain(ident, xmlDefs))
		if err != nil {
			panic("failed to load embedded positional tagset: " + err.Error())
		}
		ans[ident] = ts
	}
	return ans
})
