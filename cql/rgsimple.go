// Copyright 2025 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2025 Department of Linguistics,
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
)

type RgSimple struct {
	// RgRange / RgChar / RgAlt / RgPosixClass
	origValue string
	Values    []ASTNode
}

func (r *RgSimple) String() string {
	return r.origValue
}

// CQL returns the matched regexp fragment unchanged - no normalization
// is applied within regular expressions.
func (r *RgSimple) CQL() string {
	return r.origValue
}

func (r *RgSimple) WildcardScore() float64 {
	ans := 0.0
	r.ForEachElement(r, func(parent, item ASTNode) {
		switch tItem := item.(type) {
		case *RgChar:
			if tItem.String() == "?" {
				ans += 1
			}
		}
	})
	ans += float64(strings.Count(r.String(), ".*")) * 20
	ans += float64(strings.Count(r.String(), ".+")) * 20
	return ans
}

// -------------------------------------------------

type RgGrouped struct {
	Values []*RegExpRaw // the stuff here is A|B|C...
}

func (r *RgGrouped) String() string {
	return "#RgGrouped"
}

// CQL renders "(alt1|alt2|...)". In practice this is never reached from
// Query.CQL() - RegExp.CQL() returns its origValue directly rather than
// descending into RegExpRaw/RgGrouped - but it's implemented properly
// (rather than delegating to origValue) since, unlike RgSimple, this
// node has no origValue of its own to fall back on.
func (r *RgGrouped) CQL() string {
	var ans strings.Builder
	ans.WriteString("(")
	for i, v := range r.Values {
		if i > 0 {
			ans.WriteString("|")
		}
		ans.WriteString(v.CQL())
	}
	ans.WriteString(")")
	return ans.String()
}

func (r *RgGrouped) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	for _, v := range r.Values {
		v.ForEachElement(r, fn)
	}
}

func (r *RgGrouped) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(r) {
		leave(r)
		return
	}
	for _, v := range r.Values {
		v.DFS(enter, leave)
	}
	leave(r)
}

// ----------------------------------------------------

type RgPosixClass struct {
	Value ASTString
}

func (r *RgPosixClass) String() string {
	return "RgPosixClass"
}

// CQL is never actually reached: the grammar's RgPosixClass rule
// returns a plain string that's folded into an ASTString by the caller,
// so this struct type is never instantiated by the parser. Value is
// rendered for interface completeness in case that ever changes.
func (r *RgPosixClass) CQL() string {
	return string(r.Value)
}

func (r *RgPosixClass) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r.Value)
}

func (r *RgPosixClass) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	dfsLeaf(enter, leave, r.Value)
}

// ----------------------------------------------------

type RgLook struct {
	Value ASTString
}

func (r *RgLook) String() string {
	return "#RgLook"
}

// CQL is lossy: the parser action for RgLook (lookahead/lookbehind
// assertions) discards the matched operator and inner regexp instead of
// storing them (see the TODO in the grammar), leaving Value empty. This
// is unreached from Query.CQL() in practice since RegExp.CQL() returns
// origValue directly rather than descending this far.
func (r *RgLook) CQL() string {
	return r.Value.CQL()
}

func (r *RgLook) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r.Value)
}

func (r *RgLook) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	dfsLeaf(enter, leave, r.Value)
}

// ----------------------------------------------------

type RgLookOperator struct {
}

// -----------------------------------------------------

type RgAlt struct {
	Values []*RgAltVal
	Not    bool
}

func (r *RgAlt) NumItems() int {
	return len(r.Values)
}

func (r *RgAlt) Score() float64 {
	var ans float64
	for _, v := range r.Values {
		ans += v.SrchScore()
	}
	if r.Not {
		ans *= 5 // rough estimate
	}
	return ans
}

func (r *RgAlt) String() string {
	var ans strings.Builder
	for i, v := range r.Values {
		if i > 0 {
			ans.WriteString(", ")
		}
		ans.WriteString(v.String())
	}
	return fmt.Sprintf("#RgAlt(%s)", ans.String())
}

// CQL renders "[^abc]" / "[abc]" - a bracketed regexp character
// alternative, negated if Not is set.
func (r *RgAlt) CQL() string {
	var ans strings.Builder
	ans.WriteString("[")
	if r.Not {
		ans.WriteString("^")
	}
	for _, v := range r.Values {
		ans.WriteString(v.CQL())
	}
	ans.WriteString("]")
	return ans.String()
}

func (r *RgAlt) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	for _, item := range r.Values {
		item.ForEachElement(r, fn)
	}
}

