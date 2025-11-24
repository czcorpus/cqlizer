// Copyright 2025 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2025 Department of Linguistics,
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

package dataimport

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/czcorpus/cqlizer/eval"
	"github.com/rs/zerolog/log"
)

type StatsFileProcessor interface {
	ProcessEntry(entry eval.QueryStatsRecord) error
	SetStats(numProcessed, numFailed int)
}

// ReadStatsFile reads a JSONL file where each line is a QueryStatsRecord
// and calls the processor for each entry.
func ReadStatsFile(ctx context.Context, filePath string, processor StatsFileProcessor) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	numProc := 0
	numFailed := 0
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			log.Warn().Msg("interrupting CQL file processing")
			return nil
		default:
		}
		lineNum++
		line := scanner.Bytes()

		// Skip empty lines
		if len(line) == 0 {
			continue
		}

		var record eval.QueryStatsRecord
		if err := json.Unmarshal(line, &record); err != nil {
			log.Error().Err(err).Int("line", lineNum).Msg("failed to parse JSON, skipping")
			continue
		}

		if err := processor.ProcessEntry(record); err != nil {
			log.Error().
				Err(err).
				Any("entry", record).
				Int("line", lineNum).
				Msg("failed to process CQL entry, skipping")
			numFailed++
			continue

		} else {
			numProc++
		}
	}

	for _, item := range eval.ObligatoryExamples {

		if err := processor.ProcessEntry(item); err != nil {
			log.Error().
				Err(err).
				Any("entry", item).
				Int("line", lineNum).
				Msg("failed to process CQL entry, skipping")
			numFailed++
			continue

		} else {
			numProc++
		}
	}

	processor.SetStats(numProc, numFailed)
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}
	fmt.Printf("Stats file processed. Num imported queries: %d, num failed: %d\n", numProc, numFailed)

	return nil
}
