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

func TestRgUnicodeProp(t *testing.T) {
	q1 := `[mwe_lemma=".+_\p{Lu}+" & mwe_tag=".*1"]`
	_, err := ParseCQL("#", q1)
	assert.NoError(t, err)
}

func TestRgPosixCharCls(t *testing.T) {
	q1 := `[word="^[[:alpha:]]{17}$"]`
	_, err := ParseCQL("#", q1)
	assert.NoError(t, err)
}

func TestRgUncommonChar(t *testing.T) {
	q1 := `[word="Bułka"]`
	_, err := ParseCQL("#", q1)
	assert.NoError(t, err)
}

func TestAlignedQuery(t *testing.T) {
	q1 := `[word="test"] within <text group=\"Acquis|Bible|Core|Europarl|PressEurop|Subtitles\" /> within intercorp_v12_cs:[word="Je"]`
	_, err := ParseCQL("#", q1)
	assert.NoError(t, err)
}

func TestRegress001(t *testing.T) {
	q1 := `[feats="VerbForm=Fin" & upos="VERB"]`
	_, err := ParseCQL("#", q1)
	assert.NoError(t, err)
}

func TestRgress002(t *testing.T) {
	q1 := `[(lemma="(?i)demokraticko\-liberálním" | sublemma="(?i)demokraticko\-liberálním" | word="(?i)demokraticko\-liberálním")]`
	_, err := ParseCQL("#", q1)
	assert.NoError(t, err)
}

func TestRgress003(t *testing.T) {
	q1 := `[lemma=".+t(o/ö)n"]`
	_, err := ParseCQL("#", q1)
	assert.NoError(t, err)
}

func TestRgress004(t *testing.T) {
	q1 := `(meet [col_lemma="didaktický_test"][col_lemma="didaktický_test" & lemma="didaktický"] 0 15)`
	_, err := ParseCQL("#", q1)
	assert.NoError(t, err)
}

func TestRgress005(t *testing.T) {
	q1 := `[word="ni{n,5}n"]`
	_, err := ParseCQL("#", q1)
	assert.NoError(t, err)
}
