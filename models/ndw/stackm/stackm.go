package stackm

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

type Constant float64

func (op Constant) String() string {
	return fmt.Sprintf("%01.3f", op)
}

func (op Constant) Add(other Constant) Constant {
	return Constant(float64(op) + float64(other))
}

func (op Constant) Multiply(other Constant) Constant {
	return Constant(float64(op) * float64(other))
}

func (op Constant) AsFloat64() float64 {
	return float64(op)
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

// ------------------------------

type Pop struct{}

func (op Pop) String() string {
	return "Pop"
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
		return Constant(0), ErrEmptyStack
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
		sm.evalPush(op1.Add(op2))
	case Multiply:
		op1, err := sm.evalPop()
		if err != nil {
			return err
		}
		op2, err := sm.evalPop()
		if err != nil {
			return err
		}
		sm.evalPush(op1.Multiply(op2))
	case Pop:
		_, err := sm.evalPop()
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid element or element type on stack: %s (type %s)", ate, reflect.TypeOf(ate))
	}
	sm.currStep++
	return nil
}

func (sm *StackMachine) Peek() (Constant, error) {
	if len(sm.stack) == 0 {
		return Constant(0), ErrEmptyStack
	}
	return sm.stack[len(sm.stack)-1], nil
}
