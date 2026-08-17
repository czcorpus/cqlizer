// Copyright 2024 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2024 Department of Linguistics,
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

package cql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryGetAttrs(t *testing.T) {
	q, err := ParseCQL("test", `[word="hi|hello"] [lemma="people" & tag="N.*"] within <text foo="b: ar" & zoo="b,az">`)
	assert.NoError(t, err)
	attrs := q.ExtractProps()
	assert.Equal(
		t,
		[]QueryProp{
			{Name: "word", Value: "hi|hello"},
			{Name: "lemma", Value: "people"},
			{Name: "tag", Value: "N.*"},
			{Structure: "text", Name: "foo", Value: "b: ar"},
			{Structure: "text", Name: "zoo", Value: "b,az"},
			{Structure: "text"},
		},
		attrs,
	)
}

func TestRegressionAtSign(t *testing.T) {
	q, err := ParseCQL("test", `[tag="X@.*"]`)
	assert.NoError(t, err)
	attrs := q.ExtractProps()
	assert.Equal(
		t,
		[]QueryProp{
			{Name: "tag", Value: "X@.*"},
		},
		attrs,
	)
}

func TestQueryGetAttrsSimpleStruct(t *testing.T) {
	q, err := ParseCQL("test", `[word="x"] within <s>`)
	assert.NoError(t, err)
	attrs := q.ExtractProps()
	assert.Equal(
		t,
		[]QueryProp{
			{Name: "word", Value: "x"},
			{Structure: "s", Name: "", Value: ""},
		},
		attrs,
	)
}

func TestRegexpOnlyQuery(t *testing.T) {
	q, err := ParseCQL("test", `"attr.*"`)
	assert.NoError(t, err)
	attrs := q.ExtractProps()
	assert.Equal(
		t,
		[]QueryProp{
			{Value: "attr.*"},
		},
		attrs,
	)
}

func TestNormalizedQuery(t *testing.T) {
	q, err := ParseCQL("test1", `[word = "foo" & (tag = "N.*"   | tag="X.*") ]    [word="bar"   ]   within <s   attr="x"  />`)
	assert.NoError(t, err)
	assert.Equal(t, `[word="foo" & (tag="N.*" | tag="X.*")] [word="bar"] within <s attr="x" />`, q.CQL())

	q, err = ParseCQL("test2", `(   union (  meet [   tag="N.*"  ] [tag  = "VB.*"]    -3    3  )   (    meet   [tag  ="A.*"] [tag="VB.*"]  -2    2))`)
	assert.NoError(t, err)
	assert.Equal(t, `(union (meet [tag="N.*"] [tag="VB.*"] -3 3) (meet [tag="A.*"] [tag="VB.*"] -2 2))`, q.CQL())

	// parallel corpus query
	q, err = ParseCQL("test3", `[word ="car"]    within     europarl5_de:   [word="Auto"]`)
	assert.NoError(t, err)
	assert.Equal(t, `[word="car"] within europarl5_de: [word="Auto"]`, q.CQL())
}
