package cql

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

type state int

const (
	ConstChar state = iota
	Repeat
	QMark

	RangePenalty             = 1.2
	RangeInfReplac           = 10.0
	SingleCharExhaustiveness = 10.0
	NextCharPenaltyDrop      = 0.9
	DotPenalty               = 1.2
	SingleDotExhaustiveness  = 20.0
	InfRepeatPenalty         = 5.0
	QuestionMarkPenalty      = 1.05
	AltCharPenalty           = 1.1
)

type RgSimple struct {
	// RgRange / RgChar / RgAlt / RgPosixClass
	origValue string
	Values    []ASTNode
}

func (r *RgSimple) Text() string {
	return r.origValue
}

func (r *RgSimple) Normalize() string {
	var ans strings.Builder
	for _, v := range r.Values {
		ans.WriteString(v.Normalize())
	}
	return ans.String()
}

// ExpensiveOps
// TODO consider whether the .* etc are at the beginning or end
// as it matters in index search
func (r *RgSimple) ExhaustionScore() float64 {
	var state state
	var ans float64
	for _, val := range r.Values {
		switch state {
		case ConstChar:
			switch tVal := val.(type) {
			case *RgChar:
				if tVal.IsConstant() {
					if ans == 0 {
						ans = SingleCharExhaustiveness

					} else {
						ans *= NextCharPenaltyDrop
					}

				} else if tVal.IsRgOperator(".") {
					if ans == 0 {
						ans = SingleDotExhaustiveness

					} else {
						ans *= DotPenalty
					}

				} else if tVal.IsRgOperator("?") {
					state = QMark
					ans *= QuestionMarkPenalty

				} else if tVal.IsRgOperator("+") || tVal.IsRgOperator("*") {
					state = Repeat
					ans *= InfRepeatPenalty
				}
			case *RgRange:
				v := tVal.NumericRepr()
				if v[0] > -1 {
					if v[1] > 0 {
						ans *= RangePenalty * float64(v[1]-v[0])

					} else {
						ans *= RangePenalty * float64(RangeInfReplac-v[0])
					}
				}
			case *RgAlt:
				if ans == 0 {
					ans = tVal.ExhaustionScore()

				} else {
					ans += tVal.ExhaustionScore() // TODO what about the adding operation?
				}

			case *RgPosixClass:
				// currently NOP
			default:
				log.Error().Type("inputType", val).Msg("Rg parsing error in state ConstChar")
			}
		case Repeat:
			switch tVal := val.(type) {
			case *RgChar:
				if tVal.IsConstant() {
					ans *= NextCharPenaltyDrop
					state = ConstChar

				} else if tVal.IsRgOperator(".") {
					ans *= DotPenalty
					state = ConstChar
				}
			case *RgAlt, *RgPosixClass:
				// currently NOP
			default:
				log.Error().Type("inputType", val).Msg("Rg parsing error in state Repeat")
			}
		case QMark:
			switch tVal := val.(type) {
			case *RgChar:
				if tVal.IsConstant() {
					ans *= NextCharPenaltyDrop
					state = ConstChar

				} else if tVal.IsRgOperator(".") {
					ans *= DotPenalty
					state = ConstChar
				}
			case *RgAlt, *RgPosixClass:
				// currently NOP
			default:
				log.Error().Type("inputType", val).Msg("Rg parsing error in state QMark")
			}
		}
	}
	return ans
}

func (r *RgSimple) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName        string
		Expansion       RgSimple
		ExhaustionScore float64
	}{
		RuleName:        "RgSimple",
		Expansion:       *r,
		ExhaustionScore: r.ExhaustionScore(),
	})
}

// -------------------------------------------------

type RgGrouped struct {
	Values []*RegExpRaw // the stuff here is A|B|C...
}

func (r *RgGrouped) Text() string {
	return "#RgGrouped"
}

func (r *RgGrouped) Normalize() string {
	var ans strings.Builder
	for i, v := range r.Values {
		if i > 0 {
			ans.WriteString("<OR>" + v.Normalize())

		} else {
			ans.WriteString(v.Normalize())
		}
	}
	return ans.String()
}

func (r *RgGrouped) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName        string
		Expansion       RgGrouped
		ExhaustionScore float64
	}{
		RuleName:        "RgGrouped",
		Expansion:       *r,
		ExhaustionScore: r.ExhaustionScore(),
	})
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

func (r *RgGrouped) ExhaustionScore() float64 {
	var ans float64
	for _, v := range r.Values {
		ans += v.ExhaustionScore()
	}
	return ans
}

// ---------------------------------------------------------

type RgSimpleProps struct {
	Ops        []string
	Constansts []string
	Alts       []int
}

