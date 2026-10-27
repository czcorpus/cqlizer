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

package dqd

import (
	"fmt"
	"strings"
)

const indentStr = "    "

// negation is the DQD negation prefix we produce. The grammar also accepts
// "not " (with a trailing space), but a bare "NOT" on its own line is lexed
// as a LAYER (i.e. a unit named "NOT"), so "!" is the only safe choice.
const negation = "!"

// String renders q as DQD source text (the indentation-sensitive,
// one-statement-per-line format read by `python -m lcpvian dqd`, as
// opposed to the compiled JSON form in cobquec.auto.json).
//
// The IR is expected to be well-formed (each union wrapper has exactly
// one field set, NOT has exactly one argument). A malformed IR is
// a bug in the translator and String() panics on it.
func (q *Query) String() string {
	r := &renderer{}
	r.query(q)
	return r.sb.String()
}

func invalidIR(format string, args ...any) {
	panic("dqd: invalid IR - " + fmt.Sprintf(format, args...))
}

// renderer writes DQD source line by line, tracking nothing but
// the output - indentation depth is passed explicitly to each method.
type renderer struct {
	sb strings.Builder
}

func (r *renderer) line(depth int, s string) {
	r.sb.WriteString(strings.Repeat(indentStr, depth))
	r.sb.WriteString(s)
	r.sb.WriteString("\n")
}

func (r *renderer) query(q *Query) {
	for _, item := range q.Items {
		r.queryItem(0, item)
	}
	for _, res := range q.Results {
		r.sb.WriteString("\n")
		r.result(res)
	}
}

func (r *renderer) queryItem(depth int, it QueryItem) {
	switch {
	case it.Quantification != nil:
		r.quantification(depth, it.Quantification)
	case it.Sequence != nil:
		r.sequence(depth, it.Sequence)
	case it.Set != nil:
		r.set(depth, it.Set)
	case it.Unit != nil:
		r.unit(depth, it.Unit)
	case it.Constraint != nil:
		r.constraint(depth, *it.Constraint)
	default:
		invalidIR("empty QueryItem")
	}
}

// header joins a unit/sequence/set head, e.g. "Token@s t1" or
// "sequence@s seq 0..3"
func header(keyword string, partOf []PartOf, label string, rep *Repetition) string {
	var sb strings.Builder
	sb.WriteString(keyword)
	for _, p := range partOf {
		sb.WriteString(p.String())
	}
	if label != "" {
		sb.WriteString(" " + label)
	}
	if rep != nil {
		sb.WriteString(" " + rep.Min + ".." + rep.Max)
	}
	return sb.String()
}

func (p PartOf) String() string {
	switch p.Kind {
	case PartOfStream:
		return "@" + p.Label
	case PartOfTime:
		return "%" + p.Label
	case PartOfLocation:
		return "&" + p.Label
	}
	invalidIR("unknown PartOf kind %d", p.Kind)
	return ""
}

func (r *renderer) unit(depth int, u *Unit) {
	r.line(depth, header(u.Layer, u.PartOf, u.Label, nil))
	for _, c := range u.Constraints {
		r.constraint(depth+1, c)
	}
}

func (r *renderer) sequence(depth int, s *Sequence) {
	r.line(depth, header("sequence", s.PartOf, s.Label, s.Repetition))
	r.members(depth+1, s.Members)
}

func (r *renderer) set(depth int, s *Set) {
	r.line(depth, header("set", s.PartOf, s.Label, nil))
	r.members(depth+1, s.Members)
}

func (r *renderer) members(depth int, members []Member) {
	for _, m := range members {
		switch {
		case m.Unit != nil:
			r.unit(depth, m.Unit)
		case m.Sequence != nil:
			r.sequence(depth, m.Sequence)
		case m.LogicalExpression != nil:
			r.logicalExpression(depth, m.LogicalExpression)
		default:
			invalidIR("empty Member")
		}
	}
}

func (r *renderer) constraint(depth int, c Constraint) {
	switch {
	case c.Quantification != nil:
		r.quantification(depth, c.Quantification)
	case c.Unit != nil:
		r.unit(depth, c.Unit)
	case c.Sequence != nil:
		r.sequence(depth, c.Sequence)
	case c.LogicalExpression != nil:
		r.logicalExpression(depth, c.LogicalExpression)
	case c.Comparison != nil:
		r.line(depth, c.Comparison.String())
	default:
		invalidIR("empty Constraint")
	}
}

