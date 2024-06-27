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

func (e *Executor) RullFull(overwriteBenchmarked bool) error {
	rows, err := e.statsDB.GetAllRecords(!overwriteBenchmarked)
	if err != nil {
		return fmt.Errorf("failed to run full benchmark: %w", err)
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
		return 0, fmt.Errorf("failed to perform benchmark query: %w", err)
	}
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to perform benchmark query: %w", err)
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
		return 0, fmt.Errorf("failed to perform benchmark query: %w", err)
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
