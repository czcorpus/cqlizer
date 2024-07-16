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
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Seq (_ BINOR _ Seq)* / Seq
type Sequence struct {
	effect    float64
	origValue string
	Seq       []*Seq
}

func (q *Sequence) Text() string {
	return q.origValue
}

func (q *Sequence) Effect() float64 {
	if q.effect == 0 {
		q.effect = 1
	}
	return q.effect
}

func (q *Sequence) SetEffect(v float64) {
	q.effect = v
}

func (q *Sequence) IsLeaf() bool {
	return false
}

func (q *Sequence) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Expansion Sequence
		RuleName  string
		Effect    float64
	}{
		RuleName:  "Sequence",
		Expansion: *q,
		Effect:    q.effect,
	})
}

func (q *Sequence) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, q)
	for _, item := range q.Seq {
		item.ForEachElement(q, fn)
	}
}

func (q *Sequence) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(q)
	for _, item := range q.Seq {
		item.DFS(fn, path)
	}
	fn(q, path)
	path.Pop()
}

// --------------------------------------------------------------------

type Seq struct {
	effect      float64
	origValue   string
	isOrChained bool
	Not         ASTString
	Repetition  []*Repetition
}

func (q *Seq) IsOrChained() bool {
	return q.isOrChained
}

func (q *Seq) NumPositions() int {
	return len(q.Repetition)
}

func (s *Seq) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, s)
	fn(parent, s.Not)
	for _, item := range s.Repetition {
		item.ForEachElement(s, fn)
	}
}

func (s *Seq) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(s)
	fn(s.Not, path)
	for _, item := range s.Repetition {
		item.DFS(fn, path)
	}
	fn(s, path)
	path.Pop()
}

func (s *Seq) Text() string {
	return string(s.origValue)
}

func (s *Seq) Effect() float64 {
	if s.effect == 0 {
		s.effect = 1
	}
	return s.effect
}

func (s *Seq) SetEffect(v float64) {
	s.effect = v
}

func (q *Seq) IsLeaf() bool {
	return false
}

// -----------------------------------------------------

// GlobPart
// gc:GlobCond gc2:(_ BINAND _ GlobCond)*
type GlobPart struct {
	effect   float64
	GlobCond []*GlobCond
}

func (g *GlobPart) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Expansion GlobPart
		RuleName  string
		Effect    float64
	}{
		RuleName:  "GlobPart",
		Expansion: *g,
		Effect:    g.effect,
	})
}

func (q *GlobPart) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, q)
	for _, item := range q.GlobCond {
		item.ForEachElement(q, fn)
	}
}

func (q *GlobPart) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(q)
	for _, item := range q.GlobCond {
		item.DFS(fn, path)
	}
	fn(q, path)
	path.Pop()
}

func (q *GlobPart) Text() string {
	return "#GlobPart" // TODO
}

func (q *GlobPart) Effect() float64 {
	if q.effect == 0 {
		q.effect = 1
	}
	return q.effect
}

func (q *GlobPart) SetEffect(v float64) {
	q.effect = v
}

func (q *GlobPart) IsLeaf() bool {
	return false
}

// ---------------------------------------

// WithinOrContaining
//
//	NOT? (KW_WITHIN / KW_CONTAINING) _ WithinContainingPart {
type WithinOrContaining struct {
	effect                float64
	numWithinParts        int
	numNegWithinParts     int
	numContainingParts    int
	numNegContainingParts int
	KwWithin              ASTString
	KwContaining          ASTString
	WithinContainingPart  *WithinContainingPart
}

func (w *WithinOrContaining) NumWithinParts() int {
	return w.numWithinParts
}

func (w *WithinOrContaining) NumNegWithinParts() int {
	return w.numNegWithinParts
}

func (w *WithinOrContaining) NumContainingParts() int {
	return w.numContainingParts
}

func (w *WithinOrContaining) NumNegContainingParts() int {
	return w.numNegContainingParts
}

func (w *WithinOrContaining) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Expansion WithinOrContaining
		RuleName  string
		Effect    float64
	}{
		RuleName:  "WithinOrContaining",
		Expansion: *w,
		Effect:    w.effect,
	})
}

func (w *WithinOrContaining) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, w)
	fn(w, w.KwWithin)
	fn(w, w.KwContaining)
	if w.WithinContainingPart != nil {
		w.WithinContainingPart.ForEachElement(w, fn)
	}
}

func (w *WithinOrContaining) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(w)
	fn(w.KwWithin, path)
	fn(w.KwContaining, path)
	if w.WithinContainingPart != nil {
		w.WithinContainingPart.DFS(fn, path)
	}
	fn(w, path)
	path.Pop()
}

func (w *WithinOrContaining) Text() string {
	return "#WithinOrContaining"
}

func (w *WithinOrContaining) Effect() float64 {
	if w.effect == 0 {
		w.effect = 1
	}
	return w.effect
}

func (w *WithinOrContaining) SetEffect(v float64) {
	w.effect = v
}

func (w *WithinOrContaining) IsLeaf() bool {
	return false
}

// -----------------------------------------------------

type withinContainingPartVariant1 struct {
	Sequence *Sequence
}

type withinContainingPartVariant2 struct {
	WithinNumber *WithinNumber
}

type withinContainingPartVariant3 struct {
	AlignedPart *AlignedPart
}

// WithinContainingPart
//
//	Sequence / WithinNumber / NOT? AlignedPart
type WithinContainingPart struct {
	effect float64

	variant1 *withinContainingPartVariant1

	variant2 *withinContainingPartVariant2

	variant3 *withinContainingPartVariant3
}

func (wcp *WithinContainingPart) Text() string {
	return "#WithinContainingPart"
}

func (wcp *WithinContainingPart) Effect() float64 {
	if wcp.effect == 0 {
		wcp.effect = 1
	}
	return wcp.effect
}

func (wcp *WithinContainingPart) SetEffect(v float64) {
	wcp.effect = v
}

func (wcp *WithinContainingPart) IsLeaf() bool {
	return false
}

func (wcp *WithinContainingPart) MarshalJSON() ([]byte, error) {
	if wcp.variant1 != nil {
		return json.Marshal(struct {
			Expansion withinContainingPartVariant1
			RuleName  string
			Effect    float64
		}{
			Expansion: *wcp.variant1,
			RuleName:  "WithinContainingPart",
			Effect:    wcp.effect,
		})

	} else if wcp.variant2 != nil {
		return json.Marshal(struct {
			Expansion withinContainingPartVariant2
			RuleName  string
		}{
			Expansion: *wcp.variant2,
			RuleName:  "WithinContainingPart",
		})

	} else {
		return json.Marshal(struct{}{})
	}
}

func (wcp *WithinContainingPart) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, wcp)
	if wcp.variant1 != nil {
		wcp.variant1.Sequence.ForEachElement(wcp, fn)

	} else if wcp.variant2 != nil {
		fn(wcp, wcp.variant2.WithinNumber.Value)

	} else if wcp.variant3 != nil {
		wcp.variant3.AlignedPart.ForEachElement(wcp, fn)
	}
}

func (wcp *WithinContainingPart) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(wcp)
	if wcp.variant1 != nil {
		wcp.variant1.Sequence.DFS(fn, path)

	} else if wcp.variant2 != nil {
		fn(wcp.variant2.WithinNumber.Value, path)

	} else if wcp.variant3 != nil {
		wcp.variant3.AlignedPart.DFS(fn, path)
	}
	fn(wcp, path)
	path.Pop()
}

// --------------------------------------------------

// GlobCond
//
// v1: NUMBER DOT AttName _ NOT? EQ _ NUMBER DOT AttName {
//
// v2: KW_FREQ LPAREN _ NUMBER DOT AttName _ RPAREN NOT? _ ( EQ / LEQ / GEQ / LSTRUCT / RSTRUCT ) _ NUMBER {

