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

package cql

import (
	"encoding/json"
	"regexp"
	"strings"
)

var (
	rgLong = regexp.MustCompile(`xx+`)
)

// Query
//
//	Sequence (_ BINAND _ GlobPart)? (_ WithinOrContaining)* EOF {
type Query struct {
	origValue          string
	Sequence           *Sequence
	GlobPart           *GlobPart
	WithinOrContaining []*WithinOrContaining
}

func (q *Query) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Expansion Query
		RuleName  string
	}{
		RuleName:  "Query",
		Expansion: *q,
	})
}

func (q *Query) Len() int {
	return len(q.origValue)
}

func (q *Query) Text() string {
	return q.origValue
}

func (q *Query) Normalize() string {
	var ans strings.Builder
	ans.WriteString(q.Sequence.Normalize())
	if q.GlobPart != nil {
		ans.WriteString(" & " + q.GlobPart.Normalize())
	}
	for _, v := range q.WithinOrContaining {
		ans.WriteString(v.Normalize())
	}
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

func (q *Query) DFS(fn func(v ASTNode)) {
	if q.Sequence != nil {
		q.Sequence.DFS(fn)
	}
	if q.GlobPart != nil {
		q.GlobPart.DFS(fn)
	}
	for _, item := range q.WithinOrContaining {
		item.DFS(fn)
	}
	fn(q)
}

type attval struct {
	Structure string
	Name      string
	Value     string
}

func (q *Query) GetAllAttvals() []attval {
	ans := make([]attval, 0, 10)
	parents := make(parentMap)
	q.ForEachElement(func(parent, v ASTNode) {
		parents[v] = parent
		kvn, ok := v.(*AttVal)
		if !ok {
			return
		}
		if kvn.Variant1 != nil {
			newItem := attval{
				Name:  kvn.Variant1.AttName.String(),
				Value: strings.Trim(kvn.Variant1.RawString.SimpleString.Text(), "\""),
			}
			stSrch := parents.findParentByType(kvn, &Structure{})
			if stSrch != nil {
				t, ok := stSrch.(*Structure)
				if !ok {
					// this can happen only if findParentByType is broken
					panic("found structure is not a *Structure")
				}
				newItem.Structure = t.AttName.String()
			}
			ans = append(ans, newItem)

		} else if kvn.Variant2 != nil {
			newItem := attval{
				Name:  kvn.Variant2.AttName.String(),
				Value: strings.Trim(kvn.Variant2.RegExp.Text(), "\""),
			}
			stSrch := parents.findParentByType(kvn, &Structure{})
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
	})
	return ans
}
