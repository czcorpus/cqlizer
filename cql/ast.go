package cql

import "encoding/json"

// Seq (_ BINOR _ Seq)* / Seq
type Sequence struct {
	Seq []*Seq
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

func (q *Sequence) ForEachElement(fn func(v any)) {
	fn(q)
	for _, item := range q.Seq {
		item.ForEachElement(fn)
	}
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

func (q *GlobPart) ForEachElement(fn func(v any)) {
	fn(q)
	for _, item := range q.GlobCond {
		item.ForEachElement(fn)
	}
}

// ---------------------------------------

// WithinOrContaining
//
//	NOT? (KW_WITHIN / KW_CONTAINING) _ WithinContainingPart {
type WithinOrContaining struct {
	KwWithin             string
	KwContaining         string
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

func (w *WithinOrContaining) ForEachElement(fn func(v any)) {
	fn(w)
	fn(w.KwWithin)
	fn(w.KwContaining)
	w.WithinContainingPart.ForEachElement(fn)
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

func (wcp *WithinContainingPart) ForEachElement(fn func(v any)) {
	fn(wcp)
	if wcp.variant1 != nil {
		wcp.variant1.Sequence.ForEachElement(fn)

	} else if wcp.variant2 != nil {
		fn(wcp.variant2.WithinNumber.Value)

	} else if wcp.variant3 != nil {
		wcp.variant3.AlignedPart.ForEachElement(fn)
	}
}

// --------------------------------------------------

// GlobCond
//
// v1: NUMBER DOT AttName _ NOT? EQ _ NUMBER DOT AttName {
//
// v2: KW_FREQ LPAREN _ NUMBER DOT AttName _ RPAREN NOT? _ ( EQ / LEQ / GEQ / LSTRUCT / RSTRUCT ) _ NUMBER {

type globCondVariant1 struct {
	Number1  string
	AttName3 string
	Not4     string
	Eq5      string
	Number6  string
	AttName8 string
}

type globCondVariant2 struct {
	KwFreq1   string
	Number2   string
	AttName3  string
	Not4      string
	Operator5 string
	Number6   string
}

type GlobCond struct {
	variant1 *globCondVariant1

	variant2 *globCondVariant2
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

func (gc *GlobCond) ForEachElement(fn func(v any)) {

}

// ----------------------------------------------------

// Structure
//
// AttName _ AttValList?
type Structure struct {
	AttName    string
	AttValList *AttValList
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

func (s *Structure) ForEachElement(fn func(v any)) {

}

// ---------------------------------------------------------

// AttValList
//
//	av1:AttValAnd av2:(_ BINOR _ AttValAnd)*
type AttValList struct {
	AttValAnd []*AttValAnd
}

func (a *AttValList) ForEachElement(fn func(v any)) {

}

// -----------------------------------------------------------

// NumberedPosition
//
// NUMBER COLON OnePosition
type NumberedPosition struct {
	Number      string
	Colon       string
	OnePosition *OnePosition
}

func (n *NumberedPosition) ForEachElement(fn func(v any)) {

}

// --------------------------------------------------

type onePositionVariant1 struct {
	AttValList *AttValList
}

type onePositionVariant2 struct {
	RegExp *RegExp
}

type onePositionVariant3 struct {
	Number string
	RegExp *RegExp
}

type onePositionVariant4 struct {
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

func (op *OnePosition) ForEachElement(fn func(v any)) {

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

func (p *Position) MarshalJSON() ([]byte, error) {
	if p.variant1 != nil {
		return json.Marshal(p.variant1)

	} else if p.variant2 != nil {
		return json.Marshal(p.variant2)

	} else {
		return json.Marshal(struct{}{})
	}
}

func (p *Position) ForEachElement(fn func(v any)) {

}

// -------------------------------------------------------

type RegExp struct {
	RegExpRaw *RegExpRaw
}

func (r *RegExp) ForEachElement(fn func(v any)) {

}

// --------------------------------------------------------

type MuPart struct {
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

// --------------------------------------------------------------------

type Seq struct {
	Not        bool
	Repetition []*Repetition
}

func (s *Seq) ForEachElement(fn func(v any)) {

}

// ------------------------------------------------------------

type repetitionVariant1 struct {
	RepOpt    string
	AtomQuery *AtomQuery
}

type repetitionVariant2 struct {
	OpenStructTag *OpenStructTag
}

type repetitionVariant3 struct {
	CloseStructTag *CloseStructTag
}

type Repetition struct {
	variant1 *repetitionVariant1
	variant2 *repetitionVariant2
	variant3 *repetitionVariant3
}

func (r *Repetition) MarshalJSON() ([]byte, error) {
	if r.variant1 != nil {
		return json.Marshal(r.variant1)

	} else if r.variant2 != nil {
		return json.Marshal(r.variant2)

	} else if r.variant3 != nil {
		return json.Marshal(r.variant3)

	} else {
		return json.Marshal(struct{}{})
	}
}

func (r *Repetition) ForEachElement(fn func(v any)) {

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
	variant1 *atomQueryVariant1
	variant2 *atomQueryVariant2
}

func (aq *AtomQuery) MarshalJSON() ([]byte, error) {
	if aq.variant1 != nil {
		return json.Marshal(aq.variant1)

	} else if aq.variant2 != nil {
		return json.Marshal(aq.variant1)

	} else {
		return json.Marshal(struct{}{})
	}
}

func (aq *AtomQuery) ForEachElement(fn func(v any)) {

}

// --------------------------------------------------------------

type RepOpt struct {
}

// ----------------------------------------------------------------

type OpenStructTag struct {
	Structure *Structure
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

func (ost *OpenStructTag) ForEachElement(fn func(v any)) {

}

// --------------------------------------------------------------

type CloseStructTag struct {
	Structure *Structure
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

func (ost *CloseStructTag) ForEachElement(fn func(v any)) {

}

// ---------------------------------------------------------

type AlignedPart struct {
}

func (a *AlignedPart) ForEachElement(fn func(v any)) {

}

// -----------------------------------------------------------

// AttValAnd
//
//	av1:AttVal av2:(_ BINAND _ AttVal)*
type AttValAnd struct {
	AttVal []*AttVal
}

func (a *AttValAnd) ForEachElement(fn func(v any)) {

}

// --------------------------------------------------------------

// AttName _ (NOT)? EEQ _ RawString
type attValVariant1 struct {
	AttName   string
	Not       bool
	Eeq       string
	RawString *RawString
}

// AttName (_ NOT)? _ (EQ / LEQ / GEQ / TEQ NUMBER?) _ RegExp
type attValVariant2 struct {
	AttName string
	Not     bool
	Op      string
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

func (a *AttVal) ForEachElement(fn func(v any)) {

}

// ---------------------------------------------------

type WithinNumber struct {
	Value string
}

// ----------------------------------------------------------

type RegExpRaw struct {

	// RgLook / RgGrouped / RgSimple
	Values []any
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

func (r *RegExpRaw) ForEachElement(fn func(v any)) {

}

// ------------------------------------------------------------------

type RawString struct {
	SimpleString *SimpleString
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

func (r *RawString) ForEachElement(fn func(v any)) {

}

// ------------------------------------------------------------------------

type SimpleString struct {
	Values []string
}

func (r *SimpleString) ForEachElement(fn func(v any)) {

}

// -------------------------------------------------

type RgGrouped struct {
	Value *RegExpRaw
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

func (r *RgGrouped) ForEachElement(fn func(v any)) {

}

// ---------------------------------------------------------

type RgSimple struct {
	// RgRange / RgChar / RgAlt / RgPosixClass
	Values []any
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

func (r *RgSimple) ForEachElement(fn func(v any)) {

}

// ----------------------------------------------------

type RgPosixClass struct {
}

// ----------------------------------------------------

type RgLook struct {
}

// ----------------------------------------------------

type RgLookOperator struct {
}

// -----------------------------------------------------

type RgAlt struct {
	Values []*RgAltVal
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

func (r *RgAlt) ForEachElement(fn func(v any)) {

}

// --------------------------------------------------------

type rgCharVariant1 struct {
	Value string
}

type rgCharVariant2 struct {
	Value *RgOp
}

type RgChar struct {
	variant1 *rgCharVariant1
	variant2 *rgCharVariant2
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

func (r *RgChar) ForEachElement(fn func(v any)) {

}

// -----------------------------------------------------------

type RgRange struct {
	RgRangeSpec *RgRangeSpec
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

func (r *RgRange) ForEachElement(fn func(v any)) {

}

// -------------------------------------------------------------

type RgRangeSpec struct {
	Number1 string
	Number2 string
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

func (r *RgRangeSpec) ForEachElement(fn func(v any)) {

}

// -------------------------------------------------------------

type AnyLetter struct {
}

type RgOp struct {
	Value string
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

func (r *RgOp) ForEachElement(fn func(v any)) {

}

// ----------------------------------------------------------------

type rgAltValVariant1 struct {
	RgChar *RgChar
}

type rgAltValVariant2 struct {
	Value string
}

type RgAltVal struct {
	variant1 *rgAltValVariant1
	variant2 *rgAltValVariant2
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

func (r *RgAltVal) ForEachElement(fn func(v any)) {

}
