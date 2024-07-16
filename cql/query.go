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

import "encoding/json"

// Query
//
//	Sequence (_ BINAND _ GlobPart)? (_ WithinOrContaining)* EOF {
type Query struct {
	effect             float64
	origValue          string
	Sequence           *Sequence
	GlobPart           *GlobPart
	WithinOrContaining []*WithinOrContaining
}

func (q *Query) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Expansion Query
		RuleName  string
		Effect    float64
	}{
		RuleName:  "Query",
		Expansion: *q,
		Effect:    q.effect,
	})
}

func (q *Query) Len() int {
	return len(q.origValue)
}

func (q *Query) Text() string {
	return q.origValue
}

func (q *Query) Effect() float64 {
	if q.effect == 0 {
		q.effect = 1
	}
	return q.effect
}

func (q *Query) SetEffect(v float64) {
	q.effect = v
}

func (q *Query) IsLeaf() bool {
	return false
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

func (q *Query) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(q)
	if q.Sequence != nil {
		q.Sequence.DFS(fn, path)
	}
	if q.GlobPart != nil {
		q.GlobPart.DFS(fn, path)
	}
	for _, item := range q.WithinOrContaining {
		item.DFS(fn, path)
	}
	fn(q, path)
	path.Pop()
}
