package cql

import "encoding/json"

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
