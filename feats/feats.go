// Copyright 2024 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2024 Institute of the Czech National Corpus,
// Faculty of Arts, Charles University
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package feats

import (
	"fmt"
	"reflect"

	"github.com/czcorpus/cqlizer/cql"
	"github.com/sjwhitworth/golearn/pca"
	"gonum.org/v1/gonum/mat"
)

type Record struct {
	matrix         *mat.Dense
	fullWHSize     int
	numReducedCols int
}

func NewRecord() Record {
	ans := Record{}
	ans.fullWHSize = 36
	ans.numReducedCols = 4

	return ans
}

func (rec Record) ReduceDim(from *mat.Dense) *mat.Dense {
	p := pca.NewPCA(4)
	p.Fit(from)
	return p.Transform(from)
}

func (rec Record) AsVector() []float64 {
	ans := make([]float64, rec.fullWHSize*rec.numReducedCols)
	for i := 0; i < rec.fullWHSize; i++ {
		for j := 0; j < rec.numReducedCols; j++ {
			ans[i*rec.numReducedCols+j] = rec.matrix.At(i, j)
		}
	}
	return ans
}

// GetNodeTypeIdx
// For each AST node type, we want to create a [node]->[parent] record
// in our "transition heatmap matrix"
func (rec *Record) GetNodeTypeIdx(v any) int {
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
	case *cql.MeetOp:
		return 35
	default:
		panic(fmt.Sprintf("unsupported node type: %s", reflect.TypeOf(v)))
	}
}

func (rec *Record) ImportFrom(query *cql.Query) {
	largeMatrix := mat.NewDense(rec.fullWHSize, rec.fullWHSize, nil)
	query.ForEachElement(func(parent, v cql.ASTNode) {
		switch parent.(type) {
		case *cql.Query, *cql.RgChar:
			return
		}
		switch tNode := v.(type) {
		case cql.ASTString:
			// NOP (mostly attribute names)
		case *cql.Query, *cql.RgChar:
			// NOP (the matrix itself)
		case *cql.RegExpRaw:
			i1 := rec.GetNodeTypeIdx(tNode)
			i2 := rec.GetNodeTypeIdx(parent)
			largeMatrix.Set(i1, i2, largeMatrix.At(i1, i2)+tNode.ExhaustionScore())
		case *cql.AttVal:
			i1 := rec.GetNodeTypeIdx(tNode)
			i2 := rec.GetNodeTypeIdx(parent)
			v := 1.0
			if tNode.IsProblematicAttrSearch() {
				v = 20.0
			}
			largeMatrix.Set(i1, i2, largeMatrix.At(i1, i2)+v)
		case *cql.Repetition:
			i1 := rec.GetNodeTypeIdx(tNode)
			i2 := rec.GetNodeTypeIdx(parent)
			v := 1.0
			if tNode.IsAnyPosition() {
				v = 100.0
			}
			largeMatrix.Set(i1, i2, largeMatrix.At(i1, i2)+v)
		case *cql.Structure:
			i1 := rec.GetNodeTypeIdx(tNode)
			i2 := rec.GetNodeTypeIdx(parent)
			if tNode.IsBigStructure() {
				largeMatrix.Set(i1, i2, largeMatrix.At(i1, i2)+10)

			} else {
				largeMatrix.Set(i1, i2, largeMatrix.At(i1, i2)+1)
			}
		default:
			i1 := rec.GetNodeTypeIdx(tNode)
			i2 := rec.GetNodeTypeIdx(parent)
			largeMatrix.Set(i1, i2, largeMatrix.At(i1, i2)+1)
		}
	})
	rec.matrix = rec.ReduceDim(largeMatrix)
}