type globCondVariant1 struct {
	Number1  ASTString
	AttName3 ASTString
	Not4     ASTString
	Eq5      ASTString
	Number6  ASTString
	AttName8 ASTString
}

type globCondVariant2 struct {
	KwFreq1   ASTString
	Number2   ASTString
	AttName3  ASTString
	Not4      ASTString
	Operator5 ASTString
	Number6   ASTString
}

type GlobCond struct {
	effect float64

	variant1 *globCondVariant1

	variant2 *globCondVariant2
}

func (gc *GlobCond) Text() string {
	return "#GlobCond"
}

func (gc *GlobCond) Effect() float64 {
	if gc.effect == 0 {
		gc.effect = 1
	}
	return gc.effect
}

func (gc *GlobCond) SetEffect(v float64) {
	gc.effect = v
}

func (gc *GlobCond) IsLeaf() bool {
	return false
}

func (gc *GlobCond) MarshalJSON() ([]byte, error) {
	if gc.variant1 != nil {
		return json.Marshal(struct {
			Expansion globCondVariant1
			RuleName  string
			Effect    float64
		}{
			Expansion: *gc.variant1,
			RuleName:  "GlobCond",
			Effect:    gc.effect,
		})

	} else if gc.variant2 != nil {
		return json.Marshal(struct {
			Expansion globCondVariant2
			RuleName  string
		}{
			Expansion: *gc.variant2,
			RuleName:  "GlobCond",
		})

	} else {
		return json.Marshal(struct{}{})
	}
}

func (gc *GlobCond) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, gc)
	if gc.variant1 != nil {
		fn(gc, gc.variant1.Number1)
		fn(gc, gc.variant1.AttName3)
		fn(gc, gc.variant1.Not4)
		fn(gc, gc.variant1.Eq5)
		fn(gc, gc.variant1.Number6)
		fn(gc, gc.variant1.AttName8)

	} else if gc.variant2 != nil {
		fn(gc, gc.variant2.KwFreq1)
		fn(gc, gc.variant2.Number2)
		fn(gc, gc.variant2.AttName3)
		fn(gc, gc.variant2.Not4)
		fn(gc, gc.variant2.Operator5)
		fn(gc, gc.variant2.Number6)
	}
}

func (gc *GlobCond) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(gc)
	if gc.variant1 != nil {
		fn(gc.variant1.Number1, path)
		fn(gc.variant1.AttName3, path)
		fn(gc.variant1.Not4, path)
		fn(gc.variant1.Eq5, path)
		fn(gc.variant1.Number6, path)
		fn(gc.variant1.AttName8, path)

	} else if gc.variant2 != nil {
		fn(gc.variant2.KwFreq1, path)
		fn(gc.variant2.Number2, path)
		fn(gc.variant2.AttName3, path)
		fn(gc.variant2.Not4, path)
		fn(gc.variant2.Operator5, path)
		fn(gc.variant2.Number6, path)
	}
	fn(gc, path)
	path.Pop()
}

// ----------------------------------------------------

// Structure
//
// AttName _ AttValList?
type Structure struct {
	effect     float64
	AttName    ASTString
	AttValList *AttValList
}

func (s *Structure) Text() string {
	return s.AttName.Text()
}

func (s *Structure) Effect() float64 {
	if s.effect == 0 {
		s.effect = 1
	}
	return s.effect
}

func (s *Structure) SetEffect(v float64) {
	s.effect = v
}

func (s *Structure) IsLeaf() bool {
	return false
}

func (s *Structure) isBigStructure() bool {
	v := s.AttName.Text()
	return v == "s" || v == "g" || v == "p"
}

func (s *Structure) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion Structure
		Effect    float64
	}{
		RuleName:  "Structure",
		Expansion: *s,
		Effect:    s.effect,
	})
}

func (s *Structure) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, s)
	fn(s, s.AttName)
	if s.AttValList != nil {
		s.AttValList.ForEachElement(s, fn)
	}
}

func (s *Structure) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(s)
	fn(s.AttName, path)
	if s.AttValList != nil {
		s.AttValList.DFS(fn, path)
	}
	fn(s, path)
	path.Pop()
}

// ---------------------------------------------------------

// AttValList
//
//	av1:AttValAnd av2:(_ BINOR _ AttValAnd)*
type AttValList struct {
	effect    float64
	origValue string
	AttValAnd []*AttValAnd
}

func (a *AttValList) Text() string {
	return a.origValue
}

func (a *AttValList) Effect() float64 {
	if a.effect == 0 {
		a.effect = 1
	}
	return a.effect
}

func (a *AttValList) SetEffect(v float64) {
	a.effect = v
}

func (a *AttValList) IsLeaf() bool {
	return false
}

func (a *AttValList) IsEmpty() bool {
	return a == nil || len(a.AttValAnd) == 0
}

func (a *AttValList) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, a)
	for _, v := range a.AttValAnd {
		v.ForEachElement(a, fn)
	}
}

func (a *AttValList) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(a)
	for _, v := range a.AttValAnd {
		v.DFS(fn, path)
	}
	fn(a, path)
	path.Pop()
}

func (a *AttValList) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion AttValList
		Effect    float64
	}{
		RuleName:  "AttValList",
		Expansion: *a,
		Effect:    a.effect,
	})
}

// -----------------------------------------------------------

// NumberedPosition
//
// NUMBER COLON OnePosition
type NumberedPosition struct {
	effect      float64
	Number      ASTString
	Colon       ASTString
	OnePosition *OnePosition
}

func (n *NumberedPosition) Text() string {
	return "#NumberedPosition"
}

func (n *NumberedPosition) Effect() float64 {
	if n.effect == 0 {
		n.effect = 1
	}
	return n.effect
}

func (n *NumberedPosition) SetEffect(v float64) {
	n.effect = v
}

func (n *NumberedPosition) IsLeaf() bool {
	return false
}

func (n *NumberedPosition) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, n)
	fn(n, n.Number)
	fn(n, n.Colon)
	if n.OnePosition != nil {
		n.OnePosition.ForEachElement(n, fn)
	}
}

func (n *NumberedPosition) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(n)
	fn(n.Number, path)
	fn(n.Colon, path)
	if n.OnePosition != nil {
		n.OnePosition.DFS(fn, path)
	}
	fn(n, path)
	path.Pop()
}

// --------------------------------------------------

type onePositionVariant1 struct {
	AttValList *AttValList
}

type onePositionVariant2 struct {
	RegExp *RegExp
}

type onePositionVariant3 struct {
	Number ASTString
	RegExp *RegExp
}

type onePositionVariant4 struct {
	Value ASTString
}

type onePositionVariant5 struct {
	MuPart *MuPart
}

// OnePosition
// var1: LBRACKET _ AttValList? _ RBRACKET
// var2: RegExp
// var3: TEQ NUMBER? RegExp
// var4: KW_MU
// var5: MuPart
type OnePosition struct {
	effect    float64
	origValue string
	Variant1  *onePositionVariant1
	Variant2  *onePositionVariant2
	Variant3  *onePositionVariant3
	Variant4  *onePositionVariant4
	Variant5  *onePositionVariant5
}

func (op *OnePosition) Text() string {
	return op.origValue
}

func (op *OnePosition) Effect() float64 {
	if op.effect == 0 {
		op.effect = 1
	}
	return op.effect
}

func (op *OnePosition) SetEffect(v float64) {
	op.effect = v
}

func (op *OnePosition) IsLeaf() bool {
	return false
}

