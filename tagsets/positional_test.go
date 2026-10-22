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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func mustScheme(t *testing.T, id string) Scheme {
	s, err := NewScheme(id, "cs")
	assert.NoError(t, err)
	return s
}

func TestPositionalTagsetsLoaded(t *testing.T) {
	assert.Equal(t, []string{"cs_cnc2000", "cs_cnc2000_spk", "cs_cnc2020", "cs_cnc2020_spk", "verbtag"}, PositionalTagsets())
	ts := positionalTagsets()
	assert.Equal(t, 16, ts["cs_cnc2000"].numPos)
	assert.Equal(t, 15, ts["cs_cnc2020"].numPos)
	// declared as 16-positional in the KonText XML, but data have 15 positions
	assert.Equal(t, 15, ts["cs_cnc2000_spk"].numPos)
	assert.Equal(t, "Aspect", ts["cs_cnc2000"].positions[15].label)
	assert.Equal(t, "Aspect", ts["cs_cnc2020"].positions[12].label)
	for _, tag := range ts["cs_cnc2020"].tags {
		assert.NotContains(t, tag, "|")
	}
}

func TestPositionalCNC2020(t *testing.T) {
	runCases(t, mustScheme(t, "cs_cnc2020"), []validationCase{
		{value: "N.*"},
		{value: "NN.*"},
		{value: "N...4.*"},
		{value: "N...7.*"},
		{value: "NN[MI].4.*"},
		{value: "NNFP2.*"},
		{value: "RR--7.*"},
		{value: "V.........N.*"},
		{value: "V.{9}N.*"},
		{value: "V.......R.*"},
		{value: "Vf........N.*"},
		{value: "Vi.*"},
		{value: "VB.*"},
		{value: "[VZ].*"},
		{value: "Z.*|J.*"},
		{value: "....4.*"},
		{value: "B.*"},
		{value: "V...........P.*"},
		{value: "Vf--------A-I--", literal: true},
		{value: "N...4", severities: []Severity{SeverityError}, contains: "add `.*` to the end (e.g. `N...4.*`)"},
		{value: "Vf--------A-I---", severities: []Severity{SeverityError}, contains: "describes 16 positions"},
		{value: "Nf.*", severities: []Severity{SeverityError}, contains: "possible values there: `N` (noun, general)"},
		{value: "Vf..4.*", severities: []Severity{SeverityWarning}, contains: "position 5 (Case) together with the preceding positions `Vf..`"},
		{value: "Q.*", severities: []Severity{SeverityError}, contains: "position 1 (Part of speech) cannot be `Q`"},
		{value: "M.*", severities: []Severity{SeverityError}, contains: "cannot be `M`"},
		{value: "..X.*", severities: []Severity{SeverityError}, contains: "defined by the tagset, but it does not occur"},
		{value: "V............I.*", severities: []Severity{SeverityError}, contains: "position 14"},
		{value: "V.*N.*", severities: []Severity{SeverityWarning}, contains: "positions arbitrary"},
		{value: ".*4.*", severities: []Severity{SeverityWarning}, contains: "positions arbitrary"},
		{
			value:      "NOUN",
			severities: []Severity{SeverityError, SeverityHint},
			contains:   "describes 4 positions",
		},
	})
}

func TestPositionalCNC2000(t *testing.T) {
	runCases(t, mustScheme(t, "cs_cnc2000"), []validationCase{
		{value: "N.*"},
		{value: "V..............I"},
		{value: "V.*I"},
		{value: "NN------------8-"},
		{value: "B.*", severities: []Severity{SeverityError}, contains: "cannot be `B`"},
		// aspect is at the last position in this tagset
		{value: "V...........I.*", severities: []Severity{SeverityError}, contains: "position 13 (Reserved)"},
	})
}

func TestPositionalSpoken(t *testing.T) {
	runCases(t, mustScheme(t, "cs_cnc2000_spk"), []validationCase{
		{value: "M.*"},
		{value: "H.*"},
		{value: "VpYS---XR-AA---"},
		{value: "V..............I", severities: []Severity{SeverityError}, contains: "exactly 15 positions"},
	})
}

