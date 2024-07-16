package stats

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/czcorpus/cqlizer/feats"
	"github.com/rs/zerolog/log"
)

type DBRecord struct {
	ID        string
	Datetime  int
	Query     string
	Corpname  string
	ProcTime  float64
	BenchTime float64
}

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
			"benchTime FLOAT " +
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
	query := "SELECT id, datetime, query, corpname, procTime, benchTime " +
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

func (database *Database) GetAllRecords(onlyWithoutBenchmark bool) ([]DBRecord, error) {
	qTpl := "SELECT id, datetime, query, corpname, procTime, benchTime " +
		"FROM query_stats %s ORDER BY benchTime"
	var query string
	if onlyWithoutBenchmark {
		query = fmt.Sprintf(qTpl, "WHERE benchTime IS NULL")

	} else {
		query = fmt.Sprintf(qTpl, "")
	}
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

func (database *Database) AddRecord(query, corpname string, rec feats.Record, dt time.Time, procTime float64) (int64, error) {
	ans, err := database.db.Exec(
		"INSERT OR REPLACE INTO query_stats (id, datetime, query, corpname, procTime) "+
			"VALUES (?, ?, ?, ?, ?, ?)",
		IdempotentID(dt, query), dt.Unix(), query, corpname, procTime)
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
