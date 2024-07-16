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
	"strings"
)

type Stack struct {
	data []ASTNode
}

func (stack *Stack) Push(v ASTNode) {
	stack.data = append(stack.data, v)
}

func (stack *Stack) Pop() ASTNode {
	v := stack.data[len(stack.data)-1]
	stack.data = stack.data[:len(stack.data)]
	return v
}

func (stack *Stack) Peek() ASTNode {
	return stack.data[len(stack.data)-1]
}

func (stack *Stack) Show() string {
	var wr strings.Builder
	for _, d := range stack.data {
		wr.WriteString(fmt.Sprintf("[%s: %01.2f]", d.Text(), d.Effect()) + ", ")
	}
	return wr.String()
}

func (stack *Stack) PathEffect() {
	j := 2.0
	effProp := stack.Peek().Effect()
	//fmt.Println("calculating effect -------")
	//fmt.Println(stack.Show())
	for i := len(stack.data) - 2; i >= 0; i-- {
		stack.data[i].SetEffect(stack.data[i].Effect() + effProp*1.0/(j*j))
		j++
	}
	//fmt.Println("AFTER -------")
	//fmt.Println(stack.Show())
}
