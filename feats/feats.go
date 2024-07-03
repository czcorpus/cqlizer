package feats

import (
	"github.com/czcorpus/cqlizer/cql"
)

// RepOpts score: for each operation like '.*', '.+', we add 100 points and
// for each preceding constant string (e.g. 'a.+', 'hit.*') we divide the initial score
// by the [prefix length]. So e.g. for 'work.*' will get 100 / 4

type Record struct {
	matrix [][]float64
}

func NewRecord() Record {
	ans := Record{}
	ans.matrix = make([][]float64, 35)
	for i := 0; i < len(ans.matrix); i++ {
		ans.matrix[i] = make([]float64, len(ans.matrix))
	}
	return ans
}

func (rec Record) AsVector() []float64 {
	ans := make([]float64, len(rec.matrix)*len(rec.matrix))
	for i := 0; i < len(rec.matrix); i++ {
		for j := 0; j < len(rec.matrix); j++ {
			ans[i*len(rec.matrix)+j] = rec.matrix[i][j]
		}
	}
	return ans
}

func (rec *Record) GetNodeTypeID(v any) int {
	switch v.(type) {
	case *cql.Sequence:
		return 0
	case *cql.Seq:
		return 1
	case *cql.GlobPart:
		return 2
	case *cql.WithinOrContaining:
		return 3
	case *cql.WithinContainingPart:
		return 4
	case *cql.GlobCond:
		return 5
	case *cql.Structure:
		return 6
	case *cql.AttValList:
		return 7
	case *cql.NumberedPosition:
		return 8
	case *cql.OnePosition:
		return 9
	case *cql.Position:
		return 10
	case *cql.RegExp:
		return 11
	case *cql.MuPart:
		return 12
	case *cql.Repetition:
		return 13
	case *cql.AtomQuery:
		return 14
	case *cql.RepOpt:
		return 15
	case *cql.OpenStructTag:
		return 16
	case *cql.CloseStructTag:
		return 17
	case *cql.AlignedPart:
		return 18
	case *cql.AttValAnd:
		return 19
	case *cql.AttVal:
		return 20
	case *cql.WithinNumber:
		return 21
	case *cql.RegExpRaw:
		return 22
	case *cql.RawString:
		return 23
	case *cql.SimpleString:
		return 24
	case *cql.RgGrouped:
		return 25
	case *cql.RgSimple:
		return 26
	case *cql.RgPosixClass:
		return 27
	case *cql.RgLook:
		return 28
	case *cql.RgAlt:
		return 29
	case *cql.RgRange:
		return 30
	case *cql.RgRangeSpec:
		return 31
	case *cql.AnyLetter:
		return 32
	case *cql.RgOp:
		return 33
	case *cql.RgAltVal:
		return 34
	default:
		panic("unsupported node type")
	}
}

func (rec *Record) ImportFrom(query *cql.Query, corpusSize int) {

	query.ForEachElement(func(parent, v cql.ASTNode) {
		switch tNode := v.(type) {
		default:
			i1 := rec.GetNodeTypeID(tNode)
			i2 := rec.GetNodeTypeID(parent)
			rec.matrix[i1][i2] += 1
		case *cql.RegExpRaw:
			i1 := rec.GetNodeTypeID(tNode)
			i2 := rec.GetNodeTypeID(parent)
			rec.matrix[i1][i2] += tNode.ExhaustionScore()
		case *cql.Repetition:
			i1 := rec.GetNodeTypeID(tNode)
			i2 := rec.GetNodeTypeID(parent)
			v := 1.0
			if tNode.IsAnyPosition() {
				v = 10.0
			}
			rec.matrix[i1][i2] += v
		}
	})

}
