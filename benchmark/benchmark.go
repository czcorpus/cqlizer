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

package benchmark

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/czcorpus/cnc-gokit/httpclient"
	"github.com/czcorpus/cqlizer/cnf"
	"github.com/czcorpus/cqlizer/stats"
	"github.com/rs/zerolog/log"
)

const (
	idleConnTimeoutSecs = 60
	requestTimeoutSecs  = 60
)

type Executor struct {
	conf    *cnf.Conf
	statsDB *stats.Database
}

func (e *Executor) RunFull(overwriteBenchmarked bool) error {
	var listFilter stats.ListFilter
	if !overwriteBenchmarked {
		listFilter = listFilter.SetBenchmarked(false)
	}
	rows, err := e.statsDB.GetAllRecords(listFilter)
	if err != nil {
		return fmt.Errorf("failed to run full benchmark \u25B6 %w", err)
	}
	for i, row := range rows {
		fmt.Printf("%d: %s\n", i, row.Query)
		dur, err := e.TestQuery("syn2020", row.Query)
		if err != nil {
			log.Error().
				Err(err).
				Int("i", i).
				Str("q", row.Query).
				Msg("failed to perform benchmark query, skipping to the next")
			continue
		}
		fmt.Println("\ttime: ", dur)
		err = e.statsDB.AddBenchmarkResult(row.ID, dur)
		if err != nil {
			log.Error().Err(err).Send()
		}
	}
	return nil
}

func (e *Executor) TestQuery(corpus, query string) (time.Duration, error) {

	fullURL, err := url.JoinPath(
		e.conf.BenchmarkMQueryURL,
		fmt.Sprintf("/term-frequency/%s", corpus),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to perform benchmark query \u25B6 %w", err)
	}
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to perform benchmark query \u25B6 %w", err)
	}
	q := req.URL.Query()
	q.Add("q", query)
	req.URL.RawQuery = q.Encode()

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = httpclient.TransportMaxIdleConns
	transport.MaxConnsPerHost = httpclient.TransportMaxConnsPerHost
	transport.MaxIdleConnsPerHost = httpclient.TransportMaxIdleConnsPerHost
	transport.IdleConnTimeout = time.Duration(idleConnTimeoutSecs) * time.Second
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout:   time.Duration(requestTimeoutSecs) * time.Second,
		Transport: transport,
	}
	t0 := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to perform benchmark query \u25B6 %w", err)
	}
	defer resp.Body.Close()
	return time.Since(t0), nil

}

func NewExecutor(conf *cnf.Conf, statsDB *stats.Database) *Executor {
	return &Executor{
		conf:    conf,
		statsDB: statsDB,
	}
}
