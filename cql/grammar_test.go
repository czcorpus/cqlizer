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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegexQuery(t *testing.T) {
	q1 := "[word=\"moto[a-z]\"]"
	p, err := ParseCQL("#", q1)
	assert.NoError(t, err)
	fmt.Println("p: ", p)

}
