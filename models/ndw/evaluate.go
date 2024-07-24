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
		case *cql.Seq:
			for i := 1; i < tNode.NumPositions(); i++ {
				sm.Push(stackm.Multiply{})
			}
		case *cql.GlobPart:
		case *cql.WithinOrContaining:
		case *cql.WithinContainingPart:
		case *cql.GlobCond:
			sm.Push(stackm.Constant(params.GlobCond))
			sm.Push(stackm.Multiply{})
		case *cql.Structure:
			sm.Push(stackm.Constant(params.Structure))
			sm.Push(stackm.Multiply{})
		case *cql.NumberedPosition:
		case *cql.RegExp:
			// NOP
		case *cql.MuPart:
			// TODO
		case *cql.Repetition:
			// TODO NOP?
		case *cql.AtomQuery:
		case *cql.RepOpt:
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
				sm.Push(stackm.Constant(params.AttValAnd))
				sm.Push(stackm.Multiply{})
			}
		case *cql.AttVal:
			sm.Push(stackm.Constant(params.AttVal))
			sm.Push(stackm.Multiply{})
			if tNode.IsNegation() {
				sm.Push(stackm.Constant(params.AttValVariantNeg))
				sm.Push(stackm.Multiply{})
			}
		case *cql.WithinNumber:
		case *cql.RegExpRaw:
			// NOP
		case *cql.RawString:
		case *cql.SimpleString:
		case *cql.RgGrouped:
			for range tNode.Values[1:] {
				sm.Push(stackm.Add{})
			}
			sm.Push(stackm.Constant(params.RgSimple))
			sm.Push(stackm.Multiply{})
		case *cql.RgSimple:
			for range tNode.Values[1:] {
				sm.Push(stackm.Multiply{})
			}
			sm.Push(stackm.Constant(params.RgSimple))
			sm.Push(stackm.Multiply{})
		case *cql.RgPosixClass:
		case *cql.RgLook:
		case *cql.RgChar:
			sm.Push(stackm.Constant(1)) // TODO
		case *cql.RgAlt:
			for range tNode.Values[1:] {
				sm.Push(stackm.Add{})
			}
			sm.Push(stackm.Constant(params.RgAlt))
			sm.Push(stackm.Multiply{})
		case *cql.RgRange:
			sm.Push(stackm.Constant(params.RgRange)) // TODO
		case *cql.RgRangeSpec:
		case *cql.AnyLetter:
			sm.Push(stackm.Constant(params.AnyLetter))
		case *cql.RgOp:
			sm.Push(stackm.Constant(params.RgOp))
		case *cql.RgAltVal:
			// TODO not very accurate (variants: a-z, negation etc. not reflected)
			sm.Push(stackm.Constant(params.RgAltVal))
		case *cql.RgAny:
			sm.Push(stackm.Constant(params.RgAny))
		case *cql.RgQM:
			sm.Push(stackm.Constant(params.RgQM))
		case *cql.RgRepeat:
			sm.Push(stackm.Constant(params.RgRepeat))
		default:
			//fmt.Println("@ ", reflect.TypeOf(tNode))
		}
	})
	return &sm

}
