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
	"strings"
)

// langIfKnown returns the language in case UD data contain
// language-specific values for it (selected by `has`), otherwise "".
func langIfKnown(lang string, has bool) string {
	if has {
		return lang
	}
	return ""
}

// ---------------------- UPOS ---------------------

type udUPOS struct {
	set *closedSet
}

func newUDUPOS() *udUPOS {
	return &udUPOS{
		set: newClosedSet("UD part of speech (UPOS)", udUPOSValues, "", nil, true, false),
	}
}

func (s *udUPOS) ID() string {
	return SchemeUDUPOS
}

func (s *udUPOS) Validate(v Value) []Issue {
	value, literal := v.Pattern, v.Literal
	return s.set.validate(value, literal)
}

// ---------------------- deprel ---------------------

type udDeprel struct {
	set *closedSet
}

func newUDDeprel(lang string) *udDeprel {
	values, langSpecific := udData().deprelValues(lang)
	return &udDeprel{
		set: newClosedSet(
			"UD dependency relation",
			values,
			langIfKnown(lang, langSpecific),
			udData().allDeprels,
			false,
			false,
		),
	}
}

func (s *udDeprel) ID() string {
	return SchemeUDDeprel
}

func (s *udDeprel) Validate(v Value) []Issue {
	value, literal := v.Pattern, v.Literal
	issues := s.set.validate(value, literal)
	issues = append(issues, s.subtypeHints(value, literal)...)
	return issues
}

// subtypeHints warns about values which match a basic relation but
// not its subtypes (e.g. `nsubj` vs. `nsubj:pass`) as this is often
// unintended.
func (s *udDeprel) subtypeHints(value string, literal bool) []Issue {
	rx, err := compileWhole(value, literal)
	if err != nil {
		return nil
	}
	var issues []Issue
	for _, base := range s.set.values {
		if strings.Contains(base, ":") || !rx.MatchString(base) {
			continue
		}
		var missed []string
		for _, sub := range s.set.values {
			if strings.HasPrefix(sub, base+":") && !rx.MatchString(sub) {
				missed = append(missed, sub)
			}
		}
		if len(missed) > 0 {
			issues = append(issues, Issue{
				Severity: SeverityHint,
				Message: fmt.Sprintf(
					"`%s` does not include its subtypes (%s); use e.g. `%s(:.*)?` to include them",
					base, listValues(missed), base),
			})
		}
	}
	return issues
}

// ---------------------- single feature ---------------------

// udSingleFeat describes attributes containing values of a single
// UD feature (e.g. `case` with values `Nom`, `Gen`, ...)
type udSingleFeat struct {
	name string
	set  *closedSet
}

func newUDSingleFeat(name, lang string) *udSingleFeat {
	values, langSpecific, found := udData().featValues(lang, name)
	if !found {
		// the feature is not used in the language (according to UD),
		// so we use values from other languages
		values, langSpecific = udData().allFeats[name], false
	}
	return &udSingleFeat{
		name: name,
		set: newClosedSet(
			fmt.Sprintf("value of the UD feature %s", name),
			values,
			langIfKnown(lang, langSpecific),
			udData().allFeats[name],
			false,
			true,
		),
	}
}

func (s *udSingleFeat) ID() string {
	return SchemeUDFeatPrefix + s.name
}

func (s *udSingleFeat) Validate(v Value) []Issue {
	value, literal := v.Pattern, v.Literal
	return s.set.validate(value, literal)
}
