package stats

type ListFilter struct {
	Benchmarked         *bool
	TrainingExcluded    *bool
	WithNormalizedQuery *bool
	AnyCorpus           *bool
}

func (filter ListFilter) SetBenchmarked(v bool) ListFilter {
	filter.Benchmarked = &v
	return filter
}

func (filter ListFilter) SetTrainingExcluded(v bool) ListFilter {
	filter.TrainingExcluded = &v
	return filter
}

func (filter ListFilter) SetWithNormalizedQuery(v bool) ListFilter {
	filter.WithNormalizedQuery = &v
	return filter
}

func (filter ListFilter) SetAnyCorpus(v bool) ListFilter {
	filter.AnyCorpus = &v
	return filter
}
