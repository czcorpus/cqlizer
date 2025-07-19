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

package dataimport

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONDataImport(t *testing.T) {
	q := `{"lines_groups": {"sorted": false, "data": []}, "q": ["aword,[lemma=\"mimo\" & tag=\"R.*\" & tag!=\"D\"][pos=\"N\" & tag!=\"....2.*\" & tag!=\"....4.*\"|pos= \"A\" & tag!=\"....2.*\" & tag!=\"....4.*\"|pos=\"P\" & tag!=\"....2.*\" & tag!=\"....4.*\"]  "], "corpora": ["syn_v7"], "user_id": 14843, "lastop_form": {"selected_text_types": {}, "bib_mapping": {}, "curr_default_attr_values": {"syn_v7": "word"}, "curr_query_types": {"syn_v7": "cql"}, "curr_pcq_pos_neg_values": {"syn_v7": "pos"}, "form_type": "query", "curr_queries": {"syn_v7": "[lemma=\"mimo\" & tag=\"R.*\" & tag!=\"D\"][pos=\"N\" & tag!=\"....2.*\" & tag!=\"....4.*\"|pos= \"A\" & tag!=\"....2.*\" & tag!=\"....4.*\"|pos=\"P\" & tag!=\"....2.*\" & tag!=\"....4.*\"]"}, "curr_qmcase_values": {"syn_v7": false}, "curr_include_empty_values": {"syn_v7": false}, "curr_lpos_values": {"syn_v7": ""}}, "persist_level": 1, "usesubcorp": "", "id": "3BiSXng4oJ22"}`
	var cp ConcPersistence
	rec, err := cp.importJSONRecord(q)
	assert.NoError(t, err)
	assert.True(t, len(rec.AdvancedQueries()) > 0)
}
