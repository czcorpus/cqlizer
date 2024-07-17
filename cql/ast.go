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
	origValue string
	Seq       []*Seq
}

func (q *Sequence) Text() string {
	return q.origValue
}

func (q *Sequence) Normalize() string {
	var ans strings.Builder
	for i, v := range q.Seq {
		if i > 0 {
			ans.WriteString("<OR>" + v.Normalize())

		} else {
			ans.WriteString(v.Normalize())
		}
	}
	return ans.String()
}

func (q *Sequence) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Expansion Sequence
		RuleName  string
	}{
		RuleName:  "Sequence",
		Expansion: *q,
	})
}

func (q *Sequence) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, q)
	for _, item := range q.Seq {
		item.ForEachElement(q, fn)
	}
}

func (q *Sequence) DFS(fn func(v ASTNode)) {
	for _, item := range q.Seq {
		item.DFS(fn)
	}
	fn(q)
}

// --------------------------------------------------------------------

// NOT? r1:Repetition r2:(_ Repetition)*
type Seq struct {
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

func (s *Seq) DFS(fn func(v ASTNode)) {
	fn(s.Not)
	for _, item := range s.Repetition {
		item.DFS(fn)
	}
	fn(s)
}

func (s *Seq) Text() string {
	return string(s.origValue)
}

func (q *Seq) Normalize() string {
	var ans strings.Builder
	for _, v := range q.Repetition {
		ans.WriteString(v.Normalize())
	}
	return ans.String()
}

// -----------------------------------------------------

// GlobPart
// gc:GlobCond gc2:(_ BINAND _ GlobCond)*
type GlobPart struct {
	GlobCond []*GlobCond
}

func (g *GlobPart) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Expansion GlobPart
		RuleName  string
	}{
		RuleName:  "GlobPart",
		Expansion: *g,
	})
}

func (q *GlobPart) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, q)
	for _, item := range q.GlobCond {
		item.ForEachElement(q, fn)
	}
}

func (q *GlobPart) DFS(fn func(v ASTNode)) {
	for _, item := range q.GlobCond {
		item.DFS(fn)
	}
	fn(q)
}

func (q *GlobPart) Text() string {
	return "#GlobPart" // TODO
}

func (q *GlobPart) Normalize() string {
	var ans strings.Builder
	for _, v := range q.GlobCond {
		ans.WriteString(v.Normalize())
	}
	return ans.String()
}

// ---------------------------------------

