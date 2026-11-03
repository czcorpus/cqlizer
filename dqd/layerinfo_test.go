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

func TestLoadLayerInfoWritten(t *testing.T) {
	li, err := LoadLayerInfo("testdata/layerinfo_written.json")
	require.NoError(t, err)

	assert.Equal(t, FirstClass{Token: "Token", Segment: "Segment", Document: "Document"}, li.FirstClass)
	assert.Equal(t,
		[]string{"DepRel", "Division", "Document", "NamedEntity", "Paragraph", "Segment", "Token", "Token@de"},
		li.LayerNames())

	token := li.Layers["Token"]
	assert.Equal(t, "Token", token.Name)
	assert.Equal(t, LayerTypeUnit, token.LayerType)
	assert.True(t, token.Abstract)
	assert.Equal(t, &Anchoring{Stream: true}, token.Anchoring)
	assert.Equal(t, []string{"form", "lemma", "ufeat", "upos", "xpos"}, token.AttrNames())
	assert.Empty(t, token.Meta)

	xpos := token.Attributes["xpos"]
	assert.Equal(t, "xpos", xpos.Name)
	assert.Equal(t, "categorical", xpos.Type)
	assert.Equal(t, "Sample tagset", xpos.DisplayName)
	assert.Equal(t, []string{"NN", "VB", "JJ"}, xpos.Values)
	assert.True(t, xpos.Nullable)
	assert.False(t, xpos.IsGlobal)

	upos := token.Attributes["upos"]
	assert.True(t, upos.IsGlobal)
	assert.Empty(t, upos.Values)
	assert.Equal(t, []string{"ADJ", "ADP", "ADV", "NOUN", "VERB"}, li.GlobalAttrValues["upos"])

	// meta attributes are separated from the regular ones
	doc := li.Layers["Document"]
	assert.Equal(t, LayerTypeSpan, doc.LayerType)
	assert.Equal(t, "Division", doc.Contains)
	assert.Equal(t, []string{"pubdate"}, doc.AttrNames())
	assert.Equal(t, []string{"classCode", "title", "year"}, doc.MetaNames())
	assert.Equal(t, "classCode", doc.Meta["classCode"].Name)
	assert.Equal(t, "integer", doc.Meta["year"].Type)

	// language-specific variant of a layer
	tokenDE := li.Layers["Token@de"]
	assert.Equal(t, "Token", tokenDE.PartOf)
	assert.Equal(t, LayerType(""), tokenDE.LayerType)

	// relation endpoints
	deprel := li.Layers["DepRel"]
	assert.Equal(t, LayerTypeRelation, deprel.LayerType)
	assert.True(t, deprel.Attributes["source"].IsRelationEndpoint())
	assert.Equal(t, "Token", deprel.Attributes["target"].Entity)
	assert.Equal(t, "head", deprel.Attributes["target"].DisplayName)
	assert.False(t, deprel.Attributes["udep"].IsRelationEndpoint())

	assert.Equal(t,
		[][]string{
			{"Document", "Division", "Paragraph", "Segment", "Token"},
			{"NamedEntity", "Token"},
		},
		li.Hierarchies())
}

func TestLoadLayerInfoSpoken(t *testing.T) {
	li, err := LoadLayerInfo("testdata/layerinfo_spoken.json")
	require.NoError(t, err)

	assert.Equal(t, FirstClass{Token: "Token", Segment: "Utterance"}, li.FirstClass)
	assert.Equal(t, &Anchoring{Stream: true, Time: true}, li.Layers["Utterance"].Anchoring)

	speaker := li.Layers["Utterance"].Attributes["speaker"]
	assert.True(t, speaker.IsRef())
	assert.Equal(t, "speaker", speaker.Ref)
	assert.Empty(t, speaker.Type)

	gattr := li.GlobalAttrs["speaker"]
	require.NotNil(t, gattr)
	assert.Equal(t, "dict", gattr.Type)
	assert.Equal(t, "region", gattr.Keys["region"].Name)
	assert.Equal(t, "text", gattr.Keys["birthYear"].Type)

	assert.Equal(t, [][]string{{"Interview", "Utterance", "Token"}}, li.Hierarchies())
}

func TestParseLayerInfoErrors(t *testing.T) {
	for name, data := range map[string]string{
		"invalid JSON":       `{"layer": `,
		"no layers":          `{"firstClass": {"token": "Token"}}`,
		"empty layer":        `{"layer": {"Token": null}}`,
		"invalid attribute":  `{"layer": {"Token": {"attributes": {"form": "text"}}}}`,
		"invalid meta group": `{"layer": {"Doc": {"attributes": {"meta": ["title"]}}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ParseLayerInfo([]byte(data))
			assert.Error(t, err)
		})
	}
}

func TestLayerInfoHierarchiesCycle(t *testing.T) {
	li, err := ParseLayerInfo([]byte(`{"layer": {
		"Root": {"contains": "A"},
		"A": {"contains": "B"},
		"B": {"contains": "A"}
	}}`))
	require.NoError(t, err)
	assert.Equal(t, [][]string{{"Root", "A", "B"}}, li.Hierarchies())
}
