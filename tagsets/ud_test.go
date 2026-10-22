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
	"testing"

	"github.com/stretchr/testify/assert"
)

type validationCase struct {
	value    string
	literal  bool
	multiSep string

	// expected severities of found issues (in order)
	severities []Severity

	// substring expected in the first issue message (if non-empty)
	contains string
}

func runCases(t *testing.T, scheme Scheme, cases []validationCase) {
	for _, c := range cases {
		issues := scheme.Validate(Value{Pattern: c.value, Literal: c.literal, MultiSep: c.multiSep})
		sevs := make([]Severity, len(issues))
		for i, iss := range issues {
			sevs[i] = iss.Severity
		}
		if len(c.severities) == 0 {
			assert.Empty(t, issues, "value: %s", c.value)
			continue
		}
		assert.Equal(t, c.severities, sevs, "value: %s, issues: %v", c.value, issues)
		if c.contains != "" && len(issues) > 0 {
			assert.Contains(t, issues[0].Message, c.contains, "value: %s", c.value)
		}
	}
}

func TestUPOS(t *testing.T) {
	runCases(t, newUDUPOS(), []validationCase{
		{value: "NOUN"},
		{value: "NOUN|PROPN"},
		{value: "N.*"},
		{value: "(?i)noun"},
		{value: "noun", severities: []Severity{SeverityError}, contains: "did you mean `NOUN`"},
		{value: "NOUN|ADJECTIVE", severities: []Severity{SeverityError}, contains: "`ADJECTIVE`"},
		{value: "Q.*", severities: []Severity{SeverityError}, contains: "matches none"},
		{value: "NOUN", literal: true},
		{value: "N.*", literal: true, severities: []Severity{SeverityError}},
	})
}

func TestDeprel(t *testing.T) {
	runCases(t, newUDDeprel("cs"), []validationCase{
		{value: "nsubj", severities: []Severity{SeverityHint}, contains: "nsubj:pass"},
		{value: "nsubj(:.*)?"},
		{value: "nsubj.*"},
		{value: "obl:agent"},
		{value: "subj", severities: []Severity{SeverityWarning}, contains: "not a known"},
		{value: "NSUBJ:PASS", severities: []Severity{SeverityWarning}, contains: "did you mean `nsubj:pass`"},
	})
}

func TestFeats(t *testing.T) {
	runCases(t, newUDFeats("cs"), []validationCase{
		{value: ".*Case=Acc.*"},
		{value: ".*Case=Acc.*Number=Plur.*"},
		{value: ".*Case=Acc\\|.*Number=Plur.*"},
		{value: ".*Case=(Acc|Gen).*"},
		{value: ".*Case=A.*"},
		{value: ".*Case=.*"},
		{value: ".*Gender=Fem,Neut.*"},
		{value: ".*Gender\\[psor\\]=Fem.*"},
		{value: ".*Number=Sing.*NumType=Card.*"},
		{value: ".*"},
		{value: "_"},
		{value: "(?i).*case=acc.*"},
		{value: "Case=Acc|Number=Plur", literal: true},
		{value: ".*Number=Plur.*Case=Acc.*", severities: []Severity{SeverityError}, contains: "required order: Case, Number"},
		{value: ".*NumType=Card.*Number=Sing.*", severities: []Severity{SeverityError}, contains: "required order: Number, NumType"},
		{value: ".*Case=Acc.*Case=Gen.*", severities: []Severity{SeverityError}, contains: "twice"},
		{value: ".*Case=acc.*", severities: []Severity{SeverityError}, contains: "did you mean `Acc`"},
		{value: ".*case=Acc.*", severities: []Severity{SeverityError}, contains: "did you mean `Case`"},
		{value: ".*Gender[psor]=Fem.*", severities: []Severity{SeverityError}, contains: "Gender\\[psor\\]"},
		{value: ".*Case=Foo.*", severities: []Severity{SeverityWarning}, contains: "known values: Acc"},
		{value: ".*Case=(Acc|Foo).*", severities: []Severity{SeverityWarning}, contains: "`Foo`"},
		{value: ".*Clusivity=In.*", severities: []Severity{SeverityWarning}, contains: "not used for language `cs`"},
		{value: ".*Foo=Bar.*", severities: []Severity{SeverityWarning}, contains: "not a known UD feature"},
		{value: "Plur", severities: []Severity{SeverityWarning}, contains: "no `Name=Value`"},
		{value: "VerbForm=Inf", severities: []Severity{SeverityWarning}, contains: "exactly these features"},
		{value: "Animacy=Inan\\|Case=Gen.*", severities: []Severity{SeverityWarning}, contains: "first (alphabetically) feature is Animacy"},
		{value: ".*Voice=Act", severities: []Severity{SeverityWarning}, contains: "last (alphabetically) feature is Voice"},
		{value: "Case=Acc|Number=Plur", severities: []Severity{SeverityWarning}, contains: "\\|"},
	})
}

