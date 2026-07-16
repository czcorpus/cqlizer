package mcp

import (
	"fmt"
	"strings"
)

type attr struct {
	Name    string `json:"name"`
	Label   string `json:"label"`
	AttrDoc string `json:"attrDoc"`
}

type structure struct {
	Name       string `json:"name"`
	Attributes []attr `json:"attributes"`
}

type corpusInfo struct {
	Name                 string      `json:"name"`
	Info                 string      `json:"info"`
	PositionalAttributes []attr      `json:"positionalAttributes"`
	Structures           []structure `json:"structures"`
	TagsetDoc            string      `json:"tagsetDoc"`
}

func (ci *corpusInfo) AsMarkdown() string {
	var buff strings.Builder

	buff.WriteString(fmt.Sprintf("# Structure of the corpus %s\n\n", ci.Name))
	buff.WriteString("## Positional Attributes\n\n")
	for _, p := range ci.PositionalAttributes {
		buff.WriteString(fmt.Sprintf("### %s\n\n", p.Name))
		if p.Label != "" {
			buff.WriteString(fmt.Sprintf("label: %s\n", p.Label))
		}
		if p.AttrDoc != "" {
			buff.WriteString(fmt.Sprintf("attribute doc reference: %s\n", p.AttrDoc))
		}
		buff.WriteByte('\n')
	}

	buff.WriteString("## Structures and their attributes\n\n")

	for _, st := range ci.Structures {
		buff.WriteString(fmt.Sprintf("* %s\n", st.Name))
		for _, sta := range st.Attributes {
			buff.WriteString(fmt.Sprintf("  * %s\n", sta.Name))
		}
		buff.WriteByte('\n')
	}
	buff.WriteByte('\n')

	return buff.String()
}
