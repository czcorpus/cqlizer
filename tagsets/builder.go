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
	"sort"
	"strings"
)

// Morphology is a tagset-neutral description of a token
// in terms of Universal Dependencies
type Morphology struct {

	// UPOS contains alternative parts of speech (e.g. NOUN, PROPN)
	UPOS []string

	// Feats maps feature names to alternative values
	Feats map[string][]string
}

// ParseMorphology parses UPOS values separated by `|` and features in the
// form `Name=Value|Name=Value2,Value3` where comma-separated values are
// alternatives
func ParseMorphology(upos, feats string) (Morphology, error) {
	ans := Morphology{Feats: make(map[string][]string)}
	for _, u := range strings.Split(upos, "|") {
		if u = strings.TrimSpace(u); u != "" {
			ans.UPOS = append(ans.UPOS, strings.ToUpper(u))
		}
	}
	for _, f := range strings.Split(feats, "|") {
		if f = strings.TrimSpace(f); f == "" {
			continue
		}
		name, vals, ok := strings.Cut(f, "=")
		if !ok || name == "" || vals == "" {
			return ans, fmt.Errorf("invalid feature `%s`, expected `Name=Value`", f)
		}
		for _, v := range strings.Split(vals, ",") {
			if v = strings.TrimSpace(v); v != "" && !slices.Contains(ans.Feats[name], v) {
				ans.Feats[name] = append(ans.Feats[name], v)
			}
		}
	}
	if len(ans.UPOS) == 0 && len(ans.Feats) == 0 {
		return ans, fmt.Errorf("no part of speech nor features specified")
	}
	return ans, nil
}

func (m Morphology) has(name, value string) bool {
	return slices.Contains(m.Feats[name], value)
}

// BuildResult is a CQL condition built from a Morphology
type BuildResult struct {

	// CQL is a condition usable inside a CQL position, e.g. `tag="N...4.*"`
	CQL string

	// Issues contain notes on approximations and untranslatable features
	// as well as validation issues of the result
	Issues []Issue
}

// BuildCondition translates a Morphology into a CQL condition for a corpus.
// In case attr is empty, the main morphological attribute(s) of the corpus
// are used (`tag` with a positional tagset or UD `upos` + `feats`).
func BuildCondition(m Morphology, attr string, cc CorpusContext) (BuildResult, error) {
	if attr != "" {
		scheme, err := SchemeFor(attr, cc)
		if err != nil {
			return BuildResult{}, err
		}
		switch s := scheme.(type) {
		case *positionalScheme:
			return buildPositional(m, attr, s, cc), nil
		case *udUPOS:
			return buildUD(m, attr, "", cc), nil
		case *udFeats:
			return buildUD(m, "", attr, cc), nil
		case nil:
			return BuildResult{}, fmt.Errorf("attribute %s has no known tagset", attr)
		default:
			return BuildResult{}, fmt.Errorf(
				"building conditions for attribute %s (%s) is not supported", attr, scheme.ID())
		}
	}
	if s, _ := SchemeFor("tag", cc); s != nil && cc.hasAttr("tag") {
		if ps, ok := s.(*positionalScheme); ok {
			return buildPositional(m, "tag", ps, cc), nil
		}
	}
	uposAttr, featsAttr := "", ""
	if s, _ := SchemeFor("upos", cc); s != nil && cc.hasAttr("upos") {
		uposAttr = "upos"
	}
	if s, _ := SchemeFor("feats", cc); s != nil && cc.hasAttr("feats") {
		featsAttr = "feats"
	}
	if uposAttr != "" || featsAttr != "" {
		return buildUD(m, uposAttr, featsAttr, cc), nil
	}
	return BuildResult{}, fmt.Errorf(
		"the corpus has no attribute with a known tagset (for positional tags, a tagset must be configured)")
}

func issuesOf(attr, value string, issues []Issue) []Issue {
	for i := range issues {
		issues[i].Attr = attr
		issues[i].Value = value
	}
	return issues
}

// ------------------- UD -------------------

