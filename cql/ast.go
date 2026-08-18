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
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/rs/zerolog/log"
)

var (
	problematicAttributes = []string{
		"tag", "pos", "verbtag", "upos", "afun", "case",
	}
)

// Seq (_ BINOR _ Seq)* / Seq
type Sequence struct {
	origValue string
	Seq       []*Seq
}

func (q *Sequence) String() string {
	var ans strings.Builder
	ans.WriteString("Sequence( ")
	for i, s := range q.Seq {
		if i > 0 {
			ans.WriteString(", ")
		}
		ans.WriteString(s.String())
	}
	ans.WriteString(" )")
	return ans.String()
}

// CQL renders "seq1 | seq2 | ..." - BINOR-joined alternative n-gram
// sequences.
func (q *Sequence) CQL() string {
	var ans strings.Builder
	for i, s := range q.Seq {
		if i > 0 {
			ans.WriteString(" | ")
		}
		ans.WriteString(s.CQL())
	}
	return ans.String()
}

func (q *Sequence) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, q)
	for _, item := range q.Seq {
		item.ForEachElement(q, fn)
	}
}

func (q *Sequence) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(q) {
		leave(q)
		return
	}
	for _, item := range q.Seq {
		item.DFS(enter, leave)
	}
	leave(q)
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

func (s *Seq) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(s) {
		leave(s)
		return
	}
	dfsLeaf(enter, leave, s.Not)
	for _, item := range s.Repetition {
		item.DFS(enter, leave)
	}
	leave(s)
}

func (s *Seq) String() string {
	var ans strings.Builder
	ans.WriteString("Seq( ")
	if s.Not != "" {
		ans.WriteString(s.Not.String())
	}
	for i, r := range s.Repetition {
		if i > 0 {
			ans.WriteString(", ")
		}
		ans.WriteString(r.String())
	}
	ans.WriteString(" )")
	return ans.String()
}

// CQL renders "!pos1 pos2 ..." - an optional leading negation followed
// by the space-separated positions of an n-gram.
func (s *Seq) CQL() string {
	var ans strings.Builder
	if s.Not != "" {
		ans.WriteString(s.Not.CQL())
	}
	for i, r := range s.Repetition {
		if i > 0 {
			ans.WriteString(" ")
		}
		ans.WriteString(r.CQL())
	}
	return ans.String()
}

// -----------------------------------------------------

// GlobPart
// gc:GlobCond gc2:(_ BINAND _ GlobCond)*
type GlobPart struct {
	GlobCond []*GlobCond
}

func (q *GlobPart) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, q)
	for _, item := range q.GlobCond {
		item.ForEachElement(q, fn)
	}
}

func (q *GlobPart) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(q) {
		leave(q)
		return
	}
	for _, item := range q.GlobCond {
		item.DFS(enter, leave)
	}
	leave(q)
}

func (q *GlobPart) String() string {
	var ans strings.Builder
	ans.WriteString("GlobPart( ")
	for i, gc := range q.GlobCond {
		if i > 0 {
			ans.WriteString(", ")
		}
		ans.WriteString(gc.String())
	}
	ans.WriteString(" )")
	return ans.String()
}

// CQL renders "cond1 & cond2 & ..." - BINAND-joined global conditions.
func (q *GlobPart) CQL() string {
	var ans strings.Builder
	for i, gc := range q.GlobCond {
		if i > 0 {
			ans.WriteString(" & ")
		}
		ans.WriteString(gc.CQL())
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
	// Keyword holds whichever of "within" / "containing" actually
	// matched - the grammar doesn't track which one separately.
	Keyword              ASTString
	WithinContainingPart *WithinContainingPart
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

func (w *WithinOrContaining) IsNegated() bool {
	return w.not
}

func (w *WithinOrContaining) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, w)
	fn(w, w.Keyword)
	if w.WithinContainingPart != nil {
		w.WithinContainingPart.ForEachElement(w, fn)
	}
}

func (w *WithinOrContaining) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(w) {
		leave(w)
		return
	}
	dfsLeaf(enter, leave, w.Keyword)
	if w.WithinContainingPart != nil {
		w.WithinContainingPart.DFS(enter, leave)
	}
	leave(w)
}

func (w *WithinOrContaining) String() string {
	var ans strings.Builder
	ans.WriteString("WithinOrContaining")
	if w.not {
		ans.WriteString("[!]")
	}
	ans.WriteString("( ")
	ans.WriteString(w.Keyword.String())
	ans.WriteString(" ")
	if w.WithinContainingPart != nil {
		ans.WriteString(w.WithinContainingPart.String())
	}
	ans.WriteString(" )")
	return ans.String()
}

