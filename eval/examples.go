package eval

var ObligatoryExamples = []QueryStatsRecord{
	{Corpus: "syn_v13", CorpusSize: 6400899055, TimeProc: 500, Query: "aword,[word=\".*\"]"},
	{Corpus: "syn_v13", CorpusSize: 6400899055, TimeProc: 500, Query: "aword,[lemma=\".*\"]"},
	{Corpus: "syn_v13", CorpusSize: 6400899055, TimeProc: 500, Query: "aword,[lc=\".*\"]"},
	{Corpus: "syn_v13", CorpusSize: 6400899055, TimeProc: 500, Query: "aword,[word=\".+\"]"},
	{Corpus: "syn_v13", CorpusSize: 6400899055, TimeProc: 500, Query: "aword,[tag=\"N.*\"]"},
}
