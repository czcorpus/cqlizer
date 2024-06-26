package cql

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Seq (_ BINOR _ Seq)* / Seq
type Sequence struct {
	origValue string
	Seq       []*Seq
}

func (q *Sequence) Text() string {
	return q.origValue
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

// --------------------------------------------------------------------

type Seq struct {
	origValue   string
	isOrChained bool
	Not         ASTString
	Repetition  []*Repetition
}

func (q *Seq) IsOrChained() bool {
	return q.isOrChained
}

func (s *Seq) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, s)
	fn(parent, s.Not)
	for _, item := range s.Repetition {
		item.ForEachElement(s, fn)
	}
}

func (s *Seq) Text() string {
	return string(s.origValue)
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

func (q *GlobPart) Text() string {
	return "#GlobPart" // TODO
}

// ---------------------------------------

// WithinOrContaining
//
//	NOT? (KW_WITHIN / KW_CONTAINING) _ WithinContainingPart {
type WithinOrContaining struct {
	KwWithin             ASTString
	KwContaining         ASTString
	WithinContainingPart *WithinContainingPart
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

func (w *WithinOrContaining) Text() string {
	return "#WithinOrContaining"
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

// ----------------------------------------------------

// Structure
//
// AttName _ AttValList?
type Structure struct {
	AttName    ASTString
	AttValList *AttValList
}

func (s *Structure) Text() string {
	return "#Structure"
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

// ---------------------------------------------------------

// AttValList
//
//	av1:AttValAnd av2:(_ BINOR _ AttValAnd)*
type AttValList struct {
	AttValAnd []*AttValAnd
}

func (a *AttValList) Text() string {
	return "#AttValList"
}

func (a *AttValList) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, a)
	for _, v := range a.AttValAnd {
		v.ForEachElement(a, fn)
	}
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

func (n *NumberedPosition) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, n)
	fn(n, n.Number)
	fn(n, n.Colon)
	if n.OnePosition != nil {
		n.OnePosition.ForEachElement(n, fn)
	}
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
	variant1 *onePositionVariant1
	variant2 *onePositionVariant2
	variant3 *onePositionVariant3
	variant4 *onePositionVariant4
	variant5 *onePositionVariant5
}

func (op *OnePosition) Text() string {
	return "#OnePosition"
}

func (op *OnePosition) MarshalJSON() ([]byte, error) {
	if op.variant1 != nil {
		return json.Marshal(op.variant1)

	} else if op.variant2 != nil {
		return json.Marshal(op.variant2)

	} else if op.variant3 != nil {
		return json.Marshal(op.variant3)

	} else if op.variant4 != nil {
		return json.Marshal(op.variant4)

	} else if op.variant5 != nil {
		return json.Marshal(op.variant5)

	} else {
		return json.Marshal(struct{}{})
	}
}

func (op *OnePosition) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, op)
	if op.variant1 != nil {
		op.variant1.AttValList.ForEachElement(op, fn)

	} else if op.variant2 != nil {
		op.variant2.RegExp.ForEachElement(op, fn)

	} else if op.variant3 != nil {
		fn(op, op.variant3.Number)
		op.variant3.RegExp.ForEachElement(op, fn)

	} else if op.variant4 != nil {
		fn(op, op.variant4.Value)

	} else if op.variant5 != nil {
		op.variant5.MuPart.ForEachElement(op, fn)
	}
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
	variant1 *positionVariant1

	variant2 *positionVariant2
}

func (p *Position) Text() string {
	return "#Position"
}

