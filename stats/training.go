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
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

func (database *Database) createTrainingTable() error {
	_, err := database.db.Exec(
		"CREATE TABLE training (" +
			"id INTEGER PRIMARY KEY NOT NULL, " +
			"threshold FLOAT, " +
			"precision FLOAT, " +
			"recall FLOAT " +
			")",
	)
	if err != nil {
		return fmt.Errorf("failed to create table training: %w", err)
	}
	log.Info().Msg("created table `training`")
	return nil
}

func (database *Database) createTrainingQSTable() error {
	_, err := database.db.Exec(
		"CREATE TABLE training_query_stats (" +
			"training_id datetime INTEGER NOT NULL, " +
			"query_stats_id TEXT, " +
			"is_validation INT NOT NULL DEFAULT 0, " +
			"result INT, " +
			"PRIMARY KEY(training_id, query_stats_id) " +
			")",
	)
	if err != nil {
		return fmt.Errorf("failed to create table training_query_stats: %w", err)
	}
	log.Info().Msg("created table `training_query_stats`")
	return nil
}

func (database *Database) CreateNewTraining(threshold float64) (int, error) {
	t0 := time.Now()
	ans, err := database.db.Exec("INSERT INTO training (id, threshold) VALUES (?, ?)",
		t0.Unix(), threshold)
	if err != nil {
		return -1, fmt.Errorf("failed to create new training: %w", err)
	}
	v, err := ans.LastInsertId()
	if err != nil {
		return -1, fmt.Errorf("failed to create new training: %w", err)
	}
	return int(v), nil
}

func (database *Database) SetTrainingQuery(training_id int, qid string) error {
	_, err := database.db.Exec(
		"INSERT INTO training_query_stats (training_id, query_stats_id, is_validation) "+
			"VALUES (?, ?, 0)", training_id, qid)
	if err != nil {
		return fmt.Errorf("failed to set new training query: %w", err)
	}
	return nil
}

func (database *Database) SetValidationQuery(training_id int, qid string, result bool) error {
	r := 0
	if result {
		r = 1
	}
	_, err := database.db.Exec(
		"INSERT INTO training_query_stats (training_id, query_stats_id, is_validation, result) "+
			"VALUES (?, ?, 1, ?)", training_id, qid, r)
	if err != nil {
		return fmt.Errorf("failed to set new validation query: %w", err)
	}
	return nil
}

type TrainingResult struct {
	IsValidation bool
	Prediction   int
	Truth        int
	BenchTime    float64
	Query        string
	QueryID      string
}

func (database *Database) GetTrainingThreshold(trainingId int) (float64, error) {
	row := database.db.QueryRow("SELECT threshold FROM training WHERE id = ?", trainingId)
	var ans sql.NullFloat64
	err := row.Scan(&ans)
	if err == sql.ErrNoRows {
		return -1, fmt.Errorf("training not found: %w", err)

	} else if err != nil {
		return -1, fmt.Errorf("failed to get training threshold: %w", err)
	}
	if ans.Valid {
		return ans.Float64, nil
	}
	return -1, nil
}

func (database *Database) GetTrainingValidationData(trainingID int) ([]TrainingResult, error) {
	rows, err := database.db.Query(
		"SELECT tqs.is_validation, qs.benchTime, qs.query, tqs.result, "+
			"qs.benchTime > t.threshold, qs.id "+
			"FROM training AS t "+
			"JOIN training_query_stats AS tqs ON t.id = tqs.training_id "+
			"JOIN query_stats AS qs on qs.id = tqs.query_stats_id "+
			"WHERE tqs.training_id = ? AND tqs.is_validation = 1",
		trainingID,
	)
	if err != nil {
		return []TrainingResult{}, fmt.Errorf("failed to get validation data: %w", err)
	}
	ans := make([]TrainingResult, 0, 1000)
	for rows.Next() {
		var v TrainingResult
		err := rows.Scan(&v.IsValidation, &v.BenchTime, &v.Query, &v.Prediction, &v.Truth, &v.QueryID)
		if err != nil {
			return []TrainingResult{}, fmt.Errorf("failed to get validation data: %w", err)
		}
		ans = append(ans, v)
	}
	return ans, nil
}

func (database *Database) GetTrainingData(trainingId int) ([]DBRecord, error) {
	rows, err := database.db.Query(
		"SELECT qs.id, qs.datetime, qs.query, qs.corpname, "+
			"qs.benchTime "+
			"FROM query_stats AS qs "+
			"JOIN training_query_stats AS tqs ON qs.id = tqs.query_stats_id "+
			" WHERE tqs.training_id = ? AND qs.benchTime IS NOT NULL AND tqs.is_validation = 0 "+
			"ORDER BY qs.benchTime",
		trainingId,
	)
	if err != nil {
		return []DBRecord{}, fmt.Errorf("failed to fetch all records: %w", err)
	}
	ans := make([]DBRecord, 0, 1000)
	for rows.Next() {
		var rec DBRecord
		err := rows.Scan(
			&rec.ID,
			&rec.Datetime,
			&rec.Query,
			&rec.Corpname,
			&rec.BenchTime,
		)
		if err != nil {
			return []DBRecord{}, fmt.Errorf("failed to fetch all records: %w", err)
		}
		ans = append(ans, rec)
	}
	return ans, nil
}