// CQL renders "!within <part>" / "containing <part>" etc.
func (w *WithinOrContaining) CQL() string {
	var ans strings.Builder
	if w.not {
		ans.WriteString("!")
	}
	ans.WriteString(w.Keyword.CQL())
	ans.WriteString(" ")
	if w.WithinContainingPart != nil {
		ans.WriteString(w.WithinContainingPart.CQL())
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

func (wcp *WithinContainingPart) String() string {
	var ans strings.Builder
	ans.WriteString("WithinContainingPart( ")
	if wcp.variant1 != nil {
		ans.WriteString(wcp.variant1.Sequence.String())
	}
	if wcp.variant2 != nil {
		ans.WriteString(wcp.variant2.WithinNumber.String())
	}
	if wcp.variant3 != nil {
		ans.WriteString(wcp.variant3.AlignedPart.String())
	}
	ans.WriteString(" )")
	return ans.String()
}

// CQL renders the sequence/number/aligned-part payload of a within or
// containing clause. Note: as with String(), a NOT before the
// aligned-part alternative is matched by the grammar but not retained
// on withinContainingPartVariant3, so that negation is not reproduced
// here.
func (wcp *WithinContainingPart) CQL() string {
	if wcp.variant1 != nil {
		return wcp.variant1.Sequence.CQL()
	}
	if wcp.variant2 != nil {
		return wcp.variant2.WithinNumber.CQL()
	}
	if wcp.variant3 != nil {
		return wcp.variant3.AlignedPart.CQL()
	}
	return ""
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

func (wcp *WithinContainingPart) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(wcp) {
		leave(wcp)
		return
	}
	if wcp.variant1 != nil {
		wcp.variant1.Sequence.DFS(enter, leave)

	} else if wcp.variant2 != nil {
		dfsLeaf(enter, leave, wcp.variant2.WithinNumber.Value)

	} else if wcp.variant3 != nil {
		wcp.variant3.AlignedPart.DFS(enter, leave)
	}
	leave(wcp)
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

func (gc *GlobCond) String() string {
	var ans strings.Builder
	ans.WriteString("GlobCond( ")
	if gc.variant1 != nil {
		var ans strings.Builder
		ans.WriteString(gc.variant1.Number1.String())
		ans.WriteString(".")
		ans.WriteString(gc.variant1.AttName3.String())
		if gc.variant1.Not4 != "" {
			ans.WriteString("!")
		}
		ans.WriteString(gc.variant1.Eq5.String())
		ans.WriteString(gc.variant1.Number6.String())
		ans.WriteString(".")
		ans.WriteString(gc.variant1.AttName8.String())
	}
	if gc.variant2 != nil {
		var ans strings.Builder
		ans.WriteString(gc.variant2.KwFreq1.String())
		ans.WriteString("(")
		ans.WriteString(gc.variant2.Number2.String())
		ans.WriteString(".")
		ans.WriteString(gc.variant2.AttName3.String())
		ans.WriteString(")")
		if gc.variant2.Not4 != "" {
			ans.WriteString("!")
		}
		ans.WriteString(gc.variant2.Operator5.String())
		ans.WriteString(gc.variant2.Number6.String())
	}
	ans.WriteString(" )")
	return ans.String()
}

// CQL renders "N.attr=M.attr" or "f(N.attr)!=M" global conditions.
func (gc *GlobCond) CQL() string {
	if gc.variant1 != nil {
		var ans strings.Builder
		ans.WriteString(gc.variant1.Number1.CQL())
		ans.WriteString(".")
		ans.WriteString(gc.variant1.AttName3.CQL())
		if gc.variant1.Not4 != "" {
			ans.WriteString("!")
		}
		ans.WriteString(gc.variant1.Eq5.CQL())
		ans.WriteString(gc.variant1.Number6.CQL())
		ans.WriteString(".")
		ans.WriteString(gc.variant1.AttName8.CQL())
		return ans.String()
	}
	if gc.variant2 != nil {
		var ans strings.Builder
		ans.WriteString(gc.variant2.KwFreq1.CQL())
		ans.WriteString("(")
		ans.WriteString(gc.variant2.Number2.CQL())
		ans.WriteString(".")
		ans.WriteString(gc.variant2.AttName3.CQL())
		ans.WriteString(")")
		if gc.variant2.Not4 != "" {
			ans.WriteString("!")
		}
		ans.WriteString(gc.variant2.Operator5.CQL())
		ans.WriteString(gc.variant2.Number6.CQL())
		return ans.String()
	}
	return ""
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

func (gc *GlobCond) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(gc) {
		leave(gc)
		return
	}
	if gc.variant1 != nil {
		dfsLeaf(enter, leave, gc.variant1.Number1)
		dfsLeaf(enter, leave, gc.variant1.AttName3)
		dfsLeaf(enter, leave, gc.variant1.Not4)
		dfsLeaf(enter, leave, gc.variant1.Eq5)
		dfsLeaf(enter, leave, gc.variant1.Number6)
		dfsLeaf(enter, leave, gc.variant1.AttName8)

	} else if gc.variant2 != nil {
		dfsLeaf(enter, leave, gc.variant2.KwFreq1)
		dfsLeaf(enter, leave, gc.variant2.Number2)
		dfsLeaf(enter, leave, gc.variant2.AttName3)
		dfsLeaf(enter, leave, gc.variant2.Not4)
		dfsLeaf(enter, leave, gc.variant2.Operator5)
		dfsLeaf(enter, leave, gc.variant2.Number6)
	}
	leave(gc)
}

// ----------------------------------------------------

// Structure
//
// AttName _ AttValList?
type Structure struct {
	AttName    ASTString
	AttValList *AttValList
}

func (s *Structure) String() string {
	var ans strings.Builder
	ans.WriteString(fmt.Sprintf("Structure[%s]( ", s.AttName.String()))
	if s.AttValList != nil {
		ans.WriteString(s.AttValList.String())
	}
	ans.WriteString(" )")
	return ans.String()
}

// CQL renders "name" or "name attval1 & attval2 ..." - a structure's
// attribute name followed by its (optional) attribute-value list.
func (s *Structure) CQL() string {
	if s.AttValList == nil {
		return s.AttName.CQL()
	}
	return s.AttName.CQL() + " " + s.AttValList.CQL()
}

func (s *Structure) IsBigStructure() bool {
	v := s.AttName.String()
	return v == "s" || v == "g" || v == "p"
}

func (s *Structure) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, s)
	fn(s, s.AttName)
	if s.AttValList != nil {
		s.AttValList.ForEachElement(s, fn)
	}
}

func (s *Structure) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(s) {
		leave(s)
		return
	}
	dfsLeaf(enter, leave, s.AttName)
	if s.AttValList != nil {
		s.AttValList.DFS(enter, leave)
	}
	leave(s)
}

