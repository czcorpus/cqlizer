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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRgSimpleExhaustionScoreConstFirst(t *testing.T) {
	rgs := RgSimple{
		Values: []any{},
	}

	rgch := &RgChar{
		variant1: &rgCharVariant1{Value: ASTString("N")},
	}
	rgs.Values = append(rgs.Values, rgch)

	rgop := &RgOp{
		Value: ASTString("."),
	}
	rgch = &RgChar{
		variant2: &rgCharVariant2{Value: rgop},
	}
	rgs.Values = append(rgs.Values, rgch)

	rgop = &RgOp{
		Value: ASTString("*"),
	}
	rgch = &RgChar{
		variant2: &rgCharVariant2{Value: rgop},
	}
	rgs.Values = append(rgs.Values, rgch)
	assert.Equal(t, 15, rgs.ExhaustionScore())
}

func TestRgSimpleExhaustionScoreForWildcardFirst(t *testing.T) {
	rgs := RgSimple{
		Values: []any{},
	}

	rgop := &RgOp{
		Value: ASTString("."),
	}
	rgch := &RgChar{
		variant2: &rgCharVariant2{Value: rgop},
	}
	rgs.Values = append(rgs.Values, rgch)

	rgop = &RgOp{
		Value: ASTString("*"),
	}
	rgch = &RgChar{
		variant2: &rgCharVariant2{Value: rgop},
	}
	rgs.Values = append(rgs.Values, rgch)

	rgch = &RgChar{
		variant1: &rgCharVariant1{Value: ASTString("N")},
	}
	rgs.Values = append(rgs.Values, rgch)

	assert.Equal(t, 40, rgs.ExhaustionScore())
}

func TestRgSimpleExhaustionScoreRanges(t *testing.T) {
	rgs := RgSimple{
		Values: []any{},
	}

	rgch := &RgChar{
		variant1: &rgCharVariant1{Value: ASTString("N")},
	}
	rgs.Values = append(rgs.Values, rgch)

	rgrng := &RgRange{
		RgRangeSpec: &RgRangeSpec{
			Number1: ASTString("3"),
			Number2: ASTString("7"),
		},
	}
	rgs.Values = append(rgs.Values, rgrng)

	assert.Equal(t, 15, rgs.ExhaustionScore())
}