func (op *OnePosition) MarshalJSON() ([]byte, error) {
	if op.Variant1 != nil {
		return json.Marshal(struct {
			RuleName  string
			RawValue  string
			Expansion *onePositionVariant1
			Effect    float64
		}{
			RuleName:  "OnePosition",
			RawValue:  op.Text(),
			Expansion: op.Variant1,
			Effect:    op.effect,
		})

	} else if op.Variant2 != nil {
		return json.Marshal(struct {
			RuleName  string
			RawValue  string
			Expansion *onePositionVariant2
		}{
			RuleName:  "OnePosition",
			RawValue:  op.Text(),
			Expansion: op.Variant2,
		})

	} else if op.Variant3 != nil {
		return json.Marshal(struct {
			RuleName  string
			RawValue  string
			Expansion *onePositionVariant3
		}{
			RuleName:  "OnePosition",
			RawValue:  op.Text(),
			Expansion: op.Variant3,
		})

	} else if op.Variant4 != nil {
		return json.Marshal(struct {
			RuleName  string
			RawValue  string
			Expansion *onePositionVariant4
		}{
			RuleName:  "OnePosition",
			RawValue:  op.Text(),
			Expansion: op.Variant4,
		})

	} else if op.Variant5 != nil {
		return json.Marshal(struct {
			RuleName  string
			RawValue  string
			Expansion *onePositionVariant5
		}{
			RuleName:  "OnePosition",
			RawValue:  op.Text(),
			Expansion: op.Variant5,
		})

	} else {
		return json.Marshal(struct{}{})
	}
}

func (op *OnePosition) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, op)
	if op.Variant1 != nil && op.Variant1.AttValList != nil {
		op.Variant1.AttValList.ForEachElement(op, fn)

	} else if op.Variant2 != nil {
		op.Variant2.RegExp.ForEachElement(op, fn)

	} else if op.Variant3 != nil {
		fn(op, op.Variant3.Number)
		op.Variant3.RegExp.ForEachElement(op, fn)

	} else if op.Variant4 != nil {
		fn(op, op.Variant4.Value)

	} else if op.Variant5 != nil {
		op.Variant5.MuPart.ForEachElement(op, fn)
	}
}

func (op *OnePosition) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(op)
	if op.Variant1 != nil && op.Variant1.AttValList != nil {
		op.Variant1.AttValList.DFS(fn, path)

	} else if op.Variant2 != nil {
		op.Variant2.RegExp.DFS(fn, path)

	} else if op.Variant3 != nil {
		fn(op.Variant3.Number, path)
		op.Variant3.RegExp.DFS(fn, path)

	} else if op.Variant4 != nil {
		fn(op.Variant4.Value, path)

	} else if op.Variant5 != nil {
		op.Variant5.MuPart.DFS(fn, path)
	}
	fn(op, path)
	path.Pop()
}

// -----------------------------------------------------

type positionVariant1 struct {
	OnePosition *OnePosition
}

type positionVariant2 struct {
	NumberedPosition *NumberedPosition
}

// Position
//
//	OnePosition / NumberedPosition
type Position struct {
	effect    float64
	origValue string
	variant1  *positionVariant1
	variant2  *positionVariant2
}

func (p *Position) Text() string {
	return p.origValue
}

func (p *Position) Effect() float64 {
	if p.effect == 0 {
		p.effect = 1
	}
	return p.effect
}

func (p *Position) SetEffect(v float64) {
	p.effect = v
}

func (p *Position) IsLeaf() bool {
	return false
}

func (p *Position) MarshalJSON() ([]byte, error) {
	if p.variant1 != nil {
		return json.Marshal(struct {
			RuleName  string
			Expansion *positionVariant1
			Effect    float64
		}{
			RuleName:  "Position",
			Expansion: p.variant1,
			Effect:    p.effect,
		})

	} else if p.variant2 != nil {
		return json.Marshal(struct {
			RuleName  string
			Expansion *positionVariant2
		}{
			RuleName:  "Position",
			Expansion: p.variant2,
		})

	} else {
		return json.Marshal(struct{}{})
	}
}

func (p *Position) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, p)
	if p.variant1 != nil {
		p.variant1.OnePosition.ForEachElement(p, fn)

	} else if p.variant2 != nil {
		p.variant2.NumberedPosition.ForEachElement(p, fn)
	}
}

func (p *Position) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(p)
	if p.variant1 != nil {
		p.variant1.OnePosition.DFS(fn, path)

	} else if p.variant2 != nil {
		p.variant2.NumberedPosition.DFS(fn, path)
	}
	fn(p, path)
	path.Pop()
}

// -------------------------------------------------------

type RegExp struct {
	effect    float64
	origValue string
	RegExpRaw []*RegExpRaw // these are A|B|C
}

func (r *RegExp) Text() string {
	return r.origValue
}

func (r *RegExp) Effect() float64 {
	if r.effect == 0 {
		r.effect = 1
	}
	return r.effect
}

func (r *RegExp) SetEffect(v float64) {
	r.effect = v
}

func (r *RegExp) IsLeaf() bool {
	return false
}

func (r *RegExp) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	for _, v := range r.RegExpRaw {
		v.ForEachElement(r, fn)
	}
}

func (r *RegExp) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(r)
	for _, v := range r.RegExpRaw {
		v.DFS(fn, path)
	}
	fn(r, path)
	path.Pop()
}

// --------------------------------------------------------

type muPartVariant1 struct {
	UnionOp *UnionOp
}

type muPartVariant2 struct {
	MeetOp *MeetOp
}

type MuPart struct {
	effect    float64
	origValue string
	Variant1  *muPartVariant1
	Variant2  *muPartVariant2
}

func (m *MuPart) Text() string {
	return m.origValue
}

func (m *MuPart) Effect() float64 {
	if m.effect == 0 {
		m.effect = 1
	}
	return m.effect
}

func (m *MuPart) SetEffect(v float64) {
	m.effect = v
}

func (m *MuPart) IsLeaf() bool {
	return false
}

func (m *MuPart) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion MuPart
		Effect    float64
	}{
		RuleName:  "MuPart",
		Expansion: *m,
		Effect:    m.effect,
	})
}

func (m *MuPart) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, m)
	if m.Variant1 != nil {
		m.Variant1.UnionOp.ForEachElement(m, fn)

	} else if m.Variant2 != nil {
		m.Variant2.MeetOp.ForEachElement(m, fn)
	}
}

func (m *MuPart) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(m)
	if m.Variant1 != nil {
		m.Variant1.UnionOp.DFS(fn, path)

	} else if m.Variant2 != nil {
		m.Variant2.MeetOp.DFS(fn, path)
	}
	fn(m, path)
	path.Pop()
}

// --------------------------------------------------------------

type UnionOp struct {
	effect    float64
	origValue string
	Position1 *Position
	Position2 *Position
}

func (m *UnionOp) Text() string {
	return m.origValue
}

func (m *UnionOp) Effect() float64 {
	if m.effect == 0 {
		m.effect = 1
	}
	return m.effect
}

func (m *UnionOp) SetEffect(v float64) {
	m.effect = v
}

func (m *UnionOp) IsLeaf() bool {
	return false
}

func (m *UnionOp) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, m)
	m.Position1.ForEachElement(m, fn)
	m.Position2.ForEachElement(m, fn)
}

func (m *UnionOp) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(m)
	m.Position1.DFS(fn, path)
	m.Position2.DFS(fn, path)
	fn(m, path)
	path.Pop()
}

// ---------------------------------------------------------------

type MeetOp struct {
	effect    float64
	origValue string
	Position1 *Position
	Position2 *Position
}

func (m *MeetOp) Text() string {
	return m.origValue
}

func (m *MeetOp) Effect() float64 {
	if m.effect == 0 {
		m.effect = 1
	}
	return m.effect
}

