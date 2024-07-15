package cql

import (
	"encoding/json"
	"fmt"

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
	Values    []any
}

func (r *RgSimple) Text() string {
	return r.origValue
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

func (r *RgSimple) GetWildcards() RgSimpleProps {
	var state int
	ans := RgSimpleProps{
		Ops:        make([]string, 0, 5),
		Constansts: make([]string, 0, 20),
	}
	for _, val := range r.Values {
		switch tVal := val.(type) {
		case *RgChar:
			if tVal.variant2 != nil {
				if tVal.variant2.Value.Value == "." {
					if state == 0 {
						state = 1

					} else if state == 1 {
						ans.Ops = append(ans.Ops, ".")

					} else if state == 2 {
						state = 1
					}

				} else if tVal.variant2.Value.Value == "+" || tVal.variant2.Value.Value == "*" {
					if state == 1 {
						ans.Ops = append(
							ans.Ops, fmt.Sprintf(".%s", tVal.variant2.Value.Value))
						state = 2
					}

				} else if tVal.variant2.Value.Value == "|" {
					if state == 0 {
						ans.Ops = append(ans.Ops, "|")

					} else if state == 1 {
						ans.Ops = append(ans.Ops, ".")
						ans.Ops = append(ans.Ops, "|")
						state = 0

					} else if state == 2 {
						ans.Ops = append(ans.Ops, "|")
						state = 0
					}
				}

			} else if tVal.variant1 != nil {
				ans.Constansts = append(ans.Constansts, tVal.variant1.Value.Text())
			}
		case *RgAlt:
			ans.Alts = append(ans.Alts, len(tVal.Values))
		}
	}
	if state == 1 {
		ans.Ops = append(ans.Ops, ".")
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
