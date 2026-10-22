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

	"github.com/czcorpus/cqlizer/cql"
	"github.com/czcorpus/cqlizer/eval/feats"
	"github.com/czcorpus/cqlizer/tagsets"
	"github.com/czcorpus/rexplorer/parser"
	"github.com/stretchr/testify/assert"
)

const testUDRegistry = `NAME "Test UD corpus"
ATTRIBUTE word
ATTRIBUTE lemma
ATTRIBUTE upos
ATTRIBUTE feats
ATTRIBUTE deprel
ATTRIBUTE tag
`

const testUDMultiValueRegistry = `NAME "Test UD corpus with multivalue feats"
ATTRIBUTE word
ATTRIBUTE upos {
	MULTIVALUE y
	MULTISEP "|"
}
ATTRIBUTE feats {
	MULTIVALUE y
	MULTISEP "|"
}
`

const testCNCRegistry = `NAME "Test CNC corpus"
ATTRIBUTE word
ATTRIBUTE lemma
ATTRIBUTE tag
ATTRIBUTE ep_tag {
	MULTIVALUE y
	MULTISEP "|"
}
`

func testRegistryIssues(t *testing.T, registry, q string, props feats.CorpusProps) []tagsets.Issue {
	reg, err := parser.ParseRegistryBytes("test", []byte(registry))
	assert.NoError(t, err)
	query, err := cql.ParseCQL("", q)
	assert.NoError(t, err)
	return checkTagsets("test", query.ExtractProps(), props, reg)
}

func testTagsetIssues(t *testing.T, q string, props feats.CorpusProps) []tagsets.Issue {
	return testRegistryIssues(t, testUDRegistry, q, props)
}

func TestCheckTagsetsOK(t *testing.T) {
	issues := testTagsetIssues(
		t,
		`[upos="NOUN" & feats=".*Case=Acc.*Number=Plur.*"] [lemma="hrad"]`,
		feats.CorpusProps{Lang: "cs"},
	)
	assert.Empty(t, issues)
	assert.False(t, hasTagsetErrors(issues))
}

func TestCheckTagsetsErrors(t *testing.T) {
	issues := testTagsetIssues(
		t,
		`[upos="noun" & feats=".*Number=Plur.*Case=Acc.*"] [upos="noun"]`,
		feats.CorpusProps{Lang: "cs"},
	)
	// the repeated `upos="noun"` is reported only once
	assert.Len(t, issues, 2)
	assert.True(t, hasTagsetErrors(issues))
	assert.Equal(t, "upos", issues[0].Attr)
	assert.Equal(t, "feats", issues[1].Attr)
}

func TestCheckTagsetsConfig(t *testing.T) {
	q := `[upos="noun" & tag="whatever"]`
	issues := testTagsetIssues(t, q, feats.CorpusProps{Tagsets: map[string]string{"upos": "none"}})
	assert.Empty(t, issues)

	// an invalid scheme in config is logged and the attribute skipped
	issues = testTagsetIssues(t, q, feats.CorpusProps{Tagsets: map[string]string{"tag": "foo"}})
	assert.Len(t, issues, 1)
	assert.Equal(t, "upos", issues[0].Attr)
}

func TestMultiValueSep(t *testing.T) {
	reg, err := parser.ParseRegistryBytes("test", []byte(testUDMultiValueRegistry))
	assert.NoError(t, err)
	for _, pa := range reg.PosAttrs {
		sep, ok := multiValueSep(pa.Entries)
		if pa.Name == "word" {
			assert.False(t, ok)

		} else {
			assert.True(t, ok, "attr: %s", pa.Name)
			assert.Equal(t, "|", sep)
		}
	}
}

func TestCheckTagsetsMultiValueFeats(t *testing.T) {
	props := feats.CorpusProps{Lang: "en"}
	// matches individual features in a multivalue attribute
	assert.Empty(t, testRegistryIssues(t, testUDMultiValueRegistry, `[feats="VerbForm=Inf"]`, props))
	// in a single-value attribute, this matches only tokens with exactly this feature
	issues := testRegistryIssues(t, testUDRegistry, `[feats="VerbForm=Inf"]`, props)
	assert.Len(t, issues, 1)
	assert.Equal(t, tagsets.SeverityWarning, issues[0].Severity)
}

func TestCheckTagsetsPositional(t *testing.T) {
	props := feats.CorpusProps{Lang: "cs", Tagsets: map[string]string{"tag": "cs_cnc2020"}}
	assert.Empty(t, testRegistryIssues(t, testCNCRegistry, `[tag="N...4.*" & ep_tag="V.*"]`, props))
	issues := testRegistryIssues(t, testCNCRegistry, `[tag="N...4"] [ep_tag="Nf.*"]`, props)
	if assert.Len(t, issues, 2) {
		assert.Equal(t, "tag", issues[0].Attr)
		assert.Equal(t, "ep_tag", issues[1].Attr)
		assert.True(t, hasTagsetErrors(issues))
	}
	// without configuration, positional tags are not validated
	assert.Empty(t, testRegistryIssues(t, testCNCRegistry, `[tag="N...4"]`, feats.CorpusProps{Lang: "cs"}))
}
