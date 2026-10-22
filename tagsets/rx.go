// Copyright 2026 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2026 Department of Linguistics,
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

package tagsets

import (
	"fmt"
	"regexp"
	"regexp/syntax"
	"strings"
)

const (
	// maxListedValues limits the number of valid values listed in a message
	maxListedValues = 60

	// maxRxAlternatives limits expansion of regexp alternatives
	// (e.g. `Case=(Nom|Acc)`) when analyzing a pattern
	maxRxAlternatives = 64

	// maxExpandedRepeat limits expansion of exact repetitions (e.g. `.{9}`)
	maxExpandedRepeat = 32
)

// ------------------- linearized regexp -------------------

type atomKind int

const (
	atomRune atomKind = iota

	// atomAnySeq is an arbitrary sequence of characters (`.*`, `.+`)
	atomAnySeq

	// atomOpaque is anything else (a character class, an optional part, ...)
	atomOpaque
)

type rxAtom struct {
	kind atomKind
	r    rune
	fold bool
	src  string

	// class is set for opaque atoms matching exactly one character
	// (`.`, `[NA]`)
	class *syntax.Regexp
}

// isSingleChar tells whether the atom matches exactly one character
func (a rxAtom) isSingleChar() bool {
	return a.kind == atomRune || a.class != nil
}

// matchesRune tests a single-character atom against a character
func (a rxAtom) matchesRune(r rune) bool {
	switch {
	case a.kind == atomRune:
		return a.r == r || a.fold && strings.EqualFold(string(a.r), string(r))
	case a.class == nil:
		return false
	case a.class.Op == syntax.OpAnyChar:
		return true
	case a.class.Op == syntax.OpAnyCharNotNL:
		return r != '\n'
	}
	for i := 0; i+1 < len(a.class.Rune); i += 2 {
		if a.class.Rune[i] <= r && r <= a.class.Rune[i+1] {
			return true
		}
	}
	return false
}

func isAnyChar(re *syntax.Regexp) bool {
	return re.Op == syntax.OpAnyChar || re.Op == syntax.OpAnyCharNotNL
}

// concatAlts joins alternatives of consecutive regexp parts
// (a cartesian product)
func concatAlts(ans, subAlts [][]rxAtom, limit int) ([][]rxAtom, bool) {
	if len(ans)*len(subAlts) > limit {
		return nil, false
	}
	next := make([][]rxAtom, 0, len(ans)*len(subAlts))
	for _, a := range ans {
		for _, b := range subAlts {
			seq := make([]rxAtom, 0, len(a)+len(b))
			seq = append(seq, a...)
			next = append(next, append(seq, b...))
		}
	}
	return next, true
}

// linearize expands a parsed regexp into alternatives, each of them
// being a flat sequence of atoms. In case the number of alternatives
// exceeds the limit, false is returned.
func linearize(re *syntax.Regexp, limit int) ([][]rxAtom, bool) {
	switch re.Op {
	case syntax.OpLiteral:
		seq := make([]rxAtom, len(re.Rune))
		for i, r := range re.Rune {
			seq[i] = rxAtom{kind: atomRune, r: r, fold: re.Flags&syntax.FoldCase != 0}
		}
		return [][]rxAtom{seq}, true

	case syntax.OpEmptyMatch, syntax.OpBeginText, syntax.OpEndText,
		syntax.OpBeginLine, syntax.OpEndLine:
		return [][]rxAtom{{}}, true

	case syntax.OpCapture:
		return linearize(re.Sub[0], limit)

	case syntax.OpConcat:
		ans := [][]rxAtom{{}}
		for _, sub := range re.Sub {
			subAlts, ok := linearize(sub, limit)
			if !ok {
				return nil, false
			}
			if ans, ok = concatAlts(ans, subAlts, limit); !ok {
				return nil, false
			}
		}
		return ans, true

	case syntax.OpAlternate:
		var ans [][]rxAtom
		for _, sub := range re.Sub {
			subAlts, ok := linearize(sub, limit)
			if !ok || len(ans)+len(subAlts) > limit {
				return nil, false
			}
			ans = append(ans, subAlts...)
		}
		return ans, true

	case syntax.OpAnyChar, syntax.OpAnyCharNotNL, syntax.OpCharClass:
		return [][]rxAtom{{{kind: atomOpaque, src: re.String(), class: re}}}, true

	case syntax.OpStar, syntax.OpPlus:
		if isAnyChar(re.Sub[0]) {
			return [][]rxAtom{{{kind: atomAnySeq, src: re.String()}}}, true
		}

	case syntax.OpRepeat:
		if re.Max == -1 && isAnyChar(re.Sub[0]) {
			return [][]rxAtom{{{kind: atomAnySeq, src: re.String()}}}, true
		}
		if re.Min == re.Max && re.Max <= maxExpandedRepeat {
			subAlts, ok := linearize(re.Sub[0], limit)
			if !ok {
				return nil, false
			}
			ans := [][]rxAtom{{}}
			for i := 0; i < re.Min; i++ {
				if ans, ok = concatAlts(ans, subAlts, limit); !ok {
					return nil, false
				}
			}
			return ans, true
		}
	}
	return [][]rxAtom{{{kind: atomOpaque, src: re.String()}}}, true
}

