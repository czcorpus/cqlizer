// Copyright 2024 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2024 Department of Linguistics,
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

package cql

import (
	"strings"

	"github.com/czcorpus/cnc-gokit/collections"
)

// TTEnterFunc is called when DFS descends into a node. If it returns
// false, DFS does not descend into that node's children - it goes
// straight to calling leave on it instead.
type TTEnterFunc func(node ASTNode) bool
type TTLeaveFunc func(node ASTNode)

// dfsLeaf visits a leaf AST node (one with no DFS method of its own,
// e.g. an ASTString) by calling enter immediately followed by leave.
// Its enter return value is ignored - a leaf has no children to skip.
func dfsLeaf(enter TTEnterFunc, leave TTLeaveFunc, node ASTNode) {
	enter(node)
	leave(node)
}

// QueryProp is a generalized query property:
// a) positional attribute with a value
// b) structural attribute with a value
// c) structure
type QueryProp struct {
	Structure string
	Name      string
	Value     string
}

func (qp QueryProp) IsStructure() bool {
	return qp.Structure != "" && qp.Name == "" && qp.Value == ""
}

func (qp QueryProp) IsStructAttr() bool {
	return qp.Structure != "" && qp.Name != "" && qp.Value != ""
}

func (qp QueryProp) IsPosattr() bool {
	// we do not test qp.Name here as the query can
	// be also just a regexp expecting a default attribute
	return qp.Structure == "" && qp.Name != "" && qp.Value != ""
}

// Query represents root node of a CQL syntax tree.
//
//	Sequence (_ BINAND _ GlobPart)? (_ WithinOrContaining)* EOF {
type Query struct {
	origValue          string
	Sequence           *Sequence
	GlobPart           *GlobPart
	WithinOrContaining []*WithinOrContaining
}

func (q *Query) Len() int {
	return len(q.origValue)
}

func (q *Query) String() string {
	var ans strings.Builder
	ans.WriteString("Query( ")
	if q.Sequence != nil {
		ans.WriteString(q.Sequence.String())
	}
	if q.GlobPart != nil {
		ans.WriteString(q.GlobPart.String())
	}
	ans.WriteString(" )")
	return ans.String()
}

func (q *Query) ForEachElement(fn func(parent, v ASTNode)) {
	fn(nil, q)
	if q.Sequence != nil {
		q.Sequence.ForEachElement(q, fn)
	}
	if q.GlobPart != nil {
		q.GlobPart.ForEachElement(q, fn)
	}
	for _, item := range q.WithinOrContaining {
		item.ForEachElement(q, fn)
	}
}

// CQL renders the query as normalized, valid CQL source, rebuilt
// recursively from each node's own fields rather than echoed from the
// original input.
func (q *Query) CQL() string {
	if q == nil {
		return ""
	}
	var ans strings.Builder
	if q.Sequence != nil {
		ans.WriteString(q.Sequence.CQL())
	}
	if q.GlobPart != nil {
		ans.WriteString(" & ")
		ans.WriteString(q.GlobPart.CQL())
	}
	for _, wc := range q.WithinOrContaining {
		ans.WriteString(" ")
		ans.WriteString(wc.CQL())
	}
	return ans.String()
}

func (q *Query) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(q) {
		leave(q)
		return
	}
	if q.Sequence != nil {
		q.Sequence.DFS(enter, leave)
	}
	if q.GlobPart != nil {
		q.GlobPart.DFS(enter, leave)
	}
	for _, item := range q.WithinOrContaining {
		item.DFS(enter, leave)
	}
	leave(q)
}

func (q *Query) ExtractProps() []QueryProp {
	ans := make([]QueryProp, 0, 10)
	parents := make(parentMap)
	structs := collections.NewSet[string]()
	q.ForEachElement(func(parent, v ASTNode) {
		parents[v] = parent
		switch typedV := v.(type) {
		case *AttVal:
			if typedV.Variant1 != nil {
				newItem := QueryProp{
					Name:  typedV.Variant1.AttName.String(),
					Value: strings.Trim(typedV.Variant1.RawString.SimpleString.String(), "\""),
				}
				stSrch := parents.findParentByType(typedV, &Structure{}, 0)
				if stSrch != nil {
					t, ok := stSrch.(*Structure)
					if !ok {
						// this can happen only if findParentByType is broken
						panic("found structure is not a *Structure")
					}
					newItem.Structure = t.AttName.String()
				}
				ans = append(ans, newItem)

			} else if typedV.Variant2 != nil {
				newItem := QueryProp{
					Name:  typedV.Variant2.AttName.String(),
					Value: strings.Trim(typedV.Variant2.RegExp.String(), "\""),
				}
				stSrch := parents.findParentByType(typedV, &Structure{}, 0)
				if stSrch != nil {
					t, ok := stSrch.(*Structure)
					if !ok {
						// this can happen only if findParentByType() is broken
						panic("found structure is not a *Structure")
					}
					newItem.Structure = t.AttName.String()
				}
				ans = append(ans, newItem)

			}
		case *Structure:
			structs.Add(typedV.AttName.String())
		case *RegExp:
			srch := parents.findParentByType(typedV, &OnePosition{}, 1)
			if srch != nil {
				val := make([]string, len(typedV.RegExpRaw))
				for i, v := range typedV.RegExpRaw {
					val[i] = v.String()
				}
				ans = append(ans, QueryProp{Value: strings.Join(val, " ")})
			}
		}
	})
	for _, v := range structs.ToSlice() {
		ans = append(ans, QueryProp{Structure: v})
	}
	return ans
}
