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
	"sort"
	"strings"
	"unicode"
)

var (
	// unescapedLayerRx matches e.g. `Gender[psor]=` where `[psor]` is
	// (most likely unintentionally) a character class
	unescapedLayerRx = regexp.MustCompile(`([A-Za-z]+)\[([a-z]+)\]=`)

	trivialFeatsRx = regexp.MustCompile(`^(\.\*|\.\+|_|)$`)
)

// ------------------- feature occurrences -------------------

type boundary int

const (
	boundaryStart boundary = iota
	boundaryEnd
	boundaryAny
	boundarySep
	boundaryOther
)

// featOcc is a `Name=Value` found in a linearized pattern
type featOcc struct {
	name   string
	value  []rxAtom
	before boundary
	after  boundary
	fold   bool
}

func isNameRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '[' || r == ']'
}

func isSep(a rxAtom) bool {
	return a.kind == atomRune && a.r == '|'
}

func extractFeats(seq []rxAtom) []featOcc {
	var ans []featOcc
	for k, a := range seq {
		if a.kind != atomRune || a.r != '=' {
			continue
		}
		j := k
		for j > 0 && seq[j-1].kind == atomRune && isNameRune(seq[j-1].r) {
			j--
		}
		if j == k {
			continue
		}
		occ := featOcc{}
		var name strings.Builder
		for _, x := range seq[j:k] {
			name.WriteRune(x.r)
			occ.fold = occ.fold || x.fold
		}
		occ.name = name.String()
		switch {
		case j == 0:
			occ.before = boundaryStart
		case seq[j-1].kind == atomAnySeq:
			occ.before = boundaryAny
		case isSep(seq[j-1]):
			occ.before = boundarySep
		default:
			occ.before = boundaryOther
		}
		m := k + 1
		for m < len(seq) && seq[m].kind != atomAnySeq && !isSep(seq[m]) {
			occ.fold = occ.fold || seq[m].fold
			m++
		}
		occ.value = seq[k+1 : m]
		switch {
		case m == len(seq):
			occ.after = boundaryEnd
		case seq[m].kind == atomAnySeq:
			occ.after = boundaryAny
		default:
			occ.after = boundarySep
		}
		ans = append(ans, occ)
	}
	return ans
}

// ------------------- the scheme -------------------

type udFeats struct {
	lang string
}

func newUDFeats(lang string) *udFeats {
	return &udFeats{lang: lang}
}

func (s *udFeats) ID() string {
	return SchemeUDFeats
}

// separateConds suggests a CQL expression testing each feature
// in a separate condition (for multivalue feats attributes)
func separateConds(occs []featOcc) string {
	conds := make([]string, len(occs))
	for i, occ := range occs {
		val := atomsSource(occ.value, false)
		if val == "" {
			val = ".*"
		}
		conds[i] = fmt.Sprintf(`feats="%s=%s"`, strings.ReplaceAll(occ.name, "[", "\\["), val)
		conds[i] = strings.ReplaceAll(conds[i], "]=", "\\]=")
	}
	return strings.Join(conds, " & ")
}

func hasSepAtom(seq []rxAtom) bool {
	for _, a := range seq {
		if isSep(a) {
			return true
		}
	}
	return false
}

