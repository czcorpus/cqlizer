package pcalc

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

var (
	ErrProgramInvalidState = errors.New("program invalid state")
	ErrEmptyStack          = errors.New("empty stack")
)

type Constant struct {
	Value  float64
	Weight float64
}

func (op Constant) String() string {
	if op.Weight > 0 {
		return fmt.Sprintf("%01.3f (weight: %01.3f)", op.Value, op.Weight)
	}
	return fmt.Sprintf("%01.3f", op.Value)
}

func (op Constant) MergeWeight(other Constant) float64 {
	return (op.Weight + other.Weight) / 2
}

// -----------------------------

type Avg struct{}

func (op Avg) String() string {
	return "([-1] + [-2])/2"
}

// -----------------------------

type Add struct{}

func (op Add) String() string {
	return "P(X | Y)"
}

// ------------------------------

type Multiply struct{}

func (op Multiply) String() string {
	return "[-1] * [-2]"
}

// --------------------------------

type MultiplyOrWeightSum struct{}

func (op MultiplyOrWeightSum) String() string {
	return "if [-1].weight: [-1].weight * [-1] + (1 - [-1].weight) * [-2]; else: [-1] * [-2]"
}

type Divide struct{}

func (op Divide) String() string {
	return "[-2] / [-1]"
}

// ------------------------------

type Pop struct{}

func (op Pop) String() string {
	return "Pop"
}

// ------------------------------

type NegProb struct{}

func (op NegProb) String() string {
	return "P(not X)"
}

// ------------------------------

type StackElement interface {
	String() string
}

// ----------------------------------

type StackMachine struct {
	program  []StackElement
	stack    []Constant
	lock     sync.Mutex
	currStep int
}

func (sm *StackMachine) Clone() *StackMachine {
	ans := new(StackMachine)
	copy(ans.program, sm.program)
	ans.currStep = 0
	ans.stack = make([]Constant, 0, 30)
	return ans
}

func (sm *StackMachine) Push(se StackElement) {
	sm.lock.Lock()
	sm.program = append(sm.program, se)
	sm.lock.Unlock()
}

func (sm *StackMachine) PrintProgram() {
	for i, c := range sm.program {
		fmt.Printf("%d) %s\n", i, c.String())
	}
	fmt.Println("------------------------------")
}

func (sm *StackMachine) evalPush(v Constant) {
	sm.stack = append(sm.stack, v)
}

func (sm *StackMachine) evalPop() (Constant, error) {
	if len(sm.stack) == 0 {
		return Constant{}, ErrEmptyStack
	}
	v := sm.stack[len(sm.stack)-1]
	sm.stack = sm.stack[:len(sm.stack)-1]
	return v, nil
}

func (sm *StackMachine) Run() error {
	for range sm.program {
		if err := sm.NextStep(); err != nil {
			return err
		}
	}
	return nil
}

func (sm *StackMachine) NextStep() error {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	if sm.currStep >= len(sm.program) {
		return ErrProgramInvalidState
	}
	ate := any(sm.program[sm.currStep])
	switch tate := ate.(type) {
	case Constant:
		sm.evalPush(tate)
	case Add:
		op1, err := sm.evalPop()
		if err != nil {
			return err
		}
		op2, err := sm.evalPop()
		if err != nil {
			return err
		}
		sm.evalPush(Constant{Value: op1.Value + op2.Value, Weight: op1.MergeWeight(op2)})
	case Multiply:
		op1, err := sm.evalPop()
		if err != nil {
			return err
		}
		op2, err := sm.evalPop()
		if err != nil {
			return err
		}
		sm.evalPush(Constant{Value: op1.Value * op2.Value, Weight: op1.MergeWeight(op2)})
	case MultiplyOrWeightSum:
		op1, err := sm.evalPop()
		if err != nil {
			return err
		}
		op2, err := sm.evalPop()
		if err != nil {
			return err
		}
		var ans float64
		if op1.Weight > 0 && op2.Weight > 0 {
			ans = (op1.Value + op2.Value) / 2

		} else if op1.Weight > 0 {
			ans = op1.Weight*op1.Value + (1-op1.Weight)*op2.Value

		} else {
			ans = op1.Value * op2.Value
		}
		sm.evalPush(Constant{Value: ans, Weight: op1.MergeWeight(op2)})
	case Divide:
		op1, err := sm.evalPop()
		if err != nil {
			return err
		}
		op2, err := sm.evalPop()
		if err != nil {
			return err
		}
		sm.evalPush(Constant{Value: op2.Value / op1.Value, Weight: op1.MergeWeight(op2)})
	case Pop:
		_, err := sm.evalPop()
		if err != nil {
			return err
		}
	case Avg:
		op1, err := sm.evalPop()
		if err != nil {
			return err
		}
		op2, err := sm.evalPop()
		if err != nil {
			return err
		}
		sm.evalPush(Constant{Value: (op1.Value + op2.Value) / 2, Weight: op1.MergeWeight(op2)})
	case NegProb:
		op1, err := sm.evalPop()
		if err != nil {
			return err
		}
		sm.evalPush(Constant{Value: 1 - op1.Value, Weight: op1.Weight})
	default:
		return fmt.Errorf("invalid element or element type on stack: %s (type %s)", ate, reflect.TypeOf(ate))
	}
	sm.currStep++
	return nil
}

func (sm *StackMachine) Peek() (Constant, error) {
	if len(sm.stack) == 0 {
		return Constant{}, ErrEmptyStack
	}
	return sm.stack[len(sm.stack)-1], nil
}
