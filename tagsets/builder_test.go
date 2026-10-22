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

func cncContext(tagset string) CorpusContext {
	return CorpusContext{
		Lang:    "cs",
		Attrs:   []string{"word", "lemma", "tag", "verbtag"},
		Schemes: map[string]string{"tag": tagset},
	}
}

func severities(issues []Issue) []Severity {
	ans := make([]Severity, len(issues))
	for i, iss := range issues {
		ans[i] = iss.Severity
	}
	return ans
}

func TestParseMorphology(t *testing.T) {
	m, err := ParseMorphology("noun|PROPN", "Case=Acc,Gen|Number=Plur")
	assert.NoError(t, err)
	assert.Equal(t, []string{"NOUN", "PROPN"}, m.UPOS)
	assert.Equal(t, map[string][]string{"Case": {"Acc", "Gen"}, "Number": {"Plur"}}, m.Feats)

	_, err = ParseMorphology("", "Case")
	assert.Error(t, err)
	_, err = ParseMorphology("", "")
	assert.Error(t, err)
}

func TestBuildCondition(t *testing.T) {
	udMulti := CorpusContext{
		Lang:           "cs",
		Attrs:          []string{"word", "upos", "feats"},
		MultiValueSeps: map[string]string{"feats": "|"},
	}
	udSingle := CorpusContext{Lang: "cs", Attrs: []string{"word", "upos", "feats"}}
	for _, c := range []struct {
		upos, feats, attr string
		cc                CorpusContext
		cql               string
		severities        []Severity
	}{
		{"NOUN", "Case=Acc|Number=Plur", "", cncContext("cs_cnc2020"), `tag="N..P4.*"`, nil},
		{"NOUN|PROPN", "Case=Acc,Gen|Gender=Masc|Animacy=Anim", "", cncContext("cs_cnc2020"), `tag="N.M.[24].*"`, nil},
		{"NOUN", "Gender=Masc", "", cncContext("cs_cnc2020"), `tag="N.[IM].*"`, nil},
		{"PROPN", "", "", cncContext("cs_cnc2020"), `tag="N.*"`, []Severity{SeverityHint}},
		{"VERB", "VerbForm=Inf|Polarity=Neg", "", cncContext("cs_cnc2020"), `tag="Vf........N.*"`, nil},
		{"VERB", "VerbForm=Inf|Aspect=Perf", "", cncContext("cs_cnc2020"), `tag="Vf..........P.*"`, nil},
		{"VERB", "VerbForm=Inf|Aspect=Perf", "", cncContext("cs_cnc2000"), `tag="Vf.............P"`, nil},
		{"VERB", "VerbForm=Part|Voice=Pass", "", cncContext("cs_cnc2020"), `tag="Vs.*"`, nil},
		{"VERB", "VerbForm=Conv|Tense=Past", "", cncContext("cs_cnc2020"), `tag="Vm.*"`, nil},
		{"VERB", "Mood=Cnd|Person=1", "", cncContext("cs_cnc2020"), `tag="Vc.....1.*"`, nil},
		{"VERB", "VerbForm=Inf|Case=Acc", "", cncContext("cs_cnc2020"), `tag="Vf..4.*"`, []Severity{SeverityWarning}},
		{"VERB", "Mood=Ind", "", cncContext("cs_cnc2020"), `tag="V.*"`, []Severity{SeverityWarning}},
		{"PRON", "PronType=Prs|Reflex=Yes", "", cncContext("cs_cnc2020"), `tag="P[678].*"`, nil},
		{"PRON", "PronType=Dem", "", cncContext("cs_cnc2020"), `tag="PD.*"`, nil},
		{"NUM", "NumType=Ord", "", cncContext("cs_cnc2020"), `tag="Cr.*"`, nil},
		{"NOUN", "Abbr=Yes", "", cncContext("cs_cnc2020"), `tag="BN.*"`, nil},
		{"NOUN", "Abbr=Yes", "", cncContext("cs_cnc2000"), `tag="N.............8.*"`, nil},
		{"CCONJ", "", "", cncContext("cs_cnc2020"), `tag="J\^.*"`, nil},
		{"", "Foreign=Yes", "", cncContext("cs_cnc2020"), `tag="F.*"`, nil},
		{"NOUN", "Case=Acc|Foo=Bar", "", cncContext("cs_cnc2020"), `tag="N...4.*"`, []Severity{SeverityWarning}},
		{"NOUN", "Polite=Form", "", cncContext("cs_cnc2020"), `tag="N.*"`, []Severity{SeverityWarning}},
		// verbtag
		{"VERB", "Mood=Ind|Tense=Past|Voice=Pass", "verbtag", cncContext("cs_cnc2020"), `verbtag="VDP..R"`, nil},
		{"VERB", "Mood=Ind|Person=2|Polite=Form", "verbtag", cncContext("cs_cnc2020"), `verbtag="VD.2v.*"`, nil},
		{"AUX", "", "verbtag", cncContext("cs_cnc2020"), `verbtag="A.*"`, nil},
		{"VERB", "VerbForm=Inf", "verbtag", cncContext("cs_cnc2020"), `verbtag="VF.*"`, nil},
		// UD
		{"NOUN", "Number=Plur|Case=Acc", "", udMulti, `upos="NOUN" & feats="Case=Acc" & feats="Number=Plur"`, nil},
		{"NOUN", "Number=Plur|Case=Acc,Gen", "", udSingle, `upos="NOUN" & feats=".*Case=(Acc|Gen).*Number=Plur.*"`, nil},
		{"", "Gender[psor]=Fem", "", udSingle, `feats=".*Gender\[psor\]=Fem.*"`, nil},
		{"NOUN", "Case=Acc", "feats", udSingle, `feats=".*Case=Acc.*"`, []Severity{SeverityWarning}},
	} {
		m, err := ParseMorphology(c.upos, c.feats)
		assert.NoError(t, err)
		res, err := BuildCondition(m, c.attr, c.cc)
		assert.NoError(t, err)
		assert.Equal(t, c.cql, res.CQL, "input: %s %s", c.upos, c.feats)
		if len(c.severities) == 0 {
			assert.Empty(t, res.Issues, "input: %s %s", c.upos, c.feats)

		} else {
			assert.Equal(t, c.severities, severities(res.Issues), "input: %s %s, issues: %v", c.upos, c.feats, res.Issues)
		}
	}
}

