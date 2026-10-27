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
	"unicode"

	"github.com/czcorpus/cqlizer/cql"
)

// Translation of CQL attribute values to DQD operands.
//
// Important semantic difference: a CQL regexp always matches the whole
// attribute value while a DQD regexp matches any substring (verified on
// an LCP instance). So each regexp is translated as /^(...)$/.
//
// Values which match just themselves are translated to DQD strings.
// But DQD strings have no escaping, so a value containing a double quote
// or a backslash is translated to an (anchored) regexp instead.
//
// Slashes inside a DQD regexp need no escaping (/a/b/ is a valid regexp
// matching "a/b" - verified on an LCP instance).
//
// Known limitations of the CQL parser (cql/grammar.peg) affecting values
// we can translate (i.e. queries which fail to parse already):
//   - an escaped backslash (\\) is not accepted inside a regexp value
//   - a literal value (attr=="...") accepts only letters, digits and
//     escaped characters - e.g. [lemma=="a.b"] or a value containing "/"
//     cannot be parsed even though a literal has no regexp meaning

const (
	caseInsensitivePrefix = "(?i)"
	rgSpecialChars        = `.[]()*+?{}|^$\`
)

// regExpOperand translates a CQL regexp value (attr="...") to a DQD operand.
func regExpOperand(rg *cql.RegExp) (Operand, error) {
	src := strings.TrimSuffix(strings.TrimPrefix(rg.String(), `"`), `"`)
	// KonText expresses case insensitivity via a leading inline flag.
	// DQD (or PostgreSQL behind it) allows inline flags only at the
	// beginning of a pattern, so we have to use DQD's own flag instead.
	caseInsensitive := strings.HasPrefix(src, caseInsensitivePrefix)
	if caseInsensitive {
		src = strings.TrimPrefix(src, caseInsensitivePrefix)

	} else if lit, ok := regExpLiteral(src); ok && isDQDStringSafe(lit) {
		return Operand{String: &StringLit{Value: lit}}, nil
	}
	if strings.Contains(src, `\p{`) {
		return Operand{}, unsupported(`unicode properties (\p{...}) in regular expressions`)
	}
	if hasInlineFlags(src) {
		return Operand{}, unsupported("inline flags (?...) in regular expressions (other than a leading (?i))")
	}
	return Operand{Regex: &Regex{
		Pattern:         "^(" + src + ")$",
		CaseInsensitive: caseInsensitive,
	}}, nil
}

// literalOperand translates a CQL literal value (attr=="...") to a DQD operand.
func literalOperand(rs *cql.RawString) Operand {
	var lit string
	if rs.SimpleString != nil {
		lit = unescape(rs.SimpleString.String())
	}
	if isDQDStringSafe(lit) {
		return Operand{String: &StringLit{Value: lit}}
	}
	return Operand{Regex: &Regex{Pattern: "^" + quoteRegExp(lit) + "$"}}
}

// regExpLiteral returns the value a CQL regexp matches in case it
// contains no regexp operators (escaped characters are allowed and
// unescaped). Otherwise, false is returned.
func regExpLiteral(src string) (string, bool) {
	var ans strings.Builder
	chars := []rune(src)
	for i := 0; i < len(chars); i++ {
		switch {
		case chars[i] == '\\':
			// an escaped non-alphanumeric char is a literal, while
			// e.g. \p{...} is a character class
			if i+1 >= len(chars) || unicode.IsLetter(chars[i+1]) || unicode.IsDigit(chars[i+1]) {
				return "", false
			}
			i++
			ans.WriteRune(chars[i])
		case strings.ContainsRune(rgSpecialChars, chars[i]):
			return "", false
		default:
			ans.WriteRune(chars[i])
		}
	}
	return ans.String(), true
}

// unescape removes backslashes from escaped characters
// of a CQL literal (e.g. a\"b => a"b).
func unescape(s string) string {
	var ans strings.Builder
	chars := []rune(s)
	for i := 0; i < len(chars); i++ {
		if chars[i] == '\\' && i+1 < len(chars) {
			i++
		}
		ans.WriteRune(chars[i])
	}
	return ans.String()
}

// quoteRegExp escapes all the regexp operators in s.
func quoteRegExp(s string) string {
	var ans strings.Builder
	for _, c := range s {
		if strings.ContainsRune(rgSpecialChars, c) {
			ans.WriteRune('\\')
		}
		ans.WriteRune(c)
	}
	return ans.String()
}

// hasInlineFlags tests for (?...) groups other than lookahead/lookbehind
// assertions ((?=, (?!, (?<=, (?<!).
func hasInlineFlags(src string) bool {
	for i := 0; i < len(src); i++ {
		if src[i] == '\\' {
			i++
			continue
		}
		if strings.HasPrefix(src[i:], "(?") {
			rest := src[i+2:]
			if !strings.HasPrefix(rest, "=") && !strings.HasPrefix(rest, "!") &&
				!strings.HasPrefix(rest, "<=") && !strings.HasPrefix(rest, "<!") {
				return true
			}
		}
	}
	return false
}

// isDQDStringSafe tests whether s can be written as a DQD string
// which has no escaping (and we don't know how LCP would treat
// backslashes in it).
func isDQDStringSafe(s string) bool {
	return !strings.ContainsAny(s, "\"\\\n")
}