func (p RgSimpleProps) ContainsWildcards() bool {
	for _, v := range p.Ops {
		if v == ".+" || v == ".*" {
			return true
		}
	}
	return false
}

// ----------------------------------------------------

type RgPosixClass struct {
	Value ASTString
}

func (r *RgPosixClass) Text() string {
	return r.Value.Normalize()
}

func (r *RgPosixClass) Normalize() string {
	return "#RgPosixClass"
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

func (r *RgLook) Normalize() string {
	return r.Value.Normalize()
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

func (r *RgAlt) Text() string {
	return "#RgAlt"
}

func (r *RgAlt) Normalize() string {
	var ans strings.Builder
	if r.Not {
		ans.WriteString("(rgalt <NEGATION> ")

	} else {
		ans.WriteString("(rgalt ")
	}
	for _, v := range r.Values {
		ans.WriteString(v.Normalize())
	}
	ans.WriteString(")")
	return ans.String()
}

func (r *RgAlt) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RuleName        string
		Expansion       RgAlt
		ExhaustionScore float64
	}{
		RuleName:        "RgAlt",
		Expansion:       *r,
		ExhaustionScore: r.ExhaustionScore(),
	})
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

func (r *RgAlt) ExhaustionScore() float64 {
	ans := 0.0
	for _, v := range r.Values {
		ans += v.ExhaustionScore()
	}
	return ans
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
	variant1 *rgCharVariant1
	variant2 *rgCharVariant2
	variant3 *rgCharVariant3
	variant4 *rgCharVariant4
	variant5 *rgCharVariant5
}

func (rc *RgChar) Text() string {
	return "#RgChar"
}

func (rc *RgChar) Normalize() string {
	if rc.variant1 != nil {
		return rc.variant1.Value.Normalize()
	}
	if rc.variant2 != nil {
		return rc.variant2.RgOp.Normalize()
	}
	if rc.variant3 != nil {
		return rc.variant3.RgRepeat.Normalize()

	} else if rc.variant4 != nil {
		return rc.variant4.RgAny.Normalize()

	} else if rc.variant5 != nil {
		return rc.variant5.RgQM.Normalize()
	}
	return ""
}

func (rc *RgChar) IsRgOperator(v string) bool {
	return rc.variant2 != nil && rc.variant2.RgOp.Value.String() == v
}

func (rc *RgChar) IsConstant() bool {
	return rc.variant1 != nil
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
	}{
		RuleName:  "RgChar",
		Expansion: variant,
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

func (rr *RgRepeat) Normalize() string {
	return "<REPEAT>"
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

func (rr *RgQM) Normalize() string {
	return rr.Value.Normalize()
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

func (rr *RgAny) Normalize() string {
	return "<ANY>"
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

func (r *RgRange) Normalize() string {
	return fmt.Sprintf("rgrange(%s)", r.RgRangeSpec.Normalize())
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

func (r *RgRangeSpec) Normalize() string {
	return fmt.Sprintf("%s, %s", r.Number1.Normalize(), r.Number2.Normalize())
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

func (a *AnyLetter) Normalize() string {
	return a.Value.Normalize()
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

func (r *RgOp) Normalize() string {
	return "x" // TODO
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

func (rc *RgAltVal) Text() string {
	var ans strings.Builder
	if rc.variant1 != nil {
		ans.WriteString(rc.variant1.RgChar.Normalize())

	} else if rc.variant2 != nil {
		ans.WriteString(rc.variant2.Value.Normalize())

	} else if rc.variant3 != nil {
		ans.WriteString(rc.variant3.From.Normalize() + ", " + rc.variant3.To.Normalize())
	}
	return ans.String()
}

func (rc *RgAltVal) Normalize() string {
	var ans strings.Builder
	if rc.variant1 != nil {
		return rc.variant1.RgChar.Normalize()
	}
	if rc.variant2 != nil {
		return rc.variant2.Value.Normalize()
	}
	if rc.variant3 != nil {
		return fmt.Sprintf("(chrng %s-%s)", rc.variant3.From.Normalize(), rc.variant3.To.Normalize())
	}
	return ans.String()
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
		RuleName        string
		Expansion       any
		ExhaustionScore float64
	}{
		RuleName:        "RgAltVal",
		Expansion:       variant,
		ExhaustionScore: rc.ExhaustionScore(),
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

func (r *RgAltVal) ExhaustionScore() float64 {
	if r.variant1 != nil {
		return 2.0 // TODO
	}
	if r.variant2 != nil {
		return 2
	}
	if r.variant3 != nil {
		ch1 := []rune(r.variant3.From.String())
		ch2 := []rune(r.variant3.To.String())
		return float64(ch2[0]-ch1[0]+1) * 1.05
	}
	return 0
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
