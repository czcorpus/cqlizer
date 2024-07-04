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

package feats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChromosomeCrossover(t *testing.T) {
	ch1 := Chromosome{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0}
	ch2 := Chromosome{10.0, 20.0, 30.0, 40.0, 50.0, 60.0, 70.0, 80.0, 90.0, 100.0}
	ch3 := ch1.Crossover(ch2)
	assert.NotEqual(t, ch1, ch3)
	assert.NotEqual(t, ch2, ch3)
	assert.Equal(t, len(ch1), len(ch3))
}

func TestMutate(t *testing.T) {
	ch1 := Chromosome{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0}
	ch2 := mutate(ch1, 1.0)
	assert.Equal(t, ch1, ch2)
}

func TestZeroStuff(t *testing.T) {
	ch1 := randomVector(4)
	ch2 := randomVector(4)
	ch3 := ch1.Crossover(ch2)
	assert.Equal(t, 0, ch3)
}