func (m *MeetOp) SetEffect(v float64) {
	m.effect = v
}

func (m *MeetOp) IsLeaf() bool {
	return false
}

func (m *MeetOp) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, m)
	m.Position1.ForEachElement(m, fn)
	m.Position2.ForEachElement(m, fn)
}

func (m *MeetOp) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(m)
	m.Position1.DFS(fn, path)
	m.Position2.DFS(fn, path)
	fn(m, path)
	path.Pop()
}

// --------------------------------------------------------------------------

type Integer struct {
}

// ------------------------------------------------------------

type repetitionVariant1 struct {
	AtomQuery *AtomQuery
	RepOpt    *RepOpt
}

type repetitionVariant2 struct {
	OpenStructTag *OpenStructTag
}

type repetitionVariant3 struct {
	CloseStructTag *CloseStructTag
}

type Repetition struct {
	effect         float64
	origValue      string
	isTailPosition bool
	Variant1       *repetitionVariant1
	Variant2       *repetitionVariant2
	Variant3       *repetitionVariant3
}

func (r *Repetition) isAnyPosition() bool {
	if r.Variant1 != nil && r.Variant1.AtomQuery.variant1 != nil &&
		r.Variant1.AtomQuery.variant1.Position.variant1 != nil &&
		r.Variant1.AtomQuery.variant1.Position.variant1.OnePosition.Variant1 != nil {
		return r.Variant1.AtomQuery.variant1.Position.variant1.OnePosition.Variant1.AttValList == nil ||
			len(r.Variant1.AtomQuery.variant1.Position.variant1.OnePosition.Variant1.AttValList.AttValAnd) == 0
	}
	return false
}

func (r *Repetition) Text() string {
	return r.origValue
}

func (r *Repetition) Effect() float64 {
	if r.effect == 0 {
		r.effect = 1
	}
	return r.effect
}

func (r *Repetition) SetEffect(v float64) {
	r.effect = v
}

func (r *Repetition) IsLeaf() bool {
	return false
}

func (r *Repetition) GetRepOpt() string {
	if r.Variant1 != nil && r.Variant1.RepOpt != nil {
		return string(r.Variant1.RepOpt.Text())
	}
	return ""
}

func (r *Repetition) GetReptOptRange() [2]int {
	if r.Variant1 != nil && r.Variant1.RepOpt != nil && r.Variant1.RepOpt.Variant2 != nil {
		v1, err := strconv.Atoi(string(r.Variant1.RepOpt.Variant2.From))
		if err != nil {
			panic("failed to parse ReptOpt range")
		}
		ans := [2]int{v1, -1}
		if r.Variant1.RepOpt.Variant2.To != "" {
			v2, err := strconv.Atoi(string(r.Variant1.RepOpt.Variant2.To))
			if err != nil {
				panic("failed to parse ReptOpt range")
			}
			ans[1] = v2
		}
		return ans
	}
	return [2]int{-1, -1}
}

func (r *Repetition) IsTailPosition() bool {
	return r.isTailPosition
}

func (r *Repetition) MarshalJSON() ([]byte, error) {
	var variant any
	if r.Variant1 != nil {
		variant = r.Variant1

	} else if r.Variant2 != nil {
		variant = r.Variant2

	} else if r.Variant3 != nil {
		variant = r.Variant3

	} else {
		variant = struct{}{}
	}
	return json.Marshal(
		struct {
			RuleName      string
			Expansion     any
			IsAnyPosition bool
			Effect        float64
		}{
			RuleName:      "Repetition",
			Expansion:     variant,
			IsAnyPosition: r.isAnyPosition(),
			Effect:        r.effect,
		})
}

func (r *Repetition) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	if r.Variant1 != nil {
		r.Variant1.AtomQuery.ForEachElement(r, fn)
		if r.Variant1.RepOpt != nil {
			fn(r, r.Variant1.RepOpt)
		}

	} else if r.Variant2 != nil {
		r.Variant2.OpenStructTag.ForEachElement(r, fn)

	} else if r.Variant3 != nil {
		r.Variant3.CloseStructTag.ForEachElement(r, fn)
	}
}

func (r *Repetition) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(r)
	if r.Variant1 != nil {
		r.Variant1.AtomQuery.DFS(fn, path)
		fn(r.Variant1.RepOpt, path)

	} else if r.Variant2 != nil {
		r.Variant2.OpenStructTag.DFS(fn, path)

	} else if r.Variant3 != nil {
		r.Variant3.CloseStructTag.DFS(fn, path)
	}
	fn(r, path)
	path.Pop()
}

// ----------------------------------------------------------------

type atomQueryVariant1 struct {
	Position *Position
}

type atomQueryVariant2 struct {
	Sequence             *Sequence
	WithinContainingPart *WithinContainingPart
}

// AtomQuery
// var1: Position
// var2: LPAREN _ Sequence (_ NOT? (KW_WITHIN / KW_CONTAINING) _ WithinContainingPart)* _ RPAREN {
type AtomQuery struct {
	effect                float64
	origValue             string
	variant1              *atomQueryVariant1
	variant2              *atomQueryVariant2
	numWithinParts        int
	numNegWithinParts     int
	numContainingParts    int
	numNegContainingParts int
}

func (aq *AtomQuery) Text() string {
	return aq.origValue
}

func (aq *AtomQuery) Effect() float64 {
	if aq.effect == 0 {
		aq.effect = 1
	}
	return aq.effect
}

func (aq *AtomQuery) SetEffect(v float64) {
	aq.effect = v
}

func (aq *AtomQuery) IsLeaf() bool {
	return false
}

func (aq *AtomQuery) MarshalJSON() ([]byte, error) {
	if aq.variant1 != nil {
		return json.Marshal(struct {
			RuleName  string
			Expansion *atomQueryVariant1
			Effect    float64
		}{
			RuleName:  "AtomQuery",
			Expansion: aq.variant1,
			Effect:    aq.effect,
		})

	} else if aq.variant2 != nil {
		return json.Marshal(struct {
			RuleName  string
			Expansion *atomQueryVariant2
		}{
			RuleName:  "AtomQuery",
			Expansion: aq.variant2,
		})

	} else {
		return json.Marshal(struct{}{})
	}
}

func (aq *AtomQuery) NumWithinParts() int {
	return aq.numWithinParts
}

func (aq *AtomQuery) NumNegWithinParts() int {
	return aq.numNegWithinParts
}

func (aq *AtomQuery) NumContainingParts() int {
	return aq.numContainingParts
}

func (aq *AtomQuery) NumNegContainingParts() int {
	return aq.numNegContainingParts
}

func (aq *AtomQuery) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, aq)
	if aq.variant1 != nil {
		aq.variant1.Position.ForEachElement(aq, fn)

	} else if aq.variant2 != nil {
		aq.variant2.Sequence.ForEachElement(aq, fn)
		if aq.variant2.WithinContainingPart != nil {
			aq.variant2.WithinContainingPart.ForEachElement(aq, fn)
		}
	}
}

func (aq *AtomQuery) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(aq)
	if aq.variant1 != nil {
		aq.variant1.Position.DFS(fn, path)

	} else if aq.variant2 != nil {
		aq.variant2.Sequence.DFS(fn, path)
		if aq.variant2.WithinContainingPart != nil {
			aq.variant2.WithinContainingPart.DFS(fn, path)
		}
	}
	fn(aq, path)
	path.Pop()
}

// --------------------------------------------------------------

type repOptVariant1 struct {
	Value ASTString
}

type repOptVariant2 struct {
	From ASTString
	To   ASTString
}

type RepOpt struct {
	effect   float64
	Variant1 *repOptVariant1
	Variant2 *repOptVariant2
}

