package feats

import (
	"github.com/czcorpus/cqlizer/cql"
	"github.com/czcorpus/cqlizer/pcalc"
)

func calcRgSimpleProb(params Params, props cql.RgSimpleProps) float64 {
	if props.ContainsWildcards() {
		if len(props.Constansts) == 0 {
			return params.PureWildcardScore
		}
		return params.WildcardedChunkLenScore / float64(len(props.Constansts)+len(props.Ops))
	}

	if len(props.Constansts)+len(props.Ops) > 0 {
		return params.ChunkLenScore / float64(len(props.Constansts)+len(props.Ops))
	}
	return float64(len(props.Alts)) * params.ChunkLenScore
}

func Evaluate(query *cql.Query, params Params) *pcalc.StackMachine {
	var sm pcalc.StackMachine
	query.DFS(func(node cql.ASTNode) {
		switch tNode := node.(type) {
		case *cql.Sequence:
		case *cql.Seq:
			for i := 1; i < tNode.NumPositions(); i++ {
				sm.Push(pcalc.MultiplyOrWeightSum{})
			}
		case *cql.GlobPart:
		case *cql.WithinOrContaining:
		case *cql.WithinContainingPart:
		case *cql.GlobCond:
			sm.Push(pcalc.Constant{Value: params.GlobCondPenalty})
			sm.Push(pcalc.Multiply{})
		case *cql.Structure:
		case *cql.AttValList:
			for i := 1; i < len(tNode.AttValAnd); i++ {
				sm.Push(pcalc.Add{})
			}
		case *cql.NumberedPosition:
		case *cql.OnePosition:
			if tNode.Variant1 != nil {
				if tNode.Variant1.AttValList.IsEmpty() {
					sm.Push(pcalc.Constant{Value: params.AnyPositionScore}) // `[]`
				}

			} else if tNode.Variant2 != nil {
				// NOOP
			}

		case *cql.Position:
		case *cql.RegExp:
			// NOP
		case *cql.MuPart:
			if tNode.Variant1 != nil {
				if tNode.Variant1.UnionOp != nil {
					sm.Push(pcalc.Avg{})
				}

			} else if tNode.Variant2 != nil {
				if tNode.Variant2.MeetOp != nil {
					sm.Push(pcalc.Avg{})
					sm.Push(pcalc.Constant{Value: params.MeetScore})
					sm.Push(pcalc.Multiply{})
				}
			}
		case *cql.Repetition:
			if tNode.Variant1 != nil {
				if tNode.Variant1.RepOpt != nil {
					if tNode.Variant1.RepOpt.DefinesInfReps() {
						sm.Push(pcalc.Pop{})
						sm.Push(pcalc.Constant{Value: params.PositionInfRepScore, Weight: 0.9})
						//sm.Push(pcalc.Avg{})

					} else {
						sm.Push(pcalc.Pop{})
						sm.Push(pcalc.Constant{Value: params.PositionFewRepScore, Weight: 0.7})
						//sm.Push(pcalc.Avg{})
					}
				}

			} else if tNode.Variant2 != nil {
				if tNode.Variant2.OpenStructTag.Structure.IsBigStructure() {
					sm.Push((pcalc.Constant{Value: params.BigOpenStructScore}))

				} else {
					sm.Push((pcalc.Constant{Value: params.SmallOpenStructTagProb}))
				}
			}
		case *cql.AtomQuery:
		case *cql.RepOpt:
		case *cql.OpenStructTag:
		case *cql.CloseStructTag:
		case *cql.AlignedPart:
		case *cql.AttValAnd:
			for i := 1; i < len(tNode.AttVal); i++ {
				sm.Push(pcalc.MultiplyOrWeightSum{})
			}
		case *cql.AttVal:
			if tNode.Variant1 != nil {
				if tNode.Variant1.Not {
					sm.Push(pcalc.Pop{})
					sm.Push(pcalc.Constant{Value: params.NegationPenalty, Weight: 0.9})
				}
				if tNode.IsProblematicAttrSearch() {
					sm.Push(pcalc.Pop{})
					sm.Push(pcalc.Constant{Value: params.SmallValSetPenalty, Weight: 0.9})
				}

			} else if tNode.Variant2 != nil {
				if tNode.Variant2.Not {
					sm.Push(pcalc.Constant{Value: params.NegationPenalty})
					sm.Push(pcalc.Multiply{})
				}
				if tNode.IsProblematicAttrSearch() {
					sm.Push(pcalc.Pop{})
					sm.Push(pcalc.Constant{Value: params.SmallValSetPenalty, Weight: 0.7})
					//fmt.Println("adding small penalty ", params.SmallValSetPenalty)
					//x, _ := sm.Peek()
					//fmt.Println("PEEK SM: ", x)
				}
			}
		case *cql.WithinNumber:
		case *cql.RegExpRaw:
			// NOP
		case *cql.RawString:
		case *cql.SimpleString:
		case *cql.RgGrouped:
		case *cql.RgSimple:
			wcProps := tNode.GetWildcards()
			prob := calcRgSimpleProb(params, wcProps)
			sm.Push(pcalc.Constant{Value: prob})
		case *cql.RgPosixClass:
		case *cql.RgLook:
		case *cql.RgAlt:
		case *cql.RgRange:
		case *cql.RgRangeSpec:
		case *cql.AnyLetter:
		case *cql.RgOp:
			// TODO ??? sm.Push(pcalc.Constant{Value: 0.7})
		case *cql.RgAltVal:
		default:
			//fmt.Println("@ ", reflect.TypeOf(tNode))
		}
	})
	return &sm

}
