package feats

import (
	"fmt"
	"math/rand"
	"sort"
)

const (
	maxParamValue = 1.0 //60.0
)

var (
	rgen = rand.New(rand.NewSource(0))
)

type Chromosome []float64

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

func randomVector() Chromosome {
	ans := make(Chromosome, ParamsSize)
	for i := 0; i < ParamsSize; i++ {
		ans[i] = rand.Float64() * maxParamValue
	}
	return ans
}

func mutate(ch Chromosome, probMut float64) Chromosome {
	ans := make(Chromosome, len(ch))
	for i := 0; i < len(ch); i++ {
		if rand.Float64() < probMut {
			ans[i] = rand.Float64() * maxParamValue
		}
	}
	return ans
}

type populItem struct {
	ch    Chromosome
	score Result
}

type Result struct {
	Score     float64
	Precision float64
	Recall    float64
}

func Optimize(populationSize int, maxNumIter int, probMutation float64, fn func(inp Chromosome) Result) {
	population := make([]populItem, populationSize)
	var bestSoFar populItem
	for i := 0; i < populationSize; i++ {
		population[i] = populItem{ch: randomVector()}
	}
	for i := 0; i < maxNumIter; i++ {
		fmt.Println("GENERATION: ", i)
		for j := 0; j < populationSize; j++ {
			population[j].score = fn(population[j].ch)
			//fmt.Println("item ", j, " score: ", population[j].score)
		}
		sort.Slice(population, func(i, j int) bool {
			return population[i].score.Score < population[j].score.Score
		})

		fmt.Println("best; ", population[0].score, ", worst; ", population[len(population)-1].score)
		fmt.Printf("winner: %#v\n", population[0].ch)
		fmt.Printf("   %#v\n", population[0].score)
		if bestSoFar.score.Score == 0 || population[0].score.Score < bestSoFar.score.Score {
			bestSoFar = population[0]
		}
		fmt.Printf(">>> BEST SO FAR: %#v\n", bestSoFar)

		newPopulation := make([]populItem, populationSize)
		for i := 0; i < populationSize; i++ {
			ch1 := population[rand.Intn(populationSize/10)]
			ch2 := population[rand.Intn(populationSize/10)]
			newPopulation[i] = populItem{ch: mutate(ch1.ch.Crossover(ch2.ch), probMutation)}

		}
		population = newPopulation

	}

}