func (r *RepOpt) DefinesInfReps() bool {
	return r.Variant1 != nil && (r.Variant1.Value == "+" || r.Variant1.Value == "*")
}

func (r *RepOpt) Text() string {
	if r.Variant1 != nil {
		return r.Variant1.Value.Text()

	} else if r.Variant2 != nil {
		return fmt.Sprintf("{%s, %s}", r.Variant2.From, r.Variant2.To)
	}
	return ""
}

func (r *RepOpt) Effect() float64 {
	if r.effect == 0 {
		r.effect = 1
	}
	return r.effect
}

func (r *RepOpt) SetEffect(v float64) {
	r.effect = v
}

func (r *RepOpt) IsLeaf() bool {
	return false
}

func (r *RepOpt) MarshalJSON() ([]byte, error) {
	if r.Variant1 != nil {
		return json.Marshal(struct {
			RuleName  string
			Expansion repOptVariant1
			Effect    float64
		}{
			RuleName:  "RepOpt",
			Expansion: *r.Variant1,
			Effect:    r.effect,
		})

	} else if r.Variant2 != nil {
		return json.Marshal(struct {
			RuleName  string
			Expansion repOptVariant2
		}{
			RuleName:  "RepOpt",
			Expansion: *r.Variant2,
		})

	} else {
		return json.Marshal(struct{}{})
	}
}

func (r *RepOpt) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	if r.Variant1 != nil {
		fn(r, r.Variant1.Value)

	} else if r.Variant2 != nil {
		fn(r, r.Variant2.From)
		fn(r, r.Variant2.To)
	}
}

func (r *RepOpt) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(r)
	if r.Variant1 != nil {
		fn(r.Variant1.Value, path)

	} else if r.Variant2 != nil {
		fn(r.Variant2.From, path)
		fn(r.Variant2.To, path)
	}
	fn(r, path)
	path.Pop()
}

// ----------------------------------------------------------------

type OpenStructTag struct {
	effect    float64
	origValue string
	Structure *Structure
}

func (ost *OpenStructTag) Text() string {
	return ost.origValue
}

func (ost *OpenStructTag) Effect() float64 {
	if ost.effect == 0 {
		ost.effect = 1
	}
	return ost.effect
}

func (ost *OpenStructTag) SetEffect(v float64) {
	ost.effect = v
}

func (ost *OpenStructTag) IsLeaf() bool {
	return false
}

func (ost *OpenStructTag) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion OpenStructTag
		Effect    float64
	}{
		RuleName:  "OpenStructTag",
		Expansion: *ost,
		Effect:    ost.effect,
	})
}

func (ost *OpenStructTag) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, ost)
	ost.Structure.ForEachElement(ost, fn)
}

func (ost *OpenStructTag) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(ost)
	ost.Structure.DFS(fn, path)
	fn(ost, path)
	path.Pop()
}

// --------------------------------------------------------------

type CloseStructTag struct {
	effect    float64
	Structure *Structure
}

func (ost *CloseStructTag) Text() string {
	return "#CloseStructTag"
}

func (ost *CloseStructTag) Effect() float64 {
	if ost.effect == 0 {
		ost.effect = 1
	}
	return ost.effect
}

func (ost *CloseStructTag) SetEffect(v float64) {
	ost.effect = v
}

func (ost *CloseStructTag) IsLeaf() bool {
	return false
}

func (ost *CloseStructTag) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion CloseStructTag
		Effect    float64
	}{
		RuleName:  "CloseStructTag",
		Expansion: *ost,
		Effect:    ost.effect,
	})
}

func (ost *CloseStructTag) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, ost)
	ost.Structure.ForEachElement(ost, fn)
}

func (ost *CloseStructTag) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(ost)
	ost.Structure.DFS(fn, path)
	fn(ost, path)
	path.Pop()
}

// ---------------------------------------------------------

type AlignedPart struct {
	effect float64
}

func (a *AlignedPart) Text() string {
	return "#AlignedPart"
}

func (a *AlignedPart) Effect() float64 {
	if a.effect == 0 {
		a.effect = 1000000
	}
	return a.effect
}

func (a *AlignedPart) SetEffect(v float64) {
	a.effect = v
}

func (a *AlignedPart) IsLeaf() bool {
	return true // TODO
}

func (a *AlignedPart) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, a)
	// TODO
}

func (a *AlignedPart) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(a)
	fn(a, path)
	path.Pop()
}

// -----------------------------------------------------------

// AttValAnd
//
//	av1:AttVal av2:(_ BINAND _ AttVal)*
type AttValAnd struct {
	effect float64
	AttVal []*AttVal
}

func (a *AttValAnd) Text() string {
	return "#AttValAnd"
}

func (a *AttValAnd) Effect() float64 {
	if a.effect == 0 {
		a.effect = 1
	}
	return a.effect
}

func (a *AttValAnd) SetEffect(v float64) {
	a.effect = v
}

func (a *AttValAnd) IsLeaf() bool {
	return false
}

func (a *AttValAnd) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, a)
	for _, item := range a.AttVal {
		item.ForEachElement(a, fn)
	}
}

func (a *AttValAnd) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(a)
	for _, item := range a.AttVal {
		item.DFS(fn, path)
	}
	fn(a, path)
	path.Pop()
}

// --------------------------------------------------------------

// AttName _ (NOT)? EEQ _ RawString
type attValVariant1 struct {
	AttName   ASTString
	Not       bool
	Eeq       ASTString
	RawString *RawString
}

// AttName (_ NOT)? _ (EQ / LEQ / GEQ / TEQ NUMBER?) _ RegExp
type attValVariant2 struct {
	AttName ASTString
	Not     bool
	Op      ASTString
	RegExp  *RegExp
}

// POSNUM NUMBER DASH NUMBER
type attValVariant3 struct {
}

// POSNUM NUMBER
type attValVariant4 struct {
}

// NOT AttVal
type attValVariant5 struct {
	AttVal *AttVal
}

// LPAREN _ AttValList _ RPAREN
type attValVariant6 struct {
	AttValList *AttValList
}

// (KW_WS / KW_TERM) LPAREN _ (NUMBER COMMA NUMBER / RegExp COMMA RegExp COMMA RegExp) _ RPAREN
type attValVariant7 struct {
}

// KW_SWAP LPAREN _ NUMBER COMMA AttValList _ RPAREN
type attValVariant8 struct {
}

// KW_CCOLL LPAREN _ NUMBER COMMA NUMBER COMMA AttValList _ RPAREN
type attValVariant9 struct {
}

type AttVal struct {
	effect    float64
	origValue string
	Variant1  *attValVariant1
	Variant2  *attValVariant2
	Variant3  *attValVariant3
	Variant4  *attValVariant4
	Variant5  *attValVariant5
	Variant6  *attValVariant6
	Variant7  *attValVariant7
	Variant8  *attValVariant8
	Variant9  *attValVariant9
}

func (a *AttVal) Text() string {
	return a.origValue
}

func (a *AttVal) Effect() float64 {
	if a.effect == 0 {
		if a.isNegation() {
			a.effect = -1

		} else {
			a.effect = 1
		}
	}
	return a.effect
}

func (a *AttVal) SetEffect(v float64) {
	a.effect = v
}

func (a *AttVal) IsLeaf() bool {
	return false
}

func (a *AttVal) isNegation() bool {
	return a.Variant1 != nil && a.Variant1.Not ||
		a.Variant2 != nil && a.Variant2.Not
}

