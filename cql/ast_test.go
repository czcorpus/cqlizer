package cql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRgSimpleExhaustionScore(t *testing.T) {
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
	assert.Equal(t, 40, rgs.ExhaustionScore())
}
