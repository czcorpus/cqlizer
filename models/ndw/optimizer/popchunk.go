package optimizer

import (
	"fmt"
	"sync"
)

type PopulationChunk struct {
	msgInput      <-chan int
	objectiveFunc func(Chromosome) Result
	population    []*PopulItem
	wg            *sync.WaitGroup
}

func (pc *PopulationChunk) goListen() {
	go func() {
		for range pc.msgInput {
			for _, item := range pc.population {
				item.Result = pc.objectiveFunc(item.Ch)
				//fmt.Println("item ", j, " score: ", population[j].score)
			}
			pc.wg.Done()
		}
	}()
}

func (pc *PopulationChunk) PrepareForNextRun(items []*PopulItem, wg *sync.WaitGroup) {
	pc.population = items
	pc.wg = wg
}

func newPopulationChunk(msg <-chan int, objectiveFunc func(Chromosome) Result) *PopulationChunk {
	ans := &PopulationChunk{
		msgInput:      msg,
		objectiveFunc: objectiveFunc,
	}
	ans.goListen()
	return ans
}

func splitPopulation(
	popul []*PopulItem,
	numPieces int,
) [][]*PopulItem {
	chunkSize := len(popul) / numPieces
	sizes := make([]int, numPieces)
	for i := 0; i < numPieces; i++ {
		sizes[i] = chunkSize
	}
	missing := len(popul) - numPieces*chunkSize
	for i := 0; i < missing; i++ {
		sizes[i]++
	}
	fmt.Printf("we have chunks: %#v\n", sizes)
	ans := make([][]*PopulItem, numPieces)
	var cut int
	for i, size := range sizes {
		ans[i] = popul[cut : cut+size]
		cut += size
	}
	return ans
}