func (p *Position) MarshalJSON() ([]byte, error) {
	if p.variant1 != nil {
		return json.Marshal(p.variant1)

	} else if p.variant2 != nil {
		return json.Marshal(p.variant2)

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

// -------------------------------------------------------

type RegExp struct {
	RegExpRaw *RegExpRaw
}

func (r *RegExp) Text() string {
	return "#RegExp"
}

func (r *RegExp) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	r.RegExpRaw.ForEachElement(r, fn)
}

// --------------------------------------------------------

type MuPart struct {
}

func (m *MuPart) Text() string {
	return "#MuPart"
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
	// TODO
}

// --------------------------------------------------------------

type UnionOp struct {
}

// ---------------------------------------------------------------

type MeetOp struct {
}

// --------------------------------------------------------------------------

type Integer struct {
}

// ------------------------------------------------------------

type repetitionVariant1 struct {
	RepOpt    *RepOpt
	AtomQuery *AtomQuery
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
	variant1       *repetitionVariant1
	variant2       *repetitionVariant2
	variant3       *repetitionVariant3
}

func (r *Repetition) Text() string {
	return r.origValue
}

func (r *Repetition) GetRepOpt() string {
	if r.variant1 != nil && r.variant1.RepOpt != nil {
		return string(r.variant1.RepOpt.Text())
	}
	return ""
}

func (r *Repetition) IsTailPosition() bool {
	return r.isTailPosition
}

func (r *Repetition) MarshalJSON() ([]byte, error) {
	if r.variant1 != nil {
		return json.Marshal(struct {
			RuleName  string
			Expansion repetitionVariant1
		}{
			RuleName:  "Repetition",
			Expansion: *r.variant1,
		})

	} else if r.variant2 != nil {
		return json.Marshal(struct {
			RuleName  string
			Expansion repetitionVariant2
		}{
			RuleName:  "Repetition",
			Expansion: *r.variant2,
		})

	} else if r.variant3 != nil {
		return json.Marshal(struct {
			RuleName  string
			Expansion repetitionVariant3
		}{
			RuleName:  "Repetition",
			Expansion: *r.variant3,
		})

	} else {
		return json.Marshal(struct{}{})
	}
}

func (r *Repetition) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	if r.variant1 != nil {
		r.variant1.AtomQuery.ForEachElement(r, fn)
		fn(r, r.variant1.RepOpt)

	} else if r.variant2 != nil {
		r.variant2.OpenStructTag.ForEachElement(r, fn)

	} else if r.variant3 != nil {
		r.variant3.CloseStructTag.ForEachElement(r, fn)
	}
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
	origValue string
	variant1  *atomQueryVariant1
	variant2  *atomQueryVariant2
}

func (aq *AtomQuery) Text() string {
	return aq.origValue
}

func (aq *AtomQuery) MarshalJSON() ([]byte, error) {
	if aq.variant1 != nil {
		return json.Marshal(aq.variant1)

	} else if aq.variant2 != nil {
		return json.Marshal(aq.variant2)

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
		if aq.variant2.WithinContainingPart != nil {
			aq.variant2.WithinContainingPart.ForEachElement(aq, fn)
		}
	}
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
	variant1 *repOptVariant1
	variant2 *repOptVariant2
}

func (r *RepOpt) Text() string {
	if r.variant1 != nil {
		return r.variant1.Value.Text()

	} else if r.variant2 != nil {
		return fmt.Sprintf("{%s, %s}", r.variant2.From, r.variant2.To)
	}
	return ""
}

func (r *RepOpt) MarshalJSON() ([]byte, error) {
	if r.variant1 != nil {
		return json.Marshal(struct {
			RuleName  string
			Expansion repOptVariant1
		}{
			RuleName:  "RepOpt",
			Expansion: *r.variant1,
		})

	} else if r.variant2 != nil {
		return json.Marshal(struct {
			RuleName  string
			Expansion repOptVariant2
		}{
			RuleName:  "RepOpt",
			Expansion: *r.variant2,
		})

	} else {
		return json.Marshal(struct{}{})
	}
}

func (r *RepOpt) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	if r.variant1 != nil {
		fn(r, r.variant1.Value)

	} else if r.variant2 != nil {
		fn(r, r.variant2.From)
		fn(r, r.variant2.To)
	}
}

// ----------------------------------------------------------------

type OpenStructTag struct {
	Structure *Structure
}

func (ost *OpenStructTag) Text() string {
	return "#OpenStructTag"
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

// --------------------------------------------------------------

type CloseStructTag struct {
	Structure *Structure
}

func (ost *CloseStructTag) Text() string {
	return "#CloseStructTag"
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

// ---------------------------------------------------------

type AlignedPart struct {
}

func (a *AlignedPart) Text() string {
	return "#AlignedPart"
}

func (a *AlignedPart) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, a)
	// TODO
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

func (a *AttValAnd) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, a)
	for _, item := range a.AttVal {
		item.ForEachElement(a, fn)
	}
}

// --------------------------------------------------------------

// AttName _ (NOT)? EEQ _ RawString
type attValVariant1 struct {
	AttName   ASTString
	Not       ASTString
	Eeq       ASTString
	RawString *RawString
}

// AttName (_ NOT)? _ (EQ / LEQ / GEQ / TEQ NUMBER?) _ RegExp
type attValVariant2 struct {
	AttName ASTString
	Not     ASTString
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
}