// ---------------------------------------------------------

// AttValList
//
//	av1:AttValAnd av2:(_ BINOR _ AttValAnd)*
type AttValList struct {
	origValue string
	AttValAnd []*AttValAnd
}

func (a *AttValList) String() string {
	var ans strings.Builder
	ans.WriteString("AttValList(")
	for i, v := range a.AttValAnd {
		if i > 0 {
			ans.WriteString(", ")
		}
		ans.WriteString(v.String())
	}
	ans.WriteString(" )")
	return ans.String()
}

// CQL renders "attval1 & attval2 | attval3 & attval4 | ..." -
// BINOR-joined AttValAnd groups.
func (a *AttValList) CQL() string {
	var ans strings.Builder
	for i, v := range a.AttValAnd {
		if i > 0 {
			ans.WriteString(" | ")
		}
		ans.WriteString(v.CQL())
	}
	return ans.String()
}

func (a *AttValList) NumAttVals() int {
	if a == nil {
		return 0
	}
	return len(a.AttValAnd)
}

func (a *AttValList) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, a)
	for _, v := range a.AttValAnd {
		v.ForEachElement(a, fn)
	}
}

func (a *AttValList) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(a) {
		leave(a)
		return
	}
	for _, v := range a.AttValAnd {
		v.DFS(enter, leave)
	}
	leave(a)
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

func (n *NumberedPosition) String() string {
	var ans strings.Builder
	ans.WriteString(fmt.Sprintf("NumberedPosition[%s]( ", n.Number.String()))
	if n.OnePosition != nil {
		ans.WriteString(n.OnePosition.String())
	}
	ans.WriteString(" )")
	return ans.String()
}

// CQL renders "N:position" - a numbered n-gram position.
func (n *NumberedPosition) CQL() string {
	return n.Number.CQL() + ":" + n.OnePosition.CQL()
}

func (n *NumberedPosition) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, n)
	fn(n, n.Number)
	fn(n, n.Colon)
	if n.OnePosition != nil {
		n.OnePosition.ForEachElement(n, fn)
	}
}

func (n *NumberedPosition) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(n) {
		leave(n)
		return
	}
	dfsLeaf(enter, leave, n.Number)
	dfsLeaf(enter, leave, n.Colon)
	if n.OnePosition != nil {
		n.OnePosition.DFS(enter, leave)
	}
	leave(n)
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

// -------------------------------------------------

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

func (op *OnePosition) String() string {
	var ans strings.Builder
	ans.WriteString("OnePosition( ")
	switch {
	case op.Variant1 != nil:
		ans.WriteString("[")
		if op.Variant1.AttValList != nil {
			ans.WriteString(op.Variant1.AttValList.String())
		}
		ans.WriteString("]")

	case op.Variant2 != nil:
		ans.WriteString(op.Variant2.RegExp.String())

	case op.Variant3 != nil:
		ans.WriteString(fmt.Sprintf("~%s", op.Variant3.Number.String()))
		ans.WriteString(op.Variant3.RegExp.String())

	case op.Variant4 != nil:
		ans.WriteString(op.Variant4.Value.String())

	case op.Variant5 != nil:
		ans.WriteString(op.Variant5.MuPart.String())
	}
	ans.WriteString(" )")
	return ans.String()
}

