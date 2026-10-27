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
	// regexps (anchored - DQD regexps match substrings, CQL ones the whole value)
	{`[word="pes.*"]`, singleTokenDQD("    form = /^(pes.*)$/\n")},
	{`"pes.*"`, singleTokenDQD("    form = /^(pes.*)$/\n")},
	{`[word="pes|kočka"]`, singleTokenDQD("    form = /^(pes|kočka)$/\n")},
	{`[word="pes|"]`, singleTokenDQD("    form = /^(pes|)$/\n")},
	{`[word!="p.*"]`, singleTokenDQD("    form != /^(p.*)$/\n")},
	{`[word="(?i)pes"]`, singleTokenDQD("    form = /^(pes)$/i\n")},
	{`[word="(?i)p.*"]`, singleTokenDQD("    form = /^(p.*)$/i\n")},
	{`[word="[[:alpha:]]+"]`, singleTokenDQD("    form = /^([[:alpha:]]+)$/\n")},
	{`[word="(?!a)."]`, singleTokenDQD("    form = /^((?!a).)$/\n")},
	{`[word="a/b.*"]`, singleTokenDQD("    form = /^(a/b.*)$/\n")},
	{`[tag="N.{4}1.*"]`, singleTokenDQD("    xpos = /^(N.{4}1.*)$/\n")},
	// escaped characters in regexps
	{`[word="\."]`, singleTokenDQD("    form = \".\"\n")},
	{`[word="a/b"]`, singleTokenDQD("    form = \"a/b\"\n")},
	{`[word="a\"b"]`, singleTokenDQD("    form = /^(a\\\"b)$/\n")},
	// literals (==)
	{`[lemma=="a\.b"]`, singleTokenDQD("    lemma = \"a.b\"\n")},
	{`[lemma=="a\"b"]`, singleTokenDQD("    lemma = /^a\"b$/\n")},
	{`[lemma=="a\\b\.c"]`, singleTokenDQD("    lemma = /^a\\\\b\\.c$/\n")},
	// boolean logic inside a position
	{`[lemma="pes" & tag="N.*"]`, singleTokenDQD(`    lemma = "pes"
    xpos = /^(N.*)$/
`)},
	{`[lemma="pes" | lemma="kočka"]`, singleTokenDQD(`    OR
        lemma = "pes"
        lemma = "kočka"
`)},
	{`[(lemma="pes" | lemma="kočka") & tag="N.*"]`, singleTokenDQD(`    OR
        lemma = "pes"
        lemma = "kočka"
    xpos = /^(N.*)$/
`)},
	{`[lemma="pes" & tag="N.*" | word="kočka"]`, singleTokenDQD(`    OR
        AND
            lemma = "pes"
            xpos = /^(N.*)$/
        form = "kočka"
`)},
	{`[!lemma="pes"]`, singleTokenDQD("    lemma != \"pes\"\n")},
	{`[!lemma!="pes"]`, singleTokenDQD("    lemma = \"pes\"\n")},
	{`[!(lemma="pes" | lemma="kočka")]`, singleTokenDQD(`    !
        OR
            lemma = "pes"
            lemma = "kočka"
`)},
	{`[!(lemma="pes" & tag="N.*")]`, singleTokenDQD(`    !
        AND
            lemma = "pes"
            xpos = /^(N.*)$/
`)},
	{`[(lemma="pes")]`, singleTokenDQD("    lemma = \"pes\"\n")},
	{`[((lemma="a" & tag="b") & word="c")]`, singleTokenDQD(`    lemma = "a"
    xpos = "b"
    form = "c"
`)},
	{`[word="a" | (lemma="b" | lemma="c")]`, singleTokenDQD(`    OR
        form = "a"
        OR
            lemma = "b"
            lemma = "c"
`)},
	// sequences
	{`[lemma="dobrý"] [tag="N.*"]`, `Segment s
sequence seq1
    Token@s t1
        lemma = "dobrý"
    Token@s t2
        xpos = /^(N.*)$/

res => plain
    context
        s
    entities
        t1
        t2
`},
	{`"velmi" [] [lemma="pes" | lemma="kočka"] [word!="a"]`, `Segment s
sequence seq1
    Token@s t1
        form = "velmi"
    Token@s t2
    Token@s t3
        OR
            lemma = "pes"
            lemma = "kočka"
    Token@s t4
        form != "a"

res => plain
    context
        s
    entities
        t1
        t2
        t3
        t4
`},
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
		`[word="\p{Lu}"]`,                // unicode class
		`[word="a(?i)b"]`,                // non-leading inline flag
		`[word="a" & ws(1,2)]`,           // ws() condition
		`![word="a"][word="b"]`,          // negated sequence
		`[word="a"][word="b.*"]{2}`,      // repetition in a sequence
		`[word="a"][foo="b"]`,            // unknown attribute in a sequence
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
