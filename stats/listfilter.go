package stats

type ListFilter struct {
	Benchmarked      *bool
	TrainingExcluded *bool
}

func (filter ListFilter) SetBenchmarked(v bool) ListFilter {
	filter.Benchmarked = &v
	return filter
}

func (filter ListFilter) SetTrainingExcluded(v bool) ListFilter {
	filter.TrainingExcluded = &v
	return filter
}
