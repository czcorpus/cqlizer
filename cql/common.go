// Copyright 2024 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2024 Institute of the Czech National Corpus,
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

func (s ASTString) Text() string {
	return string(s)
}

func (s ASTString) String() string {
	return string(s)
}

type ASTNode interface {
	Text() string
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
		panic("expecting a slice")
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
