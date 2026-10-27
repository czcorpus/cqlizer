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
	"strings"

	"github.com/czcorpus/cqlizer/cql"
)

// plainRegExpValue returns the value of a CQL regexp which contains
// no special characters at all (no operators, no escapes, no character
// classes) - i.e. one which matches just itself and can be safely
// represented as a DQD string. For any other regexp, false is returned.
// An empty regexp ("") is a plain empty string.
func plainRegExpValue(rg *cql.RegExp) (string, bool) {
	if len(rg.RegExpRaw) == 0 {
		return "", true
	}
	if len(rg.RegExpRaw) > 1 {
		return "", false
	}
	var ans strings.Builder
	for _, item := range rg.RegExpRaw[0].Values {
		simple, ok := item.(*cql.RgSimple)
		if !ok {
			return "", false
		}
		for _, v := range simple.Values {
			ch, ok := v.(*cql.RgChar)
			if !ok || !ch.IsConstant() || ch.IsUnicodeClass() {
				return "", false
			}
			lit, _ := ch.Literal()
			if hasEscapes(lit.String()) {
				return "", false
			}
			ans.WriteString(lit.String())
		}
	}
	// The CQL parser silently drops some parts of a regexp (e.g. empty
	// alternatives in "a|"), so we make sure the value we've collected
	// is really the whole original regexp.
	if `"`+ans.String()+`"` != rg.String() {
		return "", false
	}
	return ans.String(), true
}

func hasEscapes(s string) bool {
	return strings.Contains(s, `\`)
}