// CQL renders "[attvals]" / a regexp / "~N regexp" / "MU" / a mu-part,
// depending on which grammar alternative matched. KW_MU carries no
// fields of its own (the parser never populates onePositionVariant4),
// so it's also the fallback when no other variant is set.
func (op *OnePosition) CQL() string {
	if op.Variant1 != nil {
		if op.Variant1.AttValList != nil {
			return "[" + op.Variant1.AttValList.CQL() + "]"
		}
		return "[]"
	}
	if op.Variant2 != nil {
		return op.Variant2.RegExp.CQL()
	}
	if op.Variant3 != nil {
		var ans strings.Builder
		ans.WriteString("~")
		if op.Variant3.Number != "" {
			ans.WriteString(op.Variant3.Number.CQL())
		}
		ans.WriteString(op.Variant3.RegExp.CQL())
		return ans.String()
	}
	if op.Variant5 != nil {
		return op.Variant5.MuPart.CQL()
	}
	return "MU"
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

func (op *OnePosition) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(op) {
		leave(op)
		return
	}
	if op.Variant1 != nil && op.Variant1.AttValList != nil {
		op.Variant1.AttValList.DFS(enter, leave)

	} else if op.Variant2 != nil {
		op.Variant2.RegExp.DFS(enter, leave)

	} else if op.Variant3 != nil {
		dfsLeaf(enter, leave, op.Variant3.Number)
		op.Variant3.RegExp.DFS(enter, leave)

	} else if op.Variant4 != nil {
		dfsLeaf(enter, leave, op.Variant4.Value)

	} else if op.Variant5 != nil {
		op.Variant5.MuPart.DFS(enter, leave)
	}
	leave(op)
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

func (p *Position) String() string {
	var ans strings.Builder
	ans.WriteString("Position( ")
	if p.variant1 != nil {
		ans.WriteString(p.variant1.OnePosition.String())
	}
	if p.variant2 != nil {
		ans.WriteString(p.variant2.NumberedPosition.String())
	}
	ans.WriteString(" )")
	return ans.String()
}

// CQL renders whichever of OnePosition / NumberedPosition matched.
func (p *Position) CQL() string {
	if p.variant1 != nil {
		return p.variant1.OnePosition.CQL()
	}
	if p.variant2 != nil {
		return p.variant2.NumberedPosition.CQL()
	}
	return ""
}

func (p *Position) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, p)
	if p.variant1 != nil {
		p.variant1.OnePosition.ForEachElement(p, fn)

	} else if p.variant2 != nil {
		p.variant2.NumberedPosition.ForEachElement(p, fn)
	}
}

func (p *Position) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(p) {
		leave(p)
		return
	}
	if p.variant1 != nil {
		p.variant1.OnePosition.DFS(enter, leave)

	} else if p.variant2 != nil {
		p.variant2.NumberedPosition.DFS(enter, leave)
	}
	leave(p)
}

// -------------------------------------------------------

type RegExp struct {
	origValue string
	RegExpRaw []*RegExpRaw // these are A|B|C
}

func (r *RegExp) String() string {
	return r.origValue
}

// CQL returns the regexp literal unchanged (quotes included) - no
// normalization is applied within regular expressions.
func (r *RegExp) CQL() string {
	return r.origValue
}

func (r *RegExp) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	for _, v := range r.RegExpRaw {
		v.ForEachElement(r, fn)
	}
}

func (r *RegExp) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(r) {
		leave(r)
		return
	}
	for _, v := range r.RegExpRaw {
		v.DFS(enter, leave)
	}
	leave(r)
}

// --------------------------------------------------------

type muPartVariant1 struct {
	UnionOp *UnionOp
}

type muPartVariant2 struct {
	MeetOp *MeetOp
}

// MuPart represents
// LPAREN _ op:(UnionOp / MeetOp) _ RPAREN
// (where Variant1 is the UnionOp and Variant2 is the MeetOp)
type MuPart struct {
	origValue string
	Variant1  *muPartVariant1
	Variant2  *muPartVariant2
}

func (m *MuPart) String() string {
	var ans strings.Builder
	ans.WriteString("MuPart( ")
	if m.Variant1 != nil {
		ans.WriteString(m.Variant1.UnionOp.String())
	}
	if m.Variant2 != nil {
		ans.WriteString(m.Variant2.MeetOp.String())
	}
	ans.WriteString(" )")
	return ans.String()
}

// CQL renders "(union p1 p2)" / "(meet p1 p2)".
func (m *MuPart) CQL() string {
	if m.Variant1 != nil {
		return "(" + m.Variant1.UnionOp.CQL() + ")"
	}
	if m.Variant2 != nil {
		return "(" + m.Variant2.MeetOp.CQL() + ")"
	}
	return ""
}

func (m *MuPart) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, m)
	if m.Variant1 != nil {
		m.Variant1.UnionOp.ForEachElement(m, fn)

	} else if m.Variant2 != nil {
		m.Variant2.MeetOp.ForEachElement(m, fn)
	}
}

func (m *MuPart) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(m) {
		leave(m)
		return
	}
	if m.Variant1 != nil {
		m.Variant1.UnionOp.DFS(enter, leave)

	} else if m.Variant2 != nil {
		m.Variant2.MeetOp.DFS(enter, leave)
	}
	leave(m)
}

// --------------------------------------------------------------

// UnionOp represents
// KW_UNION _ p1:Position _ p2:Position
type UnionOp struct {
	origValue string
	Position1 *Position
	Position2 *Position
}

func (m *UnionOp) String() string {
	var ans strings.Builder
	ans.WriteString("UnionOp( ")
	ans.WriteString(m.Position1.String())
	ans.WriteString(", ")
	ans.WriteString(m.Position2.String())
	ans.WriteString(" )")
	return ans.String()
}

func (m *UnionOp) CQL() string {
	var ans strings.Builder
	ans.WriteString("union ")
	ans.WriteString(m.Position1.CQL())
	ans.WriteString(" ")
	ans.WriteString(m.Position2.CQL())
	return ans.String()
}

func (m *UnionOp) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, m)
	m.Position1.ForEachElement(m, fn)
	m.Position2.ForEachElement(m, fn)
}