// atomsSource converts atoms back to a regexp source. With quote set
// to false, literal characters are not escaped (for use in messages).
func atomsSource(atoms []rxAtom, quote bool) string {
	var ans strings.Builder
	for _, a := range atoms {
		if a.kind == atomRune {
			if quote {
				ans.WriteString(regexp.QuoteMeta(string(a.r)))
			} else {
				ans.WriteRune(a.r)
			}
		} else if !quote && a.class != nil && isAnyChar(a.class) {
			// Go renders `.` as `(?-s:.)`
			ans.WriteString(".")
		} else {
			ans.WriteString(a.src)
		}
	}
	return ans.String()
}

// parseLinear parses a CQL value (a regexp or a literal string)
// and expands it into linear alternatives
func parseLinear(value string, literal bool) ([][]rxAtom, bool) {
	src := value
	if literal {
		src = regexp.QuoteMeta(value)
	}
	re, err := syntax.Parse(src, syntax.Perl)
	if err != nil {
		// possibly a Manatee-specific syntax
		return nil, false
	}
	return linearize(re, maxRxAlternatives)
}

var plainValueRx = regexp.MustCompile(`^[A-Za-z0-9_:,\-]+$`)

// splitTopLevelAlts splits a regular expression by `|` which are not
// escaped and not nested in a group or a character class.
func splitTopLevelAlts(rx string) []string {
	var ans []string
	var depth int
	var inClass, escaped bool
	start := 0
	for i, c := range rx {
		switch {
		case escaped:
			escaped = false
		case c == '\\':
			escaped = true
		case inClass:
			if c == ']' {
				inClass = false
			}
		case c == '[':
			inClass = true
		case c == '(':
			depth++
		case c == ')':
			depth--
		case c == '|' && depth == 0:
			ans = append(ans, rx[start:i])
			start = i + 1
		}
	}
	return append(ans, rx[start:])
}

// compileWhole compiles a CQL attribute value so it behaves like in
// Manatee, i.e. it must match the whole attribute value.
func compileWhole(value string, literal bool) (*regexp.Regexp, error) {
	if literal {
		value = regexp.QuoteMeta(value)
	}
	return regexp.Compile(`^(?:` + value + `)$`)
}

func listValues(values []string) string {
	if len(values) > maxListedValues {
		return strings.Join(values[:maxListedValues], ", ") + ", ..."
	}
	return strings.Join(values, ", ")
}

func findFold(values []string, v string) string {
	for _, x := range values {
		if strings.EqualFold(x, v) {
			return x
		}
	}
	return ""
}

// closedSet validates attributes with a finite set of possible values
type closedSet struct {

	// what is a name of a value used in messages
	what string

	values []string
	index  map[string]bool

	// lang is non-empty if values are language-specific
	lang string

	// fallback contains values for all languages (used in messages)
	fallback map[string]bool

	// certain means the set is known to be complete so any other value
	// is an error (and not just a warning)
	certain bool

	// multi allows comma-separated combinations of values (UD multivalues)
	multi bool
}

func newClosedSet(what string, values []string, lang string, fallback []string, certain, multi bool) *closedSet {
	ans := &closedSet{
		what:    what,
		values:  values,
		index:   make(map[string]bool),
		lang:    lang,
		certain: certain,
		multi:   multi,
	}
	for _, v := range values {
		ans.index[v] = true
	}
	if lang != "" {
		ans.fallback = make(map[string]bool)
		for _, v := range fallback {
			ans.fallback[v] = true
		}
	}
	return ans
}

func (c *closedSet) isValid(v string) bool {
	if c.index[v] {
		return true
	}
	if c.multi && strings.Contains(v, ",") {
		for _, p := range strings.Split(v, ",") {
			if !c.index[p] {
				return false
			}
		}
		return true
	}
	return false
}

func (c *closedSet) severity() Severity {
	if c.certain {
		return SeverityError
	}
	return SeverityWarning
}

func (c *closedSet) unknownValueIssue(v string) Issue {
	if fv := findFold(c.values, v); fv != "" {
		return Issue{
			Severity: c.severity(),
			Message:  fmt.Sprintf("`%s` is not a valid %s - values are case-sensitive, did you mean `%s`?", v, c.what, fv),
		}
	}
	if c.fallback[v] {
		return Issue{
			Severity: SeverityWarning,
			Message: fmt.Sprintf(
				"`%s` is not a known %s for language `%s` in UD (it is used in other languages)", v, c.what, c.lang),
		}
	}
	return Issue{
		Severity: c.severity(),
		Message:  fmt.Sprintf("`%s` is not a known %s; known values: %s", v, c.what, listValues(c.values)),
	}
}

// validate tests a CQL value. Alternatives consisting of plain values
// are tested one by one so each invalid one is reported. Other
// expressions are tested as a whole for matching at least one valid value.
func (c *closedSet) validate(value string, literal bool) []Issue {
	var issues []Issue
	if literal {
		if !c.isValid(value) {
			issues = append(issues, c.unknownValueIssue(value))
		}
		return issues
	}
	allPlain := true
	for _, p := range splitTopLevelAlts(value) {
		if !plainValueRx.MatchString(p) {
			allPlain = false
			continue
		}
		if !c.isValid(p) {
			issues = append(issues, c.unknownValueIssue(p))
		}
	}
	if allPlain {
		return issues
	}
	rx, err := compileWhole(value, false)
	if err != nil {
		// the regexp syntax may be specific to Manatee, so we cannot say more
		return issues
	}
	for _, v := range c.values {
		if rx.MatchString(v) {
			return issues
		}
	}
	return append(issues, Issue{
		Severity: c.severity(),
		Message: fmt.Sprintf(
			"the pattern matches none of the known values (%s); note that the pattern must match the whole value",
			listValues(c.values)),
	})
}
