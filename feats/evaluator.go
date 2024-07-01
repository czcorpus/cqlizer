package feats

import (
	"github.com/czcorpus/cqlizer/cql"
	"github.com/czcorpus/cqlizer/pcalc"
)

var chunkProbs = []float64{1, 0.2, 0.1, 0.07, 0.1, 0.1, 0.1, 0.1, 0.07, 0.04, 0.01, 0.01, 0.005}

func calcExpandVariantsProb(origLen int) float64 {
	if origLen >= len(chunkProbs) {
		return 0.003
	}
	prob := chunkProbs[origLen]
	for i := origLen + 1; i < len(chunkProbs); i++ {
		prob += chunkProbs[i]
	}
	return prob
}

func calcQuestMarkProb(origLen int) float64 {
	if origLen <= len(chunkProbs) {
		return chunkProbs[origLen-2]
	}
	return 0.005
}

func calcRgSimpleProb(props cql.RgSimpleProps) float64 {
	if props.ContainsWildcards() {
		if len(props.Constansts) == 0 {
			return 1.0
		}
		return calcExpandVariantsProb(len(props.Constansts))
	}
	if len(props.Constansts) >= len(chunkProbs) {
		return 0.003
	}
	if len(props.Alts) > 0 {
		p := chunkProbs[len(chunkProbs)-1]
		idx := len(props.Constansts) + 1
		if idx < len(chunkProbs)-1 {
			p = chunkProbs[idx]
		}
		for range props.Alts {
			p = min(1, 2*p)
		}
		return p

	} else {
		return chunkProbs[len(props.Constansts)]
	}
}

func Evaluate(query *cql.Query) *pcalc.StackMachine {
	var sm pcalc.StackMachine
	query.DFS(func(node cql.ASTNode) {
		switch tNode := node.(type) {
		case *cql.Sequence:
		case *cql.Seq:
			for i := 1; i < tNode.NumPositions(); i++ {
				sm.Push(pcalc.Multiply{})
			}
		case *cql.GlobPart:
		case *cql.WithinOrContaining:
		case *cql.WithinContainingPart:
		case *cql.GlobCond:
		case *cql.Structure:
		case *cql.AttValList:
		case *cql.NumberedPosition:
		case *cql.OnePosition:
			if tNode.Variant1 != nil {
				if tNode.Variant1.AttValList.IsEmpty() {
					sm.Push(pcalc.Constant{Value: 1.0}) // `[]`
				}

			} else if tNode.Variant2 != nil {
				// NOOP
			}

		case *cql.Position:
		case *cql.RegExp:
			// NOP
		case *cql.MuPart:
		case *cql.Repetition:
			if tNode.Variant1 != nil {
				c := 1.0
				if tNode.Variant1.RepOpt != nil {
					i1, i2 := tNode.Variant1.RepOpt.GetNumRepEstimate()
					for i := i1; i <= i2; i++ {
						c = min(1, c+1.0/float64(i))
					}
					sm.Push(pcalc.Constant{Value: c})
					sm.Push(pcalc.Multiply{})
				}
			}
		case *cql.AtomQuery:
		case *cql.RepOpt:
		case *cql.OpenStructTag:
		case *cql.CloseStructTag:
		case *cql.AlignedPart:
		case *cql.AttValAnd:
		case *cql.AttVal:
			if tNode.Variant1 != nil && tNode.Variant1.Not {
				sm.Push(pcalc.NegProb{})

			} else if tNode.Variant2 != nil && tNode.Variant2.Not {
				sm.Push(pcalc.NegProb{})
			}
		case *cql.WithinNumber:
		case *cql.RegExpRaw:
			// NOP
		case *cql.RawString:
		case *cql.SimpleString:
		case *cql.RgGrouped:
		case *cql.RgSimple:
			wcProps := tNode.GetWildcards()
			prob := calcRgSimpleProb(wcProps)
			sm.Push(pcalc.Constant{Value: prob})
		case *cql.RgPosixClass:
		case *cql.RgLook:
		case *cql.RgAlt:
		case *cql.RgRange:
		case *cql.RgRangeSpec:
		case *cql.AnyLetter:
		case *cql.RgOp:
			sm.Push(pcalc.Constant{Value: 0.7})
		case *cql.RgAltVal:
		default:
			//fmt.Println("@ ", reflect.TypeOf(tNode))
		}
	})
	return &sm

}
