package feats

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/czcorpus/cqlizer/cql"
	"github.com/davecgh/go-spew/spew"
)

// RepOpts score: for each operation like '.*', '.+', we add 100 points and
// for each preceding constant string (e.g. 'a.+', 'hit.*') we divide the initial score
// by the [prefix length]. So e.g. for 'work.*' will get 100 / 4

type Record struct {
	CorpusSize                     int     `json:"corpusSize"`
	TextLen                        int     `json:"textLen"`
	NumPositions                   int     `json:"numAtomQueries"`
	NumRepOpts                     int     `json:"numRepOpts"`
	AvgConstStringSize             float64 `json:"avgConstStringSize"`
	NumDisjunctElementsPerSequence float64 `json:"numDisjunctElementsPerSequence"`

	// NumExpensiveRgOp (e.g. RgRange, .*, .+)
	NumExpensiveRgOp int `json:"numExpensiveRgOp"`

	NumGlobCond int `json:"numGlobCond"`

	NumContaining int `json:"numContaining"`

	NumNegContaining int `json:"numNegContaining"`

	NumWithin int `json:"numWithin"`

	NumNegWithin int `json:"numNegWithin"`
}

func (rec Record) AsVector() []float64 {
	return []float64{
		float64(rec.CorpusSize),
		float64(rec.TextLen),
		float64(rec.NumPositions),
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
	var rootSequence *cql.Sequence
	var rootSeq *cql.Seq
	var numOrChainedSeq int
	query.ForEachElement(func(parent, v cql.ASTNode) {
		switch tNode := v.(type) {
		case *cql.Query:
			if tNode.GlobPart != nil {
				fmt.Println("########## WE HAVE GLOB")
			}
		case *cql.Sequence:
			fmt.Println("##### <Sequence>: ", tNode.Text())
			if parent == query {
				fmt.Println("   @@@@@@@@ we have a root ...")
				rootSequence = tNode
			}
			fmt.Println("   his parent: ", reflect.TypeOf(parent), parent.Text())
		case *cql.Seq:
			fmt.Println("##### <Seq>: ", tNode.Text())
			if parent == rootSequence {
				rootSeq = tNode
			}
			fmt.Println("   or chained? ", tNode.IsOrChained())
			if tNode.IsOrChained() {
				numOrChainedSeq++
			}
			fmt.Println("   his parent: ", reflect.TypeOf(parent), parent.Text())
		case *cql.AtomQuery:
			fmt.Println("##### <AtomQuery>: ", tNode.Text())
			fmt.Println("   his parent: ", reflect.TypeOf(parent), parent.Text())
			rec.NumWithin += tNode.NumWithinParts()
			rec.NumNegWithin += tNode.NumNegWithinParts()
			rec.NumContaining += tNode.NumContainingParts()
			rec.NumNegContaining += tNode.NumNegContainingParts()
		case *cql.Repetition:
			fmt.Println("### <Repetition>: ", tNode.Text())
			if parent == rootSeq {
				fmt.Println("   we have a POSITION Repetition!!!")
				rec.NumPositions++
			}
			fmt.Println("   RepOpt: ", tNode.GetRepOpt())
			fmt.Println("   tail position? ", tNode.IsTailPosition())
			fmt.Println("   his parent: ", reflect.TypeOf(parent), parent.Text())
			if tNode.GetRepOpt() != "" {
				rec.NumRepOpts++
			}
		case *cql.RgSimple:

		case *cql.GlobCond:
			rec.NumGlobCond++
		case *cql.WithinOrContaining:
			rec.NumWithin += tNode.NumWithinParts()
			rec.NumNegWithin += tNode.NumNegWithinParts()
			rec.NumContaining += tNode.NumContainingParts()
			rec.NumNegContaining += tNode.NumNegContainingParts()
		}
	})
	rec.NumDisjunctElementsPerSequence = (float64(rec.NumPositions) + float64(numOrChainedSeq)) / float64(rec.NumPositions)
	spew.Dump(rec)

}

func (rec Record) AsJSONString() string {
	ans, err := json.Marshal(rec)
	if err != nil {
		panic(fmt.Sprintf("failed to serialize feats.Record: %s", err))
	}
	return string(ans)
}