func buildUD(m Morphology, uposAttr, featsAttr string, cc CorpusContext) BuildResult {
	var ans BuildResult
	var conds []string
	note := func(sev Severity, msg string) {
		ans.Issues = append(ans.Issues, Issue{Severity: sev, Message: msg})
	}
	validate := func(attr, value string) {
		if issues, err := ValidateValue(attr, value, false, cc); err == nil {
			ans.Issues = append(ans.Issues, issues...)
		}
	}
	if len(m.UPOS) > 0 {
		if uposAttr == "" {
			note(SeverityWarning, "part of speech skipped - the target attribute does not contain UPOS")

		} else {
			value := strings.Join(m.UPOS, "|")
			conds = append(conds, fmt.Sprintf(`%s="%s"`, uposAttr, value))
			validate(uposAttr, value)
		}
	}
	if len(m.Feats) > 0 {
		if featsAttr == "" {
			note(SeverityWarning, "features skipped - the target attribute does not contain UD features")

		} else {
			names := sortedKeys(m.Feats)
			sort.SliceStable(names, func(i, j int) bool {
				return strings.ToLower(names[i]) < strings.ToLower(names[j])
			})
			items := make([]string, len(names))
			for i, name := range names {
				vals := m.Feats[name]
				v := vals[0]
				if len(vals) > 1 {
					v = "(" + strings.Join(vals, "|") + ")"
				}
				items[i] = regexp.QuoteMeta(name) + "=" + v
			}
			if cc.MultiValueSeps[featsAttr] == "|" {
				// order-independent and matching individual features
				for _, item := range items {
					conds = append(conds, fmt.Sprintf(`%s="%s"`, featsAttr, item))
					validate(featsAttr, item)
				}

			} else {
				value := ".*" + strings.Join(items, ".*") + ".*"
				conds = append(conds, fmt.Sprintf(`%s="%s"`, featsAttr, value))
				validate(featsAttr, value)
			}
		}
	}
	ans.CQL = strings.Join(conds, " & ")
	return ans
}

// ------------------- positional tags -------------------

// position labels as used in the tagset XML files
const (
	lblPOS      = "Part of speech"
	lblSubPOS   = "Detailed part of speech"
	lblGender   = "Gender"
	lblNumber   = "Number"
	lblCase     = "Case"
	lblPGender  = "Possessor's gender"
	lblPNumber  = "Possessor's number"
	lblPerson   = "Person"
	lblTense    = "Tense"
	lblDegree   = "Degree of comparison"
	lblNegation = "Negation"
	lblVoice    = "Active/passive voice"
	lblAspect   = "Aspect"
	lblVariant  = "Variant, style indication etc."
	lblVTKind   = "Verb kind"
	lblVTMood   = "Mood"
	lblVTVoice  = "Voice"
)

type featTarget struct {
	labels []string
	values map[string]string
}

// simpleFeatTargets map UD features to positions identified by labels
// (the first label found in a tagset is used)
var simpleFeatTargets = map[string]featTarget{
	"Case":         {[]string{lblCase}, map[string]string{"Nom": "1", "Gen": "2", "Dat": "3", "Acc": "4", "Voc": "5", "Loc": "6", "Ins": "7"}},
	"Number":       {[]string{lblNumber}, map[string]string{"Sing": "S", "Plur": "P", "Dual": "D"}},
	"Gender[psor]": {[]string{lblPGender}, map[string]string{"Masc": "M", "Fem": "F"}},
	"Number[psor]": {[]string{lblPNumber}, map[string]string{"Sing": "S", "Plur": "P"}},
	"Person":       {[]string{lblPerson}, map[string]string{"1": "1", "2": "2", "3": "3"}},
	"Tense":        {[]string{lblTense}, map[string]string{"Pres": "P", "Past": "R", "Fut": "F", "Pqp": "Q"}},
	"Degree":       {[]string{lblDegree}, map[string]string{"Pos": "1", "Cmp": "2", "Sup": "3"}},
	"Polarity":     {[]string{lblNegation}, map[string]string{"Pos": "A", "Neg": "N"}},
	"Voice":        {[]string{lblVoice, lblVTVoice}, map[string]string{"Act": "A", "Pass": "P"}},
	"Aspect":       {[]string{lblAspect}, map[string]string{"Perf": "P", "Imp": "I", "Both": "B"}},
	"Polite":       {[]string{lblNumber}, map[string]string{"Form": "v"}},
}