func TestVerbtag(t *testing.T) {
	ts := positionalTagsets()["verbtag"]
	assert.Equal(t, 6, ts.numPos)
	assert.Equal(t, "Mood", ts.positions[1].label)
	assert.Equal(t, []string{"syn2020", "syn_r25"}, ts.sources)
	runCases(t, mustScheme(t, "verbtag"), []validationCase{
		{value: "VDA3SR"},
		{value: "V.*"},
		{value: "VD.*R"},
		{value: "A-----"},
		{value: "------"},
		// used in SYN2020 only
		{value: "--p---"},
		{value: "V.A..[PF]"},
		{value: "VDA3S", severities: []Severity{SeverityError}, contains: "exactly 6 positions"},
		{value: "VX.*", severities: []Severity{SeverityError}, contains: "position 2 (Mood) cannot be `X`"},
		{value: "VFA3.*", severities: []Severity{SeverityWarning}, contains: "position 4 (Person)"},
		{value: "V..............I", severities: []Severity{SeverityError}, contains: "describes 16 positions"},
	})
}

func TestPositionalSpoken2020(t *testing.T) {
	ts := positionalTagsets()["cs_cnc2020_spk"]
	assert.Equal(t, 15, ts.numPos)
	// inherited from cs_cnc2020
	assert.Equal(t, "Aspect", ts.positions[12].label)
	assert.Equal(t, "noun", ts.positions[0].describe('N'))
	// added by the extension
	assert.Contains(t, ts.positions[0].describe('E'), "word fragment")
	assert.Contains(t, ts.positions[0].describe('W'), "spoken-specific")
	runCases(t, mustScheme(t, "cs_cnc2020_spk"), []validationCase{
		{value: "M.*"},
		{value: "H.*"},
		{value: "E.*"},
		{value: "W.*"},
		{value: "F%.*"},
		{value: "VB-S---3P-AAI--"},
		{value: "V............I.*", severities: []Severity{SeverityError}, contains: "position 14"},
		{value: "Q.*", severities: []Severity{SeverityError}, contains: "`E` (word fragment"},
	})
}

func TestDescChain(t *testing.T) {
	defs := map[string]*ktTagset{
		"a": {Ident: "a"},
		"b": {Ident: "b", Extends: "a"},
		"c": {Ident: "c", Extends: "c"},
	}
	chain := descChain("b", defs)
	if assert.Len(t, chain, 2) {
		assert.Equal(t, "a", chain[0].Ident)
		assert.Equal(t, "b", chain[1].Ident)
	}
	// self-reference does not loop
	assert.Len(t, descChain("c", defs), 1)
	assert.Empty(t, descChain("x", defs))
}

func TestPositionalSchemeFor(t *testing.T) {
	cc := CorpusContext{
		Lang:    "cs",
		Attrs:   []string{"word", "tag", "p_tag", "ep_tag", "pos"},
		Schemes: map[string]string{"tag": "cs_cnc2020"},
	}
	for _, attr := range []string{"tag", "p_tag", "ep_tag"} {
		s, err := SchemeFor(attr, cc)
		assert.NoError(t, err)
		if assert.NotNil(t, s, "attr: %s", attr) {
			assert.Equal(t, "cs_cnc2020", s.ID())
		}
	}
	s, err := SchemeFor("pos", cc)
	assert.NoError(t, err)
	assert.Nil(t, s)

	// a parent attribute can still be configured explicitly
	cc.Schemes["ep_tag"] = SchemeNone
	s, err = SchemeFor("ep_tag", cc)
	assert.NoError(t, err)
	assert.Nil(t, s)

	// no inference for positional tags
	s, err = SchemeFor("tag", CorpusContext{Attrs: []string{"tag"}})
	assert.NoError(t, err)
	assert.Nil(t, s)

	// ... except for verbtag which has a unique name and a single tagset
	s, err = SchemeFor("verbtag", CorpusContext{Attrs: []string{"tag", "verbtag"}})
	assert.NoError(t, err)
	if assert.NotNil(t, s) {
		assert.Equal(t, SchemeVerbtag, s.ID())
	}
}

func TestPositionalMessagesUseDots(t *testing.T) {
	issues := mustScheme(t, "cs_cnc2020").Validate(Value{Pattern: "Vf..4.*"})
	if assert.Len(t, issues, 1) {
		assert.False(t, strings.Contains(issues[0].Message, "(?-s:"), issues[0].Message)
	}
}
