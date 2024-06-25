package feats

import (
	"encoding/json"
	"fmt"

	"github.com/czcorpus/cqlizer/cql"
)

// RepOpts score: for each operation like '.*', '.+', we add 100 points and
// for each preceding constant string (e.g. 'a.+', 'hit.*') we divide the initial score
// by the [prefix length]. So e.g. for 'work.*' will get 100 / 4

type Record struct {
	CorpusSize         int     `json:"corpusSize"`
	TextLen            int     `json:"textLen"`
	NumAtomQueries     int     `json:"numAtomQueries"`
	NumRepOpts         int     `json:"numRepOpts"`
	AvgConstStringSize float64 `json:"avgConstStringSize"`

	// NumExpensiveRgOp (e.g. RgRange, .*, .+)
	NumExpensiveRgOp int `json:"numExpensiveRgOp"`

	NumGlobCond int `json:"numGlobCond"`

	NumContaining int `json:"numContaining"`

	NumWithin int `json:"numWithin"`
}

func (rec Record) AsVector() []float64 {
	return []float64{
		float64(rec.CorpusSize),
		float64(rec.TextLen),
		float64(rec.NumAtomQueries),
		float64(rec.NumRepOpts),
		rec.AvgConstStringSize,
		float64(rec.NumExpensiveRgOp),
		float64(rec.NumGlobCond),
		float64(rec.NumContaining),
		float64(rec.NumWithin),
	}
}

func (rec *Record) ImportFrom(query *cql.Query, corpusSize int) {
	rec.CorpusSize = corpusSize
	rec.TextLen = query.Len()

}

func (rec Record) AsJSONString() string {
	ans, err := json.Marshal(rec)
	if err != nil {
		panic(fmt.Sprintf("failed to serialize feats.Record: %s", err))
	}
	return string(ans)
}
