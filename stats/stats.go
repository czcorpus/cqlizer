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

package stats

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
)

type Database struct {
	db         *sql.DB
	tx         *sql.Tx
	sizesCache map[string]int
}

func (database *Database) createQueryStatsTable() error {
	_, err := database.db.Exec(
		"CREATE TABLE query_stats (" +
			"id TEXT PRIMARY KEY NOT NULL, " +
			"datetime INTEGER NOT NULL, " +
			"query TEXT NOT NULL, " +
			"corpname TEXT NOT NULL, " +
			"procTime FLOAT NOT NULL," +
			"benchTime FLOAT, " +
			"trainingExclude INT NOT NULL DEFAULT 1" +
			")",
	)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}
	log.Info().Msg("created table `query_stats`")
	return nil
}

func (database *Database) createCorpusSizeTable() error {
	_, err := database.db.Exec(
		"CREATE TABLE corpus_size (" +
			"id TEXT PRIMARY KEY NOT NULL, " +
			"size INTEGER NOT NULL" +
			")",
	)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	log.Info().Msg("created table `corpus_size`")
	return nil
}

func (database *Database) AddBenchmarkResult(id string, dur time.Duration) error {
	tx, err := database.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to add benchmark result: %w", err)
	}
	_, err = tx.Exec(
		"UPDATE query_stats SET benchTime = ? WHERE id = ?",
		dur.Seconds(),
		id,
	)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to add benchmark result: %w", err)
	}
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to add benchmark result: %w", err)
	}
	return nil
}

func (database *Database) GetCzechBenchmarkedRecords() ([]DBRecord, error) {
	query := "SELECT id, datetime, query, corpname, procTime, benchTime, trainingExclude " +
		"FROM query_stats " +
		"ORDER BY benchTime"
	rows, err := database.db.Query(query)
	if err != nil {
		return []DBRecord{}, fmt.Errorf("failed to fetch all records: %w", err)
	}
	ans := make([]DBRecord, 0, 500)
	for rows.Next() {
		var rec DBRecord
		var benchTime sql.NullFloat64
		err := rows.Scan(
			&rec.ID,
			&rec.Datetime,
			&rec.Query,
			&rec.Corpname,
			&rec.ProcTime,
			&benchTime,
			&rec.TrainingExclude,
		)
		if err != nil {
			return []DBRecord{}, fmt.Errorf("failed to fetch all records: %w", err)
		}
		if benchTime.Valid {
			rec.BenchTime = benchTime.Float64
		}
		ans = append(ans, rec)
	}
	return ans, nil
}

// GetAllRecords loads stats records containing imported queries with their
// benchmark times (if already benchmarked).
func (database *Database) GetAllRecords(filter ListFilter) ([]DBRecord, error) {
	query := "SELECT id, datetime, query, corpname, procTime, benchTime, trainingExclude " +
		"FROM query_stats WHERE %s ORDER BY benchTime"
	whereChunks := make([]string, 0, 3)
	whereChunks = append(whereChunks, "1 = 1")
	if filter.Benchmarked != nil {
		if *filter.Benchmarked {
			whereChunks = append(whereChunks, "benchTime IS NOT NULL")

		} else {
			whereChunks = append(whereChunks, "benchTime IS NULL")
		}
	}
	if filter.TrainingExcluded != nil {
		if *filter.TrainingExcluded {
			whereChunks = append(whereChunks, "trainingExclude = 1")

		} else {
			whereChunks = append(whereChunks, "trainingExclude = 0")
		}
	}

	rows, err := database.db.Query(fmt.Sprintf(query, strings.Join(whereChunks, " AND ")))
	if err != nil {
		return []DBRecord{}, fmt.Errorf("failed to fetch all records: %w", err)
	}
	ans := make([]DBRecord, 0, 500)
	for rows.Next() {
		var rec DBRecord
		var benchTime sql.NullFloat64
		err := rows.Scan(
			&rec.ID,
			&rec.Datetime,
			&rec.Query,
			&rec.Corpname,
			&rec.ProcTime,
			&benchTime,
			&rec.TrainingExclude,
		)
		if err != nil {
			return []DBRecord{}, fmt.Errorf("failed to fetch all records: %w", err)
		}
		if benchTime.Valid {
			rec.BenchTime = benchTime.Float64
		}
		ans = append(ans, rec)
	}
	return ans, nil
}

func (database *Database) tableExists(tn string) (bool, error) {
	ans := database.db.QueryRow(
		fmt.Sprintf("SELECT name FROM sqlite_master WHERE type='table' AND name='%s'", tn))
	var nm sql.NullString
	err := ans.Scan(&nm)
	if err == sql.ErrNoRows {
		return false, nil

	} else if err != nil {
		return false, fmt.Errorf("failed to determine existence of table %s: %w", tn, err)
	}
	return true, nil
}

