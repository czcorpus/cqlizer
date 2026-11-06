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
	"testing"

	"github.com/czcorpus/cqlizer/dqd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadWritten(t *testing.T) *dqd.LayerInfo {
	li, err := dqd.LoadLayerInfo("../dqd/testdata/layerinfo_written.json")
	require.NoError(t, err)
	return li
}

func TestLCPCorpusInfo(t *testing.T) {
	info, err := lcpCorpusInfo("written", loadWritten(t))
	require.NoError(t, err)
	assert.Equal(t, "written", info.Name)
	assert.NotEmpty(t, info.Notes)

	assert.Equal(t,
		[]attr{
			{Name: "form"},
			{Name: "lemma"},
			{Name: "upos", Values: []string{"ADJ", "ADP", "ADV", "NOUN", "VERB"}}, // global values
			{Name: "word", Label: "an alias of `form`"},
			{Name: "xpos", Label: "Sample tagset", Values: []string{"NN", "VB", "JJ"}},
		},
		info.PositionalAttributes)

	assert.Equal(t,
		[]structure{
			{Name: "Division", Note: "contains segments", Attributes: []attr{
				{Name: "type", Values: []string{"chapter", "section"}}}},
			{Name: "Document", Note: "contains segments", Attributes: []attr{
				{Name: "classCode"}, {Name: "title"}}},
			{Name: "NamedEntity", Note: "inside segments", Attributes: []attr{
				{Name: "type", Values: []string{"PER", "LOC", "ORG", "MISC"}}}},
			{Name: "Paragraph", Note: "contains segments", Attributes: []attr{}},
			{Name: "Segment", Note: "the segment", Attributes: []attr{}},
		},
		info.Structures)

	md := info.AsMarkdown()
	assert.Contains(t, md, "* This is an LCP corpus")
	assert.Contains(t, md, "### word\n\nlabel: an alias of `form`\n")
	assert.Contains(t, md, "values: `NN`, `VB`, `JJ`\n")
	assert.Contains(t, md, "* NamedEntity (inside segments)\n  * type (values: `PER`, `LOC`, `ORG`, `MISC`)\n")
}

func TestValidateLCPQuery(t *testing.T) {
	li := loadWritten(t)
	for _, q := range []string{
		`[lemma="dog" & upos="NOUN"]`,
		`[upos="N.*"]`,         // regexps are not checked against the values
		`[xpos!="JJ"] "barks"`, // a valid value, a bare value (default attribute)
		`"dog" within <NamedEntity type="PER"/>`,
		`<Document classCode="S.*"/> containing "dog"`,
	} {
		issues, err := validateLCPQuery("written", li, q)
		assert.NoError(t, err, q)
		assert.Empty(t, issues, q)
	}

	for q, expected := range map[string]string{
		`[tag="N.*"]`:                          "unsupported positional attribute `tag`",
		`"dog" within <Article/>`:              "unsupported structure <Article>",
		`"dog" within <Document foo="x"/>`:     "unsupported structural attribute <Document foo=\"...\">",
		`"dog" !within <Document/>`:            "negated within",
		`[upos="SUBST"]`:                       "`SUBST` is not a valid value of attribute `upos` (possible values: `ADJ`, `ADP`, `ADV`, `NOUN`, `VERB`)",
		`"dog" within <NamedEntity type="X"/>`: "`X` is not a valid value of attribute `type` of structure <NamedEntity>",
	} {
		issues, err := validateLCPQuery("written", li, q)
		assert.NoError(t, err, q)
		if assert.Len(t, issues, 1, q) {
			assert.Contains(t, issues[0], expected, q)
		}
	}

	_, err := validateLCPQuery("written", li, `[lemma=`)
	assert.Error(t, err)
}

func TestTranslateLCPQuery(t *testing.T) {
	li := loadWritten(t)
	ans, err := translateLCPQuery("written", li, `"dog" within <Document classCode="S.*"/>`)
	require.NoError(t, err)
	assert.Equal(t, `Document document1
    classCode = /^(S.*)$/
Segment@document1 s
Token@s t1
    form = "dog"

res => plain
    context
        s
    entities
        t1
`, ans)

	_, err = translateLCPQuery("written", li, `"dog" !within <Document/>`)
	var uf *dqd.UnsupportedFeature
	assert.ErrorAs(t, err, &uf)
}
