package predict

type Prediction struct {
	Votes          []float64
	PredictedClass int
}

func (p Prediction) SlowQueryVote() float64 {
	return p.Votes[1]
}