func (m *UnionOp) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(m) {
		leave(m)
		return
	}
	m.Position1.DFS(enter, leave)
	m.Position2.DFS(enter, leave)
	leave(m)
}

// ---------------------------------------------------------------

type MeetRange struct {
	lft string
	rgt string
}

func (m *MeetRange) CQL() string {
	return fmt.Sprintf("%s %s", m.lft, m.rgt)
}

func (m *MeetRange) String() string {
	return fmt.Sprintf("MeetRange(%s %s)", m.lft, m.rgt)
}

// --------------------------------------------------------------

// MeetOp represents the rule
// KW_MEET _ p1:Position _ p2:Position _ (Integer _ Integer)?
type MeetOp struct {
	origValue string
	Position1 *Position
	Position2 *Position
	Range     *MeetRange
}

func (m *MeetOp) String() string {
	var ans strings.Builder
	ans.WriteString(m.Position1.String())
	ans.WriteString(", ")
	ans.WriteString(m.Position2.String())
	ans.WriteString(", ")
	ans.WriteString(m.Range.String())
	ans.WriteString(" )")
	return ans.String()
}

func (m *MeetOp) CQL() string {
	var ans strings.Builder
	ans.WriteString("meet ")
	ans.WriteString(m.Position1.CQL())
	ans.WriteString(" ")
	ans.WriteString(m.Position2.CQL())
	ans.WriteString(" ")
	ans.WriteString(m.Range.CQL())
	return ans.String()
}

func (m *MeetOp) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, m)
	m.Position1.ForEachElement(m, fn)
	m.Position2.ForEachElement(m, fn)
}

func (m *MeetOp) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(m) {
		leave(m)
		return
	}
	m.Position1.DFS(enter, leave)
	m.Position2.DFS(enter, leave)
	leave(m)
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

// Repetition represents rules
// AtomQuery RepOpt?
// OpenStructTag
// CloseStructTag
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

func (r *Repetition) SubcorpusDefScore() float64 {
	if r.Variant2 != nil && r.Variant2.OpenStructTag != nil {
		return r.Variant2.OpenStructTag.SubcorpusDefScore()
	}
	return 0
}

func (r *Repetition) String() string {
	var ans strings.Builder
	ans.WriteString("Repetition( ")
	if r.Variant1 != nil {
		ans.WriteString(r.Variant1.AtomQuery.String())
		if r.Variant1.RepOpt != nil {
			ans.WriteString(" ")
			ans.WriteString(r.Variant1.RepOpt.String())
		}
	}
	if r.Variant2 != nil {
		ans.WriteString(r.Variant2.OpenStructTag.String())
	}
	if r.Variant3 != nil {
		ans.WriteString(r.Variant3.CloseStructTag.String())
	}
	ans.WriteString(" )")
	return ans.String()
}

// CQL renders an atom query with its optional repetition suffix, or an
// open/close struct tag.
func (r *Repetition) CQL() string {
	if r.Variant1 != nil {
		ans := r.Variant1.AtomQuery.CQL()
		if r.Variant1.RepOpt != nil {
			ans += r.Variant1.RepOpt.CQL()
		}
		return ans
	}
	if r.Variant2 != nil {
		return r.Variant2.OpenStructTag.CQL()
	}
	if r.Variant3 != nil {
		return r.Variant3.CloseStructTag.CQL()
	}
	return ""
}

func (r *Repetition) RepetitionScore() float64 {
	if r.Variant1 != nil && r.Variant1.RepOpt != nil {
		return r.Variant1.RepOpt.RepetitionScore()
	}
	return 0
}