var uposToCNC = map[string]string{
	"NOUN": "N", "PROPN": "N", "ADJ": "A", "PRON": "P", "DET": "P", "NUM": "C",
	"VERB": "V", "AUX": "V", "ADV": "D", "ADP": "R", "CCONJ": "J", "SCONJ": "J",
	"PART": "T", "INTJ": "I", "PUNCT": "Z", "X": "X", "SYM": "X",
}

var uposApprox = map[string]string{
	"PROPN": "positional tags do not distinguish proper nouns (`N` covers all nouns)",
	"DET":   "positional tags have no determiners (`P` covers all pronouns)",
	"AUX":   "positional tags do not distinguish auxiliary verbs (`V` covers all verbs; see the `verbtag` attribute)",
	"SYM":   "positional tags have no symbols (`X` is used for unknown words)",
}

// uposApproxPartner is a UPOS whose CNC counterpart covers an approximated one
var uposApproxPartner = map[string]string{
	"PROPN": "NOUN", "DET": "PRON", "AUX": "VERB", "SYM": "X",
}

// subPOSFeats map features to values of the detailed part of speech;
// multiple features intersect (e.g. PronType=Prs + Reflex=Yes)
var subPOSFeats = map[string]map[string]string{
	// in UD Czech, reflexive and possessive pronouns are Prs too
	"PronType": {"Prs": "PH5678S", "Dem": "D", "Ind": "Z", "Neg": "W", "Tot": "L", "Rel": "J149E", "Int": "4KQ"},
	"Reflex":   {"Yes": "678"},
	"Poss":     {"Yes": "S81U"},
	"NumType":  {"Card": "lnz", "Ord": "r", "Mult": "uv", "Frac": "y", "Sets": "dj"},
	"NumForm":  {"Digit": "=", "Roman": "}"},
	"AdpType":  {"Voc": "V", "Prep": "RV"},
}

// tagBuilder collects constraints on positions of a positional tag
type tagBuilder struct {
	ts     *positionalTagset
	sets   [][]rune // nil = any value
	issues []Issue
}

func (b *tagBuilder) note(sev Severity, format string, args ...any) {
	b.issues = append(b.issues, Issue{Severity: sev, Message: fmt.Sprintf(format, args...)})
}

func (b *tagBuilder) posIndex(labels ...string) int {
	for _, l := range labels {
		for i, p := range b.ts.positions {
			if p.label == l {
				return i
			}
		}
	}
	return -1
}

// constrain restricts a position to provided values (filtered by values
// used in the tagset). It returns false if the position does not exist.
func (b *tagBuilder) constrain(what string, chars string, labels ...string) bool {
	idx := b.posIndex(labels...)
	if idx < 0 {
		return false
	}
	var vals []rune
	for _, c := range chars {
		if slices.Contains(b.ts.positions[idx].used, c) && !slices.Contains(vals, c) {
			vals = append(vals, c)
		}
	}
	if len(vals) == 0 {
		b.note(SeverityWarning, "%s cannot be expressed in the %s tagset (position %d does not use `%s`) - skipped",
			what, b.ts.ident, idx+1, chars)
		return true
	}
	if b.sets[idx] != nil {
		var common []rune
		for _, c := range vals {
			if slices.Contains(b.sets[idx], c) {
				common = append(common, c)
			}
		}
		if len(common) == 0 {
			b.note(SeverityWarning, "%s conflicts with other constraints at position %d - skipped", what, idx+1)
			return true
		}
		vals = common
	}
	slices.Sort(vals)
	b.sets[idx] = vals
	return true
}

// constrainOrNote is like constrain but notes features without a target position
func (b *tagBuilder) constrainOrNote(what string, chars string, labels ...string) {
	if !b.constrain(what, chars, labels...) {
		b.note(SeverityWarning, "%s has no counterpart in the %s tagset - skipped", what, b.ts.ident)
	}
}