func (s *udFeats) Validate(v Value) []Issue {
	value, literal := v.Pattern, v.Literal
	// with `|` as a multivalue separator, Manatee matches patterns also
	// against individual features
	perFeature := v.MultiSep == "|"
	if !literal {
		if m := unescapedLayerRx.FindStringSubmatch(value); m != nil {
			return []Issue{{
				Severity: SeverityError,
				Message: fmt.Sprintf(
					"brackets in a layered feature name must be escaped (`%s\\[%s\\]=`), otherwise `[%s]` is a character class",
					m[1], m[2], m[2]),
			}}
		}
	}
	alts, ok := parseLinear(value, literal)
	if !ok {
		return nil
	}
	var issues []Issue
	seen := make(map[string]bool)
	add := func(iss Issue) {
		if !seen[iss.Message] {
			seen[iss.Message] = true
			issues = append(issues, iss)
		}
	}
	var anyFeat bool
	singleFeatPaths := true
	featNames := make(map[string]bool)
	for _, seq := range alts {
		occs := extractFeats(seq)
		if len(occs) == 0 {
			singleFeatPaths = false
			continue
		}
		anyFeat = true
		s.checkOrder(occs, perFeature, add)
		for _, occ := range occs {
			s.checkFeature(occ, add)
			featNames[occ.name] = true
		}
		// a pattern spanning more features can match only the whole value
		wholeValue := !perFeature || len(occs) > 1 || hasSepAtom(seq)
		if wholeValue {
			singleFeatPaths = false
		}
		if !literal && wholeValue {
			s.checkAnchoring(value, occs, perFeature, add)
		}
	}
	if perFeature && !literal && singleFeatPaths && len(featNames) > 1 &&
		len(splitTopLevelAlts(value)) > 1 {
		add(Issue{
			Severity: SeverityHint,
			Message: "the unescaped `|` is a regexp alternative, so the pattern matches tokens having " +
				"any of the features; to require all of them, use separate conditions " +
				"(e.g. `feats=\"Case=Acc\" & feats=\"Number=Plur\"`)",
		})
	}
	if !anyFeat && !trivialFeatsRx.MatchString(value) && !strings.Contains(value, "=") {
		add(Issue{
			Severity: SeverityWarning,
			Message: "values of the feats attribute have the form `Name=Value|Name=Value...` " +
				"(e.g. `Case=Acc|Number=Plur`), but the pattern contains no `Name=Value` pair",
		})
	}
	return issues
}

// checkOrder tests whether features are required in the order
// they appear in feats values (sorted case-insensitively by name)
func (s *udFeats) checkOrder(occs []featOcc, perFeature bool, add func(Issue)) {
	alternative := ""
	if perFeature {
		alternative = "; alternatively, use separate conditions which do not depend on the order: `" +
			separateConds(occs) + "`"
	}
	for i := 1; i < len(occs); i++ {
		a, b := strings.ToLower(occs[i-1].name), strings.ToLower(occs[i].name)
		if a == b {
			add(Issue{
				Severity: SeverityError,
				Message: fmt.Sprintf(
					"feature %s is required twice in a sequence, but each feature occurs at most once "+
						"in a feats value (multiple values are written as e.g. `Gender=Fem,Neut`), "+
						"so the pattern never matches",
					occs[i].name),
			})
			// no alternative suggested here, as `feats="Case=Acc" & feats="Case=Gen"`
			// would not match either

		} else if a > b {
			names := make([]string, len(occs))
			for j, occ := range occs {
				names[j] = occ.name
			}
			sort.Slice(names, func(x, y int) bool {
				return strings.ToLower(names[x]) < strings.ToLower(names[y])
			})
			add(Issue{
				Severity: SeverityError,
				Message: fmt.Sprintf(
					"features in feats values are sorted alphabetically (case-insensitive), so %s "+
						"cannot follow %s and the pattern never matches; required order: %s%s",
					occs[i].name, occs[i-1].name, strings.Join(names, ", "), alternative),
			})
		}
	}
}

