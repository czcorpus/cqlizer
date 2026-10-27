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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// singleTokenDQD creates the expected output for a single token query
// with the default corpus props.
func singleTokenDQD(constraints string) string {
	return "Segment s\nToken@s t1\n" + constraints + `
res => plain
    context
        s
    entities
        t1
`
}

var translateCases = []struct {
	cql      string
	expected string
}{
	{`[lemma="pes"]`, singleTokenDQD("    lemma = \"pes\"\n")},
	{`[word!="pes"]`, singleTokenDQD("    form != \"pes\"\n")},
	{`[lemma=="pes"]`, singleTokenDQD("    lemma = \"pes\"\n")},
	{`[lemma!=="pes"]`, singleTokenDQD("    lemma != \"pes\"\n")},
	{`"pes"`, singleTokenDQD("    form = \"pes\"\n")},
	{`[tag="NNMS1-----A----"]`, singleTokenDQD("    xpos = \"NNMS1-----A----\"\n")},
	{`[word=""]`, singleTokenDQD("    form = \"\"\n")},
	{`[]`, singleTokenDQD("")},
}

func TestCQL2DQD(t *testing.T) {
	for _, tc := range translateCases {
		t.Run(tc.cql, func(t *testing.T) {
			ans, err := CQL2DQD(DefaultCorpusProps(), ViewOptions{}, tc.cql)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, ans)
		})
	}
}

func TestCQL2DQDSyntax(t *testing.T) {
	skipIfNoDQDCheck(t)
	for _, tc := range translateCases {
		t.Run(tc.cql, func(t *testing.T) {
			ans, err := CQL2DQD(DefaultCorpusProps(), ViewOptions{}, tc.cql)
			assert.NoError(t, err)
			checkDQDSyntax(t, ans)
		})
	}
}

func TestCQL2DQDCustomProps(t *testing.T) {
	cp := CorpusProps{
		PosAttrLayer: "Word",
		SegmentLayer: "Sentence",
		DefaultAttr:  "lc",
		PosAttrs:     map[string]string{"lc": "text_lc"},
	}
	ans, err := CQL2DQD(cp, ViewOptions{ResultLabel: "kwic"}, `"pes"`)
	assert.NoError(t, err)
	assert.Equal(t, `Sentence s
Word@s t1
    text_lc = "pes"

kwic => plain
    context
        s
    entities
        t1
`, ans)
}

func TestCQL2DQDUnsupported(t *testing.T) {
	queries := []string{
		`[word="p.*"]`,                   // regexp
		`[word="a|"]`,                    // regexp
		`[word="\."]`,                    // escaped char
		`[word="\p{Lu}"]`,                // unicode class
		`[word="a" & lemma="b"]`,         // AND inside a position
		`[word="a" | lemma="b"]`,         // OR inside a position
		`[word="a"][word="b"]`,           // more positions
		`[word="a"]{2}`,                  // repetition
		`[word="a"] within <s/>`,         // within
		`1:[word="a"] & 1.tag = 1.lemma`, // numbered position, global condition
		`[foo="a"]`,                      // attribute without mapping
		`[word<="a"]`,                    // other operator
		`<s>[word="a"]`,                  // structure tag in a sequence
		`[word="a"] | [word="b"]`,        // alternative sequences
		`([word="a"])`,                   // parenthesized sub-query
	}
	for _, q := range queries {
		t.Run(q, func(t *testing.T) {
			_, err := CQL2DQD(DefaultCorpusProps(), ViewOptions{}, q)
			var unsupported *UnsupportedFeature
			assert.ErrorAs(t, err, &unsupported)
		})
	}
}

func TestCQL2DQDPropagatesParseError(t *testing.T) {
	_, err := CQL2DQD(DefaultCorpusProps(), ViewOptions{}, `[word=`)
	assert.Error(t, err)
	var unsupported *UnsupportedFeature
	assert.NotErrorAs(t, err, &unsupported)
}

func TestLoadCorpusProps(t *testing.T) {
	path := filepath.Join(t.TempDir(), "props.json")
	err := os.WriteFile(path, []byte(`{"segmentLayer": "Sentence", "posAttrs": {"word": "text", "pos": "upos"}}`), 0o644)
	assert.NoError(t, err)
	cp, err := LoadCorpusProps(path)
	assert.NoError(t, err)
	assert.Equal(t, "Sentence", cp.SegmentLayer)
	assert.Equal(t, "Token", cp.PosAttrLayer) // default kept
	assert.Equal(t, "text", cp.PosAttrs["word"])
	assert.Equal(t, "upos", cp.PosAttrs["pos"])
	assert.Equal(t, "lemma", cp.PosAttrs["lemma"]) // default kept
}
