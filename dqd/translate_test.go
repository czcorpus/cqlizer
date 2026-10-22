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
)

func TestCQL2DQDPropagatesParseError(t *testing.T) {
	_, err := CQL2DQD(CorpusProps{}, ViewOptions{}, `[word=`)
	assert.Error(t, err)
}

func TestCQL2DQDStubNotYetImplemented(t *testing.T) {
	// The query itself is valid CQL - this pins down that a well-formed
	// query reaches (and fails at) the translation stub, not the parser,
	// until translateQuery is actually implemented.
	_, err := CQL2DQD(CorpusProps{}, ViewOptions{}, `[word="a"]`)
	var unsupported *UnsupportedFeature
	assert.ErrorAs(t, err, &unsupported)
}
