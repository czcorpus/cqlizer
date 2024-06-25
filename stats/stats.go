package stats

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/czcorpus/cqlizer/feats"
	"github.com/rs/zerolog/log"
)

type Database struct {
	db *sql.DB
}

func (database *Database) CreateTables() error {
	_, err := database.db.Exec(
		"CREATE TABLE query_stats (" +
			"id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, " +
			"datetime INTEGER NOT NULL, " +
			"query TEXT NOT NULL, " +
			"procTime FLOAT NOT NULL," +
			"featsJSON TEXT" +
			")",
	)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}
	log.Info().Msg("created new database")
	return nil
}

func (database *Database) Init() error {
	ans := database.db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='table' AND name='query_stats'")
	var nm sql.NullString
	err := ans.Scan(&nm)
	if err == sql.ErrNoRows {
		return database.CreateTables()

	} else {
		log.Info().Msg("working database already exists")
	}
	if err != nil {
		return fmt.Errorf("failed to determine working data tables existence: %w", err)
	}
	return nil
}

func (database *Database) AddRecord(query string, rec feats.Record, procTime float64) (int64, error) {
	ans, err := database.db.Exec(
		"INSERT INTO query_stats (datetime, query, procTime, featsJSON) "+
			"VALUES (?, ?, ?, ?)", time.Now().Unix(), query, procTime, rec.AsJSONString())
	if err != nil {
		return -1, fmt.Errorf("failed to add record: %w", err)
	}
	lastID, err := ans.LastInsertId()
	if err != nil {
		return -1, fmt.Errorf("failed to add record: %w", err)
	}
	return lastID, err
}

func NewDatabase(db *sql.DB) *Database {
	return &Database{db: db}
}