// WithinOrContaining
//
//	NOT? (KW_WITHIN / KW_CONTAINING) _ WithinContainingPart {
type WithinOrContaining struct {
	not                   bool
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
	}{
		RuleName:  "WithinOrContaining",
		Expansion: *w,
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

func (w *WithinOrContaining) DFS(fn func(v ASTNode)) {
	fn(w.KwWithin)
	fn(w.KwContaining)
	if w.WithinContainingPart != nil {
		w.WithinContainingPart.DFS(fn)
	}
	fn(w)
}

func (w *WithinOrContaining) Text() string {
	return "#WithinOrContaining"
}

func (w *WithinOrContaining) Normalize() string {
	//	NOT? (KW_WITHIN / KW_CONTAINING) _ WithinContainingPart {
	var ans strings.Builder
	if w.not {
		ans.WriteString("<NEGATION>")
	}
	if w.numNegWithinParts > 0 {
		ans.WriteString("within")
	}
	if w.numContainingParts > 0 {
		ans.WriteString("containing")
	}
	return ans.String()
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
	variant1 *withinContainingPartVariant1

	variant2 *withinContainingPartVariant2

	variant3 *withinContainingPartVariant3
}

func (wcp *WithinContainingPart) Text() string {
	return "#WithinContainingPart"
}

func (wcp *WithinContainingPart) Normalize() string {
	if wcp.variant1 != nil {
		return wcp.variant1.Sequence.Normalize()
	}
	if wcp.variant2 != nil {
		return wcp.variant2.WithinNumber.Normalize()
	}
	if wcp.variant3 != nil {
		wcp.variant3.AlignedPart.Normalize() // TODO missing NOT
	}
	return "#WithinContainingPart"
}

func (wcp *WithinContainingPart) MarshalJSON() ([]byte, error) {
	if wcp.variant1 != nil {
		return json.Marshal(struct {
			Expansion withinContainingPartVariant1
			RuleName  string
		}{
			Expansion: *wcp.variant1,
			RuleName:  "WithinContainingPart",
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

func (wcp *WithinContainingPart) DFS(fn func(v ASTNode)) {
	if wcp.variant1 != nil {
		wcp.variant1.Sequence.DFS(fn)

	} else if wcp.variant2 != nil {
		fn(wcp.variant2.WithinNumber.Value)

	} else if wcp.variant3 != nil {
		wcp.variant3.AlignedPart.DFS(fn)
	}
	fn(wcp)
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
	variant1 *globCondVariant1

	variant2 *globCondVariant2
}

func (gc *GlobCond) Text() string {
	return "#GlobCond"
}

func (gc *GlobCond) Normalize() string {
	// NUMBER DOT AttName _ NOT? EQ _ NUMBER DOT AttName
	// KW_FREQ LPAREN _ NUMBER DOT AttName _ RPAREN NOT? _ ( EQ / LEQ / GEQ / LSTRUCT / RSTRUCT ) _ NUMBER {
	if gc.variant1 != nil {
		var ans strings.Builder
		ans.WriteString("att ")
		ans.WriteString(gc.variant1.Not4.Normalize())
		ans.WriteString("att ")
		return ans.String()
	}
	if gc.variant2 != nil {
		var ans strings.Builder
		ans.WriteString(gc.variant2.Number2.Normalize())
		ans.WriteString("att ")
		ans.WriteString(gc.variant2.Number6.Normalize())
		return ans.String()
	}
	return "#GlobCond"
}

func (gc *GlobCond) MarshalJSON() ([]byte, error) {
	if gc.variant1 != nil {
		return json.Marshal(struct {
			Expansion globCondVariant1
			RuleName  string
		}{
			Expansion: *gc.variant1,
			RuleName:  "GlobCond",
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

func (gc *GlobCond) DFS(fn func(v ASTNode)) {
	if gc.variant1 != nil {
		fn(gc.variant1.Number1)
		fn(gc.variant1.AttName3)
		fn(gc.variant1.Not4)
		fn(gc.variant1.Eq5)
		fn(gc.variant1.Number6)
		fn(gc.variant1.AttName8)

	} else if gc.variant2 != nil {
		fn(gc.variant2.KwFreq1)
		fn(gc.variant2.Number2)
		fn(gc.variant2.AttName3)
		fn(gc.variant2.Not4)
		fn(gc.variant2.Operator5)
		fn(gc.variant2.Number6)
	}
	fn(gc)
}

// ----------------------------------------------------

// Structure
//
// AttName _ AttValList?
type Structure struct {
	AttName    ASTString
	AttValList *AttValList
}

func (s *Structure) Text() string {
	return s.AttName.Text()
}

func (s *Structure) Normalize() string {
	// // AttName _ AttValList?
	if s.AttValList != nil {
		return fmt.Sprintf("(struct %s)", s.AttValList.Normalize())
	}
	return "(struct)"
}

func (s *Structure) IsBigStructure() bool {
	v := s.AttName.Text()
	return v == "s" || v == "g" || v == "p"
}

func (s *Structure) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion Structure
	}{
		RuleName:  "Structure",
		Expansion: *s,
	})
}

func (s *Structure) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, s)
	fn(s, s.AttName)
	if s.AttValList != nil {
		s.AttValList.ForEachElement(s, fn)
	}
}

func (s *Structure) DFS(fn func(v ASTNode)) {
	fn(s.AttName)
	if s.AttValList != nil {
		s.AttValList.DFS(fn)
	}
	fn(s)
}

// ---------------------------------------------------------

// AttValList
//
//	av1:AttValAnd av2:(_ BINOR _ AttValAnd)*
type AttValList struct {
	origValue string
	AttValAnd []*AttValAnd
}

func (a *AttValList) Text() string {
	return a.origValue
}

func (a *AttValList) Normalize() string {
	// //	av1:AttValAnd av2:(_ BINOR _ AttValAnd)*
	var ans strings.Builder
	for i, v := range a.AttValAnd {
		if i > 0 {
			ans.WriteString("<OR>" + v.Normalize())

		} else {
			ans.WriteString(v.Normalize())
		}
	}
	return ans.String()
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

func (a *AttValList) DFS(fn func(v ASTNode)) {
	for _, v := range a.AttValAnd {
		v.DFS(fn)
	}
	fn(a)
}

func (a *AttValList) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion AttValList
	}{
		RuleName:  "AttValList",
		Expansion: *a,
	})
}

// -----------------------------------------------------------

// NumberedPosition
//
// NUMBER COLON OnePosition
type NumberedPosition struct {
	Number      ASTString
	Colon       ASTString
	OnePosition *OnePosition
}

func (n *NumberedPosition) Text() string {
	return "#NumberedPosition"
}

func (n *NumberedPosition) Normalize() string {
	return fmt.Sprintf("(%s %s %s)", n.Number.Normalize(), n.Colon.Normalize(), n.OnePosition.Normalize())
}

func (n *NumberedPosition) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, n)
	fn(n, n.Number)
	fn(n, n.Colon)
	if n.OnePosition != nil {
		n.OnePosition.ForEachElement(n, fn)
	}
}