func (r *Repetition) GetRepOpt() string {
	if r.Variant1 != nil && r.Variant1.RepOpt != nil {
		return string(r.Variant1.RepOpt.String())
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

func (r *Repetition) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(r) {
		leave(r)
		return
	}
	if r.Variant1 != nil {
		r.Variant1.AtomQuery.DFS(enter, leave)
		dfsLeaf(enter, leave, r.Variant1.RepOpt)

	} else if r.Variant2 != nil {
		r.Variant2.OpenStructTag.DFS(enter, leave)

	} else if r.Variant3 != nil {
		r.Variant3.CloseStructTag.DFS(enter, leave)
	}
	leave(r)
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

func (aq *AtomQuery) String() string {
	var ans strings.Builder
	ans.WriteString("AtomQuery( ")
	if aq.variant1 != nil {
		ans.WriteString(aq.variant1.Position.String())
	}
	if aq.variant2 != nil {
		ans.WriteString(aq.variant2.Sequence.String())
		for _, v := range aq.variant2.WCBlock {
			ans.WriteString(", ")
			if v.Not {
				ans.WriteString("!")
			}
			ans.WriteString(v.Keyword.String())
			ans.WriteString(" ")
			ans.WriteString(v.WithinContainingPart.String())
		}
	}
	ans.WriteString(" )")
	return ans.String()
}

// CQL renders a bare position, or a parenthesized sequence with its
// trailing within/containing blocks.
func (aq *AtomQuery) CQL() string {
	if aq.variant1 != nil {
		return aq.variant1.Position.CQL()
	}
	if aq.variant2 != nil {
		var ans strings.Builder
		ans.WriteString("(")
		ans.WriteString(aq.variant2.Sequence.CQL())
		for _, wc := range aq.variant2.WCBlock {
			ans.WriteString(" ")
			if wc.Not {
				ans.WriteString("!")
			}
			ans.WriteString(wc.Keyword.CQL())
			ans.WriteString(" ")
			ans.WriteString(wc.WithinContainingPart.CQL())
		}
		ans.WriteString(")")
		return ans.String()
	}
	return ""
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

func (aq *AtomQuery) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(aq) {
		leave(aq)
		return
	}
	if aq.variant1 != nil {
		aq.variant1.Position.DFS(enter, leave)

	} else if aq.variant2 != nil {
		aq.variant2.Sequence.DFS(enter, leave)
		if aq.variant2.WCBlock != nil {
			for _, v := range aq.variant2.WCBlock {
				if v.Not {
					dfsLeaf(enter, leave, ASTString("!"))
				}
				dfsLeaf(enter, leave, v.Keyword)
				v.WithinContainingPart.DFS(enter, leave)
			}
		}
	}
	leave(aq)
}

// --------------------------------------------------------------

type repOptVariant1 struct {
	Value ASTString
}

type repOptVariant2 struct {
	From ASTString
	To   ASTString
	// HasComma distinguishes "{n}" (exact count, HasComma == false) from
	// "{n,}" (at least n, HasComma == true but To == "") - both leave To
	// empty, so that alone can't tell them apart.
	HasComma bool
}

type RepOpt struct {
	Variant1 *repOptVariant1
	Variant2 *repOptVariant2
}

func (r *RepOpt) RepetitionScore() float64 {
	if r.Variant1 != nil && (r.Variant1.Value == "+" || r.Variant1.Value == "*") ||
		r.Variant2 != nil && r.Variant2.From.String() != "" && r.Variant2.To.String() == "" {
		return 100
	}
	if r.Variant2 != nil && r.Variant2.From.String() != "" && r.Variant2.To.String() != "" {
		toInt, err := strconv.Atoi(r.Variant2.To.String())
		if err != nil {
			// TODO
			log.Error().Err(err).Msg("failed to determine position repetition score")
			return 0
		}
		return float64(toInt)
	}
	return 0
}

func (r *RepOpt) String() string {
	var ans strings.Builder
	ans.WriteString("RepOpt( ")
	if r.Variant1 != nil {
		ans.WriteString(r.Variant1.Value.String())

	} else if r.Variant2 != nil {
		ans.WriteString(fmt.Sprintf("{%s, %s}", r.Variant2.From, r.Variant2.To))
	}
	ans.WriteString(" )")
	return ans.String()
}

// CQL renders "*" / "+" / "?" / "{n}" / "{n,}" / "{n,m}".
// It distinguishes an exact count ("{n}", HasComma false)
// from "at least n" ("{n,}", HasComma true with To empty) - both leave
// To empty, so HasComma is what tells them apart.
func (r *RepOpt) CQL() string {
	if r.Variant1 != nil {
		return r.Variant1.Value.CQL()
	}
	if r.Variant2 != nil {
		if !r.Variant2.HasComma {
			return fmt.Sprintf("{%s}", r.Variant2.From.CQL())
		}
		if r.Variant2.To == "" {
			return fmt.Sprintf("{%s,}", r.Variant2.From.CQL())
		}
		return fmt.Sprintf("{%s,%s}", r.Variant2.From.CQL(), r.Variant2.To.CQL())
	}
	return ""
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

func (r *RepOpt) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(r) {
		leave(r)
		return
	}
	if r.Variant1 != nil {
		dfsLeaf(enter, leave, r.Variant1.Value)

	} else if r.Variant2 != nil {
		dfsLeaf(enter, leave, r.Variant2.From)
		dfsLeaf(enter, leave, r.Variant2.To)
	}
	leave(r)
}

// ----------------------------------------------------------------

type OpenStructTag struct {
	origValue string
	Structure *Structure
}

func (ost *OpenStructTag) String() string {
	var ans strings.Builder
	ans.WriteString("OpenStructTag")
	if ost.IsSelfClosing() {
		ans.WriteString("[/]")
	}
	ans.WriteString("( ")
	if ost.Structure != nil {
		ans.WriteString(ost.Structure.String())
	}
	ans.WriteString(" )")
	return ans.String()
}

// CQL renders "<structure>" / "<structure />".
func (ost *OpenStructTag) CQL() string {
	if ost.IsSelfClosing() {
		return "<" + ost.Structure.CQL() + " />"
	}
	return "<" + ost.Structure.CQL() + ">"
}

func (ost *OpenStructTag) IsSelfClosing() bool {
	return strings.HasSuffix(ost.origValue, "/>")
}

func (ost *OpenStructTag) SubcorpusDefScore() float64 {
	if ost.Structure != nil && ost.Structure.AttValList != nil {
		return float64(ost.Structure.AttValList.NumAttVals())
	}
	return 0
}

func (ost *OpenStructTag) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, ost)
	ost.Structure.ForEachElement(ost, fn)
}

func (ost *OpenStructTag) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(ost) {
		leave(ost)
		return
	}
	ost.Structure.DFS(enter, leave)
	leave(ost)
}

