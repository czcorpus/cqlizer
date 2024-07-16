package cql

import (
	"encoding/json"
)

type state int

const (
	ConstChar state = iota
	Repeat
	QMark
)

type RgSimple struct {
	effect float64
	// RgRange / RgChar / RgAlt / RgPosixClass
	origValue string
	Values    []any
}

func (r *RgSimple) Text() string {
	return r.origValue
}

func (r *RgSimple) Effect() float64 {
	if r.effect == 0 {
		r.effect = 1
	}
	return r.effect
}

func (r *RgSimple) SetEffect(v float64) {
	r.effect = v
}

func (r *RgSimple) IsLeaf() bool {
	return false
}

func (r *RgSimple) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion RgSimple
		Effect    float64
	}{
		RuleName:  "RgSimple",
		Expansion: *r,
		Effect:    r.effect,
	})
}

func (r *RgSimple) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	for _, item := range r.Values {
		switch tItem := item.(type) {
		case *RgRange:
			tItem.ForEachElement(r, fn)
		case *RgChar:
			tItem.ForEachElement(r, fn)
		case *RgAlt:
			tItem.ForEachElement(r, fn)
		case *RgPosixClass:
			tItem.ForEachElement(r, fn)
		}
	}
}

func (r *RgSimple) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(r)
	for _, item := range r.Values {
		switch tItem := item.(type) {
		case *RgRange:
			tItem.DFS(fn, path)
		case *RgChar:
			tItem.DFS(fn, path)
		case *RgAlt:
			tItem.DFS(fn, path)
		case *RgPosixClass:
			tItem.DFS(fn, path)
		}
	}
	fn(r, path)
	path.Pop()

}