func (n *NumberedPosition) DFS(fn func(v ASTNode)) {
	fn(n.Number)
	fn(n.Colon)
	if n.OnePosition != nil {
		n.OnePosition.DFS(fn)
	}
	fn(n)
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

func (op *OnePosition) Normalize() string {
	// var1: LBRACKET _ AttValList? _ RBRACKET
	// var2: RegExp
	// var3: TEQ NUMBER? RegExp
	// var4: KW_MU
	// var5: MuPart
	var ans strings.Builder
	if op.Variant1 != nil {
		if op.Variant1.AttValList != nil {
			ans.WriteString("(pos " + op.Variant1.AttValList.Normalize() + ")")

		} else {
			ans.WriteString("(pos ())")
		}

	} else if op.Variant2 != nil {
		ans.WriteString("(pos " + op.Variant2.RegExp.Normalize() + ")")

	} else if op.Variant3 != nil {
		ans.WriteString("(pos " + op.Variant3.RegExp.origValue + ")")

	} else if op.Variant4 != nil {
		ans.WriteString("(pos " + op.Variant4.Value.Normalize() + ")")

	} else if op.Variant5 != nil {
		ans.WriteString("(pos " + op.Variant5.MuPart.Normalize() + ")")
	}
	return ans.String()
}

func (op *OnePosition) MarshalJSON() ([]byte, error) {
	if op.Variant1 != nil {
		return json.Marshal(struct {
			RuleName  string
			RawValue  string
			Expansion *onePositionVariant1
		}{
			RuleName:  "OnePosition",
			RawValue:  op.Text(),
			Expansion: op.Variant1,
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

func (op *OnePosition) DFS(fn func(v ASTNode)) {
	if op.Variant1 != nil && op.Variant1.AttValList != nil {
		op.Variant1.AttValList.DFS(fn)

	} else if op.Variant2 != nil {
		op.Variant2.RegExp.DFS(fn)

	} else if op.Variant3 != nil {
		fn(op.Variant3.Number)
		op.Variant3.RegExp.DFS(fn)

	} else if op.Variant4 != nil {
		fn(op.Variant4.Value)

	} else if op.Variant5 != nil {
		op.Variant5.MuPart.DFS(fn)
	}
	fn(op)
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
	origValue string
	variant1  *positionVariant1
	variant2  *positionVariant2
}

func (p *Position) Text() string {
	return p.origValue
}

func (p *Position) Normalize() string {
	var ans strings.Builder
	if p.variant1 != nil {
		ans.WriteString(p.variant1.OnePosition.Normalize())

	} else if p.variant2 != nil {
		ans.WriteString(p.variant2.NumberedPosition.Normalize())
	}
	return ans.String()
}

func (p *Position) MarshalJSON() ([]byte, error) {
	if p.variant1 != nil {
		return json.Marshal(struct {
			RuleName  string
			Expansion *positionVariant1
		}{
			RuleName:  "Position",
			Expansion: p.variant1,
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

func (p *Position) DFS(fn func(v ASTNode)) {
	if p.variant1 != nil {
		p.variant1.OnePosition.DFS(fn)

	} else if p.variant2 != nil {
		p.variant2.NumberedPosition.DFS(fn)
	}
	fn(p)
}

// -------------------------------------------------------

type RegExp struct {
	origValue string
	RegExpRaw []*RegExpRaw // these are A|B|C
}

func (r *RegExp) Text() string {
	return r.origValue
}

func (r *RegExp) Normalize() string {
	var ans strings.Builder
	for i, v := range r.RegExpRaw {
		if i > 0 {
			ans.WriteString("<OR>" + v.Normalize())

		} else {
			ans.WriteString(v.Normalize())
		}
	}
	return ans.String()
}

func (r *RegExp) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	for _, v := range r.RegExpRaw {
		v.ForEachElement(r, fn)
	}
}

func (r *RegExp) DFS(fn func(v ASTNode)) {
	for _, v := range r.RegExpRaw {
		v.DFS(fn)
	}
	fn(r)
}

// --------------------------------------------------------

type muPartVariant1 struct {
	UnionOp *UnionOp
}

type muPartVariant2 struct {
	MeetOp *MeetOp
}

type MuPart struct {
	origValue string
	Variant1  *muPartVariant1
	Variant2  *muPartVariant2
}

func (m *MuPart) Text() string {
	return m.origValue
}

func (m *MuPart) Normalize() string {
	var ans strings.Builder
	if m.Variant1 != nil {
		ans.WriteString(m.Variant1.UnionOp.Normalize())

	} else if m.Variant2 != nil {
		ans.WriteString(m.Variant2.MeetOp.Normalize())
	}
	return ans.String()
}

func (m *MuPart) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion MuPart
	}{
		RuleName:  "MuPart",
		Expansion: *m,
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

func (m *MuPart) DFS(fn func(v ASTNode)) {
	if m.Variant1 != nil {
		m.Variant1.UnionOp.DFS(fn)

	} else if m.Variant2 != nil {
		m.Variant2.MeetOp.DFS(fn)
	}
	fn(m)
}

// --------------------------------------------------------------

type UnionOp struct {
	origValue string
	Position1 *Position
	Position2 *Position
}

func (m *UnionOp) Text() string {
	return m.origValue
}

func (m *UnionOp) Normalize() string {
	return fmt.Sprintf("%s %s", m.Position1.Normalize(), m.Position2.Normalize())
}

func (m *UnionOp) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, m)
	m.Position1.ForEachElement(m, fn)
	m.Position2.ForEachElement(m, fn)
}

func (m *UnionOp) DFS(fn func(v ASTNode)) {
	m.Position1.DFS(fn)
	m.Position2.DFS(fn)
	fn(m)
}

// ---------------------------------------------------------------

type MeetOp struct {
	origValue string
	Position1 *Position
	Position2 *Position
}

func (m *MeetOp) Text() string {
	return m.origValue
}

func (m *MeetOp) Normalize() string {
	return fmt.Sprintf("%s %s", m.Position1.Normalize(), m.Position2.Normalize())
}

func (m *MeetOp) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, m)
	m.Position1.ForEachElement(m, fn)
	m.Position2.ForEachElement(m, fn)
}

func (m *MeetOp) DFS(fn func(v ASTNode)) {
	m.Position1.DFS(fn)
	m.Position2.DFS(fn)
	fn(m)
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
	origValue      string
	isTailPosition bool
	Variant1       *repetitionVariant1
	Variant2       *repetitionVariant2
	Variant3       *repetitionVariant3
}

func (r *Repetition) IsAnyPosition() bool {
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

func (r *Repetition) Normalize() string {
	var ans strings.Builder
	if r.Variant1 != nil {
		ans.WriteString(r.Variant1.AtomQuery.Normalize() + " " + r.Variant1.RepOpt.Normalize())

	} else if r.Variant2 != nil {
		ans.WriteString(r.Variant2.OpenStructTag.Normalize())

	} else if r.Variant3 != nil {
		ans.WriteString(r.Variant3.CloseStructTag.Normalize())
	}
	return ans.String()
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
		}{
			RuleName:      "Repetition",
			Expansion:     variant,
			IsAnyPosition: r.IsAnyPosition(),
		})
}

func (r *Repetition) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	if r.Variant1 != nil {
		r.Variant1.AtomQuery.ForEachElement(r, fn)
		fn(r, r.Variant1.RepOpt)

	} else if r.Variant2 != nil {
		r.Variant2.OpenStructTag.ForEachElement(r, fn)

	} else if r.Variant3 != nil {
		r.Variant3.CloseStructTag.ForEachElement(r, fn)
	}
}

func (r *Repetition) DFS(fn func(v ASTNode)) {
	if r.Variant1 != nil {
		r.Variant1.AtomQuery.DFS(fn)
		fn(r.Variant1.RepOpt)

	} else if r.Variant2 != nil {
		r.Variant2.OpenStructTag.DFS(fn)

	} else if r.Variant3 != nil {
		r.Variant3.CloseStructTag.DFS(fn)
	}
	fn(r)
}

// ----------------------------------------------------------------

type atomQueryVariant1 struct {
	Position *Position
}

type withinContainingBlock struct {
	Not                  bool
	Keyword              ASTString // either `within` or `containing`
	WithinContainingPart *WithinContainingPart
}

type atomQueryVariant2 struct {
	Sequence *Sequence
	WCBlock  []*withinContainingBlock
}

// AtomQuery
// var1: Position
// var2: LPAREN _ Sequence (_ NOT? (KW_WITHIN / KW_CONTAINING) _ WithinContainingPart)* _ RPAREN {
type AtomQuery struct {
	origValue string
	variant1  *atomQueryVariant1
	variant2  *atomQueryVariant2
}

func (aq *AtomQuery) Text() string {
	return aq.origValue
}

func (aq *AtomQuery) Normalize() string {
	var ans strings.Builder
	if aq.variant1 != nil {
		ans.WriteString(aq.variant1.Position.Normalize())

	} else if aq.variant2 != nil {
		ans.WriteString(aq.variant2.Sequence.Normalize())
	}
	return ans.String()
}

func (aq *AtomQuery) MarshalJSON() ([]byte, error) {
	if aq.variant1 != nil {
		return json.Marshal(struct {
			RuleName  string
			Expansion *atomQueryVariant1
		}{
			RuleName:  "AtomQuery",
			Expansion: aq.variant1,
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

func (aq *AtomQuery) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, aq)
	if aq.variant1 != nil {
		aq.variant1.Position.ForEachElement(aq, fn)

	} else if aq.variant2 != nil {
		aq.variant2.Sequence.ForEachElement(aq, fn)
		if aq.variant2.WCBlock != nil {
			for _, v := range aq.variant2.WCBlock {
				if v.Not {
					fn(aq, ASTString("!"))
				}
				fn(aq, v.Keyword)
				v.WithinContainingPart.ForEachElement(aq, fn)
			}
		}
	}
}

func (aq *AtomQuery) DFS(fn func(v ASTNode)) {
	if aq.variant1 != nil {
		aq.variant1.Position.DFS(fn)

	} else if aq.variant2 != nil {
		aq.variant2.Sequence.DFS(fn)
		if aq.variant2.WCBlock != nil {
			for _, v := range aq.variant2.WCBlock {
				if v.Not {
					fn(ASTString("!"))
				}
				fn(v.Keyword)
				v.WithinContainingPart.DFS(fn)
			}
		}
	}
	fn(aq)
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

func (r *RepOpt) Normalize() string {
	if r == nil {
		return ""
	}
	var ans strings.Builder
	if r.Variant1 != nil {
		ans.WriteString(r.Variant1.Value.Normalize())

	} else if r.Variant2 != nil {
		ans.WriteString(r.Variant2.From.Normalize())
		ans.WriteString(r.Variant2.To.Normalize())
	}
	return ans.String()
}

func (r *RepOpt) MarshalJSON() ([]byte, error) {
	if r.Variant1 != nil {
		return json.Marshal(struct {
			RuleName  string
			Expansion repOptVariant1
		}{
			RuleName:  "RepOpt",
			Expansion: *r.Variant1,
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

func (r *RepOpt) DFS(fn func(v ASTNode)) {
	if r.Variant1 != nil {
		fn(r.Variant1.Value)

	} else if r.Variant2 != nil {
		fn(r.Variant2.From)
		fn(r.Variant2.To)
	}
	fn(r)
}

// ----------------------------------------------------------------

type OpenStructTag struct {
	origValue string
	Structure *Structure
}

func (ost *OpenStructTag) Text() string {
	return ost.origValue
}

func (ost *OpenStructTag) Normalize() string {
	return ost.Structure.Normalize()
}

func (ost *OpenStructTag) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion OpenStructTag
	}{
		RuleName:  "OpenStructTag",
		Expansion: *ost,
	})
}

func (ost *OpenStructTag) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, ost)
	ost.Structure.ForEachElement(ost, fn)
}

func (ost *OpenStructTag) DFS(fn func(v ASTNode)) {
	ost.Structure.DFS(fn)
	fn(ost)
}

// --------------------------------------------------------------

type CloseStructTag struct {
	Structure *Structure
}

func (ost *CloseStructTag) Text() string {
	return "#CloseStructTag"
}

func (ost *CloseStructTag) Normalize() string {
	return ost.Structure.Normalize()
}

func (ost *CloseStructTag) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion CloseStructTag
	}{
		RuleName:  "CloseStructTag",
		Expansion: *ost,
	})
}

func (ost *CloseStructTag) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, ost)
	ost.Structure.ForEachElement(ost, fn)
}

