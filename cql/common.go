// Copyright 2024 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2024 Department of Linguistics,
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

package cql

import (
	"fmt"
	"reflect"
)

type ASTString string

// Text renders the node as literal CQL source. For ASTString (a bare
// token - a keyword, operator, attribute name, number, ...) the text
// content already *is* valid CQL, so it's returned unchanged.
func (s ASTString) String() string {
	return string(s)
}

func (s ASTString) CQL() string {
	return string(s)
}

// ASTNode is implemented by every node of the CQL AST.
//
// String returns a debugging representation - for some node types this
// is the raw matched source, for others (ones with several grammar
// variants) it's a placeholder such as "#AttVal[...]"; it is not meant
// to be parseable.
//
// CQL renders the node as valid, literal CQL source, rebuilt from the
// node's own (possibly normalized) fields rather than echoed from the
// original input. Requiring it here - rather than adding it ad hoc -
// means a new AST type fails to compile until it defines both methods,
// so it can't quietly end up without a way to regenerate CQL.
type ASTNode interface {
	String() string
	CQL() string
}

func fromIdxOfUntypedSlice[T any](arr any, idx int) T {
	if arr == nil {
		var t T
		return t
	}
	tmp, ok := arr.([]any)
	if !ok {
		panic("value must be a slice")
	}
	v := tmp[idx]
	if v == nil {
		var t T
		return t
	}
	vt, ok := v.(T)
	if !ok {
		panic(fmt.Sprintf("value with idx %d has invalid type %s", idx, reflect.TypeOf(v)))
	}
	return vt
}

func anyToSlice(v any) []any {
	if v == nil {
		return []any{}
	}
	vt, ok := v.([]any)
	if !ok {
		panic(fmt.Sprintf("expecting an []any slice, got %s", reflect.TypeOf(v)))
	}
	return vt
}

func typedOrPanic[T any](v any) T {
	if v == nil {
		var ans T
		return ans
	}
	vt, ok := v.(T)
	if !ok {
		panic(fmt.Sprintf("unexpected type %s of: %v", reflect.TypeOf(v), v))
	}
	return vt
}
