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
	"slices"
	"strings"
)

// maxListedTagValues limits the number of described values in a message
const maxListedTagValues = 40

// uposToPositional maps UD parts of speech to the first position
// of the Czech positional tags (approximately, see the Alpha project)
var uposToPositional = map[string]string{
	"NOUN": "N", "PROPN": "N", "ADJ": "A", "PRON": "P", "DET": "P", "NUM": "C",
	"VERB": "V", "AUX": "V", "ADV": "D", "ADP": "R", "CCONJ": "J^", "SCONJ": "J,",
	"PART": "T", "INTJ": "I", "PUNCT": "Z", "X": "X", "SYM": "X",
}

// positionalScheme validates values of positional tags (e.g. `tag`,
// `p_tag`, `ep_tag` in CNC corpora). A pattern is considered valid
// if it matches at least one tag found in the reference data.
// Otherwise, the reason is searched for and explained.
type positionalScheme struct {
	ts *positionalTagset
}

func (s *positionalScheme) ID() string {
	return s.ts.ident
}

func (s *positionalScheme) anyTagMatches(rx *regexp.Regexp) bool {
	for _, tag := range s.ts.tags {
		if rx.MatchString(tag) {
			return true
		}
	}
	return false
}

func (s *positionalScheme) Validate(v Value) []Issue {
	value, literal := v.Pattern, v.Literal
	paths, ok := parseLinear(value, literal)
	if !ok {
		// we cannot analyze the pattern structure, so we can only test it as a whole
		rx, err := compileWhole(value, literal)
		if err != nil || s.anyTagMatches(rx) {
			return nil
		}
		return []Issue{s.noMatchIssue()}
	}
	var issues []Issue
	seen := make(map[string]bool)
	for _, p := range paths {
		if iss, ok := s.validatePath(p); ok && !seen[iss.Message] {
			seen[iss.Message] = true
			issues = append(issues, iss)
		}
	}
	if len(issues) > 0 && !literal {
		if letter, ok := uposToPositional[strings.TrimSuffix(value, ".*")]; ok {
			issues = append(issues, Issue{
				Severity: SeverityHint,
				Message: fmt.Sprintf(
					"`%s` looks like a UD part of speech, but this attribute uses the positional tagset %s "+
						"(the closest equivalent is `%s.*`)",
					value, s.ts.ident, letter),
			})
		}
	}
	return issues
}

func (s *positionalScheme) noMatchIssue() Issue {
	return Issue{
		Severity: SeverityWarning,
		Message: fmt.Sprintf(
			"the pattern matches none of %d tags of the %s tagset found in reference corpora (%s)",
			len(s.ts.tags), s.ts.ident, strings.Join(s.ts.sources, ", ")),
	}
}

// fixedPrefix returns the leading single-character atoms of a path, i.e.
// the part of the pattern where positions can be determined. The `complete`
// flag tells whether the prefix covers the whole path.
func fixedPrefix(path []rxAtom) (prefix []rxAtom, complete bool) {
	for i, a := range path {
		if !a.isSingleChar() {
			return path[:i], false
		}
	}
	return path, true
}

func describeAtom(a rxAtom) string {
	if a.kind == atomRune {
		return "`" + string(a.r) + "`"
	}
	return "`" + a.src + "`"
}

// listTagValues describes provided values of a position
func (s *positionalScheme) listTagValues(pos int, values []rune) string {
	tp := s.ts.positions[pos]
	items := make([]string, 0, len(values))
	for i, v := range values {
		if i == maxListedTagValues {
			items = append(items, "...")
			break
		}
		if d := tp.describe(v); d != "" {
			items = append(items, fmt.Sprintf("`%c` (%s)", v, d))

		} else {
			items = append(items, fmt.Sprintf("`%c`", v))
		}
	}
	return strings.Join(items, ", ")
}

func (s *positionalScheme) posLabel(pos int) string {
	return fmt.Sprintf("position %d (%s)", pos+1, s.ts.positions[pos].label)
}

// validatePath tests a single linear alternative of a pattern. In case
// the alternative matches no tag, the most specific explanation found is
// returned.
func (s *positionalScheme) validatePath(path []rxAtom) (Issue, bool) {
	rx, err := regexp.Compile("^(?:" + atomsSource(path, true) + ")$")
	if err != nil {
		return Issue{}, false
	}
	if s.anyTagMatches(rx) {
		return s.checkFloatingPositions(path)
	}
	numPos := s.ts.numPos
	prefix, complete := fixedPrefix(path)

	// 1) length
	if complete && len(prefix) != numPos || len(prefix) > numPos {
		msg := fmt.Sprintf(
			"tags of the %s tagset have exactly %d positions and the pattern must match the whole tag, "+
				"but the pattern describes %d positions",
			s.ts.ident, numPos, len(prefix))
		if len(prefix) < numPos {
			msg += fmt.Sprintf("; add `.*` to the end (e.g. `%s.*`)", atomsSource(prefix, false))
		}
		return Issue{Severity: SeverityError, Message: msg}, true
	}

	// 2) values impossible at a position
	for i, a := range prefix {
		tp := s.ts.positions[i]
		if slices.ContainsFunc(tp.used, a.matchesRune) {
			continue
		}
		msg := fmt.Sprintf("%s cannot be %s", s.posLabel(i), describeAtom(a))
		if d, ok := tp.values[a.r]; ok && a.kind == atomRune {
			msg = fmt.Sprintf(
				"%s: `%c` (%s) is defined by the tagset, but it does not occur in the corpus data",
				s.posLabel(i), a.r, d)
		}
		msg += "; possible values: " + s.listTagValues(i, tp.used)
		return Issue{Severity: SeverityError, Message: msg}, true
	}

	// 3) values incompatible with preceding positions
	cands := s.ts.tags
	for i, a := range prefix {
		var next []string
		for _, tag := range cands {
			if a.matchesRune(rune(tag[i])) {
				next = append(next, tag)
			}
		}
		if len(next) > 0 {
			cands = next
			continue
		}
		possible := make(map[rune]bool)
		for _, tag := range cands {
			possible[rune(tag[i])] = true
		}
		values := make([]rune, 0, len(possible))
		for r := range possible {
			values = append(values, r)
		}
		slices.Sort(values)
		// the first two positions (part of speech and its subtype) are
		// fully covered by reference data, other combinations are
		// considered less certain
		sev := SeverityWarning
		if i <= 1 {
			sev = SeverityError
		}
		return Issue{
			Severity: sev,
			Message: fmt.Sprintf(
				"no tag has %s at %s together with the preceding positions `%s`; possible values there: %s",
				describeAtom(a), s.posLabel(i), atomsSource(prefix[:i], false), s.listTagValues(i, values)),
		}, true
	}
	return s.noMatchIssue(), true
}

// checkFloatingPositions warns about conditions surrounded by `.*`
// (e.g. `V.*N.*`) as this makes them position-independent. Conditions
// after the last `.*` are fine as they are anchored to the tag end
// (e.g. `V.*I` for aspect at the last position).
func (s *positionalScheme) checkFloatingPositions(path []rxAtom) (Issue, bool) {
	anySeen, floating := false, false
	for _, a := range path {
		if a.kind != atomAnySeq {
			floating = floating || anySeen
			continue
		}
		if floating {
			return Issue{
				Severity: SeverityWarning,
				Message: "`.*` followed by more characters makes their positions arbitrary " +
					"(e.g. `V.*N.*` matches `N` at any position after `V`, not only negation); " +
					"use `.` for each skipped position instead (e.g. `V.........N.*`)",
			}, true
		}
		anySeen = true
	}
	return Issue{}, false
}
