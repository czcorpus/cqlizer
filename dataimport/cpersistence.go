package dataimport

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/czcorpus/cqlizer/cql"
	"github.com/czcorpus/cqlizer/index"
	"github.com/dgraph-io/badger/v4"
	"github.com/go-sql-driver/mysql"
	"github.com/rs/zerolog/log"
)

var (
	ErrNoDataToImport = errors.New("no data to import found")
)

type DBConf struct {
	Host   string `json:"host"`
	User   string `json:"user"`
	Passwd string `json:"passwd"`
	Name   string `json:"db"`
}

type CPResult struct {
	index.SearchResult
	Error error
}

type ConcPersistence struct {
	conn *sql.DB
}

func (cp *ConcPersistence) importDataChunk(ctx context.Context, fromDate, toDate time.Time) (chan CPResult, error) {
	rows, err := cp.conn.QueryContext(
		ctx,
		"SELECT data, num_access FROM kontext_conc_persistence WHERE created BETWEEN ? AND ?",
		fromDate,
		toDate,
	)
	ans := make(chan CPResult, 100)
	if err != nil {
		return ans, fmt.Errorf("failed to fetch stored queries: %w", err)
	}
	go func() {
		defer close(ans)
		i := 0
		for rows.Next() {
			var data string
			var freq int
			if err := rows.Scan(&data, &freq); err != nil {
				ans <- CPResult{
					Error: err,
				}
				return
			}

			var rec queryPersistenceRecord
			if err := json.Unmarshal([]byte(data), &rec); err != nil {
				// this likely means we're dealing with a different type of query
				// (the one we're not interested in)
				continue
			}
			for _, q := range rec.AdvancedQueries() {
				ans <- CPResult{
					SearchResult: index.SearchResult{
						Value: q,
						Freq:  uint32(freq),
					},
				}
				i++
			}
		}
		log.Info().
			Time("fromDate", fromDate).
			Time("toDate", toDate).
			Int("numberOfItems", i).
			Msg("imported queries chunk from kontext_conc_persistence")
	}()
	return ans, nil
}

const (
	importLastPositionKey = "importLastPosition"
	defaultStartDate      = "2014-01-01"
	importChunkDuration   = 7 * 24 * time.Hour // one week
)

func ImportFromConcPersistence(
	ctx context.Context,
	concPers *ConcPersistence,
	db *index.DB,
	w2vSourceFilePath string,
	fromDateUser string,
) error {
	var fromDate time.Time
	var err error
	if fromDateUser != "" {
		fromDate, err = time.Parse("2006-01-02", fromDateUser)
		if err != nil {
			return fmt.Errorf("failed to parse user-defined start date: %w", err)
		}

	} else {
		fromDate, err = db.ReadTimestamp(importLastPositionKey)
		if err != nil {
			fromDate, err = time.Parse("2006-01-02", defaultStartDate)
			if err != nil {
				return fmt.Errorf("failed to parse default start date: %w", err)
			}
		}
	}
	toDate := fromDate.Add(importChunkDuration)
	now := time.Now()
	if toDate.After(now) {
		toDate = now
	}
	if fromDate.After(now) || fromDate.Equal(toDate) {
		return ErrNoDataToImport
	}

	resultChan, err := concPers.importDataChunk(ctx, fromDate, toDate)
	if err != nil {
		return fmt.Errorf("failed to import data chunk: %w", err)
	}

	err = db.Update(func(txn *badger.Txn) error {
		for result := range resultChan {
			if result.Error != nil {
				return fmt.Errorf("error reading data chunk: %w", result.Error)
			}
			parsed, parseErr := cql.ParseCQL("persistence_import", result.SearchResult.Value)
			if parseErr != nil {
				log.Error().
					Str("query", result.SearchResult.Value).
					Err(parseErr).
					Msg("failed to parse query")
				continue
			}
			if err := AppendToFile(
				w2vSourceFilePath,
				parsed.Normalize(),
			); err != nil {
				log.Error().
					Err(err).
					Msg("failed to store query to the auxiliary file, ignoring")
			}
			if err := db.StoreQueryTx(
				txn,
				result.SearchResult.Value,
				result.SearchResult.Freq,
			); err != nil {
				return fmt.Errorf("failed to store query: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update index with imported data: %w", err)
	}

	if err := db.StoreTimestamp(importLastPositionKey, toDate); err != nil {
		return fmt.Errorf("failed to store last import position: %w", err)
	}

	return nil
}

func NewConcPersistence(
	dbConf DBConf,
) (*ConcPersistence, error) {
	conf := mysql.NewConfig()
	conf.Net = "tcp"
	conf.Addr = dbConf.Host
	conf.User = dbConf.User
	conf.Passwd = dbConf.Passwd
	conf.DBName = dbConf.Name
	conf.ParseTime = true
	conf.Loc = time.Local
	db, err := sql.Open("mysql", conf.FormatDSN())
	if err != nil {
		return nil, err
	}
	return &ConcPersistence{
		conn: db,
	}, nil
}
