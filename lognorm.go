package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/czcorpus/cnc-gokit/collections"
	"github.com/czcorpus/cqlizer/cnf"
	"github.com/czcorpus/cqlizer/dataimport"
	"github.com/czcorpus/cqlizer/eval"
	"github.com/czcorpus/cqlizer/eval/feats"
	"github.com/rs/zerolog/log"
)

func getMQueryCorpora(mqueryURL string) ([]string, error) {
	urlObj, err := url.Parse(mqueryURL)
	if err != nil {
		return []string{}, fmt.Errorf("cannot measure request: %w", err)
	}
	urlObj = urlObj.JoinPath("/corplist")
	resp, err := http.Get(urlObj.String())
	if err != nil {
		return []string{}, fmt.Errorf("failed to fetch installed corpora from MQuery: %w", err)
	}
	var respObj corporaResp
	rawResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return []string{}, fmt.Errorf("failed to fetch installed corpora from MQuery: %w", err)
	}
	if err := json.Unmarshal(rawResp, &respObj); err != nil {
		return []string{}, fmt.Errorf("failed to fetch installed corpora from MQuery: %w", err)
	}
	ans := make([]string, len(respObj.Corpora))
	for i, rc := range respObj.Corpora {
		ans[i] = rc.ID
	}
	return ans, nil
}

type missingRecFixer struct {
	ctx                context.Context
	mqueryURL          string
	uniqEntries        map[string]eval.QueryStatsRecord
	onlyAllowedCorpora []string
	batchSize          int
	batchOffset        int
	corporaProps       map[string]feats.CorpusProps
}

func (fixer *missingRecFixer) ProcessEntry(entry eval.QueryStatsRecord) error {
	if !slices.Contains(fixer.onlyAllowedCorpora, entry.Corpus) {
		return nil
	}
	if entry.TimeProc == 0 || entry.CorpusSize == 0 {
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
	var isIncompleteProc bool
	if fixer.batchSize == 0 {
		fixer.batchSize = len(procEntries)

	} else {
		if len(procEntries) > fixer.batchOffset+fixer.batchSize {
			isIncompleteProc = true
		}
		fixer.batchSize = min(len(procEntries), fixer.batchSize)
	}
	for _, entry := range procEntries[fixer.batchOffset:fixer.batchSize] {
		select {
		case <-fixer.ctx.Done():
			return
		default:
		}

		if entry.CorpusSize == 0 && entry.Corpus != "" { // legacy records with just corpnames
			entry.CorpusSize = int64(fixer.corporaProps[entry.Corpus].Size)
		}
		if entry.TimeProc > 0 {
			continue
			// we are also dealing with records with just missing "CorpusSize" property which is fixable
			// without benchmarking
		}

		if entry.CorpusSize == 0 {
			log.Warn().Str("corpname", entry.Corpus).Str("q", entry.Query).Msg("entry ignored due to missing corpus size")
			continue
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
	if isIncompleteProc {
		fmt.Fprintf(
			os.Stderr,
			"Finished the current batch (%d ... %d), More data are available (next offset: %d)",
			fixer.batchOffset,
			fixer.batchSize-1,
			fixer.batchOffset+fixer.batchSize,
		)
	} else {
		fmt.Fprintf(
			os.Stderr,
			"Finished the current batch (%d ... %d). No more data are available.",
			fixer.batchOffset,
			fixer.batchSize-1,
		)
	}
}

type corporaRespCorpus struct {
	ID string `json:"id"`
}

type corporaResp struct {
	Corpora []corporaRespCorpus `json:"corpora"`
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
		return -1, fmt.Errorf("failed to perform MQuery search (corpus: %s, q: %s): %w", corpname, q, err)
	}
	t0 := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return -1, fmt.Errorf("failed to perform MQuery search (corpus: %s, q: %s): %w", corpname, q, err)
	}
	if resp.StatusCode != 200 {
		return -1, fmt.Errorf("failed to perform MQuery search (corpus: %s, q: %s) - status %s", corpname, q, resp.Status)
	}
	return float64(time.Since(t0).Seconds()), nil
}

func runActionBenchmarkMissing(
	ctx context.Context,
	conf *cnf.Conf,
	srcPath string,
	onlyAllowedCorpora []string,
	batchSize int,
	batchOffset int,
) {
	if len(onlyAllowedCorpora) == 0 {
		mqCorpora, err := getMQueryCorpora(conf.MQueryBenchmarkingURL)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to run benchmark action")
			return
		}
		onlyAllowedCorpora = mqCorpora
	}
	fmt.Fprintln(os.Stderr, "Only queries for the following corpora will be tested:")
	for _, v := range onlyAllowedCorpora {
		fmt.Fprintf(os.Stderr, "\t%s\n", v)
	}
	fixer := &missingRecFixer{
		ctx:                ctx,
		mqueryURL:          conf.MQueryBenchmarkingURL,
		uniqEntries:        make(map[string]eval.QueryStatsRecord),
		onlyAllowedCorpora: onlyAllowedCorpora,
		batchSize:          batchSize,
		batchOffset:        batchOffset,
		corporaProps:       conf.CorporaProps,
	}
	dataimport.ReadStatsFile(ctx, srcPath, fixer)
	fmt.Fprintf(os.Stderr, "queries loaded and deduplicated, num processable queries: %d\n", len(fixer.uniqEntries))
	fixer.RunBenchmark()
}

// -------------------------------------------------

type zeroRemover struct {
	ctx          context.Context
	numProcessed int
	numZero      int
	foundCorpora map[string]int
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
		remover.foundCorpora[entry.Corpus]++
	}
	return nil
}

func (remover *zeroRemover) SetStats(numProcessed, numFailed int) {

}

type corpAndSize struct {
	c string
	s int
}

func runActionRemoveZero(
	ctx context.Context,
	conf *cnf.Conf,
	srcPath string,
) {
	rm := &zeroRemover{
		ctx:          ctx,
		foundCorpora: make(map[string]int),
	}

	var mqueryCorpora []string
	var err error
	if conf.MQueryBenchmarkingURL != "" {
		mqueryCorpora, err = getMQueryCorpora(conf.MQueryBenchmarkingURL)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get MQuery corpora, skipping the feature")
		}
	}

	dataimport.ReadStatsFile(ctx, srcPath, rm)
	corpora := collections.MapToEntriesSorted(
		rm.foundCorpora,
		func(a, b collections.MapEntry[string, int]) int {
			return b.V - a.V
		},
	)
	corpora2 := make([]collections.MapEntry[string, int], 0, len(corpora))
	for _, corp := range corpora {
		if slices.ContainsFunc(mqueryCorpora, func(v string) bool {
			return v == corp.K
		}) || len(mqueryCorpora) == 0 {
			corpora2 = append(corpora2, corp)
		}
	}
	fmt.Fprintln(os.Stderr, "\nCorpora with benchmarkable zero time requests:")
	if conf.MQueryBenchmarkingURL == "" {
		fmt.Fprintln(os.Stderr, "(without MQuery check - it is not known which corpora are installed for benchmarking)")
	}
	for _, entry := range corpora2 {
		fmt.Fprintf(os.Stderr, "\t%s: %d\n", entry.K, entry.V)
	}
	log.Info().
		Int("numProcessed", rm.numProcessed).
		Int("numZero", rm.numZero).
		Msg("removed zero entries")
}
