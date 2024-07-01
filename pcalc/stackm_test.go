package pcalc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMultiplyOperation(t *testing.T) {
	sm := StackMachine{}
	sm.Push(Constant{Value: 10})
	sm.Push(Constant{Value: 27})
	sm.Push(Multiply{})
	err := sm.Run()
	assert.NoError(t, err)
	nm, err := sm.Peek()
	assert.NoError(t, err)
	assert.InDelta(t, 270.0, nm.Value, 0.001)
}

func TestAddOperation(t *testing.T) {
	sm := StackMachine{}
	sm.Push(Constant{Value: 5})
	sm.Push(Constant{Value: -2.7})
	sm.Push(Add{})
	err := sm.Run()
	assert.NoError(t, err)
	nm, err := sm.Peek()
	assert.NoError(t, err)
	assert.InDelta(t, 2.3, 0.001, nm.Value)
}

func TestMoreComplexOperation(t *testing.T) {
	// 3 * (2.5 + 20) => RPN 3, 2.5, 20, +, *
	sm := StackMachine{}
	sm.Push(Constant{Value: 3})
	sm.Push(Constant{Value: 2.5})
	sm.Push(Constant{Value: 20})
	sm.Push(Add{})
	sm.Push(Multiply{})
	err := sm.Run()
	assert.NoError(t, err)
	nm, err := sm.Peek()
	assert.NoError(t, err)
	assert.InDelta(t, 67.5, 0.001, nm.Value)
}
