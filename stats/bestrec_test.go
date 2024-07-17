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

package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBestRecInitial(t *testing.T) {
	best := NewBestMatches(4)
	best.TryAdd(&DBRecord{ID: "a"}, 5)
	best.TryAdd(&DBRecord{ID: "b"}, 9)
	best.TryAdd(&DBRecord{ID: "c"}, 8)
	best.TryAdd(&DBRecord{ID: "d"}, 4)
	assert.Equal(t, "d", best.At(0).Record.ID)
	assert.Equal(t, "a", best.At(1).Record.ID)
	assert.Equal(t, "c", best.At(2).Record.ID)
	assert.Equal(t, "b", best.At(3).Record.ID)
}

func TestBestRecFull(t *testing.T) {
	best := NewBestMatches(4)
	best.data = []MatchItem{
		{Record: &DBRecord{ID: "a"}, Distance: 1},
		{Record: &DBRecord{ID: "b"}, Distance: 4},
		{Record: &DBRecord{ID: "c"}, Distance: 8},
		{Record: &DBRecord{ID: "d"}, Distance: 12},
	}
	best.TryAdd(&DBRecord{ID: "x"}, 6)
	assert.Equal(t, "x", best.At(2).Record.ID)
}
