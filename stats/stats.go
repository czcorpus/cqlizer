package stats

import (
	"bufio"
	"database/sql"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/czcorpus/cqlizer/feats"
	"github.com/rs/zerolog/log"
)

type Database struct {
	db         *sql.DB
	tx         *sql.Tx
	sizesCache map[string]int
}

func (database *Database) CreateQueryStatsTable() error {
	_, err := database.db.Exec(
		"CREATE TABLE query_stats (" +
			"id TEXT PRIMARY KEY NOT NULL, " +
			"datetime INTEGER NOT NULL, " +
			"query TEXT NOT NULL, " +
			"corpname TEXT NOT NULL, " +
			"procTime FLOAT NOT NULL," +
			"ptPercentile INTEGER, " +
			"featsJSON TEXT" +
			")",
	)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}
	log.Info().Msg("created table `query_stats`")
	return nil
}

func (database *Database) CreateCorpusSizeTable() error {
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

func (database *Database) Init() error {
	ans := database.db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='table' AND name='query_stats'")
	var nm sql.NullString
	err := ans.Scan(&nm)
	if err == sql.ErrNoRows {
		if err := database.CreateQueryStatsTable(); err != nil {
			return fmt.Errorf("failed to create table `query_stats`: %w", err)
		}

	} else if err != nil {
		return fmt.Errorf("failed to determine existence of `query_stats`: %w", err)
	}
	log.Info().Msg("table `query_stats` already exists")

	ans = database.db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='table' AND name='corpus_size'")
	err = ans.Scan(&nm)
	if err == sql.ErrNoRows {
		return database.CreateCorpusSizeTable()

	} else if err != nil {
		return fmt.Errorf("failed to determine existence of `corpus_size`: %w", err)
	}

	log.Info().Msg("table `corpus_size` already exists")

	return nil
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

func (database *Database) AddRecord(query, corpname string, rec feats.Record, dt time.Time, procTime float64) (int64, error) {
	if rec.CorpusSize == 0 {
		rec.CorpusSize = database.GetCorpusSize(corpname)
	}
	ans, err := database.db.Exec(
		"INSERT OR REPLACE INTO query_stats (id, datetime, query, corpname, procTime, featsJSON) "+
			"VALUES (?, ?, ?, ?, ?, ?)",
		IdempotentID(dt, query), dt.Unix(), query, corpname, procTime, rec.AsJSONString())
	if err != nil {
		return -1, fmt.Errorf("failed to add record: %w", err)
	}
	lastID, err := ans.LastInsertId()
	if err != nil {
		return -1, fmt.Errorf("failed to add record: %w", err)
	}
	return lastID, err
}

func (database *Database) RecalculatePercentiles() error {
	tx, err := database.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to recalculate percentiles: %w", err)
	}

	row := tx.QueryRow("SELECT COUNT(*) FROM query_stats")
	var total int
	err = row.Scan(&total)
	if err != nil {
		return fmt.Errorf("failed to recalculate percentiles: %w", err)
	}

	rows, err := tx.Query("SELECT id FROM query_stats ORDER BY procTime DESC")
	if err != nil {
		return fmt.Errorf("failed to recalculate percentiles: %w", err)
	}
	var i int
	for rows.Next() {
		var id string
		err := rows.Scan(&id)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to recalculate percentiles: %w", err)
		}
		perc := int(math.RoundToEven(float64(i) / float64(total) * 100))
		fmt.Println(">>> ABOUTTO UPD ", id, perc)
		tx.Exec("UPDATE query_stats SET ptPercentile = ? WHERE id = ?", perc, id)
		i++
	}
	return tx.Commit()
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
