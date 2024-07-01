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
	CorpusSize   int `json:"corpusSize"`
	TextLen      int `json:"textLen"`
	NumPositions int `json:"numAtomQueries"`

	//
	// []{1,3}, []+ [tag="N.*"]*
	PositionExhaustionScore        float64 `json:"positionExhaustionScore"`
	AvgConstStringSize             float64 `json:"avgConstStringSize"`
	NumDisjunctElementsPerSequence float64 `json:"numDisjunctElementsPerSequence"`
	NumDisjunctPerRawRegexp        float64 `json:"numDisjunctPerRawRegexp"`
	AvgUppercaseRatio              float64 `json:"avgUppercaseRatio"`

	// RegExpExhaustionScore (problems inside regexps: e.g. .*, .+)
	RegExpExhaustionScore float64 `json:"regExpExhaustionScore"`

	NumGlobCond int `json:"numGlobCond"`

	NumContaining int `json:"numContaining"`

	NumNegContaining int `json:"numNegContaining"`

	NumWithin int `json:"numWithin"`

	NumNegWithin int `json:"numNegWithin"`

	NumOpenStructTag int `json:"numOpenStructTag"`
}

func calcAvg(data []float64) float64 {
	var ans float64
	for _, v := range data {
		ans += v
	}
	return ans / float64(len(data))
}

func (rec Record) AsVector() []float64 {
	return []float64{
		float64(rec.CorpusSize),
		//float64(rec.TextLen),
		float64(rec.NumPositions),
		rec.PositionExhaustionScore,
		rec.AvgConstStringSize,
		rec.RegExpExhaustionScore,
		float64(rec.NumGlobCond),
		float64(rec.NumContaining),
		float64(rec.NumWithin),
		rec.AvgConstStringSize,
		rec.NumDisjunctElementsPerSequence,
		rec.NumDisjunctPerRawRegexp,
		rec.AvgUppercaseRatio,
		float64(rec.NumOpenStructTag),
	}
}

func (rec *Record) ImportFrom(query *cql.Query, corpusSize int) {
	rec.CorpusSize = corpusSize
	rec.TextLen = query.Len()
	var rootSequence *cql.Sequence
	var rootSeq *cql.Seq
	var numOrChainedSeq int
	var currRegexp *cql.RegExp
	var numRegexpRaw int
	var numRegexpRawExpOps int
	uppercaseRatioItems := make([]float64, 0, 50)
	query.ForEachElement(func(parent, v cql.ASTNode) {
		switch tNode := v.(type) {
		case *cql.Query:
			if tNode.GlobPart != nil {
				//fmt.Println("########## WE HAVE GLOB")
			}
		case *cql.Sequence:
			//fmt.Println("##### <Sequence>: ", tNode.Text())
			if parent == query {
				//fmt.Println("   @@@@@@@@ we have a root ...")
				rootSequence = tNode
			}
			//fmt.Println("   his parent: ", reflect.TypeOf(parent), parent.Text())
		case *cql.Seq:
			//fmt.Println("##### <Seq>: ", tNode.Text())
			if parent == rootSequence {
				rootSeq = tNode
			}
			//fmt.Println("   or chained? ", tNode.IsOrChained())
			if tNode.IsOrChained() {
				numOrChainedSeq++
			}
			//fmt.Println("   his parent: ", reflect.TypeOf(parent), parent.Text())
		case *cql.AtomQuery:
			//fmt.Println("##### <AtomQuery>: ", tNode.Text())
			//fmt.Println("   his parent: ", reflect.TypeOf(parent), parent.Text())
			rec.NumWithin += tNode.NumWithinParts()
			rec.NumNegWithin += tNode.NumNegWithinParts()
			rec.NumContaining += tNode.NumContainingParts()
			rec.NumNegContaining += tNode.NumNegContainingParts()
		case *cql.Repetition:
			//fmt.Println("### <Repetition>: ", tNode.Text())
			if parent == rootSeq {
				rec.NumPositions++
			}
			//fmt.Println("   RepOpt: ", tNode.GetRepOpt())
			//fmt.Println("   tail position? ", tNode.IsTailPosition())
			//fmt.Println("   his parent: ", reflect.TypeOf(parent), parent.Text())
			if tNode.GetRepOpt() == "+" || tNode.GetRepOpt() == "*" {
				rec.PositionExhaustionScore += 10

			} else if tNode.GetRepOpt() == "?" {
				rec.PositionExhaustionScore += 2
			}
			if tNode.IsAnyPosition() {
				rec.PositionExhaustionScore += 20
			}
			rng := tNode.GetReptOptRange()
			if rng[0] > -1 && rng[1] > -1 {
				rec.PositionExhaustionScore += float64((rng[1] - rng[0]) * 3)

			} else if rng[0] > -1 {
				rec.PositionExhaustionScore += 10
			}

		case *cql.RgSimple:
			rec.RegExpExhaustionScore += tNode.ExhaustionScore()
		case *cql.GlobCond:
			rec.NumGlobCond++
		case *cql.WithinOrContaining:
			rec.NumWithin += tNode.NumWithinParts()
			rec.NumNegWithin += tNode.NumNegWithinParts()
			rec.NumContaining += tNode.NumContainingParts()
			rec.NumNegContaining += tNode.NumNegContainingParts()

		case *cql.RegExp:
			currRegexp = tNode

		case *cql.RegExpRaw:
			if parent == currRegexp {
				numRegexpRaw++
				//fmt.Println("======= <RegExpRaw>: ", tNode.Text())
				//fmt.Println("   his parent: ", reflect.TypeOf(parent), parent.Text())
				//fmt.Println("   EXPENSIVE ops: ", tNode.ExpensiveOps())
				//rec.RegExpExhaustionScore += tNode.ExhaustionScore() // TODO isn't this duplicate of stuff in RgSimple?
			}
		case *cql.SimpleString:
			uppercaseRatioItems = append(uppercaseRatioItems, tNode.UppercaseRatio())
		case *cql.OpenStructTag:
			rec.NumOpenStructTag++
		}
	})
	if len(uppercaseRatioItems) > 0 {
		rec.AvgUppercaseRatio = calcAvg(uppercaseRatioItems)
	}
	if numRegexpRaw > 0 {
		rec.NumDisjunctPerRawRegexp = float64(numRegexpRawExpOps) / float64(numRegexpRaw)
	}
	rec.NumDisjunctElementsPerSequence = float64(numOrChainedSeq) / float64(rec.NumPositions)
	rec.RegExpExhaustionScore /= float64(rec.NumPositions * rec.NumPositions)

}

func (rec Record) AsJSONString() string {
	ans, err := json.Marshal(rec)
	if err != nil {
		panic(fmt.Sprintf("failed to serialize feats.Record: %s", err))
	}
	return string(ans)
}