// LPAREN _ AttValList _ RPAREN
type attValVariant6 struct {
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
	variant1 *attValVariant1
	variant2 *attValVariant2
	variant3 *attValVariant3
	variant4 *attValVariant4
	variant5 *attValVariant5
	variant6 *attValVariant6
	variant7 *attValVariant7
	variant8 *attValVariant8
	variant9 *attValVariant9
}

func (r *AttVal) Text() string {
	return "#AttVal"
}

func (r *AttVal) MarshalJSON() ([]byte, error) {
	if r.variant1 != nil {
		return json.Marshal(r.variant1)

	} else if r.variant2 != nil {
		return json.Marshal(r.variant2)

	} else if r.variant3 != nil {
		return json.Marshal(r.variant3)

	} else if r.variant4 != nil {
		return json.Marshal(r.variant4)

	} else if r.variant5 != nil {
		return json.Marshal(r.variant5)

	} else if r.variant6 != nil {
		return json.Marshal(r.variant6)

	} else if r.variant7 != nil {
		return json.Marshal(r.variant7)

	} else if r.variant8 != nil {
		return json.Marshal(r.variant8)

	} else if r.variant9 != nil {
		return json.Marshal(r.variant9)

	} else {
		return json.Marshal(struct{}{})
	}
}

func (a *AttVal) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, a)
	if a.variant1 != nil {
		fn(a, a.variant1.AttName)
		fn(a, a.variant1.Not)
		fn(a, a.variant1.Eeq)
		a.variant1.RawString.ForEachElement(a, fn)

	} else if a.variant2 != nil {
		fn(a, a.variant2.AttName)
		fn(a, a.variant2.Not)
		fn(a, a.variant2.Op)
		a.variant2.RegExp.ForEachElement(a, fn)

	} else if a.variant3 != nil {
		// TODO a.variant3

	} else if a.variant4 != nil {
		// TODO a.variant4

	} else if a.variant5 != nil {
		// TODO a.variant5

	} else if a.variant6 != nil {
		// TODO a.variant6

	} else if a.variant7 != nil {
		// TODO a.variant7

	} else if a.variant8 != nil {
		// TODO a.variant8

	} else if a.variant9 != nil {
		// TODO a.variant9
	}
}

// ---------------------------------------------------

type WithinNumber struct {
	Value ASTString
}

func (w *WithinNumber) Text() string {
	return "#WithinNumber"
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

// ----------------------------------------------------------

type RegExpRaw struct {

	// RgLook / RgGrouped / RgSimple
	Values []any
}

func (r *RegExpRaw) Text() string {
	return "#RegExpRaw"
}

func (r *RegExpRaw) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion RegExpRaw
	}{
		RuleName:  "RegExpRaw",
		Expansion: *r,
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

// ------------------------------------------------------------------

type RawString struct {
	SimpleString *SimpleString
}

func (r *RawString) Text() string {
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

// ------------------------------------------------------------------------

type SimpleString struct {
	Values []ASTString
}

func (r *SimpleString) Text() string {
	var ans strings.Builder
	for _, v := range r.Values {
		ans.WriteString(string(v))
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

// -------------------------------------------------

type RgGrouped struct {
	Value *RegExpRaw
}

func (r *RgGrouped) Text() string {
	return "#RgGrouped"
}

func (r *RgGrouped) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion RgGrouped
	}{
		RuleName:  "RgGrouped",
		Expansion: *r,
	})
}

func (r *RgGrouped) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	r.Value.ForEachElement(r, fn)
}

// ---------------------------------------------------------

type RgSimple struct {
	// RgRange / RgChar / RgAlt / RgPosixClass
	Values []any
}

func (r *RgSimple) Text() string {
	return "#RgSimple"
}

func (r *RgSimple) ExpensiveOps() []string {
	var state int
	ans := make([]string, 0, 10)
	for _, val := range r.Values {
		switch tVal := val.(type) {
		case *RgChar:
			if tVal.variant2 != nil {
				if tVal.variant2.Value.Value == "." {
					if state == 0 {
						state = 1

					} else if state == 2 {
						state = 1
					}

				} else if tVal.variant2.Value.Value == "+" || tVal.variant2.Value.Value == "*" {
					if state == 1 {
						ans = append(ans, fmt.Sprintf(".%s", tVal.variant2.Value.Value))
					}
				}
			}
		}
	}
	return ans
}

func (r *RgSimple) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion RgSimple
	}{
		RuleName:  "RgSimple",
		Expansion: *r,
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

// ----------------------------------------------------

type RgPosixClass struct {
	Value ASTString
}

func (r *RgPosixClass) Text() string {
	return "#RgPosixClass"
}

func (r *RgPosixClass) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Value)
}

