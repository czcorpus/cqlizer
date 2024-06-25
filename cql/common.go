// Copyright 2024 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2024 Institute of the Czech National Corpus,
//                Faculty of Arts, Charles University
//   This file is part of CQLIZER.
//
//  CQLIZER is free software: you can redistribute it and/or modify
//  it under the terms of the GNU General Public License as published by
//  the Free Software Foundation, either version 3 of the License, or
//  (at your option) any later version.
//
//  CQLIZER is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of
//  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//  GNU General Public License for more details.
//
//  You should have received a copy of the GNU General Public License
//  along with CQLIZER.  If not, see <https://www.gnu.org/licenses/>.

package cql

import (
	"fmt"
	"reflect"
)

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
		panic(fmt.Sprintf("unexpected type %s", reflect.TypeOf(v)))
	}
	return vt
}
