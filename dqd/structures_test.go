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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// plainResultDQD creates the expected plain result block
func plainResultDQD(entities ...string) string {
	ans := "\nres => plain\n    context\n        s\n    entities\n"
	for _, e := range entities {
		ans += "        " + e + "\n"
	}
	return ans
}

// structureCases use props created from testdata/layerinfo_written.json
// (Document > Division > Paragraph > Segment > Token, NamedEntity > Token)
var structureCases = []struct {
	cql      string
	expected string
}{
	// within a structure containing segments
	{`[lemma="dog"] within <Document classCode="S.*"/>`, `Document document1
    classCode = /^(S.*)$/
Segment@document1 s
Token@s t1
    lemma = "dog"
` + plainResultDQD("t1")},
	// multiple within clauses, structure without attributes,
	// boolean logic in structural attributes
	{`"dog" within <Document classCode="S.*" & !title="x"/> within <Division/>`, `Document document1
    classCode = /^(S.*)$/
    title != "x"
Division division1
Segment@document1@division1 s
Token@s t1
    form = "dog"
` + plainResultDQD("t1")},
	// within the segment itself
	{`"dog" "barks" within <Segment/>`, `Segment s
sequence seq1
    Token@s t1
        form = "dog"
    Token@s t2
        form = "barks"
` + plainResultDQD("t1", "t2")},
	// within a structure inside the segment - tokens are part of it
	{`[upos="PROPN"] [upos="PROPN"]+ within <NamedEntity type="PER"/>`, `Segment s
NamedEntity@s namedEntity1
    type = "PER"
sequence seq1
    Token@s@namedEntity1 t1
        upos = "PROPN"
    sequence rep1 1..*
        Token@s@namedEntity1 t2
            upos = "PROPN"
` + plainResultDQD("t1")},
	// containing
	{`<Segment/> containing [upos="VERB"]`, `Segment s
Token@s t1
    upos = "VERB"
` + plainResultDQD("s")},
	{`<Document classCode="S.*"/> containing "dog" "barks"`, `Document document1
    classCode = /^(S.*)$/
Segment@document1 s
sequence seq1
    Token@s t1
        form = "dog"
    Token@s t2
        form = "barks"
` + plainResultDQD("document1")},
	{`<NamedEntity type="LOC"/> containing "Prague"`, `Segment s
NamedEntity@s namedEntity1
    type = "LOC"
Token@s@namedEntity1 t1
    form = "Prague"
` + plainResultDQD("namedEntity1")},
	// multiple containing clauses, negated containing
	{`<Segment/> containing [upos="VERB"] !containing "dog"`, `Segment s
Token@s t1
    upos = "VERB"
!EXISTS
    Token@s t2
        form = "dog"
` + plainResultDQD("s")},
	// containing + within
	{`<Segment/> containing "dog" within <Document classCode="S.*"/>`, `Document document1
    classCode = /^(S.*)$/
Segment@document1 s
Token@s t1
    form = "dog"
` + plainResultDQD("s")},
}

func TestCQL2DQDStructures(t *testing.T) {
	cp := propsFromFixture(t, "testdata/layerinfo_written.json")
	for _, tc := range structureCases {
		t.Run(tc.cql, func(t *testing.T) {
			ans, err := CQL2DQD(cp, ViewOptions{}, tc.cql)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, ans)
		})
	}
}

func TestCQL2DQDStructuresSyntax(t *testing.T) {
	skipIfNoDQDCheck(t)
	cp := propsFromFixture(t, "testdata/layerinfo_written.json")
	for _, tc := range structureCases {
		t.Run(tc.cql, func(t *testing.T) {
			ans, err := CQL2DQD(cp, ViewOptions{}, tc.cql)
			require.NoError(t, err)
			checkDQDSyntax(t, ans)
		})
	}
}

func TestCQL2DQDStructuresDefaultProps(t *testing.T) {
	ans, err := CQL2DQD(DefaultCorpusProps(), ViewOptions{}, `[word="a"] within <doc/> within <s/>`)
	require.NoError(t, err)
	assert.Equal(t, `Document document1
Segment@document1 s
Token@s t1
    form = "a"
`+plainResultDQD("t1"), ans)
}

func TestCorpusPropsFromLayerInfoStructures(t *testing.T) {
	cp := propsFromFixture(t, "testdata/layerinfo_written.json")
	assert.Equal(t,
		map[string]StructProps{
			// meta attributes are flattened, pubdate (date) and year (integer)
			// are not comparable using attr="value"
			"Document":    {Layer: "Document", ContainsSegments: true, Attrs: map[string]string{"classCode": "classCode", "title": "title"}},
			"Division":    {Layer: "Division", ContainsSegments: true, Attrs: map[string]string{"type": "type"}},
			"Paragraph":   {Layer: "Paragraph", ContainsSegments: true, Attrs: map[string]string{}},
			"Segment":     {Layer: "Segment", Attrs: map[string]string{}},
			"NamedEntity": {Layer: "NamedEntity", Attrs: map[string]string{"type": "type"}},
		},
		cp.Structures)

	cp = propsFromFixture(t, "testdata/layerinfo_spoken.json")
	assert.Equal(t,
		map[string]StructProps{
			// a reference to a global attribute (speaker) is not comparable
			"Interview": {Layer: "Interview", ContainsSegments: true, Attrs: map[string]string{"genre": "genre"}},
			"Utterance": {Layer: "Utterance", Attrs: map[string]string{"text": "text"}},
		},
		cp.Structures)
}