func (a *AttVal) isProblematicAttrSearch() bool {
	if a.Variant1 != nil {
		return (a.Variant1.AttName == "tag" || a.Variant1.AttName == "pos" || a.Variant1.AttName == "verbtag" ||
			a.Variant1.AttName == "upos" || a.Variant1.AttName == "afun" || a.Variant1.AttName == "case") &&
			len(a.Variant1.RawString.Text()) < 6 && // TODO
			(strings.Contains(a.Variant1.RawString.Text(), ".*") || strings.Contains(a.Variant1.RawString.Text(), ".+"))

	} else if a.Variant2 != nil {
		return (a.Variant2.AttName == "tag" || a.Variant2.AttName == "pos" || a.Variant2.AttName == "verbtag" ||
			a.Variant2.AttName == "upos" || a.Variant2.AttName == "afun" || a.Variant2.AttName == "case") &&
			len(a.Variant2.RegExp.Text()) < 6 && // TODO
			(strings.Contains(a.Variant2.RegExp.Text(), ".*") || strings.Contains(a.Variant2.RegExp.Text(), ".+"))
	}
	return false
}

func (a *AttVal) MarshalJSON() ([]byte, error) {
	var variant any
	if a.Variant1 != nil {
		variant = a.Variant1

	} else if a.Variant2 != nil {
		variant = a.Variant2

	} else if a.Variant3 != nil {
		variant = a.Variant3

	} else if a.Variant4 != nil {
		variant = a.Variant4

	} else if a.Variant5 != nil {
		variant = a.Variant5

	} else if a.Variant6 != nil {
		variant = a.Variant6

	} else if a.Variant7 != nil {
		variant = a.Variant7

	} else if a.Variant8 != nil {
		variant = a.Variant8

	} else if a.Variant9 != nil {
		variant = a.Variant9

	} else {
		variant = struct{}{}
	}
	return json.Marshal(struct {
		RuleName                string
		Expansion               any
		IsProblematicAttrSearch bool
		Effect                  float64
	}{
		RuleName:                "AttVal",
		Expansion:               variant,
		IsProblematicAttrSearch: a.isProblematicAttrSearch(),
		Effect:                  a.effect,
	})
}

func (a *AttVal) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, a)
	if a.Variant1 != nil {
		fn(a, a.Variant1.AttName)
		fn(a, a.Variant1.Eeq)
		a.Variant1.RawString.ForEachElement(a, fn)

	} else if a.Variant2 != nil {
		fn(a, a.Variant2.AttName)
		fn(a, a.Variant2.Op)
		a.Variant2.RegExp.ForEachElement(a, fn)

	} else if a.Variant3 != nil {
		// TODO a.variant3

	} else if a.Variant4 != nil {
		// TODO a.variant4

	} else if a.Variant5 != nil {
		a.Variant5.AttVal.ForEachElement(a, fn)

	} else if a.Variant6 != nil {
		a.Variant6.AttValList.ForEachElement(a, fn)

	} else if a.Variant7 != nil {
		// TODO a.variant7

	} else if a.Variant8 != nil {
		// TODO a.variant8

	} else if a.Variant9 != nil {
		// TODO a.variant9
	}
}

func (a *AttVal) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(a)
	if a.Variant1 != nil {
		fn(a.Variant1.AttName, path)
		fn(a.Variant1.Eeq, path)
		a.Variant1.RawString.DFS(fn, path)

	} else if a.Variant2 != nil {
		fn(a.Variant2.AttName, path)
		fn(a.Variant2.Op, path)
		a.Variant2.RegExp.DFS(fn, path)

	} else if a.Variant3 != nil {
		// TODO a.variant3

	} else if a.Variant4 != nil {
		// TODO a.variant4

	} else if a.Variant5 != nil {
		a.Variant5.AttVal.DFS(fn, path)

	} else if a.Variant6 != nil {
		a.Variant6.AttValList.DFS(fn, path)

	} else if a.Variant7 != nil {
		// TODO a.variant7

	} else if a.Variant8 != nil {
		// TODO a.variant8

	} else if a.Variant9 != nil {
		// TODO a.variant9
	}
	fn(a, path)
	path.Pop()
}

// ---------------------------------------------------

type WithinNumber struct {
	effect float64
	Value  ASTString
}

func (w *WithinNumber) Text() string {
	return "#WithinNumber"
}

func (w *WithinNumber) Effect() float64 {
	if w.effect == 0 {
		w.effect = 10000000
	}
	return w.effect
}

func (w *WithinNumber) SetEffect(v float64) {
	w.effect = v
}

func (w *WithinNumber) IsLeaf() bool {
	return true
}

func (w *WithinNumber) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion WithinNumber
		Effect    float64
	}{
		RuleName:  "WithinNumber",
		Expansion: *w,
		Effect:    w.effect,
	})
}

func (w *WithinNumber) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, w.Value)
}

func (w *WithinNumber) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(w)
	fn(w.Value, path)
	path.Pop()
}

// ----------------------------------------------------------

type RegExpRaw struct {
	effect    float64
	origValue string
	// RgLook / RgGrouped / RgSimple
	Values []any
}

func (r *RegExpRaw) Text() string {
	return r.origValue
}

func (r *RegExpRaw) Effect() float64 {
	if r.effect == 0 {
		r.effect = 1
	}
	return r.effect
}

func (r *RegExpRaw) SetEffect(v float64) {
	r.effect = v
}

func (r *RegExpRaw) IsLeaf() bool {
	return false
}

func (r *RegExpRaw) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion RegExpRaw
		Effect    float64
	}{
		RuleName:  "RegExpRaw",
		Expansion: *r,
		Effect:    r.effect,
	})
}

func (r *RegExpRaw) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	for _, item := range r.Values {
		switch tItem := item.(type) {
		case *RgLook:
			tItem.ForEachElement(r, fn)
		case *RgGrouped:
			tItem.ForEachElement(r, fn)
		case *RgSimple:
			tItem.ForEachElement(r, fn)
		}
	}
}

func (r *RegExpRaw) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(r)
	for _, item := range r.Values {
		switch tItem := item.(type) {
		case *RgLook:
			tItem.DFS(fn, path)
		case *RgGrouped:
			tItem.DFS(fn, path)
		case *RgSimple:
			tItem.DFS(fn, path)
		}
	}
	fn(r, path)
	path.Pop()
}

// ------------------------------------------------------------------

type RawString struct {
	effect       float64
	SimpleString *SimpleString
}

func (r *RawString) Text() string {
	return "#RawString"
}

func (r *RawString) Effect() float64 {
	if r.effect == 0 {
		r.effect = 1
	}
	return r.effect
}

func (r *RawString) SetEffect(v float64) {
	r.effect = v
}

func (r *RawString) IsLeaf() bool {
	return false
}

func (r *RawString) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion RawString
		Effect    float64
	}{
		RuleName:  "RawString",
		Expansion: *r,
		Effect:    r.effect,
	})
}

func (r *RawString) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	r.SimpleString.ForEachElement(r, fn)
}

func (r *RawString) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(r)
	r.SimpleString.DFS(fn, path)
	fn(r, path)
	path.Pop()
}

// ------------------------------------------------------------------------

type SimpleString struct {
	effect    float64
	origValue string
	Values    []ASTString
}

func (r *SimpleString) Text() string {
	var ans strings.Builder
	for _, v := range r.Values {
		ans.WriteString(string(v))
	}
	return ans.String()
}

func (r *SimpleString) Effect() float64 {
	if r.effect == 0 {
		r.effect = 1
	}
	return r.effect
}

func (r *SimpleString) SetEffect(v float64) {
	r.effect = v
}

func (r *SimpleString) IsLeaf() bool {
	return true
}

func (r *SimpleString) UppercaseRatio() float64 {
	var upper int
	src := []rune(r.origValue)
	for _, v := range src {
		if unicode.IsUpper(v) {
			upper++
		}
	}
	return float64(len(src)) / float64(upper)
}

