// Copyright 2026 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2026 Institute of the Czech National Corpus,
//                Faculty of Arts, Charles University
//   This file is part of MQUERY.
//
//  MQUERY is free software: you can redistribute it and/or modify
//  it under the terms of the GNU General Public License as published by
//  the Free Software Foundation, either version 3 of the License, or
//  (at your option) any later version.
//
//  MQUERY is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of
//  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//  GNU General Public License for more details.
//
//  You should have received a copy of the GNU General Public License
//  along with MQUERY.  If not, see <https://www.gnu.org/licenses/>.

package monitoring

import (
	"context"
	"time"

	"github.com/czcorpus/hltscl"
	"github.com/rs/zerolog/log"
)

/*
Expected tables:

create table cqlizer_queries_evaluations (
  "time" timestamp with time zone NOT NULL,
  votes_for int,
  votes_against int,
  avg_certainty float,
  corpus text,
  num_errors int
);
select create_hypertable('cqlizer_queries_evaluations', 'time');

*/

type VoteReport struct {
	VotesFor     int
	VotesAgainst int
	AvgCertainty float64
	Corpus       string
	IsError      bool
}

// ------------

type StatusWriter interface {
	Write(rec VoteReport)
}

// ------------

type NullStatusWriter struct{}

func (n *NullStatusWriter) Write(rec VoteReport) {}

// -------------

type TimescaleDBWriter struct {
	tableWriter *hltscl.TableWriter
	opsDataCh   chan<- hltscl.Entry
	errCh       <-chan hltscl.WriteError
	location    *time.Location
}

func (sw *TimescaleDBWriter) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("about to close StatusWriter")
				return
			case err := <-sw.errCh:
				log.Error().
					Err(err.Err).
					Str("entry", err.Entry.String()).
					Str("table", "cqlizer_queries_evaluations").
					Msg("error writing data to TimescaleDB")
			}
		}
	}()
}

func (sw *TimescaleDBWriter) Stop(ctx context.Context) error {
	log.Warn().Msg("stopping StatusWriter")
	return nil
}

func (sw *TimescaleDBWriter) Write(item VoteReport) {
	if sw.tableWriter != nil {
		var numErr int
		if item.IsError {
			numErr = 1
		}
		sw.opsDataCh <- *sw.tableWriter.NewEntry(time.Now().In(sw.location)).
			Int("votes_for", item.VotesFor).
			Int("votes_against", item.VotesAgainst).
			Float("avg_certainty", item.AvgCertainty).
			Int("num_errors", numErr).
			Str("corpus", item.Corpus)
	}
}

func NewTimescaleDBWriter(
	ctx context.Context,
	conf hltscl.PgConf,
	tz *time.Location,
	onError func(err error),
) (*TimescaleDBWriter, error) {

	conn, err := hltscl.CreatePool(conf)
	if err != nil {
		return nil, err
	}
	twriter := hltscl.NewTableWriter(conn, "cqlizer_queries_evaluations", "time", tz)
	opsDataCh, errCh := twriter.Activate(
		ctx,
		hltscl.WithTimeout(20*time.Second),
	)

	return &TimescaleDBWriter{
		tableWriter: twriter,
		opsDataCh:   opsDataCh,
		errCh:       errCh,
		location:    tz,
	}, nil
}
