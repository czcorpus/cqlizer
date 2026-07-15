package cql

import (
	"errors"
	"sort"
	"strings"
)

// wantDisplayNames maps a raw pigeon "want" string (the literal text of a
// charClassMatcher.val or litMatcher.want, e.g. the huge generated Unicode
// range for LETTER) to the display name declared for the rule that owns it
// (`Rule "display name" <- ...` in grammar.peg). Pigeon itself only uses a
// rule's display name to prefix errors raised *while that rule is still on
// the call stack* (e.g. from action code); the generic "no match found,
// expected: ..." message it builds after a failed parse is assembled purely
// from the raw matcher text, so a display name on LETTER/LETTER_PHON never
// reaches that message on its own. This map lets simplifyParseError patch
// it in afterwards.
var wantDisplayNames = buildWantDisplayNames()

func buildWantDisplayNames() map[string]string {
	m := make(map[string]string)
	for _, r := range g.rules {
		if r.displayName == "" {
			continue
		}
		collectWantStrings(r.expr, categoryLabel(r.displayName), m)
	}
	return m
}

// categoryLabel turns a rule's raw displayName (e.g. `"a letter"`, quotes
// included, as pigeon stores it) into `<a letter>` so it reads as a token
// category rather than a literal string pigeon expects verbatim.
func categoryLabel(displayName string) string {
	return "<" + strings.Trim(displayName, "\"") + ">"
}

// collectWantStrings walks a rule's expression tree, without crossing into
// referenced sub-rules (those own their own display name, if any), and
// records the raw "want" text of every literal/char-class matcher found.
func collectWantStrings(node any, displayName string, m map[string]string) {
	switch e := node.(type) {
	case *choiceExpr:
		for _, alt := range e.alternatives {
			collectWantStrings(alt, displayName, m)
		}
	case *actionExpr:
		collectWantStrings(e.expr, displayName, m)
	case *recoveryExpr:
		collectWantStrings(e.expr, displayName, m)
	case *seqExpr:
		for _, sub := range e.exprs {
			collectWantStrings(sub, displayName, m)
		}
	case *labeledExpr:
		collectWantStrings(e.expr, displayName, m)
	case *andExpr:
		collectWantStrings(e.expr, displayName, m)
	case *notExpr:
		collectWantStrings(e.expr, displayName, m)
	case *zeroOrOneExpr:
		collectWantStrings(e.expr, displayName, m)
	case *zeroOrMoreExpr:
		collectWantStrings(e.expr, displayName, m)
	case *oneOrMoreExpr:
		collectWantStrings(e.expr, displayName, m)
	case *litMatcher:
		m[e.want] = displayName
	case *charClassMatcher:
		m[e.val] = displayName
	}
}

// simplifyParseError rewrites the "no match found, expected: ..." message
// produced by the generated parser so that raw matcher text covered by a
// named rule (e.g. LETTER, LETTER_PHON) is replaced by that rule's display
// name, collapsing the huge generated Unicode character classes into a
// short, readable expectation.
func simplifyParseError(err error) error {
	el, ok := err.(errList)
	if !ok {
		return err
	}
	for i, e := range el {
		pe, ok := e.(*parserError)
		if !ok || len(pe.expected) == 0 {
			continue
		}
		seen := make(map[string]struct{}, len(pe.expected))
		friendly := make([]string, 0, len(pe.expected))
		for _, w := range pe.expected {
			name := w
			if dn, ok := wantDisplayNames[w]; ok {
				name = dn
			}
			if _, dup := seen[name]; dup {
				continue
			}
			seen[name] = struct{}{}
			friendly = append(friendly, name)
		}
		sort.Strings(friendly)
		pe.expected = friendly
		pe.Inner = errors.New("no match found, expected: " + listJoin(friendly, ", ", "or"))
		el[i] = pe
	}
	return el
}
