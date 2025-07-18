package dataimport

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/czcorpus/cqlizer/index"
	"github.com/go-sql-driver/mysql"
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

func (cp *ConcPersistence) ImportDataChunk(ctx context.Context, fromDate, toDate time.Time) (chan CPResult, error) {
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
		for rows.Next() {
			var data string
			var freq int
			if err := rows.Scan(&data, &freq); err != nil {
				ans <- CPResult{
					Error: err,
				}
				return
			}
			ans <- CPResult{
				SearchResult: index.SearchResult{
					Value: data,
					Freq:  uint32(freq),
				},
			}
		}
	}()
	return ans, nil
}

func NewCNCMySQLHandler(
	host,
	user,
	pass,
	dbName,
	corporaTableName,
	pcTableName string) (*ConcPersistence, error) {
	conf := mysql.NewConfig()
	conf.Net = "tcp"
	conf.Addr = host
	conf.User = user
	conf.Passwd = pass
	conf.DBName = dbName
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
