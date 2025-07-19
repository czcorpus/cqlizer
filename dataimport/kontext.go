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
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/czcorpus/cqlizer/cql"
	"github.com/czcorpus/cqlizer/index"
	"github.com/dgraph-io/badger/v4"
	"github.com/rs/zerolog/log"
)

const (
	qTypeSimple        queryType = "simple"
	qTypeAdvanced      queryType = "advanced"
	scanBufferCapacity           = 1024 * 1024 // some of our qeries can be quite long
)

func convertDatetimeStringWithMillisNoTZ(datetime string) time.Time {
	t, err := time.Parse("2006-01-02T15:04:05.000000", datetime)
	if err == nil {
		return t
	}
	log.Warn().Msgf("%s", err)
	return time.Time{}
}

func convertDatetimeStringWithMillis(datetime string) time.Time {
	t, err := time.Parse("2006-01-02T15:04:05.000000-07:00", datetime)
	if err == nil {
		return t
	}
	log.Warn().Msgf("%s", err)
	return time.Time{}
}

type queryType string

type queryProps struct {
	Q     string    `json:"q"`
	QType queryType `json:"qtype"`
}

type inputArgs struct {
	Corpora         []string     `json:"corpora"`
	UseRegexp       bool         `json:"use_regexp"`
	Queries         []queryProps `json:"queries"`
	TTNumAttrs      int          `json:"tt_num_attrs"`
	TTNumSelections int          `json:"tt_num_selections"`
}

func (iargs inputArgs) hasAvancedQuery() bool {
	return len(iargs.Queries) > 0 && iargs.Queries[0].QType == qTypeAdvanced
}

func (iargs inputArgs) hasSimpleRegexpQuery() bool {
	return len(iargs.Corpora) > 0 && iargs.Queries[0].QType == qTypeSimple && iargs.UseRegexp
}

func (iargs inputArgs) getFirstQuery() string {
	if len(iargs.Queries) > 0 {
		if iargs.Queries[0].QType == qTypeAdvanced {
			return iargs.Queries[0].Q
		}
		return fmt.Sprintf("\"%s\"", iargs.Queries[0].Q)
	}
	return ""
}

type logQueryRecord struct {
	Action   string    `json:"action"`
	Date     string    `json:"date"`
	Args     inputArgs `json:"args"`
	ProcTime float64   `json:"proc_time"`
	Logger   string    `json:"logger"`
}

func (rec logQueryRecord) GetTime() time.Time {
	if rec.Date[len(rec.Date)-1] == 'Z' {
		return convertDatetimeStringWithMillisNoTZ(rec.Date[:len(rec.Date)-1] + "000")
	}
	return convertDatetimeStringWithMillis(rec.Date)
}

// --------

type lastOpForm struct {
	FormType       string            `json:"form_type"`
	CurrQueryTypes map[string]string `json:"curr_query_types"`
	CurrQueries    map[string]string `json:"curr_queries"`
}

type queryPersistenceRecord struct {
	LastOpForm lastOpForm `json:"lastop_form"`
}

func (qpr *queryPersistenceRecord) AdvancedQueries() []string {
	ans := make([]string, 0, len(qpr.LastOpForm.CurrQueries))
	for k, v := range qpr.LastOpForm.CurrQueryTypes {
		if v == "advanced" {
			ans = append(ans, qpr.LastOpForm.CurrQueries[k])
		}
	}
	return ans
}

// -----

type ConcurrentErr struct {
	lock  sync.Mutex
	items []error
}

func (cerr *ConcurrentErr) Add(err error) {
	cerr.lock.Lock()
	cerr.items = append(cerr.items, err)
	cerr.lock.Unlock()
}

func (cerr *ConcurrentErr) LastErr() error {
	if len(cerr.items) > 0 {
		return cerr.items[0]
	}
	return nil
}

// --------

func ImportKontextLog(path string, db *index.DB) error {
	fr, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to import file \u25B6 %w", err)
	}

	defer fr.Close()

	scn := bufio.NewScanner(fr)
	buf := make([]byte, scanBufferCapacity)
	scn.Buffer(buf, scanBufferCapacity)
	var i int

	err = db.Update(
		func(txn *badger.Txn) error {
			for scn.Scan() {
				var rec logQueryRecord
				if err := json.Unmarshal(scn.Bytes(), &rec); err != nil {
					fmt.Fprintf(os.Stderr, "failed to parse JSON log with error: %s\n", err)
					fmt.Fprintf(os.Stderr, "   ... skipping\n")
					break
				}
				i++

				if rec.Args.hasAvancedQuery() {
					fmt.Fprintf(os.Stderr, "Q: %s\n", rec.Args.getFirstQuery())
					_, err := cql.ParseCQL("query@"+rec.Date, rec.Args.getFirstQuery())
					if err != nil {
						fmt.Fprintf(os.Stderr, "failed to parse %s with error: %s\n", rec.Args.getFirstQuery(), err)
						fmt.Fprintf(os.Stderr, "   ... skipping\n")
						continue
					}
					// TODO - use parsed query as w2v source
					db.StoreQueryTx(txn, rec.Args.getFirstQuery(), 1)
				}
			}
			return nil
		},
	)
	if err != nil {
		return fmt.Errorf("failed to update index: %w", err)
	}
	return nil
}