func (r *SimpleString) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion SimpleString
		Effect    float64
	}{
		RuleName:  "SimpleString",
		Expansion: *r,
		Effect:    r.effect,
	})
}

func (r *SimpleString) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, ASTString(r.Text()))
}

func (r *SimpleString) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(r)
	fn(ASTString(r.Text()), path)
	path.Pop()
}

// -------------------------------------------------

type RgGrouped struct {
	effect float64
	Values []*RegExpRaw // the stuff here is A|B|C...
}

func (r *RgGrouped) Text() string {
	return "#RgGrouped"
}

func (r *RgGrouped) Effect() float64 {
	if r.effect == 0 {
		r.effect = 1
	}
	return r.effect
}

func (r *RgGrouped) SetEffect(v float64) {
	r.effect = v
}

func (r *RgGrouped) IsLeaf() bool {
	return false
}

func (r *RgGrouped) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion RgGrouped
		Effect    float64
	}{
		RuleName:  "RgGrouped",
		Expansion: *r,
		Effect:    r.effect,
	})
}

func (r *RgGrouped) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	for _, v := range r.Values {
		v.ForEachElement(r, fn)
	}
}

func (r *RgGrouped) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(r)
	for _, v := range r.Values {
		v.DFS(fn, path)
	}
	fn(r, path)
	path.Pop()
}

// ----------------------------------------------------

type RgPosixClass struct {
	effect float64
	Value  ASTString
}

func (r *RgPosixClass) Text() string {
	return "#RgPosixClass"
}

func (r *RgPosixClass) Effect() float64 {
	if r.effect == 0 {
		r.effect = 1
	}
	return r.effect
}

func (r *RgPosixClass) SetEffect(v float64) {
	r.effect = v
}

func (r *RgPosixClass) IsLeaf() bool {
	return true
}

func (r *RgPosixClass) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Value)
}

func (r *RgPosixClass) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r.Value)
}

func (r *RgPosixClass) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(r)
	fn(r.Value, path)
	path.Pop()
}

// ----------------------------------------------------

type RgLook struct {
	effect float64
	Value  ASTString
}

func (r *RgLook) Text() string {
	return "#RgLook"
}

func (r *RgLook) Effect() float64 {
	return 10
}

func (r *RgLook) SetEffect(v float64) {
	r.effect = v
}

func (r *RgLook) IsLeaf() bool {
	return true
}

func (r *RgLook) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion RgLook
		Effect    float64
	}{
		RuleName:  "RgLook",
		Expansion: *r,
		Effect:    r.effect,
	})
}

func (r *RgLook) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r.Value)
}

func (r *RgLook) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(r)
	fn(r.Value, path)
	path.Pop()
}

// ----------------------------------------------------

type RgLookOperator struct {
}

// -----------------------------------------------------

type RgAlt struct {
	effect float64
	Values []*RgAltVal
}

func (r *RgAlt) NumItems() int {
	return len(r.Values)
}

func (r *RgAlt) Text() string {
	return "#RgAlt"
}

func (r *RgAlt) Effect() float64 {
	if r.effect == 0 {
		r.effect = 1
	}
	return r.effect
}

func (r *RgAlt) SetEffect(v float64) {
	r.effect = v
}

func (r *RgAlt) IsLeaf() bool {
	return false
}

func (r *RgAlt) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion RgAlt
		Effect    float64
	}{
		RuleName:  "RgAlt",
		Expansion: *r,
		Effect:    r.effect,
	})
}

func (r *RgAlt) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	for _, item := range r.Values {
		item.ForEachElement(r, fn)
	}
}

func (r *RgAlt) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(r)
	for _, item := range r.Values {
		item.DFS(fn, path)
	}
	fn(r, path)
	path.Pop()
}

// --------------------------------------------------------

type rgCharVariant1 struct {
	Value ASTString
}

type rgCharVariant2 struct {
	RgOp *RgOp
}

type rgCharVariant3 struct {
	RgRepeat *RgRepeat
}

type rgCharVariant4 struct {
	RgAny *RgAny
}

type rgCharVariant5 struct {
	RgQM *RgQM
}

type RgChar struct {
	effect   float64
	variant1 *rgCharVariant1
	variant2 *rgCharVariant2
	variant3 *rgCharVariant3
	variant4 *rgCharVariant4
	variant5 *rgCharVariant5
}

func (rc *RgChar) Text() string {
	return "#RgChar"
}

func (rc *RgChar) Effect() float64 {
	if rc.effect == 0 {
		rc.effect = 1
	}
	return rc.effect
}

func (rc *RgChar) SetEffect(v float64) {
	rc.effect = v
}

func (rc *RgChar) IsLeaf() bool {
	return false
}

func (rc *RgChar) MarshalJSON() ([]byte, error) {
	var variant any
	if rc.variant1 != nil {
		variant = rc.variant1

	} else if rc.variant2 != nil {
		variant = rc.variant2

	} else if rc.variant3 != nil {
		variant = rc.variant3

	} else if rc.variant4 != nil {
		variant = rc.variant4

	} else if rc.variant5 != nil {
		variant = rc.variant5

	} else {
		variant = struct{}{}
	}
	return json.Marshal(struct {
		RuleName  string
		Expansion any
		Effect    float64
	}{
		RuleName:  "RgChar",
		Expansion: variant,
		Effect:    rc.effect,
	})
}

func (r *RgChar) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	if r.variant1 != nil {
		fn(r, r.variant1.Value)

	} else if r.variant2 != nil {
		r.variant2.RgOp.ForEachElement(r, fn)

	} else if r.variant3 != nil {
		r.variant3.RgRepeat.ForEachElement(r, fn)

	} else if r.variant4 != nil {
		r.variant4.RgAny.ForEachElement(r, fn)

	} else if r.variant5 != nil {
		r.variant5.RgQM.ForEachElement(r, fn)
	}
}

func (r *RgChar) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(r)
	if r.variant1 != nil {
		fn(r.variant1.Value, path)

	} else if r.variant2 != nil {
		r.variant2.RgOp.DFS(fn, path)
	} else if r.variant3 != nil {
		r.variant3.RgRepeat.DFS(fn, path)

	} else if r.variant4 != nil {
		r.variant4.RgAny.DFS(fn, path)

	} else if r.variant5 != nil {
		r.variant5.RgQM.DFS(fn, path)
	}
	fn(r, path)
	path.Pop()
}

// -----------------------------------------------------------

type RgRepeat struct {
	effect float64
	Value  ASTString
}

func (rr *RgRepeat) Text() string {
	return rr.Value.String()
}

func (rr *RgRepeat) Effect() float64 {
	if rr.effect == 0 {
		rr.effect = 100
	}
	return rr.effect
}

func (rr *RgRepeat) SetEffect(v float64) {
	rr.effect = v
}

func (rr *RgRepeat) IsLeaf() bool {
	return true
}

func (rr *RgRepeat) MarshalJSON() ([]byte, error) {
	return json.Marshal(
		struct {
			RuleName  string
			Expansion string
			Effect    float64
		}{
			RuleName:  "RgRepeat",
			Expansion: string(rr.Value),
			Effect:    rr.effect,
		},
	)
}

func (rr *RgRepeat) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, rr.Value)
}

func (rr *RgRepeat) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(rr)
	fn(rr.Value, path)
	path.Pop()
}

// -----------------------------------------------------------

type RgQM struct {
	effect float64
	Value  ASTString
}

func (rr *RgQM) Text() string {
	return rr.Value.String()
}

func (rr *RgQM) Effect() float64 {
	if rr.effect == 0 {
		rr.effect = 1000
	}
	return rr.effect
}

func (rr *RgQM) SetEffect(v float64) {
	rr.effect = v
}

