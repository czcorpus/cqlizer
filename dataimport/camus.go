package dataimport

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/czcorpus/cqlizer/eval"
	"github.com/rs/zerolog/log"
)

type StatsFileProcessor interface {
	ProcessEntry(entry eval.QueryStatsRecord) error
}

// ReadStatsFile reads a JSONL file where each line is a QueryStatsRecord
// and calls the processor for each entry.
func ReadStatsFile(filePath string, processor StatsFileProcessor) error {
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

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	fmt.Printf("Stats file processed. Num imported queries: %d, num failed: %d\n", numProc, numFailed)

	return nil
}