func (s *udFeats) checkFeature(occ featOcc, add func(Issue)) {
	if occ.before == boundaryOther {
		// the name is probably only partially literal
		return
	}
	data := udData()
	name := occ.name
	if occ.fold {
		if fn := findFold(sortedKeys(data.allFeats), name); fn != "" {
			name = fn
		}
	}
	values, langSpecific, found := data.featValues(s.lang, name)
	if !found {
		if allVals, ok := data.allFeats[name]; ok {
			add(Issue{
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("feature %s is not used for language `%s` in UD", name, s.lang),
			})
			values, langSpecific = allVals, false

		} else if fn := findFold(sortedKeys(data.allFeats), name); fn != "" {
			add(Issue{
				Severity: SeverityError,
				Message:  fmt.Sprintf("`%s` is not a valid UD feature - names are case-sensitive, did you mean `%s`?", name, fn),
			})
			return

		} else {
			add(Issue{
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("`%s` is not a known UD feature", name),
			})
			return
		}
	}
	if len(occ.value) == 0 || s.valueMatches(occ, values) {
		return
	}
	where := ""
	if langSpecific {
		where = fmt.Sprintf(" for language `%s`", s.lang)
	}
	valueSrc := atomsSource(occ.value, false)
	if lit, ok := atomsLiteral(occ.value); ok {
		if fv := findFold(values, lit); fv != "" {
			add(Issue{
				Severity: SeverityError,
				Message: fmt.Sprintf(
					"`%s` is not a valid value of feature %s - values are case-sensitive, did you mean `%s`?",
					lit, name, fv),
			})
			return
		}
	}
	add(Issue{
		Severity: SeverityWarning,
		Message: fmt.Sprintf(
			"`%s` matches no known value of feature %s%s; known values: %s",
			valueSrc, name, where, listValues(values)),
	})
}

// atomsLiteral returns a value as a string in case it consists
// of literal characters only
func atomsLiteral(atoms []rxAtom) (string, bool) {
	var ans strings.Builder
	for _, a := range atoms {
		if a.kind != atomRune {
			return "", false
		}
		ans.WriteRune(a.r)
	}
	return ans.String(), true
}

// valueMatches tests whether a value pattern can match at least one
// of known values. Multivalues (`Fem,Neut`) are supported. In case
// the value is followed by `.*`, it is treated as a prefix.
func (s *udFeats) valueMatches(occ featOcc, values []string) bool {
	prefix := occ.after == boundaryAny
	if lit, ok := atomsLiteral(occ.value); ok {
		parts := strings.Split(lit, ",")
		for i, p := range parts {
			isPrefix := prefix && i == len(parts)-1
			var found bool
			for _, v := range values {
				if isPrefix && (strings.HasPrefix(v, p) || occ.fold && strings.HasPrefix(strings.ToLower(v), strings.ToLower(p))) ||
					!isPrefix && (v == p || occ.fold && strings.EqualFold(v, p)) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}
	src := "^(?:" + atomsSource(occ.value, true) + ")"
	if !prefix {
		src += "$"
	}
	if occ.fold {
		src = "(?i)" + src
	}
	rx, err := regexp.Compile(src)
	if err != nil {
		return true
	}
	for _, v := range values {
		if rx.MatchString(v) {
			return true
		}
	}
	return false
}

// checkAnchoring warns about patterns which (probably unintentionally)
// require a feature to be the first or the last one in a feats value
//
// For multivalue attributes (perFeature), this is called only for patterns
// spanning more features as single-feature patterns match individual features.
func (s *udFeats) checkAnchoring(value string, occs []featOcc, perFeature bool, add func(Issue)) {
	first, last := occs[0], occs[len(occs)-1]
	startAnchored := first.before == boundaryStart
	endAnchored := last.after == boundaryEnd
	var msg string
	switch {
	case startAnchored && endAnchored:
		msg = "the pattern must match the whole feats value; as it neither starts nor ends with `.*`, " +
			"it matches only tokens having exactly these features and no other ones"
		if !perFeature {
			msg += " (use e.g. `.*Case=Acc.*` to search for a single feature)"
		}
	case startAnchored:
		msg = fmt.Sprintf(
			"the pattern must match the whole feats value; as it does not start with `.*`, "+
				"it matches only tokens whose first (alphabetically) feature is %s", first.name)
	case endAnchored:
		msg = fmt.Sprintf(
			"the pattern must match the whole feats value; as it does not end with `.*`, "+
				"it matches only tokens whose last (alphabetically) feature is %s", last.name)
	default:
		return
	}
	if perFeature {
		msg += "; to require the features regardless of other ones, use separate conditions: `" +
			separateConds(occs) + "`"

	} else if len(splitTopLevelAlts(value)) > 1 {
		msg += "; also note that an unescaped `|` is a regexp alternative - the separator of features must be written as `\\|`"
	}
	add(Issue{Severity: SeverityWarning, Message: msg})
}
