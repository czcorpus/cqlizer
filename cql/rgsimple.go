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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type RgSimple struct {
	// RgRange / RgChar / RgAlt / RgPosixClass
	origValue string
	Values    []ASTNode
}

func (r *RgSimple) Text() string {
	return r.origValue
}

func (r *RgSimple) WildcardScore() float64 {
	ans := 0.0
	r.ForEachElement(r, func(parent, item ASTNode) {
		switch tItem := item.(type) {
		case *RgChar:
			if tItem.Text() == "?" {
				ans += 1
			}
		}
	})
	ans += float64(strings.Count(r.Text(), ".*")) * 20
	ans += float64(strings.Count(r.Text(), ".+")) * 20
	return ans
}

// -------------------------------------------------

type RgGrouped struct {
	Values []*RegExpRaw // the stuff here is A|B|C...
}

func (r *RgGrouped) Text() string {
	return "#RgGrouped"
}

func (r *RgGrouped) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	for _, v := range r.Values {
		v.ForEachElement(r, fn)
	}
}

func (r *RgGrouped) DFS(fn func(v ASTNode)) {
	for _, v := range r.Values {
		v.DFS(fn)
	}
	fn(r)
}

// ----------------------------------------------------

type RgPosixClass struct {
	Value ASTString
}

func (r *RgPosixClass) Text() string {
	return "RgPosixClass"
}

func (r *RgPosixClass) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Value)
}

func (r *RgPosixClass) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r.Value)
}

func (r *RgPosixClass) DFS(fn func(v ASTNode)) {
	fn(r.Value)
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

func (r *RgLook) DFS(fn func(v ASTNode)) {
	fn(r.Value)
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

func (r *RgAlt) Text() string {
	var ans strings.Builder
	for i, v := range r.Values {
		if i > 0 {
			ans.WriteString(", ")
		}
		ans.WriteString(v.Text())
	}
	return fmt.Sprintf("#RgAlt(%s)", ans.String())
}

func (r *RgAlt) ForEachElement(parent ASTNode, fn func(parent, v ASTNode)) {
	fn(parent, r)
	for _, item := range r.Values {
		item.ForEachElement(r, fn)
	}
}

func (r *RgAlt) DFS(fn func(v ASTNode)) {
	for _, item := range r.Values {
		item.DFS(fn)
	}
	fn(r)
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

func (rc *RgChar) Text() string {
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

func (r *RgChar) DFS(fn func(v ASTNode)) {
	if r.variant1 != nil {
		fn(r.variant1.Value)

	} else if r.variant2 != nil {
		r.variant2.RgOp.DFS(fn)
	} else if r.variant3 != nil {
		r.variant3.RgRepeat.DFS(fn)

	} else if r.variant4 != nil {
		r.variant4.RgAny.DFS(fn)

	} else if r.variant5 != nil {
		r.variant5.RgQM.DFS(fn)
	}
	fn(r)
}

// -----------------------------------------------------------

type RgRepeat struct {
	effect float64
	Value  ASTString
}

func (rr *RgRepeat) Text() string {
	return rr.Value.String()
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

func (rr *RgRepeat) DFS(fn func(ASTNode)) {
	fn(rr.Value)
}

// -----------------------------------------------------------

type RgQM struct {
	effect float64
	Value  ASTString
}

func (rr *RgQM) Text() string {
	return rr.Value.String()
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

func (rr *RgQM) DFS(fn func(ASTNode)) {
	fn(rr.Value)
}

// -----------------------------------------------------------

type RgAny struct {
	effect float64
	Value  ASTString
}

func (rr *RgAny) Text() string {
	return rr.Value.String()
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

func (rr *RgAny) DFS(fn func(ASTNode)) {
	fn(rr.Value)
}

// -----------------------------------------------------------

type RgRange struct {
	RgRangeSpec *RgRangeSpec
}

func (r *RgRange) Text() string {
	if r.RgRangeSpec != nil {
		return r.RgRangeSpec.Text()
	}
	return "RgRange{?, ?}"
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

func (r *RgRange) DFS(fn func(v ASTNode)) {
	r.RgRangeSpec.DFS(fn)
	fn(r)
}

// -------------------------------------------------------------

type RgRangeSpec struct {
	origValue string
	Number1   ASTString
	Number2   ASTString
}

func (r *RgRangeSpec) Text() string {
	return r.origValue
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
	fn(parent, r)
	fn(parent, r.Number1)
	fn(parent, r.Number2)
}

func (r *RgRangeSpec) DFS(fn func(v ASTNode)) {
	fn(r.Number1)
	fn(r.Number2)
	fn(r)
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

func (a *AnyLetter) DFS(fn func(v ASTNode)) {
	fn(a.Value)
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

func (r *RgOp) DFS(fn func(v ASTNode)) {
	fn(r.Value)
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
		textLen := float64(len(rc.variant1.RgChar.Text()))
		if rc.variant1.RgChar.IsUnicodeClass() {
			textLen *= 20
		}
		return textLen
	}
	if rc.variant2 != nil {
		return float64(len(rc.variant2.Value.Text()))
	}
	if rc.variant3 != nil {
		return float64(len(rc.variant3.From)) * 10 // TODO this is just a rough estimate
	}
	return 0
}

func (rc *RgAltVal) Text() string {
	return "#RgAltVal"
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

func (r *RgAltVal) DFS(fn func(v ASTNode)) {
	if r.variant1 != nil {
		r.variant1.RgChar.DFS(fn)

	} else if r.variant2 != nil {
		fn(r.variant2.Value)

	} else if r.variant3 != nil {
		fn(r.variant3.From)
		fn(r.variant3.To)
	}
	fn(r)
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

func (r *RgSimple) DFS(fn func(v ASTNode)) {
	for _, item := range r.Values {
		switch tItem := item.(type) {
		case *RgRange:
			tItem.DFS(fn)
		case *RgChar:
			tItem.DFS(fn)
		case *RgAlt:
			tItem.DFS(fn)
		case *RgPosixClass:
			tItem.DFS(fn)
		}
	}
	fn(r)
}
