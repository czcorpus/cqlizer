package ndw

import (
	"github.com/czcorpus/cqlizer/cql"
	"github.com/czcorpus/cqlizer/models/ndw/stackm"
)

func Evaluate(query *cql.Query, params Params) *stackm.StackMachine {
	var sm stackm.StackMachine
	query.DFS(func(node cql.ASTNode) {
		switch tNode := node.(type) {
		case *cql.Sequence:
			for i := 1; i < len(tNode.Seq); i++ {
				sm.Push(stackm.Add{})
			}
		case *cql.Seq:
			for i := 1; i < tNode.NumPositions(); i++ {
				sm.Push(stackm.Multiply{})
			}
		case *cql.OnePosition:
			if tNode.Text() == "[]" {
				sm.Push(stackm.Constant(params.AnyPosition))
				sm.Push(stackm.Constant(params.AnyPosition))
				sm.Push(stackm.Multiply{})
			}
		case *cql.GlobPart:
		case *cql.WithinOrContaining:
		case *cql.WithinContainingPart:
		case *cql.GlobCond:
			sm.Push(stackm.Constant(params.GlobCond))
			sm.Push(stackm.Multiply{})
		case *cql.Structure:
			if tNode.AttValList != nil {
				for i := 1; i < len(tNode.AttValList.AttValAnd); i++ {
					sm.Push(stackm.Multiply{})
				}
			}
			sm.Push(stackm.Constant(params.Structure))
			sm.Push(stackm.Multiply{})
		case *cql.NumberedPosition:
		case *cql.RegExp:
			for i := 1; i < len(tNode.RegExpRaw); i++ {
				sm.Push(stackm.Add{}) // TODO Adding or Multiplying?
			}
			sm.Push(stackm.Constant(params.RegExp))
			sm.Push(stackm.Multiply{})

		case *cql.MuPart:
			// TODO
		case *cql.MeetOp:
			sm.Push(stackm.Multiply{})
			sm.Push(stackm.Constant(params.MeetOp))
			sm.Push(stackm.Multiply{})
		case *cql.Repetition:
			// NOP
		case *cql.AtomQuery:
		case *cql.RepOpt:
			sm.Push(stackm.Constant(params.RepOpt))
			sm.Push(stackm.Multiply{})
		case *cql.OpenStructTag:
			sm.Push(stackm.Constant(params.OpenStructTag))
			sm.Push(stackm.Multiply{})
		case *cql.CloseStructTag:
		case *cql.AlignedPart:
			sm.Push(stackm.Constant(params.AlignedPart))
			sm.Push(stackm.Multiply{})
		case *cql.AttValList:
			for i := 1; i < len(tNode.AttValAnd); i++ {
				sm.Push(stackm.Add{})
			}
			sm.Push(stackm.Constant(params.AttValList))
			sm.Push(stackm.Multiply{})
		case *cql.AttValAnd:
			for i := 1; i < len(tNode.AttVal); i++ {
				sm.Push(stackm.Multiply{})
			}
			sm.Push(stackm.Constant(params.AttValAnd))
			sm.Push(stackm.Multiply{Tag: "AttValAnd (param)"})
		case *cql.AttVal:
			if tNode.Variant5 != nil { // NOT ...
				sm.Push(stackm.Constant(params.AttValVariantNeg))
				sm.Push(stackm.Multiply{Tag: "AttVal(v5) AttValVariantNeg"})

			} else if tNode.Variant6 != nil {
				for i := 1; i < len(tNode.Variant6.AttValList.AttValAnd); i++ {
					sm.Push(stackm.Multiply{Tag: "tNode.Variant6.AttValList.AttValAnd"})
				}
			}
			sm.Push(stackm.Constant(params.AttVal))
			sm.Push(stackm.Multiply{Tag: "AttVal itself"})
		case *cql.WithinNumber:
		case *cql.RegExpRaw:
			for i := 1; i < len(tNode.Values); i++ {
				sm.Push(stackm.Multiply{})
			}
			sm.Push(stackm.Constant(params.RegExpRaw))
			sm.Push(stackm.Multiply{})
		case *cql.RawString:
		case *cql.SimpleString:
		case *cql.RgGrouped:
			for i := 1; i < len(tNode.Values); i++ {
				sm.Push(stackm.Add{})
			}
			sm.Push(stackm.Constant(params.RgSimple))
			sm.Push(stackm.Multiply{})
		case *cql.RgSimple:
			for i := 1; i < len(tNode.Values); i++ {
				sm.Push(stackm.Multiply{})
			}
			sm.Push(stackm.Constant(params.RgSimple))
			sm.Push(stackm.Multiply{})
		case *cql.RgPosixClass:
		case *cql.RgLook:
		case *cql.RgAlt:
			for i := 1; i < len(tNode.Values); i++ {
				sm.Push(stackm.Add{})
			}
			sm.Push(stackm.Constant(params.RgAlt))
			sm.Push(stackm.Multiply{})
		case *cql.RgRange:
			sm.Push(stackm.Constant(params.RgRange)) // TODO
		case *cql.RgRangeSpec:
		case *cql.AnyLetter:
			sm.Push(stackm.Constant(params.AnyLetter))
		case *cql.RgChar:
			if tNode.Variant1 != nil {
				sm.Push(stackm.Constant(params.RgChar))

			} else if tNode.Variant2 != nil {
				sm.Push(stackm.Constant(params.RgOp))

			} else if tNode.Variant3 != nil {
				sm.Push(stackm.Constant(params.RgRepeat))

			} else if tNode.Variant4 != nil {
				sm.Push(stackm.Constant(params.RgAny))

			} else if tNode.Variant5 != nil {
				sm.Push(stackm.Constant(params.RgQM))
			}
		case *cql.RgAltVal:
			// TODO not very accurate (variants: a-z, negation etc. not reflected)
			sm.Push(stackm.Constant(params.RgAltVal))
			sm.Push(stackm.Multiply{})
		default:
			//fmt.Println("@ ", reflect.TypeOf(tNode))
		}
	})
	return &sm

}