func (r *RgAlt) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(r) {
		leave(r)
		return
	}
	for _, item := range r.Values {
		item.DFS(enter, leave)
	}
	leave(r)
}

// --------------------------------------------------------

type rgCharVariant1 struct {
	Value          ASTString
	IsUnicodeClass bool
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
	variant1 *rgCharVariant1
	variant2 *rgCharVariant2
	variant3 *rgCharVariant3
	variant4 *rgCharVariant4
	variant5 *rgCharVariant5
}

func (rc *RgChar) IsUnicodeClass() bool {
	if rc.variant1 != nil {
		return rc.variant1.IsUnicodeClass
	}
	return false
}

func (rc *RgChar) Info() string {
	if rc.variant1 != nil {
		return fmt.Sprintf("#RgChar[%s]", rc.variant1.Value.String())

	} else if rc.variant2 != nil {
		return fmt.Sprintf("#RgChar[%s]", rc.variant2.RgOp.Value.String())

	} else if rc.variant3 != nil {
		return fmt.Sprintf("#RgChar[%s]", rc.variant3.RgRepeat.Value.String())

	} else if rc.variant4 != nil {
		return fmt.Sprintf("#RgChar[%s]", rc.variant4.RgAny.Value.String())

	} else if rc.variant5 != nil {
		return fmt.Sprintf("#RgChar[%s]", rc.variant5.RgQM.Value.String())
	}
	return "#RgChar(_unknown_)"
}

func (rc *RgChar) String() string {
	if rc.variant1 != nil {
		return rc.variant1.Value.String()

	} else if rc.variant2 != nil {
		return rc.variant2.RgOp.Value.String()

	} else if rc.variant3 != nil {
		return rc.variant3.RgRepeat.Value.String()

	} else if rc.variant4 != nil {
		return rc.variant4.RgAny.Value.String()

	} else if rc.variant5 != nil {
		return rc.variant5.RgQM.Value.String()
	}
	return ""
}

func (rc *RgChar) CQL() string {
	if rc.variant1 != nil {
		return rc.variant1.Value.CQL()

	} else if rc.variant2 != nil {
		return rc.variant2.RgOp.CQL()

	} else if rc.variant3 != nil {
		return rc.variant3.RgRepeat.CQL()

	} else if rc.variant4 != nil {
		return rc.variant4.RgAny.CQL()

	} else if rc.variant5 != nil {
		return rc.variant5.RgQM.CQL()
	}
	return ""
}

func (rc *RgChar) IsRgOperator(v string) bool {
	return rc.variant2 != nil && rc.variant2.RgOp.Value.String() == v
}

func (rc *RgChar) IsConstant() bool {
	return rc.variant1 != nil
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

func (r *RgChar) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(r) {
		leave(r)
		return
	}
	if r.variant1 != nil {
		dfsLeaf(enter, leave, r.variant1.Value)

	} else if r.variant2 != nil {
		r.variant2.RgOp.DFS(enter, leave)
	} else if r.variant3 != nil {
		r.variant3.RgRepeat.DFS(enter, leave)

	} else if r.variant4 != nil {
		r.variant4.RgAny.DFS(enter, leave)

	} else if r.variant5 != nil {
		r.variant5.RgQM.DFS(enter, leave)
	}
	leave(r)
}

// -----------------------------------------------------------

type RgRepeat struct {
	effect float64
	Value  ASTString
}

func (rr *RgRepeat) String() string {
	return rr.Value.String()
}

func (rr *RgRepeat) CQL() string {
	return rr.Value.CQL()
}

func (rr *RgRepeat) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, rr.Value)
}

func (rr *RgRepeat) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	dfsLeaf(enter, leave, rr.Value)
}

// -----------------------------------------------------------

type RgQM struct {
	effect float64
	Value  ASTString
}

func (rr *RgQM) String() string {
	return rr.Value.String()
}

func (rr *RgQM) CQL() string {
	return rr.Value.CQL()
}

func (rr *RgQM) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, rr.Value)
}

func (rr *RgQM) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	dfsLeaf(enter, leave, rr.Value)
}

// -----------------------------------------------------------

type RgAny struct {
	effect float64
	Value  ASTString
}

func (rr *RgAny) String() string {
	return rr.Value.String()
}

func (rr *RgAny) CQL() string {
	return rr.Value.CQL()
}

func (rr *RgAny) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, rr.Value)
}

func (rr *RgAny) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	dfsLeaf(enter, leave, rr.Value)
}