// --------------------------------------------------------------

type CloseStructTag struct {
	Structure *Structure
}

func (ost *CloseStructTag) String() string {
	var ans strings.Builder
	ans.WriteString("CloseStructTag( ")
	if ost.Structure != nil {
		ans.WriteString(ost.Structure.String())
	}
	ans.WriteString(" )")
	return ans.String()
}

// CQL renders "</structure>".
func (ost *CloseStructTag) CQL() string {
	return "</" + ost.Structure.CQL() + ">"
}

func (ost *CloseStructTag) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, ost)
	ost.Structure.ForEachElement(ost, fn)
}

func (ost *CloseStructTag) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(ost) {
		leave(ost)
		return
	}
	ost.Structure.DFS(enter, leave)
	leave(ost)
}

// ---------------------------------------------------------

type AlignedPart struct {
	AttName  ASTString
	Sequence *Sequence
}

func (a *AlignedPart) String() string {
	var ans strings.Builder
	ans.WriteString(fmt.Sprintf("AlignedPart[%s]( ", a.AttName.String()))
	if a.Sequence != nil {
		ans.WriteString(a.Sequence.String())
	}
	ans.WriteString(" )")
	return ans.String()
}

// CQL renders "attName: sequence" - a parallel-alignment part.
func (a *AlignedPart) CQL() string {
	return a.AttName.CQL() + ": " + a.Sequence.CQL()
}

func (a *AlignedPart) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, a)
	// TODO
}

func (a *AlignedPart) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(a) {
		leave(a)
		return
	}
	leave(a)
}

// -----------------------------------------------------------

// AttValAnd
//
//	av1:AttVal av2:(_ BINAND _ AttVal)*
type AttValAnd struct {
	AttVal []*AttVal
}

func (a *AttValAnd) String() string {
	var ans strings.Builder
	ans.WriteString("AttValAnd( ")
	for i, v := range a.AttVal {
		if i > 0 {
			ans.WriteString(", ")
		}
		ans.WriteString(v.String())
	}
	ans.WriteString(" )")
	return ans.String()
}

// CQL renders "attval1 & attval2 & ..." - BINAND-joined AttVal items.
func (a *AttValAnd) CQL() string {
	var ans strings.Builder
	for i, v := range a.AttVal {
		if i > 0 {
			ans.WriteString(" & ")
		}
		ans.WriteString(v.CQL())
	}
	return ans.String()
}

func (a *AttValAnd) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, a)
	for _, item := range a.AttVal {
		item.ForEachElement(a, fn)
	}
}

func (a *AttValAnd) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(a) {
		leave(a)
		return
	}
	for _, item := range a.AttVal {
		item.DFS(enter, leave)
	}
	leave(a)
}

// --------------------------------------------------------------

// AttName _ (NOT)? EEQ _ RawString
type attValVariant1 struct {
	AttName   ASTString
	Not       bool
	Eeq       ASTString
	RawString *RawString
}

// CQL renders "attName==rawstring" (optionally negated).
func (av attValVariant1) CQL() string {
	var ans strings.Builder
	ans.WriteString(av.AttName.CQL())
	if av.Not {
		ans.WriteString("!")
	}
	ans.WriteString(av.Eeq.CQL())
	ans.WriteString(av.RawString.CQL())
	return ans.String()
}

// AttName (_ NOT)? _ (EQ / LEQ / GEQ / TEQ NUMBER?) _ RegExp
type attValVariant2 struct {
	AttName ASTString
	Not     bool
	Op      ASTString
	RegExp  *RegExp
}

