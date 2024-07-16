package stats

type ListFilter struct {
	Benchmarked      *bool
	TrainingExcluded *bool
	SynCompat        *bool
}

func (filter ListFilter) SetBenchmarked(v bool) ListFilter {
	filter.Benchmarked = &v
	return filter
}

func (filter ListFilter) SetTrainingExcluded(v bool) ListFilter {
	filter.TrainingExcluded = &v
	return filter
}

func (filter ListFilter) SetSynCompat(v bool) ListFilter {
	filter.SynCompat = &v
	return filter
}
