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
	Value float64
}

func (op Constant) String() string {
	return fmt.Sprintf("%01.3f", op.Value)
}

// -----------------------------

type Ceil1 struct{}

func (op Ceil1) String() string {
	return "Ceil1"
}

// -----------------------------

type Add struct{}

func (op Add) String() string {
	return "P(X | Y)"
}

// ------------------------------

type Multiply struct{}

func (op Multiply) String() string {
	return "P(X & Y)"
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
		sm.evalPush(Constant{Value: op1.Value + op2.Value})
	case Multiply:
		op1, err := sm.evalPop()
		if err != nil {
			return err
		}
		op2, err := sm.evalPop()
		if err != nil {
			return err
		}
		sm.evalPush(Constant{Value: op1.Value * op2.Value})
	case Ceil1:
		op1, err := sm.evalPop()
		if err != nil {
			return err
		}
		sm.evalPush(Constant{Value: min(1.0, op1.Value)})
	case NegProb:
		op1, err := sm.evalPop()
		if err != nil {
			return err
		}
		sm.evalPush(Constant{Value: 1 - op1.Value})
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