func (ost *CloseStructTag) DFS(fn func(v ASTNode)) {
	ost.Structure.DFS(fn)
	fn(ost)
}

// ---------------------------------------------------------

type AlignedPart struct {
}

func (a *AlignedPart) Text() string {
	return "#AlignedPart"
}

func (a *AlignedPart) Normalize() string {
	return "#AlignedPart"
}

func (a *AlignedPart) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, a)
	// TODO
}

func (a *AlignedPart) DFS(fn func(v ASTNode)) {
	fn(a)
}

// -----------------------------------------------------------

// AttValAnd
//
//	av1:AttVal av2:(_ BINAND _ AttVal)*
type AttValAnd struct {
	AttVal []*AttVal
}

func (a *AttValAnd) Text() string {
	return "#AttValAnd"
}

func (a *AttValAnd) Normalize() string {
	var ans strings.Builder
	for i, v := range a.AttVal {
		if i > 0 {
			ans.WriteString("<AND>" + v.Normalize())

		} else {
			ans.WriteString(v.Normalize())
		}
	}
	return ans.String()
}

func (a *AttValAnd) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, a)
	for _, item := range a.AttVal {
		item.ForEachElement(a, fn)
	}
}

func (a *AttValAnd) DFS(fn func(v ASTNode)) {
	for _, item := range a.AttVal {
		item.DFS(fn)
	}
	fn(a)
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

func (a *AttVal) IsNegation() bool {
	return a.Variant1 != nil && a.Variant1.Not ||
		a.Variant2 != nil && a.Variant2.Not
}

func (a *AttVal) IsProblematicAttrSearch() bool {
	if a.Variant1 != nil {
		return (a.Variant1.AttName == "tag" || a.Variant1.AttName == "pos" || a.Variant1.AttName == "verbtag" ||
			a.Variant1.AttName == "upos" || a.Variant1.AttName == "afun" || a.Variant1.AttName == "case")

	} else if a.Variant2 != nil {
		return (a.Variant2.AttName == "tag" || a.Variant2.AttName == "pos" || a.Variant2.AttName == "verbtag" ||
			a.Variant2.AttName == "upos" || a.Variant2.AttName == "afun" || a.Variant2.AttName == "case")
	}
	return false
}

func (a *AttVal) Text() string {
	return a.origValue
}

func (a *AttVal) Normalize() string {
	var ans strings.Builder
	if a.Variant1 != nil {
		// AttName _ (NOT)? EEQ _ RawString
		var ns string
		if a.Variant1.Not {
			ns = "<NEGATION>"
		}
		attName := "(att "
		if a.IsProblematicAttrSearch() {
			attName = "(ATT "
		}
		ans.WriteString(attName + ns + " " + a.Variant1.RawString.Normalize() + ")")

	} else if a.Variant2 != nil {
		// AttName (_ NOT)? _ (EQ / LEQ / GEQ / TEQ NUMBER?) _ RegExp
		var ns string
		if a.Variant2.Not {
			ns = "<NEGATION>"
		}
		attName := "(att "
		if a.IsProblematicAttrSearch() {
			attName = "(ATT "
		}
		ans.WriteString(attName + ns + " " + a.Variant2.RegExp.Normalize() + ")")

	} else if a.Variant5 != nil {
		ans.WriteString("<NEGATION>" + a.Variant5.AttVal.Normalize())

	} else if a.Variant6 != nil {
		ans.WriteString("(" + a.Variant6.AttValList.Normalize() + ") ")
	}

	return ans.String()
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
	}{
		RuleName:                "AttVal",
		Expansion:               variant,
		IsProblematicAttrSearch: a.IsProblematicAttrSearch(),
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

func (a *AttVal) DFS(fn func(v ASTNode)) {
	if a.Variant1 != nil {
		fn(a.Variant1.AttName)
		fn(a.Variant1.Eeq)
		a.Variant1.RawString.DFS(fn)

	} else if a.Variant2 != nil {
		fn(a.Variant2.AttName)
		fn(a.Variant2.Op)
		a.Variant2.RegExp.DFS(fn)

	} else if a.Variant3 != nil {
		// TODO a.variant3

	} else if a.Variant4 != nil {
		// TODO a.variant4

	} else if a.Variant5 != nil {
		a.Variant5.AttVal.DFS(fn)

	} else if a.Variant6 != nil {
		a.Variant6.AttValList.DFS(fn)

	} else if a.Variant7 != nil {
		// TODO a.variant7

	} else if a.Variant8 != nil {
		// TODO a.variant8

	} else if a.Variant9 != nil {
		// TODO a.variant9
	}
	fn(a)
}

// ---------------------------------------------------

type WithinNumber struct {
	Value ASTString
}

func (w *WithinNumber) Text() string {
	return "#WithinNumber"
}

func (w *WithinNumber) Normalize() string {
	return w.Value.Normalize()
}

func (w *WithinNumber) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion WithinNumber
	}{
		RuleName:  "WithinNumber",
		Expansion: *w,
	})
}