// TestFeatsMultiValue tests feats declared as MULTIVALUE with `|` separator
// where Manatee matches patterns also against individual features
func TestFeatsMultiValue(t *testing.T) {
	runCases(t, newUDFeats("cs"), []validationCase{
		{value: "VerbForm=Inf", multiSep: "|"},
		{value: "Case=Acc", multiSep: "|"},
		{value: "Case=A.*", multiSep: "|"},
		{value: "Case=(Acc|Gen)", multiSep: "|"},
		{value: "Case=Acc|Case=Gen", multiSep: "|"},
		{value: ".*Case=Acc.*Number=Plur.*", multiSep: "|"},
		{
			value: ".*Number=Plur.*Case=Acc.*", multiSep: "|",
			severities: []Severity{SeverityError},
			contains:   "`feats=\"Number=Plur\" & feats=\"Case=Acc\"`",
		},
		{
			value: "Case=Acc|Number=Plur", multiSep: "|",
			severities: []Severity{SeverityHint},
			contains:   "matches tokens having any of the features",
		},
		{
			value: "Case=Acc\\|Number=Plur", multiSep: "|",
			severities: []Severity{SeverityWarning},
			contains:   "separate conditions: `feats=\"Case=Acc\" & feats=\"Number=Plur\"`",
		},
		{value: "Case=Foo", multiSep: "|", severities: []Severity{SeverityWarning}},
		// other separators do not allow matching individual features
		{value: "VerbForm=Inf", multiSep: ",", severities: []Severity{SeverityWarning}},
	})
}

func TestFeatsUnknownLang(t *testing.T) {
	runCases(t, newUDFeats(""), []validationCase{
		{value: ".*Clusivity=In.*"},
		{value: ".*Case=Foo.*", severities: []Severity{SeverityWarning}},
	})
}

func TestSingleFeat(t *testing.T) {
	s, err := NewScheme("ud:feat:Case", "cs")
	assert.NoError(t, err)
	runCases(t, s, []validationCase{
		{value: "Gen"},
		{value: "Gen|Acc"},
		{value: "Acc,Gen"},
		{value: "gen", severities: []Severity{SeverityWarning}, contains: "did you mean `Gen`"},
		{value: "Ill", severities: []Severity{SeverityWarning}, contains: "not a known value of the UD feature Case for language `cs`"},
	})
	_, err = NewScheme("ud:feat:Foo", "cs")
	assert.Error(t, err)
}

func TestSchemeFor(t *testing.T) {
	udCorp := CorpusContext{
		Lang:  "cs",
		Attrs: []string{"word", "lemma", "upos", "feats", "deprel", "p_upos", "case_feats", "e_deprel", "case", "verbform", "tag"},
	}
	for attr, expected := range map[string]string{
		"upos":       SchemeUDUPOS,
		"p_upos":     SchemeUDUPOS,
		"feats":      SchemeUDFeats,
		"case_feats": SchemeUDFeats,
		"deprel":     SchemeUDDeprel,
		"e_deprel":   SchemeUDDeprel,
		"case":       "ud:feat:Case",
		"verbform":   "ud:feat:VerbForm",
		"word":       "",
		"tag":        "",
	} {
		s, err := SchemeFor(attr, udCorp)
		assert.NoError(t, err)
		if expected == "" {
			assert.Nil(t, s, "attr: %s", attr)

		} else if assert.NotNil(t, s, "attr: %s", attr) {
			assert.Equal(t, expected, s.ID(), "attr: %s", attr)
		}
	}

	// non-UD corpora get no inferred schemes
	cncCorp := CorpusContext{Lang: "cs", Attrs: []string{"word", "lemma", "tag", "case", "deprel"}}
	s, err := SchemeFor("case", cncCorp)
	assert.NoError(t, err)
	assert.Nil(t, s)

	// explicit configuration overrides inference
	cncCorp.Schemes = map[string]string{"deprel": SchemeUDDeprel}
	s, err = SchemeFor("deprel", cncCorp)
	assert.NoError(t, err)
	assert.Equal(t, SchemeUDDeprel, s.ID())
	udCorp.Schemes = map[string]string{"upos": SchemeNone}
	s, err = SchemeFor("upos", udCorp)
	assert.NoError(t, err)
	assert.Nil(t, s)
	udCorp.Schemes = map[string]string{"upos": "foo"}
	_, err = SchemeFor("upos", udCorp)
	assert.Error(t, err)
}

func TestValidateValueFillsAttr(t *testing.T) {
	issues, err := ValidateValue("upos", "noun", false, CorpusContext{Attrs: []string{"upos"}})
	assert.NoError(t, err)
	assert.Len(t, issues, 1)
	assert.Equal(t, "upos", issues[0].Attr)
	assert.Equal(t, "noun", issues[0].Value)
}
