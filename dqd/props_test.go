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

func propsFromFixture(t *testing.T, path string) CorpusProps {
	li, err := LoadLayerInfo(path)
	require.NoError(t, err)
	cp, err := CorpusPropsFromLayerInfo(li)
	require.NoError(t, err)
	return cp
}

func TestCorpusPropsFromLayerInfo(t *testing.T) {
	cp := propsFromFixture(t, "testdata/layerinfo_written.json")
	assert.Equal(t, "Token", cp.PosAttrLayer)
	assert.Equal(t, "Segment", cp.SegmentLayer)
	assert.Equal(t, "word", cp.DefaultAttr)
	// ufeat (dict) is not comparable using attr="value"
	assert.Equal(t,
		map[string]string{"form": "form", "lemma": "lemma", "upos": "upos", "xpos": "xpos", "word": "form"},
		cp.PosAttrs)

	cp = propsFromFixture(t, "testdata/layerinfo_spoken.json")
	assert.Equal(t, "Token", cp.PosAttrLayer)
	assert.Equal(t, "Utterance", cp.SegmentLayer)
	assert.Equal(t,
		map[string]string{"form": "form", "lemma": "lemma", "pos": "pos", "word": "form"},
		cp.PosAttrs)
}

func TestCorpusPropsFromLayerInfoWordAttr(t *testing.T) {
	// a corpus with its own "word" attribute keeps it
	li, err := ParseLayerInfo([]byte(`{
		"layer": {
			"Token": {"attributes": {"word": {"type": "text"}, "form": {"type": "text"}}},
			"Segment": {"contains": "Token"}
		},
		"firstClass": {"token": "Token", "segment": "Segment"}
	}`))
	require.NoError(t, err)
	cp, err := CorpusPropsFromLayerInfo(li)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"word": "word", "form": "form"}, cp.PosAttrs)

	// a corpus without "form" gets no alias
	li, err = ParseLayerInfo([]byte(`{
		"layer": {
			"Token": {"attributes": {"text": {"type": "text"}}},
			"Segment": {"contains": "Token"}
		},
		"firstClass": {"token": "Token", "segment": "Segment"}
	}`))
	require.NoError(t, err)
	cp, err = CorpusPropsFromLayerInfo(li)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"text": "text"}, cp.PosAttrs)
}

func TestCorpusPropsFromLayerInfoErrors(t *testing.T) {
	for name, data := range map[string]string{
		"missing firstClass":      `{"layer": {"Token": {}, "Segment": {}}}`,
		"missing segment":         `{"layer": {"Token": {}}, "firstClass": {"token": "Token"}}`,
		"undefined token layer":   `{"layer": {"Segment": {}}, "firstClass": {"token": "Word", "segment": "Segment"}}`,
		"undefined segment layer": `{"layer": {"Token": {}}, "firstClass": {"token": "Token", "segment": "Sentence"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			li, err := ParseLayerInfo([]byte(data))
			require.NoError(t, err)
			_, err = CorpusPropsFromLayerInfo(li)
			assert.Error(t, err)
		})
	}
}

func TestCQL2DQDWithLayerInfoProps(t *testing.T) {
	cp := propsFromFixture(t, "testdata/layerinfo_spoken.json")
	ans, err := CQL2DQD(cp, ViewOptions{}, `"bon" [pos="NOM" & lemma="jour"]`)
	require.NoError(t, err)
	assert.Equal(t, `Utterance s
sequence seq1
    Token@s t1
        form = "bon"
    Token@s t2
        pos = "NOM"
        lemma = "jour"

res => plain
    context
        s
    entities
        t1
        t2
`, ans)

	// the corpus has no "tag" attribute
	_, err = CQL2DQD(cp, ViewOptions{}, `[tag="N.*"]`)
	var unsupported *UnsupportedFeature
	assert.ErrorAs(t, err, &unsupported)
}
