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

// Package dqd translates CQL queries (see the cql package) into DQD, the
// query language of the LiRI LCP platform. The types in this file mirror
// LCP's DQD JSON schema (parser/cobquec.auto.json) - each DQD
// node there is a "one-key wrapper" object encoding a union (e.g.
// {"comparison": {...}}), which is modeled here the same way the cql
// package models grammar alternatives: a struct with several optional
// pointer fields, of which exactly one is set.
package dqd

// Query is the root of a DQD query, rule:
//
// query : ((quantification | sequence | set | unit | constraint) _NL*)+
type Query struct {
	Items []QueryItem

	// DQD also specifies some output properties.
	// To keep aligned with CQL we use the ViewOptions
	// type along with input CQL.
	Results []Result
}

// QueryItem wraps multiple forms:
//
// quantification | sequence | set | unit | constraint
type QueryItem struct {
	Quantification *Quantification
	Sequence       *Sequence
	Set            *Set
	Unit           *Unit
	Constraint     *Constraint
}

// PartOfOperator is the DQD @, %, & prefix relating a Unit/Sequence/Set
// to a containing label.
type PartOfOperator int

const (
	// PartOfStream is "@label" (e.g. a Token being part of a Segment).
	PartOfStream PartOfOperator = iota
	// PartOfTime is "%label".
	PartOfTime
	// PartOfLocation is "&label".
	PartOfLocation
)

// PartOf is a single "@label" / "%label" / "&label" relation.
type PartOf struct {
	Kind  PartOfOperator
	Label string
}

// Unit is a single labeled layer instance (e.g. "Token t1", "Segment
// s"), the DQD equivalent of one CQL bracket position or one CQL
// structure.
type Unit struct {
	// Layer is required, e.g. "Token", "Segment" - capitalized, per
	// DQD's LAYER token (`[A-Z][a-zA-Z0-9_]+`). Resolving a CQL
	// attribute/structure name to a DQD Layer name is corpus-specific;
	// see CorpusProps.
	Layer string
	Label string

	PartOf []PartOf

	// Constraints is implicitly AND'ed - DQD does not wrap same-unit
	// comparisons in an explicit logicalExpression (see test.dqd's
	// Turn/Token blocks). Only cross-comparison OR needs
	// LogicalExpression.
	Constraints []Constraint
}

// Sequence is an ordered list of members - the DQD equivalent of a CQL
// n-gram (a run of bracket positions).
type Sequence struct {
	Label  string
	PartOf []PartOf

	// Repetition, if set, applies to the sequence AS A WHOLE. This is a
	// structural mismatch with CQL, where {n,m}/*/+/? applies to a
	// single bracket position - translating a single repeated CQL
	// position requires wrapping it in its own one-member Sequence.
	Repetition *Repetition

	Members []Member
}

// Set is an unordered group of members.
//
// set : "set"(part_of)? label? _NL _INDENT members _DEDENT
type Set struct {
	Label   string
	PartOf  []PartOf
	Members []Member
}

// Member is one of Unit / Sequence / LogicalExpression - an entry of
// Sequence.Members or Set.Members.
//
// (unit|sequence|logical_expression) _NL*
type Member struct {
	Unit              *Unit
	Sequence          *Sequence
	LogicalExpression *LogicalExpression
}

// Repetition
//
// min".."max
type Repetition struct {
	Min string // POS_INT
	Max string // POS_INT or "*"
}

// Constraint
//
// constraint : quantification | unit | sequence | logical_expression | comparison
type Constraint struct {
	Quantification    *Quantification
	Unit              *Unit
	Sequence          *Sequence
	LogicalExpression *LogicalExpression
	Comparison        *Comparison
}

// Quantification is DQD's "EXISTS"/"ALL" (optionally negated) quantifier
// over a Unit/Sequence/Set.
//
// quantifier _NL _INDENT (unit | sequence | set) _DEDENT
type Quantification struct {
	Negated   bool
	Universal bool // false = EXISTS, true = ALL

	Unit     *Unit
	Sequence *Sequence
	Set      *Set
}

// LogicalOp is the AND/OR/NOT operator of a LogicalExpression.
type LogicalOp int

const (
	LogicalAnd LogicalOp = iota
	LogicalOr
	// LogicalNot is unary - Args holds exactly one element.
	LogicalNot
)

// LogicalExpression combines several constraints with AND/OR, or negates
// a single one with NOT.
type LogicalExpression struct {
	Op   LogicalOp
	Args []Constraint
}

// Comparison is a single "left comparator right" condition.
type Comparison struct {
	Left Operand
	Not  bool
	// Comparator is one of "=", ">", "<", ">=", "<=", "contain",
	// "overlaps".
	Comparator string
	Right      Operand
}

// Operand is one of Function / Reference / Attribute / String / Regex /
// Math - the left/right side of a Comparison, or a Function argument.
type Operand struct {
	Function *Function
	// Reference is a bare "label" or "label.attr" cross-reference (e.g.
	// the "t3"/"tx" in test.dqd's DepRel head=t3/dep=tx).
	Reference *Reference
	// Attribute is always "label.attr" - the dedicated dotted form
	// used for plain attribute access (e.g. "t1.lemma").
	Attribute *Attribute
	String    *StringLit
	Regex     *Regex
	Math      *Math
}

// Reference is a bare cross-reference to another label (or label.attr).
type Reference struct {
	Value string
}

// Attribute is a "Label.Name" attribute access.
type Attribute struct {
	Label string
	Name  string
}

// StringLit is a DQD double-quoted string literal.
type StringLit struct {
	Value string
}

// Regex is a DQD "/pattern/" regex literal, optionally case-insensitive
// ("/pattern/i" in DQD source, a separate "i" flag in the wire format).
type Regex struct {
	Pattern         string
	CaseInsensitive bool
}

// Function is a DQD function call, e.g. "start(label2)".
type Function struct {
	// Name excludes the grammar's trailing "(", e.g. "start" not
	// "start(".
	Name string
	Args []Operand
}

// Math is either a bare number literal or an Operation - the two
// alternatives of DQD's `math` rule.
type Math struct {
	Number    string // leaf case
	Operation *Operation
}

// Operation is a single "left op right" arithmetic expression.
type Operation struct {
	Left Operand // math or function only, per the DQD grammar
	// Operator is one of "+", "-", "*", "/".
	Operator string
	Right    Operand
}

// Result is one "label => plain|analysis|collocation" results block.
type Result struct {
	Label       string
	Plain       *PlainResult
	Analysis    *AnalysisResult
	Collocation *CollocationResult
}

// PlainResult lists which labels provide surrounding context and which
// are the highlighted entities of a KWIC-style result.
type PlainResult struct {
	Context  []string
	Entities []string
}

// AnalysisResult describes an aggregated (frequency table style) result.
type AnalysisResult struct {
	Attributes []Operand // attribute | function | entity(reference)
	// Functions is one or more of "frequency", "minimum", "maximum",
	// "average", "stddev".
	Functions []string
	Filter    []Comparison
}

// CollocationResult describes a collocation search, either around a
// Space (Space set) or a Center label with an explicit Window.
type CollocationResult struct {
	Space     string // mutually exclusive with Center/Window
	Center    string
	Window    *Window
	Attribute string
}

// Window is a collocation search window, e.g. "-5..+5".
type Window struct {
	Left  string // "-N" or "*"
	Right string // "N" or "*"
}
