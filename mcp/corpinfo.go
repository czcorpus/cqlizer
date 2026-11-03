// Copyright 2026 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2026 Department of Linguistics,
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

package mcp

import (
	"fmt"
	"strings"
)

type attr struct {
	Name    string `json:"name"`
	Label   string `json:"label"`
	AttrDoc string `json:"attrDoc"`

	// Values lists all the possible values of the attribute
	// (if known - currently for LCP corpora only)
	Values []string `json:"values,omitempty"`
}

type structure struct {
	Name       string `json:"name"`
	Attributes []attr `json:"attributes"`

	// Note is an optional remark about the structure
	Note string `json:"note,omitempty"`
}

type corpusInfo struct {
	Name                 string      `json:"name"`
	Info                 string      `json:"info"`
	PositionalAttributes []attr      `json:"positionalAttributes"`
	Structures           []structure `json:"structures"`
	TagsetDoc            string      `json:"tagsetDoc"`

	// Notes are optional remarks on how to write queries for the corpus
	Notes []string `json:"notes,omitempty"`
}

// formatValues renders a list of attribute values for markdown output
func formatValues(values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = "`" + v + "`"
	}
	return strings.Join(quoted, ", ")
}

func (ci *corpusInfo) AsMarkdown() string {
	var buff strings.Builder

	buff.WriteString(fmt.Sprintf("# Structure of the corpus %s\n\n", ci.Name))
	for _, note := range ci.Notes {
		buff.WriteString(fmt.Sprintf("* %s\n", note))
	}
	if len(ci.Notes) > 0 {
		buff.WriteByte('\n')
	}
	buff.WriteString("## Positional Attributes\n\n")
	for _, p := range ci.PositionalAttributes {
		buff.WriteString(fmt.Sprintf("### %s\n\n", p.Name))
		if p.Label != "" {
			buff.WriteString(fmt.Sprintf("label: %s\n", p.Label))
		}
		if p.AttrDoc != "" {
			buff.WriteString(fmt.Sprintf("attribute doc reference: %s\n", p.AttrDoc))
		}
		if len(p.Values) > 0 {
			buff.WriteString(fmt.Sprintf("values: %s\n", formatValues(p.Values)))
		}
		buff.WriteByte('\n')
	}

	buff.WriteString("## Structures and their attributes\n\n")

	for _, st := range ci.Structures {
		if st.Note != "" {
			buff.WriteString(fmt.Sprintf("* %s (%s)\n", st.Name, st.Note))

		} else {
			buff.WriteString(fmt.Sprintf("* %s\n", st.Name))
		}
		for _, sta := range st.Attributes {
			if len(sta.Values) > 0 {
				buff.WriteString(fmt.Sprintf("  * %s (values: %s)\n", sta.Name, formatValues(sta.Values)))

			} else {
				buff.WriteString(fmt.Sprintf("  * %s\n", sta.Name))
			}
		}
		buff.WriteByte('\n')
	}
	buff.WriteByte('\n')

	return buff.String()
}
