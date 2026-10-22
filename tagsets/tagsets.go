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

// Package tagsets contains knowledge about annotation schemes used
// in corpora (Universal Dependencies, Czech positional tagsets)
// and validates CQL attribute values against them.
//
// A scheme describes the value space of a single positional attribute
// (e.g. `upos`, `feats`, `deprel`). Because one corpus usually mixes
// several schemes, they are bound to (corpus, attribute) pairs - either
// explicitly via configuration or by inference from attribute names.
//
// Parts of the tagset knowledge come from the Alpha project
// by Jiří Milička.
package tagsets

import (
	"fmt"
	"strings"
)

// Severity of a found issue
type Severity string

const (
	// SeverityError means the value cannot match anything (or cannot
	// match anything the way it was most likely intended)
	SeverityError Severity = "error"

	// SeverityWarning means the value is suspicious - e.g. it is not
	// listed in the reference data, which can still be outdated
	// with respect to a concrete corpus
	SeverityWarning Severity = "warning"

	// SeverityHint is an advice which does not indicate a mistake
	SeverityHint Severity = "hint"
)

// Issue is a problem found in an attribute value
type Issue struct {
	Attr     string
	Value    string
	Severity Severity
	Message  string
}

func (iss Issue) String() string {
	return fmt.Sprintf("[%s] `%s=\"%s\"`: %s", iss.Severity, iss.Attr, iss.Value, iss.Message)
}

// Value is a CQL attribute value to be validated
type Value struct {

	// Pattern is a regular expression which must match the whole
	// attribute value or, in case Literal is true, a plain string (CQL `==`)
	Pattern string

	Literal bool

	// MultiSep is a separator of multiple values in a single attribute
	// value (registry MULTIVALUE + MULTISEP), empty for single-value
	// attributes. Manatee matches patterns of multivalue attributes
	// against the whole value and also against each of the values.
	MultiSep string
}

// Scheme validates values of a positional attribute
type Scheme interface {
	ID() string

	// Validate tests a CQL attribute value. Returned issues have their
	// Attr and Value fields empty.
	Validate(v Value) []Issue
}

// Scheme identifiers usable in configuration
const (
	SchemeUDUPOS   = "ud:upos"
	SchemeUDFeats  = "ud:feats"
	SchemeUDDeprel = "ud:deprel"

	// SchemeUDFeatPrefix is followed by a UD feature name (e.g. `ud:feat:Case`)
	// and describes attributes containing values of a single feature
	SchemeUDFeatPrefix = "ud:feat:"

	// SchemeNone disables validation of an attribute
	SchemeNone = "none"
)

// Positional tagsets are identified by their KonText names (e.g. `cs_cnc2020`),
// see PositionalTagsets().

// SchemeVerbtag describes the CNC `verbtag` attribute (whole verb forms
// incl. compound ones), it is inferred automatically for `verbtag` attributes
const SchemeVerbtag = "verbtag"

// NewScheme creates a scheme based on its identifier. The `lang`
// argument is an ISO 639 code used to select language-specific value
// sets; in case it is empty or unknown, values of all languages are used.
func NewScheme(id, lang string) (Scheme, error) {
	switch {
	case id == SchemeUDUPOS:
		return newUDUPOS(), nil
	case id == SchemeUDFeats:
		return newUDFeats(lang), nil
	case id == SchemeUDDeprel:
		return newUDDeprel(lang), nil
	case strings.HasPrefix(id, SchemeUDFeatPrefix):
		name := strings.TrimPrefix(id, SchemeUDFeatPrefix)
		if _, ok := udData().allFeats[name]; !ok {
			return nil, fmt.Errorf("unknown UD feature %s in scheme %s", name, id)
		}
		return newUDSingleFeat(name, lang), nil
	}
	if ts, ok := positionalTagsets()[id]; ok {
		return &positionalScheme{ts: ts}, nil
	}
	return nil, fmt.Errorf("unknown tagset scheme %s", id)
}

// PositionalTagsets returns identifiers of available positional tagsets
// (they are the same as tagset identifiers in KonText)
func PositionalTagsets() []string {
	return sortedKeys(positionalTagsets())
}

// parentAttrPrefixes are prefixes of attributes describing a parent
// token (e.g. `p_tag`, `ep_tag`); such attributes share the scheme
// of their base attribute
var parentAttrPrefixes = []string{"p_", "ep_"}

// CorpusContext describes a corpus as needed for resolving schemes
// of its attributes
type CorpusContext struct {

	// Lang is an ISO 639 code of the corpus language
	Lang string

	// Attrs lists all the positional attributes of the corpus
	Attrs []string

	// Schemes explicitly maps attributes to scheme identifiers. They
	// take precedence over inferred schemes. Use SchemeNone to
	// disable validation of an attribute.
	Schemes map[string]string

	// MultiValueSeps maps multivalue attributes to their
	// separators (registry MULTIVALUE + MULTISEP)
	MultiValueSeps map[string]string
}

func (cc CorpusContext) hasAttr(name string) bool {
	for _, a := range cc.Attrs {
		if a == name {
			return true
		}
	}
	return false
}

// SchemeFor returns a scheme for a corpus attribute or nil if
// the attribute is not covered by any known scheme.
func SchemeFor(attr string, cc CorpusContext) (Scheme, error) {
	id, ok := cc.Schemes[attr]
	if !ok {
		for _, prefix := range parentAttrPrefixes {
			if base, found := strings.CutPrefix(attr, prefix); found {
				if id, ok = cc.Schemes[base]; ok {
					break
				}
			}
		}
	}
	if !ok {
		id = inferSchemeID(attr, cc)
	}
	if id == "" || id == SchemeNone {
		return nil, nil
	}
	return NewScheme(id, cc.Lang)
}

// inferSchemeID guesses a scheme based on attribute names. UD schemes
// are applied only to corpora which look like UD ones (i.e. they
// have `upos` or `feats` attributes) and only to attributes with
// UD-specific names, including prefixed variants of them used e.g.
// for parent tokens (`p_upos`, `case_feats`, ...).
func inferSchemeID(attr string, cc CorpusContext) string {
	// unlike `tag`, the name `verbtag` is specific enough and there
	// is only one tagset for it
	if attr == "verbtag" {
		return SchemeVerbtag
	}
	if !cc.hasAttr("upos") && !cc.hasAttr("feats") {
		return ""
	}
	switch {
	case attr == "upos" || strings.HasSuffix(attr, "_upos"):
		return SchemeUDUPOS
	case attr == "feats" || strings.HasSuffix(attr, "_feats"):
		return SchemeUDFeats
	case attr == "deprel" || strings.HasSuffix(attr, "_deprel"):
		return SchemeUDDeprel
	}
	// attributes containing values of a single feature (`case`, `verbform`, ...)
	if name, ok := udData().featByLowerName[attr]; ok {
		return SchemeUDFeatPrefix + name
	}
	return ""
}

// ValidateValue validates an attribute value using a scheme resolved
// for the attribute. Attributes without a known scheme produce no issues.
func ValidateValue(attr, value string, literal bool, cc CorpusContext) ([]Issue, error) {
	scheme, err := SchemeFor(attr, cc)
	if err != nil || scheme == nil {
		return nil, err
	}
	issues := scheme.Validate(Value{Pattern: value, Literal: literal, MultiSep: cc.MultiValueSeps[attr]})
	for i := range issues {
		issues[i].Attr = attr
		issues[i].Value = value
	}
	return issues, nil
}