func (r *renderer) quantification(depth int, q *Quantification) {
	var kw string
	if q.Negated {
		kw = negation
	}
	if q.Universal {
		kw += "ALL"

	} else {
		kw += "EXISTS"
	}
	r.line(depth, kw)
	switch {
	case q.Unit != nil:
		r.unit(depth+1, q.Unit)
	case q.Sequence != nil:
		r.sequence(depth+1, q.Sequence)
	case q.Set != nil:
		r.set(depth+1, q.Set)
	default:
		invalidIR("empty Quantification")
	}
}

func (r *renderer) logicalExpression(depth int, e *LogicalExpression) {
	switch e.Op {
	case LogicalAnd:
		r.line(depth, "AND")
	case LogicalOr:
		r.line(depth, "OR")
	case LogicalNot:
		if len(e.Args) != 1 {
			invalidIR("NOT requires exactly one argument, got %d", len(e.Args))
		}
		r.line(depth, negation)
	default:
		invalidIR("unknown LogicalOp %d", e.Op)
	}
	for _, arg := range e.Args {
		r.constraint(depth+1, arg)
	}
}

func (c *Comparison) String() string {
	cmp := c.Comparator
	if c.Not {
		cmp = negation + cmp
	}
	return c.Left.render() + " " + cmp + " " + c.Right.render()
}

// render returns the source form of the operand. (It cannot be String()
// because Operand already has a field of that name.)
func (o Operand) render() string {
	switch {
	case o.Function != nil:
		return o.Function.String()
	case o.Reference != nil:
		return o.Reference.Value
	case o.Attribute != nil:
		return o.Attribute.Label + "." + o.Attribute.Name
	case o.String != nil:
		// Note: DQD strings have no escaping - the value must not contain
		// a double quote or a newline.
		return `"` + o.String.Value + `"`
	case o.Regex != nil:
		ans := "/" + o.Regex.Pattern + "/"
		if o.Regex.CaseInsensitive {
			ans += "i"
		}
		return ans
	case o.Math != nil:
		return o.Math.String()
	}
	invalidIR("empty Operand")
	return ""
}

func (f *Function) String() string {
	args := make([]string, len(f.Args))
	for i, a := range f.Args {
		args[i] = a.render()
	}
	return f.Name + "(" + strings.Join(args, ", ") + ")"
}

func (m *Math) String() string {
	if m.Operation != nil {
		// always parenthesized so nested operations keep their structure
		return "(" + m.Operation.Left.render() + " " + m.Operation.Operator +
			" " + m.Operation.Right.render() + ")"
	}
	return m.Number
}

func (r *renderer) result(res Result) {
	switch {
	case res.Plain != nil:
		r.line(0, res.Label+" => plain")
		r.line(1, "context")
		for _, v := range res.Plain.Context {
			r.line(2, v)
		}
		r.line(1, "entities")
		for _, v := range res.Plain.Entities {
			r.line(2, v)
		}
	case res.Analysis != nil:
		r.line(0, res.Label+" => analysis")
		r.line(1, "attributes")
		for _, v := range res.Analysis.Attributes {
			r.line(2, v.render())
		}
		r.line(1, "functions")
		for _, v := range res.Analysis.Functions {
			r.line(2, v)
		}
		if len(res.Analysis.Filter) > 0 {
			r.line(1, "filter")
			for _, v := range res.Analysis.Filter {
				r.line(2, v.String())
			}
		}
	case res.Collocation != nil:
		col := res.Collocation
		r.line(0, res.Label+" => collocation")
		if col.Space != "" {
			r.line(1, "space")
			r.line(2, col.Space)

		} else {
			if col.Window == nil {
				invalidIR("collocation with center requires a window")
			}
			r.line(1, "center")
			r.line(2, col.Center)
			r.line(1, "window")
			r.line(2, col.Window.Left+".."+col.Window.Right)
		}
		r.line(1, "attribute")
		r.line(2, col.Attribute)
	default:
		invalidIR("empty Result")
	}
}
