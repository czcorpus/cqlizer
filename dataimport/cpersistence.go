package dataimport

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/czcorpus/cqlizer/cql"
	"github.com/czcorpus/cqlizer/embedding"
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

func (cp *ConcPersistence) importJSONRecord(v string) (queryPersistenceRecord, error) {
	var rec queryPersistenceRecord
	if err := json.Unmarshal([]byte(v), &rec); err != nil {
		return queryPersistenceRecord{}, nil
	}
	return rec, nil
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
		go func() {
			close(ans)
		}()
		return ans, fmt.Errorf("failed to fetch stored queries: %w", err)
	}
	go func() {
		defer close(ans)
		i := 0
		ignored := 0
		for rows.Next() {
			var data string
			var freq int
			if err := rows.Scan(&data, &freq); err != nil {
				ans <- CPResult{
					Error: err,
				}
				return
			}
			rec, err := cp.importJSONRecord(data)
			if err != nil {
				// this likely means we're dealing with a different type of query
				// (the one we're not interested in)
				// TODO but it can also hide different kind of errors
				ignored++
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
			Int("numProcessed", i).
			Int("numIgnored", ignored).
			Msg("imported queries chunk from kontext_conc_persistence")
	}()
	return ans, nil
}

const (
	importLastPositionKey = "importLastPosition"
	defaultStartDate      = "2014-01-01"
	importChunkDuration   = 30 * 24 * time.Hour // one week
)

func importChunk(db *index.DB, chunk []CPResult, w2vSrcPath string, model *embedding.CQLEmbedder) error {
	err := db.Update(func(txn *badger.Txn) error {
		for _, result := range chunk {
			if result.Error != nil {
				return fmt.Errorf("error reading data chunk: %w", result.Error)
			}
			if len(result.SearchResult.Value) > 700 {
				log.Warn().Str("query", result.SearchResult.Value).Msg("ignoring too long query")
				continue
			}
			parsed, parseErr := cql.ParseCQL("persistence_import", result.SearchResult.Value)
			if parseErr != nil {
				log.Error().
					Str("query", result.SearchResult.Value).
					Err(parseErr).
					Msg("failed to parse query")
				continue
			}
			abstractQuery := parsed.Normalize()
			if err := AppendToFile(
				w2vSrcPath,
				abstractQuery,
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
			if model != nil {
				vec, err := model.CreateEmbedding(abstractQuery)
				if err != nil {
					log.Error().Err(err).Msg("failed to evaluate query via w2v model")
					continue
				}
				if err := db.StoreEmbeddingTx(
					txn,
					abstractQuery, // Use abstract query as key, not original
					vec.Vector,
				); err != nil {
					log.Error().Err(err).Msg("failed to store embedding")
					continue
				}
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update index with imported data: %w", err)
	}
	return nil
}

func ImportFromConcPersistence(
	ctx context.Context,
	concPers *ConcPersistence,
	db *index.DB,
	w2vSourceFilePath string,
	w2vModel *embedding.CQLEmbedder,
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
	buff := make([]CPResult, 100)
	i := 0
	for item := range resultChan {
		if i < len(buff) {
			buff[i] = item
			i++

		} else {
			if err := importChunk(db, buff, w2vSourceFilePath, w2vModel); err != nil {
				log.Error().Err(err).Msg("failed to import chunk of data")
			}
			i = 0
		}
	}
	if i > 0 {
		if err := importChunk(db, buff[:i], w2vSourceFilePath, w2vModel); err != nil {
			log.Error().Err(err).Msg("failed to import chunk of data")
		}
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
