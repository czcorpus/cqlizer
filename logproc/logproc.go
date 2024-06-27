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
	"github.com/czcorpus/cqlizer/feats"
	"github.com/czcorpus/cqlizer/stats"
	"github.com/rs/zerolog/log"
)

const (
	qTypeSimple   queryType = "simple"
	qTypeAdvanced queryType = "advanced"
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

func (iargs inputArgs) getFirstQuery() string {
	if len(iargs.Queries) > 0 {
		return iargs.Queries[0].Q
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

func ImportLog(conf *cnf.Conf, path string) error {
	data := make(chan inputRecord, 100)
	var retErr error
	var fr *os.File
	fr, retErr = os.Open(path)
	if retErr != nil {
		return fmt.Errorf("failed to import file: %w", retErr)
	}
	statsDb, retErr := stats.NewDatabase(conf.WorkingDBPath)
	if retErr != nil {
		return fmt.Errorf("failed to import file: %w", retErr)
	}
	retErr = statsDb.Init()
	if retErr != nil {
		return fmt.Errorf("failed to import file: %w", retErr)
	}
	retErr = statsDb.StartTx()
	if retErr != nil {
		return fmt.Errorf("failed to import file: %w", retErr)
	}
	defer fr.Close()
	var wg sync.WaitGroup
	wg.Add(2)
	// producer
	go func() {
		scn := bufio.NewScanner(fr)
		for scn.Scan() {
			var rec inputRecord
			if err := json.Unmarshal(scn.Bytes(), &rec); err != nil {
				retErr = fmt.Errorf("failed to decode log record: %w", err)
				break
			}
			data <- rec
		}
		close(data)
		wg.Done()
	}()

	// consumer
	go func() {
		for rec := range data {
			if rec.Action == "/query_submit" && rec.Logger == "QUERY" && rec.Args.hasAvancedQuery() {
				fmt.Println("HIT: ", rec)
				p, err := cql.ParseCQL("query@"+rec.Date, rec.Args.getFirstQuery())
				if err != nil {
					fmt.Printf("failed to parse %s with error: %s", rec.Args.getFirstQuery(), err)
					fmt.Println("   ... skipping")
				}
				var fts feats.Record
				fts.ImportFrom(p, 0) // 0 => actual value is added on the next line
				statsDb.AddRecord(rec.Args.getFirstQuery(), rec.Args.Corpora[0], fts, rec.GetTime(), rec.ProcTime)
			}
		}
		wg.Done()
	}()

	wg.Wait()
	return statsDb.CommitTx()
}