func TestBuildConditionErrors(t *testing.T) {
	m, err := ParseMorphology("NOUN", "")
	assert.NoError(t, err)
	// positional tags are not inferred
	_, err = BuildCondition(m, "", CorpusContext{Attrs: []string{"word", "tag"}})
	assert.Error(t, err)
	_, err = BuildCondition(m, "lemma", cncContext("cs_cnc2020"))
	assert.Error(t, err)
}

func TestBuiltPatternsAreValid(t *testing.T) {
	// all values of each simple feature produce patterns matching some tags
	for _, ts := range []string{"cs_cnc2000", "cs_cnc2020"} {
		for name, target := range simpleFeatTargets {
			for v := range target.values {
				m := Morphology{Feats: map[string][]string{name: {v}}}
				res, err := BuildCondition(m, "", cncContext(ts))
				assert.NoError(t, err)
				for _, iss := range res.Issues {
					assert.NotEqual(t, SeverityError, iss.Severity, "%s %s=%s: %s", ts, name, v, iss.Message)
				}
			}
		}
	}
}

func TestDescribe(t *testing.T) {
	cnc := cncContext("cs_cnc2020")
	s, err := Describe("tag", 0, cnc)
	assert.NoError(t, err)
	assert.Contains(t, s, "positional tagset cs_cnc2020")
	assert.Contains(t, s, "exactly 15 characters")
	assert.Contains(t, s, "(e.g. `N...4.*`)")
	assert.Contains(t, s, "- `f` verb, infinitive (with position 1: V)")
	// the X gender is defined but not used in data
	assert.NotContains(t, s, "any gender")

	s, err = Describe("tag", 5, cnc)
	assert.NoError(t, err)
	assert.Contains(t, s, "## Position 5: Case")
	assert.NotContains(t, s, "## Position 1")
	_, err = Describe("tag", 16, cnc)
	assert.Error(t, err)

	s, err = Describe("verbtag", 0, cnc)
	assert.NoError(t, err)
	assert.Contains(t, s, "(e.g. `A-.*`)")
	assert.Contains(t, s, "- `v` polite form")

	_, err = Describe("lemma", 0, cnc)
	assert.Error(t, err)

	ud := CorpusContext{Lang: "cs", Attrs: []string{"upos", "feats", "deprel"}, MultiValueSeps: map[string]string{"feats": "|"}}
	s, err = Describe("feats", 0, ud)
	assert.NoError(t, err)
	assert.Contains(t, s, "multivalue attribute")
	assert.Contains(t, s, "- Case: Acc, Dat, Gen, Ins, Loc, Nom, Voc")
	s, err = Describe("upos", 0, ud)
	assert.NoError(t, err)
	assert.Contains(t, s, "- `PROPN` proper noun")
	s, err = Describe("deprel", 0, ud)
	assert.NoError(t, err)
	assert.Contains(t, s, "- `nsubj` nominal subject")
	assert.Contains(t, s, "- `nsubj:pass`")
	assert.True(t, strings.HasPrefix(s, "# Attribute `deprel`"))
}