func (database *Database) Init() error {
	ex, err := database.tableExists("query_stats")
	if err != nil {
		return fmt.Errorf("failed to init table query_stats: %w", err)
	}
	if ex {
		log.Info().Str("table", "query_stats").Msg("table already exists")

	} else {
		if err := database.createQueryStatsTable(); err != nil {
			return fmt.Errorf("failed to create table query_stats: %w", err)
		}
	}

	ex, err = database.tableExists("corpus_size")
	if err != nil {
		return fmt.Errorf("failed to init table corpus_size: %w", err)
	}
	if ex {
		log.Info().Str("table", "corpus_size").Msg("table already exists")

	} else {
		if err := database.createCorpusSizeTable(); err != nil {
			return fmt.Errorf("failed to create table corpus_size: %w", err)
		}
	}

	ex, err = database.tableExists("training")
	if err != nil {
		return fmt.Errorf("failed to init table training: %w", err)
	}
	if ex {
		log.Info().Str("table", "training").Msg("table already exists")

	} else {
		if err := database.createTrainingTable(); err != nil {
			return fmt.Errorf("failed to create table training: %w", err)
		}
	}

	ex, err = database.tableExists("training_query_stats")
	if err != nil {
		return fmt.Errorf("failed to init table training_query_stats: %w", err)
	}
	if ex {
		log.Info().Str("table", "training_query_stats").Msg("table already exists")

	} else {
		if err := database.createTrainingQSTable(); err != nil {
			return fmt.Errorf("failed to create table training_query_stats: %w", err)
		}
	}

	return nil
}

func (database *Database) GetQueryAvgBenchTime(q string) (float64, error) {
	rows := database.db.QueryRow("SELECT AVG(benchTime) FROM query_stats WHERE query = ?", q)
	var ans sql.NullFloat64
	if err := rows.Scan(&ans); err != nil {
		return -1, err
	}
	if ans.Valid {
		return ans.Float64, nil
	}
	return -1, nil
}

func (database *Database) GetCorpusSize(corpname string) int {
	ans, cached := database.sizesCache[corpname]
	if cached {
		return ans
	}
	row := database.db.QueryRow("SELECT size FROM corpus_size WHERE id = ?", corpname)
	var size int
	err := row.Scan(&size)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Error().Err(err).Msg("Failed to fetch corpus size from database")
		}
		database.sizesCache[corpname] = 0
		return 0
	}
	database.sizesCache[corpname] = size
	return size
}

func (database *Database) StartTx() error {
	if database.tx != nil {
		panic("a transaction is already running")
	}
	var err error
	database.tx, err = database.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	return nil
}

func (database *Database) CommitTx() error {
	if database.tx == nil {
		panic("no transaction running")
	}
	err := database.tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (database *Database) RollbackTx() error {
	if database.tx == nil {
		panic("no transaction running")
	}
	err := database.tx.Rollback()
	if err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}
	return nil
}

func (database *Database) AddRecord(rec DBRecord) (int64, error) {
	ans, err := database.db.Exec(
		"INSERT OR REPLACE INTO query_stats (id, datetime, query, corpname, procTime, trainingExclude) "+
			"VALUES (?, ?, ?, ?, ?, ?)",
		IdempotentID(time.Unix(rec.Datetime, 0), rec.Query),
		rec.Datetime,
		rec.Query,
		rec.Corpname,
		rec.ProcTime,
		rec.TrainingExclude,
	)
	if err != nil {
		return -1, fmt.Errorf("failed to add record: %w", err)
	}
	lastID, err := ans.LastInsertId()
	if err != nil {
		return -1, fmt.Errorf("failed to add record: %w", err)
	}
	return lastID, err
}

func (database *Database) ImportCorpusSizesFromCSV(path string) error {
	fr, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to import CSV corpsizes: %w", err)
	}
	defer fr.Close()
	scnr := bufio.NewScanner(fr)
	tx, err := database.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to import CSV corpsizes: %w", err)
	}
	for scnr.Scan() {
		tmp := strings.Split(scnr.Text(), ";")
		_, err := tx.Exec("INSERT INTO corpus_size (id, size) VALUES (?, ?)", tmp[0], tmp[1])
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to import CSV corpsizes: %w", err)
		}
	}
	return tx.Commit()
}

func NewDatabase(path string) (*Database, error) {
	dbConn, err := sql.Open("sqlite3", "file:"+path)
	if err != nil {
		return nil, fmt.Errorf("failed to open stats datase: %w", err)
	}
	return &Database{
		sizesCache: make(map[string]int),
		db:         dbConn,
	}, nil
}
