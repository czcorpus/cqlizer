// Copyright 2024 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2024 Institute of the Czech National Corpus,
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