// -----------------------------------------------------------

type RgRange struct {
	RgRangeSpec *RgRangeSpec
}

func (r *RgRange) String() string {
	if r.RgRangeSpec != nil {
		return r.RgRangeSpec.String()
	}
	return "RgRange{?, ?}"
}

// CQL renders "{n}" / "{n,}" / "{n,m}".
func (r *RgRange) CQL() string {
	if r.RgRangeSpec != nil {
		return "{" + r.RgRangeSpec.CQL() + "}"
	}
	return "{}"
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

func (r *RgRange) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(r) {
		leave(r)
		return
	}
	r.RgRangeSpec.DFS(enter, leave)
	leave(r)
}

// -------------------------------------------------------------

type RgRangeSpec struct {
	origValue string
	Number1   ASTString
	Number2   ASTString
}

func (r *RgRangeSpec) String() string {
	return r.origValue
}

// CQL renders "n,m" / "n,". origValue is only set when the grammar's
// COMMA-bearing alternative matched (see grammar.peg); the bare-NUMBER
// alternative leaves it empty, which is when Number1 alone is rendered.
func (r *RgRangeSpec) CQL() string {
	if r.origValue != "" {
		return r.origValue
	}
	return r.Number1.CQL()
}

func (r *RgRangeSpec) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	fn(parent, r.Number1)
	fn(parent, r.Number2)
}

func (r *RgRangeSpec) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(r) {
		leave(r)
		return
	}
	dfsLeaf(enter, leave, r.Number1)
	dfsLeaf(enter, leave, r.Number2)
	leave(r)
}

// -------------------------------------------------------------

type AnyLetter struct {
	Value ASTString
}

func (a *AnyLetter) String() string {
	return string(a.Value)
}

func (a *AnyLetter) CQL() string {
	return string(a.Value)
}

func (a *AnyLetter) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, a.Value)
}

func (a *AnyLetter) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	dfsLeaf(enter, leave, a.Value)
}

// -------------------------------------------------------------

type RgOp struct {
	Value ASTString
}

func (r *RgOp) String() string {
	return string(r.Value)
}

func (r *RgOp) CQL() string {
	return string(r.Value)
}

func (r *RgOp) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r.Value)
}

func (r *RgOp) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	dfsLeaf(enter, leave, r.Value)
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
	variant1 *rgAltValVariant1
	variant2 *rgAltValVariant2
	variant3 *rgAltValVariant3
}

func (rc *RgAltVal) SrchScore() float64 {
	if rc.variant1 != nil {
		textLen := float64(len(rc.variant1.RgChar.String()))
		if rc.variant1.RgChar.IsUnicodeClass() {
			textLen *= 20
		}
		return textLen
	}
	if rc.variant2 != nil {
		return float64(len(rc.variant2.Value.String()))
	}
	if rc.variant3 != nil {
		return float64(len(rc.variant3.From)) * 10 // TODO this is just a rough estimate
	}
	return 0
}

func (rc *RgAltVal) String() string {
	return "#RgAltVal"
}

// CQL renders a single char, a literal "-", or a "from-to" range,
// depending on which grammar alternative matched.
func (rc *RgAltVal) CQL() string {
	if rc.variant1 != nil {
		return rc.variant1.RgChar.CQL()
	}
	if rc.variant2 != nil {
		return rc.variant2.Value.CQL()
	}
	if rc.variant3 != nil {
		return rc.variant3.From.CQL() + "-" + rc.variant3.To.CQL()
	}
	return ""
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

func (r *RgAltVal) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(r) {
		leave(r)
		return
	}
	if r.variant1 != nil {
		r.variant1.RgChar.DFS(enter, leave)

	} else if r.variant2 != nil {
		dfsLeaf(enter, leave, r.variant2.Value)

	} else if r.variant3 != nil {
		dfsLeaf(enter, leave, r.variant3.From)
		dfsLeaf(enter, leave, r.variant3.To)
	}
	leave(r)
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

func (r *RgSimple) DFS(enter TTEnterFunc, leave TTLeaveFunc) {
	if !enter(r) {
		leave(r)
		return
	}
	for _, item := range r.Values {
		switch tItem := item.(type) {
		case *RgRange:
			tItem.DFS(enter, leave)
		case *RgChar:
			tItem.DFS(enter, leave)
		case *RgAlt:
			tItem.DFS(enter, leave)
		case *RgPosixClass:
			tItem.DFS(enter, leave)
		}
	}
	leave(r)
}