// CQL renders "attName=regexp" (optionally negated; Op holds the
// matched operator: =, <=, >=, or ~).
func (av attValVariant2) CQL() string {
	var ans strings.Builder
	ans.WriteString(av.AttName.CQL())
	if av.Not {
		ans.WriteString("!")
	}
	ans.WriteString(av.Op.CQL())
	ans.WriteString(av.RegExp.CQL())
	return ans.String()
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

// CQL renders "!attval".
func (av attValVariant5) CQL() string {
	return "!" + av.AttVal.CQL()
}

// LPAREN _ AttValList _ RPAREN
type attValVariant6 struct {
	AttValList *AttValList
}

// CQL renders "(attvallist)".
func (av attValVariant6) CQL() string {
	return "(" + av.AttValList.CQL() + ")"
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

// -----------

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

func (a *AttVal) IsRecursive() bool {
	return a.Variant6 != nil || a.Variant8 != nil || a.Variant9 != nil
}

func (a *AttVal) getAttName() string {
	if a.Variant1 != nil {
		return a.Variant1.AttName.String()

	} else if a.Variant2 != nil {
		return a.Variant2.AttName.String()
	}
	return ""
}

func (a *AttVal) rawValue() string {
	if a.Variant1 != nil {
		return a.Variant1.RawString.String()
	}
	if a.Variant2 != nil {
		return a.Variant2.RegExp.String()
	}
	return ""
}

// String renders "AttVal( ... )". Variant3/4/7/8/9 carry no parsed
// fields (see the CQL()/ForEachElement comments below), so origValue -
// the raw matched source - is used for those instead.
func (a *AttVal) String() string {
	var ans strings.Builder
	ans.WriteString("AttVal( ")
	switch {
	case a.Variant1 != nil:
		ans.WriteString(a.Variant1.AttName.String())
		if a.Variant1.Not {
			ans.WriteString("!")
		}
		ans.WriteString(a.Variant1.Eeq.String())
		ans.WriteString(a.Variant1.RawString.String())

	case a.Variant2 != nil:
		ans.WriteString(a.Variant2.AttName.String())
		if a.Variant2.Not {
			ans.WriteString("!")
		}
		ans.WriteString(a.Variant2.Op.String())
		ans.WriteString(a.Variant2.RegExp.String())

	case a.Variant5 != nil:
		ans.WriteString("!")
		ans.WriteString(a.Variant5.AttVal.String())

	case a.Variant6 != nil:
		ans.WriteString(a.Variant6.AttValList.String())

	default:
		ans.WriteString(a.origValue)
	}
	ans.WriteString(" )")
	return ans.String()
}

// CQL renders whichever grammar variant matched. Variant3/4/7/8/9
// (POSNUM ranges, ws()/term()/swap()/ccoll() calls) carry no parsed
// fields - the grammar actions never populate them (see the TODOs in
// ForEachElement/DFS below) - so origValue, the raw matched source, is
// the only data available for those and is echoed unchanged.
func (a *AttVal) CQL() string {
	if a.Variant1 != nil {
		return a.Variant1.CQL()
	}
	if a.Variant2 != nil {
		return a.Variant2.CQL()
	}
	if a.Variant5 != nil {
		return a.Variant5.CQL()
	}
	if a.Variant6 != nil {
		return a.Variant6.CQL()
	}
	return a.origValue
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

func (a *AttVal) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(a) {
		leave(a)
		return
	}
	if a.Variant1 != nil {
		dfsLeaf(enter, leave, a.Variant1.AttName)
		dfsLeaf(enter, leave, a.Variant1.Eeq)
		a.Variant1.RawString.DFS(enter, leave)

	} else if a.Variant2 != nil {
		dfsLeaf(enter, leave, a.Variant2.AttName)
		dfsLeaf(enter, leave, a.Variant2.Op)
		a.Variant2.RegExp.DFS(enter, leave)

	} else if a.Variant3 != nil {
		// TODO a.variant3

	} else if a.Variant4 != nil {
		// TODO a.variant4

	} else if a.Variant5 != nil {
		a.Variant5.AttVal.DFS(enter, leave)

	} else if a.Variant6 != nil {
		a.Variant6.AttValList.DFS(enter, leave)

	} else if a.Variant7 != nil {
		// TODO a.variant7

	} else if a.Variant8 != nil {
		// TODO a.variant8

	} else if a.Variant9 != nil {
		// TODO a.variant9
	}
	leave(a)
}

// ---------------------------------------------------

type WithinNumber struct {
	Value ASTString
}

func (w *WithinNumber) String() string {
	return fmt.Sprintf("WithinNumber( %s )", w.Value.String())
}

// CQL renders the plain number.
func (w *WithinNumber) CQL() string {
	return w.Value.CQL()
}

func (w *WithinNumber) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, w.Value)
}

func (w *WithinNumber) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	dfsLeaf(enter, leave, w.Value)
}

// ----------------------------------------------------------

type RegExpRaw struct {
	origValue string
	// RgLook / RgGrouped / RgSimple
	Values []ASTNode
}

func (r *RegExpRaw) String() string {
	return r.origValue
}

// CQL returns the matched regexp fragment unchanged - no normalization
// is applied within regular expressions.
func (r *RegExpRaw) CQL() string {
	return r.origValue
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

func (r *RegExpRaw) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(r) {
		leave(r)
		return
	}
	for _, item := range r.Values {
		switch tItem := item.(type) {
		case *RgLook:
			tItem.DFS(enter, leave)
		case *RgGrouped:
			tItem.DFS(enter, leave)
		case *RgSimple:
			tItem.DFS(enter, leave)
		}
	}
	leave(r)
}

// ------------------------------------------------------------------

type RawString struct {
	SimpleString *SimpleString
}

func (r *RawString) String() string {
	if r.SimpleString != nil {
		return fmt.Sprintf("RawString( %s )", r.SimpleString.String())
	}
	return "RawString(  )"
}

// CQL renders "\"content\"" - RawString itself doesn't keep the
// surrounding quotes (unlike RegExp), so they're added back here.
func (r *RawString) CQL() string {
	if r.SimpleString != nil {
		return `"` + r.SimpleString.CQL() + `"`
	}
	return `""`
}

func (r *RawString) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	r.SimpleString.ForEachElement(r, fn)
}

func (r *RawString) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(r) {
		leave(r)
		return
	}
	r.SimpleString.DFS(enter, leave)
	leave(r)
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

func (r *SimpleString) String() string {
	var ans strings.Builder
	for _, v := range r.Values {
		ans.WriteString(string(v))
	}
	return ans.String()
}

func (r *SimpleString) CQL() string {
	return r.String()
}

func (r *SimpleString) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, ASTString(r.String()))
}

func (r *SimpleString) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	dfsLeaf(enter, leave, ASTString(r.String()))
}