func escapeTagChar(c rune, inClass bool) string {
	if inClass {
		if strings.ContainsRune(`]\^-[`, c) {
			return `\` + string(c)
		}
		return string(c)
	}
	return regexp.QuoteMeta(string(c))
}

func (b *tagBuilder) export() string {
	last := -1
	for i, s := range b.sets {
		if s != nil {
			last = i
		}
	}
	var ans strings.Builder
	for i := 0; i <= last; i++ {
		switch s := b.sets[i]; {
		case s == nil:
			ans.WriteString(".")
		case len(s) == 1:
			ans.WriteString(escapeTagChar(s[0], false))
		default:
			ans.WriteString("[")
			for _, c := range s {
				ans.WriteString(escapeTagChar(c, true))
			}
			ans.WriteString("]")
		}
	}
	if last < len(b.sets)-1 {
		ans.WriteString(".*")
	}
	return ans.String()
}

func buildPositional(m Morphology, attr string, s *positionalScheme, cc CorpusContext) BuildResult {
	b := &tagBuilder{ts: s.ts, sets: make([][]rune, s.ts.numPos)}
	isVerbtag := b.posIndex(lblVTKind) >= 0
	done := make(map[string]bool)
	feat := func(name string) []string {
		done[name] = true
		return m.Feats[name]
	}

	// part of speech
	if isVerbtag {
		var kinds string
		for _, u := range m.UPOS {
			switch u {
			case "VERB":
				kinds += "V"
			case "AUX":
				kinds += "A"
			default:
				b.note(SeverityWarning, "verbtag describes verbs only, part of speech %s skipped", u)
			}
		}
		if kinds != "" {
			b.constrainOrNote("part of speech", kinds, lblVTKind)
		}

	} else {
		var letters string
		for _, u := range m.UPOS {
			l, ok := uposToCNC[u]
			if !ok {
				b.note(SeverityWarning, "unknown part of speech %s - skipped", u)
				continue
			}
			letters += l
			// no need for a note if the broader category is requested too
			if msg, ok := uposApprox[u]; ok && !slices.Contains(m.UPOS, uposApproxPartner[u]) {
				b.note(SeverityHint, "%s", msg)
			}
		}
		abbr := slices.Contains(feat("Abbr"), "Yes")
		switch {
		case abbr && slices.Contains(b.ts.positions[0].used, 'B'):
			// abbreviations have their own part of speech, the original
			// one is moved to the next position
			b.constrainOrNote("abbreviation", "B", lblPOS)
			if letters != "" {
				b.constrainOrNote("part of speech of the abbreviation", letters, lblSubPOS)
			}
		default:
			if letters != "" {
				b.constrainOrNote("part of speech", letters, lblPOS)
			}
			if abbr {
				b.constrainOrNote("Abbr=Yes", "8", lblVariant)
			}
		}
		if len(m.UPOS) == 1 && m.UPOS[0] == "CCONJ" {
			b.constrainOrNote("CCONJ", "^", lblSubPOS)
		}
		if len(m.UPOS) == 1 && m.UPOS[0] == "SCONJ" {
			b.constrainOrNote("SCONJ", ",", lblSubPOS)
		}
	}
	if slices.Contains(feat("Foreign"), "Yes") {
		b.constrainOrNote("Foreign=Yes", "F", lblPOS)
	}

	// verb forms and moods
	verbForms := feat("VerbForm")
	for _, vf := range verbForms {
		switch vf {
		case "Inf":
			if !b.constrain("VerbForm=Inf", "F", lblVTMood) {
				b.constrainOrNote("VerbForm=Inf", "f", lblSubPOS)
			}
		case "Conv":
			if !b.constrain("VerbForm=Conv", "T", lblVTMood) {
				chars := "em"
				if t := m.Feats["Tense"]; len(t) == 1 && t[0] == "Pres" {
					chars = "e"
				} else if len(t) == 1 && t[0] == "Past" {
					chars = "m"
				}
				b.constrainOrNote("VerbForm=Conv", chars, lblSubPOS)
				done["Tense"] = true
			}
		case "Part":
			if isVerbtag {
				b.note(SeverityWarning, "verbtag describes whole verb forms - use e.g. Tense=Past or Voice=Pass instead of VerbForm=Part")
				continue
			}
			chars := "ps"
			if v := m.Feats["Voice"]; len(v) == 1 && v[0] == "Act" {
				chars = "p"
			} else if len(v) == 1 && v[0] == "Pass" {
				chars = "s"
			}
			b.constrainOrNote("VerbForm=Part", chars, lblSubPOS)
			done["Voice"] = true
		case "Fin":
			// no specific position
		default:
			b.note(SeverityWarning, "VerbForm=%s has no counterpart in the %s tagset - skipped", vf, b.ts.ident)
		}
	}
	for _, mood := range feat("Mood") {
		switch mood {
		case "Ind":
			if !b.constrain("Mood=Ind", "D", lblVTMood) {
				b.note(SeverityWarning, "indicative is not marked in the %s tagset - skipped (see the `verbtag` attribute)", b.ts.ident)
			}
		case "Cnd":
			if !b.constrain("Mood=Cnd", "C", lblVTMood) {
				b.constrainOrNote("Mood=Cnd", "c", lblSubPOS)
			}
		case "Imp":
			if !b.constrain("Mood=Imp", "I", lblVTMood) {
				b.constrainOrNote("Mood=Imp", "i", lblSubPOS)
			}
		default:
			b.note(SeverityWarning, "Mood=%s has no counterpart in the %s tagset - skipped", mood, b.ts.ident)
		}
	}

	// subtypes of pronouns, numerals and prepositions
	for _, name := range sortedKeys(subPOSFeats) {
		vals := feat(name)
		var chars string
		for _, v := range vals {
			c := subPOSFeats[name][v]
			if c == "" {
				b.note(SeverityWarning, "%s=%s has no counterpart in the %s tagset - skipped", name, v, b.ts.ident)
			}
			chars += c
		}
		if chars != "" {
			b.constrainOrNote(name+"="+strings.Join(vals, ","), chars, lblSubPOS)
		}
	}

	// gender and animacy
	genders, animacy := feat("Gender"), feat("Animacy")
	if len(genders) > 0 || len(animacy) > 0 {
		var chars string
		masc := slices.Contains(genders, "Masc") || len(genders) == 0
		if masc {
			anim, inan := slices.Contains(animacy, "Anim"), slices.Contains(animacy, "Inan")
			if anim {
				chars += "M"
			}
			if inan {
				chars += "I"
			}
			if !anim && !inan {
				chars += "MIY"
			}
		}
		if slices.Contains(genders, "Fem") {
			chars += "F"
		}
		if slices.Contains(genders, "Neut") {
			chars += "N"
		}
		b.constrainOrNote("gender", chars, lblGender)
	}

	// simple features
	for _, name := range sortedKeys(m.Feats) {
		if done[name] {
			continue
		}
		done[name] = true
		target, ok := simpleFeatTargets[name]
		if !ok {
			b.note(SeverityWarning, "feature %s has no counterpart in the %s tagset - skipped", name, b.ts.ident)
			continue
		}
		var chars string
		for _, v := range m.Feats[name] {
			c, ok := target.values[v]
			if !ok {
				b.note(SeverityWarning, "%s=%s has no counterpart in the %s tagset - skipped", name, v, b.ts.ident)
				continue
			}
			chars += c
		}
		if chars != "" {
			b.constrainOrNote(name+"="+strings.Join(m.Feats[name], ","), chars, target.labels...)
		}
	}

	value := b.export()
	if value == "" {
		b.note(SeverityError, "no constraint could be expressed in the %s tagset", b.ts.ident)
		return BuildResult{Issues: issuesOf(attr, "", b.issues)}
	}
	ans := BuildResult{
		CQL:    fmt.Sprintf(`%s="%s"`, attr, value),
		Issues: issuesOf(attr, value, b.issues),
	}
	if issues, err := ValidateValue(attr, value, false, cc); err == nil {
		ans.Issues = append(ans.Issues, issues...)
	}
	return ans
}