func (rr *RgQM) IsLeaf() bool {
	return true
}

func (rr *RgQM) MarshalJSON() ([]byte, error) {
	return json.Marshal(
		struct {
			RuleName  string
			Expansion string
			Effect    float64
		}{
			RuleName:  "RgQM",
			Expansion: string(rr.Value),
			Effect:    rr.effect,
		},
	)
}

func (rr *RgQM) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, rr.Value)
}

func (rr *RgQM) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(rr)
	fn(rr.Value, path)
	path.Pop()
}

// -----------------------------------------------------------

type RgAny struct {
	effect float64
	Value  ASTString
}

func (rr *RgAny) Text() string {
	return rr.Value.String()
}

func (rr *RgAny) Effect() float64 {
	if rr.effect == 0 {
		rr.effect = 10000
	}
	return rr.effect
}

func (rr *RgAny) SetEffect(v float64) {
	rr.effect = v
}

func (rr *RgAny) IsLeaf() bool {
	return true
}

func (rr *RgAny) MarshalJSON() ([]byte, error) {
	return json.Marshal(
		struct {
			RuleName  string
			Expansion string
			Effect    float64
		}{
			RuleName:  "RgAny",
			Expansion: string(rr.Value),
			Effect:    rr.effect,
		},
	)
}

func (rr *RgAny) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, rr.Value)
}

func (rr *RgAny) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(rr)
	fn(rr.Value, path)
	path.Pop()
}

// -----------------------------------------------------------

type RgRange struct {
	effect      float64
	RgRangeSpec *RgRangeSpec
}

func (r *RgRange) Text() string {
	if r.RgRangeSpec != nil {
		return r.RgRangeSpec.Text()
	}
	return "RgRange{?, ?}"
}

func (r *RgRange) Effect() float64 {
	if r.effect == 0 {
		r.effect = 1
	}
	return r.effect
}

func (r *RgRange) SetEffect(v float64) {
	r.effect = v
}

func (r *RgRange) IsLeaf() bool {
	return false
}

func (r *RgRange) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion RgRange
		Effect    float64
	}{
		RuleName:  "RgRange",
		Expansion: *r,
		Effect:    r.effect,
	})
}

// NumericRepr returns a numeric representation
// of a repeat range operation ({a, b}). If something
// is undefined, -1 is used.
func (r *RgRange) NumericRepr() [2]int {
	if r.RgRangeSpec == nil {
		return [2]int{-1, -1}
	}
	v1, err := strconv.Atoi(r.RgRangeSpec.Number1.String())
	if err != nil {
		panic("non-integer 1st value in RgRange") // should not happen - guaranteed by the parser
	}
	v2 := -1
	if r.RgRangeSpec.Number2 != "" {
		v2, err = strconv.Atoi(r.RgRangeSpec.Number2.String())
		if err != nil {
			panic("non-integer 2nd value in RgRange") // should not happen - guaranteed by the parser
		}
	}
	return [2]int{v1, v2}
}

func (r *RgRange) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	r.RgRangeSpec.ForEachElement(r, fn)
}

func (r *RgRange) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(r)
	r.RgRangeSpec.DFS(fn, path)
	fn(r, path)
	path.Pop()
}

// -------------------------------------------------------------

type RgRangeSpec struct {
	effect    float64
	origValue string
	Number1   ASTString
	Number2   ASTString
}

func (r *RgRangeSpec) Text() string {
	return r.origValue
}

func (r *RgRangeSpec) Effect() float64 {
	if r.effect == 0 {
		r.effect = 1

	}
	return r.effect
}

func (r *RgRangeSpec) SetEffect(v float64) {
	r.effect = v
}

func (r *RgRangeSpec) IsLeaf() bool {
	return false
}

func (r *RgRangeSpec) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion RgRangeSpec
		Effect    float64
	}{
		RuleName:  "RgRangeSpec",
		Expansion: *r,
		Effect:    r.effect,
	})
}

func (r *RgRangeSpec) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	fn(parent, r.Number1)
	fn(parent, r.Number2)
}

func (r *RgRangeSpec) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(r)
	fn(r.Number1, path)
	fn(r.Number2, path)
	fn(r, path)
	path.Pop()
}

// -------------------------------------------------------------

type AnyLetter struct {
	effect float64
	Value  ASTString
}

func (a *AnyLetter) Text() string {
	return string(a.Value)
}

func (a *AnyLetter) Effect() float64 {
	if a.effect == 0 {
		a.effect = 1
	}
	return a.effect
}

func (a *AnyLetter) SetEffect(v float64) {
	a.effect = v
}

func (a *AnyLetter) IsLeaf() bool {
	return true
}

func (a *AnyLetter) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.Value)
}

func (a *AnyLetter) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, a.Value)
}

func (a *AnyLetter) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(a)
	fn(a.Value, path)
	path.Pop()
}

// -------------------------------------------------------------

type RgOp struct {
	effect float64
	Value  ASTString
}

func (r *RgOp) Text() string {
	return string(r.Value)
}

func (r *RgOp) Effect() float64 {
	if r.effect == 0 {
		r.effect = 100000
	}
	return r.effect
}

func (r *RgOp) SetEffect(v float64) {
	r.effect = v
}

func (r *RgOp) IsLeaf() bool {
	return true
}

func (r *RgOp) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion RgOp
		Effect    float64
	}{
		RuleName:  "RgOp",
		Expansion: *r,
		Effect:    r.effect,
	})
}

func (r *RgOp) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r.Value)
}

func (r *RgOp) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(r)
	fn(r.Value, path)
	path.Pop()
}

// ----------------------------------------------------------------

type rgAltValVariant1 struct {
	RgChar *RgChar
}

type rgAltValVariant2 struct {
	Value ASTString
}

type rgAltValVariant3 struct {
	From ASTString
	To   ASTString
}

type RgAltVal struct {
	effect   float64
	variant1 *rgAltValVariant1
	variant2 *rgAltValVariant2
	variant3 *rgAltValVariant3
}

func (rc *RgAltVal) Text() string {
	return "#RgAltVal"
}

func (rc *RgAltVal) Effect() float64 {
	if rc.effect == 0 {
		rc.effect = 1
	}
	return rc.effect
}

func (rc *RgAltVal) SetEffect(v float64) {
	rc.effect = v
}

func (rc *RgAltVal) IsLeaf() bool {
	return false
}

func (rc *RgAltVal) MarshalJSON() ([]byte, error) {
	var variant any
	if rc.variant1 != nil {
		variant = rc.variant1

	} else if rc.variant2 != nil {
		variant = rc.variant2

	} else if rc.variant3 != nil {
		variant = rc.variant3

	} else {
		variant = struct{}{}
	}

	return json.Marshal(struct {
		RuleName  string
		Expansion any
		Effect    float64
	}{
		RuleName:  "RgAltVal",
		Expansion: variant,
		Effect:    rc.effect,
	})
}

func (r *RgAltVal) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	if r.variant1 != nil {
		r.variant1.RgChar.ForEachElement(r, fn)

	} else if r.variant2 != nil {
		fn(r, r.variant2.Value)

	} else if r.variant3 != nil {
		fn(r, r.variant3.From)
		fn(r, r.variant3.To)
	}
}

func (r *RgAltVal) DFS(fn func(ASTNode, *Stack), path *Stack) {
	path.Push(r)
	if r.variant1 != nil {
		r.variant1.RgChar.DFS(fn, path)

	} else if r.variant2 != nil {
		fn(r.variant2.Value, path)

	} else if r.variant3 != nil {
		fn(r.variant3.From, path)
		fn(r.variant3.To, path)
	}
	fn(r, path)
	path.Pop()
}
