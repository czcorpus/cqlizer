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
	ch1 := randomVector()
	ch2 := randomVector()
	ch3 := ch1.Crossover(ch2)
	assert.Equal(t, 0, ch3)
}
