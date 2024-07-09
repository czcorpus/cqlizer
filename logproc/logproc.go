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

package logproc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/czcorpus/cqlizer/cnf"
	"github.com/czcorpus/cqlizer/cql"
	"github.com/czcorpus/cqlizer/stats"
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
	fmt.Println("iargs: ", iargs)
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

type inputRecord struct {
	Action   string    `json:"action"`
	Date     string    `json:"date"`
	Args     inputArgs `json:"args"`
	ProcTime float64   `json:"proc_time"`
	Logger   string    `json:"logger"`
}

func (rec inputRecord) GetTime() time.Time {
	if rec.Date[len(rec.Date)-1] == 'Z' {
		return convertDatetimeStringWithMillisNoTZ(rec.Date[:len(rec.Date)-1] + "000")
	}
	return convertDatetimeStringWithMillis(rec.Date)
}

// --------

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

func ImportLog(conf *cnf.Conf, path string, addToTrainingSet bool) error {
	data := make(chan inputRecord, 100)
	retErr := new(ConcurrentErr)
	fr, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to import file: %w", err)
	}
	statsDb, err := stats.NewDatabase(conf.WorkingDBPath)
	if err != nil {
		return fmt.Errorf("failed to import file: %w", err)
	}
	err = statsDb.Init()
	if err != nil {
		return fmt.Errorf("failed to import file: %w", err)
	}
	err = statsDb.StartTx()
	if err != nil {
		return fmt.Errorf("failed to import file: %w", err)
	}
	defer fr.Close()
	var wg sync.WaitGroup
	wg.Add(2)
	// producer
	go func() {
		scn := bufio.NewScanner(fr)
		buf := make([]byte, scanBufferCapacity)
		scn.Buffer(buf, scanBufferCapacity)
		var i int
		for scn.Scan() {
			var rec inputRecord
			if err := json.Unmarshal(scn.Bytes(), &rec); err != nil {
				retErr.Add(fmt.Errorf("line %d: failed to decode log record: %w", i+1, err))
				break
			}
			i++
			data <- rec
		}
		close(data)
		wg.Done()
	}()

	// consumer
	go func() {
		for rec := range data {
			if rec.Action == "/query_submit" && rec.Logger == "QUERY" &&
				(rec.Args.hasAvancedQuery() || rec.Args.hasSimpleRegexpQuery()) {
				_, err := cql.ParseCQL("query@"+rec.Date, rec.Args.getFirstQuery())
				if err != nil {
					fmt.Printf("failed to parse %s with error: %s", rec.Args.getFirstQuery(), err)
					fmt.Println("   ... skipping")
					continue
				}
				if retErr.LastErr() != nil {
					continue // we want the consumer to run till the end
				}
				_, err = statsDb.AddRecord(stats.DBRecord{
					Query:           rec.Args.getFirstQuery(),
					Corpname:        rec.Args.Corpora[0],
					Datetime:        rec.GetTime().Unix(),
					ProcTime:        rec.ProcTime,
					TrainingExclude: !addToTrainingSet,
				})
				if err != nil {
					retErr.Add(err)
				}
			}
		}
		wg.Done()
	}()

	wg.Wait()
	if retErr.LastErr() != nil {
		return retErr.LastErr()
	}
	return statsDb.CommitTx()
}