func (r *RgPosixClass) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r.Value)
}

// ----------------------------------------------------

type RgLook struct {
	Value ASTString
}

func (r *RgLook) Text() string {
	return "#RgLook"
}

func (r *RgLook) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion RgLook
	}{
		RuleName:  "RgLook",
		Expansion: *r,
	})
}

func (r *RgLook) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r.Value)
}

// ----------------------------------------------------

type RgLookOperator struct {
}

// -----------------------------------------------------

type RgAlt struct {
	Values []*RgAltVal
}

func (r *RgAlt) Text() string {
	return "#RgAlt"
}

func (r *RgAlt) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion RgAlt
	}{
		RuleName:  "RgAlt",
		Expansion: *r,
	})
}

func (r *RgAlt) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	for _, item := range r.Values {
		item.ForEachElement(r, fn)
	}
}

// --------------------------------------------------------

type rgCharVariant1 struct {
	Value ASTString
}

type rgCharVariant2 struct {
	Value *RgOp
}

type RgChar struct {
	variant1 *rgCharVariant1
	variant2 *rgCharVariant2
}

func (rc *RgChar) Text() string {
	return "#RgChar"
}

func (rc *RgChar) MarshalJSON() ([]byte, error) {
	if rc.variant1 != nil {
		return json.Marshal(struct {
			RuleName  string
			Expansion rgCharVariant1
		}{
			RuleName:  "RgChar",
			Expansion: *rc.variant1,
		})

	} else if rc.variant2 != nil {
		return json.Marshal(struct {
			RuleName  string
			Expansion rgCharVariant2
		}{
			RuleName:  "RgChar",
			Expansion: *rc.variant2,
		})

	} else {
		return json.Marshal(struct{}{})
	}
}

func (r *RgChar) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	if r.variant1 != nil {
		fn(r, r.variant1.Value)

	} else if r.variant2 != nil {
		r.variant2.Value.ForEachElement(r, fn)
	}
}

// -----------------------------------------------------------

type RgRange struct {
	RgRangeSpec *RgRangeSpec
}

func (r *RgRange) Text() string {
	return "#RgRange"
}

func (r *RgRange) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion RgRange
	}{
		RuleName:  "RgRange",
		Expansion: *r,
	})
}

func (r *RgRange) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	r.RgRangeSpec.ForEachElement(r, fn)
}

// -------------------------------------------------------------

type RgRangeSpec struct {
	Number1 ASTString
	Number2 ASTString
}

func (r *RgRangeSpec) Text() string {
	return "#RgRangeSpec"
}

func (r *RgRangeSpec) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion RgRangeSpec
	}{
		RuleName:  "RgRangeSpec",
		Expansion: *r,
	})
}

func (r *RgRangeSpec) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r.Number1)
	fn(parent, r.Number2)
}

// -------------------------------------------------------------

type AnyLetter struct {
	Value ASTString
}

func (a *AnyLetter) Text() string {
	return string(a.Value)
}

func (a *AnyLetter) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.Value)
}

func (a *AnyLetter) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, a.Value)
}

// -------------------------------------------------------------

type RgOp struct {
	Value ASTString
}

func (r *RgOp) Text() string {
	return string(r.Value)
}

func (r *RgOp) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName  string
		Expansion RgOp
	}{
		RuleName:  "RgOp",
		Expansion: *r,
	})
}

func (r *RgOp) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r.Value)
}

// ----------------------------------------------------------------

type rgAltValVariant1 struct {
	RgChar *RgChar
}

type rgAltValVariant2 struct {
	Value ASTString
}

type RgAltVal struct {
	variant1 *rgAltValVariant1
	variant2 *rgAltValVariant2
}

func (rc *RgAltVal) Text() string {
	return "#RgAltVal"
}

func (rc *RgAltVal) MarshalJSON() ([]byte, error) {
	if rc.variant1 != nil {
		return json.Marshal(struct {
			RuleName  string
			Expansion rgAltValVariant1
		}{
			RuleName:  "RgAltVal",
			Expansion: *rc.variant1,
		})

	} else if rc.variant2 != nil {
		return json.Marshal(struct {
			RuleName  string
			Expansion rgAltValVariant2
		}{
			RuleName:  "RgAltVal",
			Expansion: *rc.variant2,
		})

	} else {
		return json.Marshal(struct{}{})
	}
}

func (r *RgAltVal) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	if r.variant1 != nil {
		r.variant1.RgChar.ForEachElement(r, fn)

	} else if r.variant2 != nil {
		fn(r, r.variant2.Value)
	}
}
