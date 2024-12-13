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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegexQuery(t *testing.T) {
	q1 := "[word=\"moto[a-z]\"]"
	p, err := ParseCQL("#", q1)
	assert.NoError(t, err)
	fmt.Println("p: ", p)
}

func TestRgOrQuery(t *testing.T) {
	q1 := "\"ſb(é|ě)r(ka|ku|ki|ze)\""
	_, err := ParseCQL("#", q1)
	assert.NoError(t, err)
}

func TestRgOrQuery2(t *testing.T) {
	q1 := "[lemma=\"de|-|\"]"
	_, err := ParseCQL("#", q1)
	assert.NoError(t, err)
}

func TestJustRgQuery(t *testing.T) {
	q1 := "\"more|less\""
	_, err := ParseCQL("#", q1)
	assert.NoError(t, err)
}

func TestParallelQuery(t *testing.T) {
	q := "[word=\"Skifahren\"] within <text group=\"Syndicate|Subtitles\" /> within " +
		"intercorp_v15_cs:[word=\"lyžování\"]"
	_, err := ParseCQL("#", q)
	assert.NoError(t, err)
}
