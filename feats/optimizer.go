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
	"fmt"
	"math/rand"
	"sort"
	"strings"
)

const (
	maxParamValue = 1.0
)

type Chromosome []float64

func (ch Chromosome) String() string {
	items := make([]string, len(ch))
	for i, v := range ch {
		items[i] = fmt.Sprintf("%01.3f", v)
	}
	return "Chromosome{" + strings.Join(items, ", ") + "}"
}

func (ch Chromosome) Crossover(ch2 Chromosome) Chromosome {
	ans := make(Chromosome, 0, len(ch))
	i0 := rand.Intn(len(ch))
	if rand.Float64() < 0.5 {
		ans = append(ans, ch[:i0]...)
		ans = append(ans, ch2[i0:]...)

	} else {
		ans = append(ans, ch2[:i0]...)
		ans = append(ans, ch[i0:]...)
	}
	return ans
}

func randomVector(dim int) Chromosome {
	ans := make(Chromosome, dim)
	for i := 0; i < dim; i++ {
		ans[i] = rand.Float64() * maxParamValue
	}
	return ans
}

func mutate(ch Chromosome, probMut float64) Chromosome {
	ans := make(Chromosome, len(ch))
	for i := 0; i < len(ch); i++ {
		if rand.Float64() < probMut {
			ans[i] = rand.Float64() * maxParamValue

		} else {
			ans[i] = ch[i]
		}
	}
	return ans
}

type PopulItem struct {
	Ch    Chromosome
	Score Result
}

type Result struct {
	Score         float64
	PredictionErr float64
	Precision     float64
	Recall        float64
}

func Optimize(populationSize int, vectorDim int, maxNumIter int, tuneAfter int, probMutation float64, fn func(inp Chromosome) Result) PopulItem {
	population := make([]PopulItem, populationSize)
	var bestSoFar PopulItem
	for i := 0; i < populationSize; i++ {
		population[i] = PopulItem{Ch: randomVector(vectorDim)}
	}
	for i := 0; i < maxNumIter; i++ {
		fmt.Println("GENERATION: ", i)
		for j := 0; j < populationSize; j++ {
			population[j].Score = fn(population[j].Ch)
			//fmt.Println("item ", j, " score: ", population[j].score)
		}
		sort.Slice(population, func(i, j int) bool {
			return population[i].Score.Score < population[j].Score.Score
		})

		fmt.Println("best: ", population[0].Score, ", ", population[0].Ch)
		fmt.Println("worst: ", population[len(population)-1].Score, ", ", population[len(population)-1].Ch)
		if bestSoFar.Score.Score == 0 || population[0].Score.Score < bestSoFar.Score.Score {
			bestSoFar = population[0]
		}
		fmt.Printf(">>> BEST SO FAR: %#v\n%s\n", bestSoFar.Score, bestSoFar.Ch)

		newPopulation := make([]PopulItem, populationSize)
		for j := 0; j < populationSize; j++ {
			ch1 := population[rand.Intn(populationSize/5)]
			ch2 := population[rand.Intn(populationSize/5)]
			if tuneAfter > 0 && tuneAfter < i {
				newPopulation[j] = PopulItem{Ch: mutate(ch1.Ch.Crossover(ch2.Ch), probMutation/2)}

			} else {
				newPopulation[j] = PopulItem{Ch: mutate(ch1.Ch.Crossover(ch2.Ch), probMutation)}
			}
			//fmt.Println("new item from: ", ch1.ch, " and ", ch2.ch)
			//fmt.Println(">>> ", newPopulation[i].ch)

		}
		population = newPopulation

	}
	return bestSoFar
}