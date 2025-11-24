// Copyright 2025 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2025 Department of Linguistics,
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

package eval

var ObligatoryExamples = []QueryStatsRecord{
	{Corpus: "syn_v13", CorpusSize: 6400899055, TimeProc: 500, Query: "aword,[]"},
	{Corpus: "syn_v13", CorpusSize: 6400899055, TimeProc: 500, Query: "aword,[word=\".*\"]"},
	{Corpus: "syn_v13", CorpusSize: 6400899055, TimeProc: 500, Query: "aword,[word=\".+\"]"},
	{Corpus: "syn_v13", CorpusSize: 6400899055, TimeProc: 500, Query: "aword,[lemma=\".*\"]"},
	{Corpus: "syn_v13", CorpusSize: 6400899055, TimeProc: 500, Query: "aword,[lemma=\".+\"]"},
	{Corpus: "syn_v13", CorpusSize: 6400899055, TimeProc: 500, Query: "aword,[lc=\".*\"]"},
	{Corpus: "syn_v13", CorpusSize: 6400899055, TimeProc: 500, Query: "aword,[lc=\".+\"]"},
	{Corpus: "syn_v13", CorpusSize: 6400899055, TimeProc: 500, Query: "aword,[tag=\"N.*\"]"},
	{Corpus: "syn_v13", CorpusSize: 6400899055, TimeProc: 500, Query: "aword,[tag=\"N.+\"]"},
	{Corpus: "syn_v13", CorpusSize: 6400899055, TimeProc: 500, Query: "aword,[pos=\"N\"]"},
}