func (w *WithinNumber) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, w.Value)
}

func (w *WithinNumber) DFS(fn func(v ASTNode)) {
	fn(w.Value)
}

// ----------------------------------------------------------

type RegExpRaw struct {
	origValue string
	// RgLook / RgGrouped / RgSimple
	Values []ASTNode
}

func (r *RegExpRaw) Text() string {
	return r.origValue
}

func (r *RegExpRaw) Normalize() string {
	var ans strings.Builder
	for _, v := range r.Values {
		ans.WriteString(v.Normalize())
	}
	return ans.String()
}

func (r *RegExpRaw) ExhaustionScore() float64 {
	var ans float64
	for _, v := range r.Values {
		switch tValue := v.(type) {
		case *RgGrouped:
			ans += tValue.ExhaustionScore()
		case *RgSimple:
			ans += tValue.ExhaustionScore()
		}
	}
	return ans
}

func (r *RegExpRaw) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName        string
		Expansion       RegExpRaw
		ExhaustionScore float64
	}{
		RuleName:        "RegExpRaw",
		Expansion:       *r,
		ExhaustionScore: r.ExhaustionScore(),
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

func (r *RegExpRaw) DFS(fn func(v ASTNode)) {
	for _, item := range r.Values {
		switch tItem := item.(type) {
		case *RgLook:
			tItem.DFS(fn)
		case *RgGrouped:
			tItem.DFS(fn)
		case *RgSimple:
			tItem.DFS(fn)
		}
	}
	fn(r)
}

// ------------------------------------------------------------------

type RawString struct {
	SimpleString *SimpleString
}

func (r *RawString) Text() string {
	return r.SimpleString.Normalize()
}

func (r *RawString) Normalize() string {
	return "#RawString"
}

func (r *RawString) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion RawString
	}{
		RuleName:  "RawString",
		Expansion: *r,
	})
}

func (r *RawString) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	r.SimpleString.ForEachElement(r, fn)
}

func (r *RawString) DFS(fn func(v ASTNode)) {
	r.SimpleString.DFS(fn)
	fn(r)
}

// ------------------------------------------------------------------------

type SimpleString struct {
	origValue string
	Values    []ASTString
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

func (r *SimpleString) Text() string {
	var ans strings.Builder
	for _, v := range r.Values {
		ans.WriteString(string(v))
	}
	return ans.String()
}

func (r *SimpleString) Normalize() string {
	var ans strings.Builder
	for _, v := range r.Values {
		ans.WriteString(v.Normalize())
	}
	return ans.String()
}

func (r *SimpleString) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion SimpleString
	}{
		RuleName:  "SimpleString",
		Expansion: *r,
	})
}

func (r *SimpleString) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, ASTString(r.Text()))
}

func (r *SimpleString) DFS(fn func(v ASTNode)) {
	fn(ASTString(r.Text()))
}
