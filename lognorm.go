package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/czcorpus/cqlizer/dataimport"
	"github.com/czcorpus/cqlizer/eval"
	"github.com/rs/zerolog/log"
)

type missingRecFixer struct {
	ctx                context.Context
	mqueryURL          string
	uniqEntries        map[string]eval.QueryStatsRecord
	onlyAllowedCorpora []string
	batchSize          int
	batchOffset        int
}

func (fixer *missingRecFixer) ProcessEntry(entry eval.QueryStatsRecord) error {
	if !slices.Contains(fixer.onlyAllowedCorpora, entry.Corpus) {
		return nil
	}
	if entry.TimeProc == 0 {
		_, ok := fixer.uniqEntries[entry.UniqKey()]
		if !ok {
			fixer.uniqEntries[entry.UniqKey()] = entry
		}
	}
	return nil
}

func (fixer *missingRecFixer) SetStats(numProcessed, numFailed int) {

}

func (fixer *missingRecFixer) RunBenchmark() {
	procEntries := make([]eval.QueryStatsRecord, len(fixer.uniqEntries))
	i := 0
	for _, entry := range fixer.uniqEntries {
		procEntries[i] = entry
		i++
	}
	slices.SortFunc(procEntries, func(v1, v2 eval.QueryStatsRecord) int {
		return strings.Compare(v1.UniqKey(), v2.UniqKey())
	})
	if fixer.batchSize == 0 {
		fixer.batchSize = len(procEntries)

	} else {
		fixer.batchSize = min(len(procEntries), fixer.batchSize)
	}
	for _, entry := range procEntries[fixer.batchOffset:fixer.batchSize] {
		select {
		case <-fixer.ctx.Done():
			return
		default:
		}
		t0, err := fixer.measureRequest(fixer.ctx, fixer.mqueryURL, entry.Corpus, entry.GetCQL())
		if err != nil {
			log.Error().Err(err).Msg("failed to perform benchmark query, skipping")
			continue
		}
		entry.TimeProc = t0
		entry.IsSynthetic = true
		data, err := json.Marshal(entry)
		if err != nil {
			log.Error().Err(err).Msg("failed to perform benchmark query, skipping")
			continue
		}
		fmt.Println(string(data))
	}
	log.Warn().Int("nextBatch", fixer.batchOffset+fixer.batchSize).Msg("Finished current batch")
}

func (fixer *missingRecFixer) measureRequest(
	ctx context.Context,
	mqueryURL, corpname, q string,
) (float64, error) {
	urlObj, err := url.Parse(mqueryURL)
	if err != nil {
		return -1, fmt.Errorf("cannot measure request: %w", err)
	}
	urlObj = urlObj.JoinPath(fmt.Sprintf("/concordance/%s", corpname))
	args := make(url.Values)
	args.Add("q", q)
	urlObj.RawQuery = args.Encode()
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		urlObj.String(),
		nil,
	)
	if err != nil {
		return -1, fmt.Errorf("failed to perform MQuery request: %w", err)
	}
	t0 := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return -1, fmt.Errorf("failed to perform MQuery request: %w", err)
	}
	if resp.StatusCode != 200 {
		return -1, fmt.Errorf("failed to perform mquery request - status %s", resp.Status)
	}
	return float64(time.Since(t0).Seconds()), nil
}

func runActionBenchmarkMissing(
	ctx context.Context,
	srcPath, mqueryURL string,
	onlyAllowedCorpora []string,
	batchSize int,
	batchOffset int,
) {
	fixer := &missingRecFixer{
		ctx:                ctx,
		mqueryURL:          mqueryURL,
		uniqEntries:        make(map[string]eval.QueryStatsRecord),
		onlyAllowedCorpora: onlyAllowedCorpora,
		batchSize:          batchSize,
		batchOffset:        batchOffset,
	}
	dataimport.ReadStatsFile(ctx, srcPath, fixer)
	log.Info().Msg("queries loaded and deduplicated")
	fixer.RunBenchmark()
}

// -------------------------------------------------

type zeroRemover struct {
	ctx          context.Context
	numProcessed int
	numZero      int
}

func (remover *zeroRemover) ProcessEntry(entry eval.QueryStatsRecord) error {
	if entry.TimeProc > 0 {
		data, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal eval.QueryStatsRecord record: %w", err)
		}
		fmt.Println(string(data))
		remover.numProcessed++

	} else {
		remover.numZero++
	}
	return nil
}

func (remover *zeroRemover) SetStats(numProcessed, numFailed int) {

}

func runActionRemoveZero(
	ctx context.Context,
	srcPath string,
) {
	rm := &zeroRemover{
		ctx: ctx,
	}
	dataimport.ReadStatsFile(ctx, srcPath, rm)
	log.Info().
		Int("numProcessed", rm.numProcessed).
		Int("numZero", rm.numZero).
		Msg("removed zero entries")
}
