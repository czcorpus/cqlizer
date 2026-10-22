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
	"context"
	"fmt"
	"strings"

	"github.com/czcorpus/cqlizer/cnf"
	"github.com/czcorpus/cqlizer/cql"
	"github.com/czcorpus/cqlizer/eval/feats"
	"github.com/czcorpus/cqlizer/tagsets"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog/log"

	regParser "github.com/czcorpus/rexplorer/parser"
)

// newCorpusContext describes a corpus for the tagsets package
// based on its configuration and registry
func newCorpusContext(corpusProps feats.CorpusProps, reg *regParser.Document) tagsets.CorpusContext {
	cc := tagsets.CorpusContext{
		Lang:           corpusProps.Lang,
		Attrs:          make([]string, 0, len(reg.PosAttrs)),
		Schemes:        corpusProps.Tagsets,
		MultiValueSeps: make(map[string]string),
	}
	for _, pa := range reg.PosAttrs {
		cc.Attrs = append(cc.Attrs, pa.Name)
		if sep, ok := multiValueSep(pa.Entries); ok {
			cc.MultiValueSeps[pa.Name] = sep
		}
	}
	return cc
}

// checkTagsets validates values of positional attributes against
// annotation schemes of the respective attributes
func checkTagsets(
	corpusID string,
	props []cql.QueryProp,
	corpusProps feats.CorpusProps,
	reg *regParser.Document,
) []tagsets.Issue {
	cc := newCorpusContext(corpusProps, reg)
	var ans []tagsets.Issue
	seen := make(map[string]bool)
	for _, prop := range props {
		if !prop.IsPosattr() {
			continue
		}
		issues, err := tagsets.ValidateValue(prop.Name, prop.Value, prop.Literal, cc)
		if err != nil {
			log.Warn().
				Err(err).
				Str("corpus", corpusID).
				Str("attr", prop.Name).
				Msg("failed to resolve tagset scheme, skipping validation")
			continue
		}
		for _, iss := range issues {
			if s := iss.String(); !seen[s] {
				seen[s] = true
				ans = append(ans, iss)
			}
		}
	}
	return ans
}

func corpusPropsOf(conf *cnf.Conf, corpusID string) feats.CorpusProps {
	if conf == nil {
		return feats.CorpusProps{}
	}
	return conf.CorporaProps[corpusID]
}

// addTagsetTools registers tools providing tagset knowledge
func addTagsetTools(srv *server.MCPServer, corpInfoProv *CorpInfoProvider, conf *cnf.Conf) {
	srv.AddTool(
		mcp.NewTool("build_tag_condition",
			mcp.WithDescription(
				"Translate a morphological description in Universal Dependencies terms (UPOS and features) "+
					"into a CQL condition using the tagset of a corpus - e.g. a Czech positional tag "+
					"(`tag=\"N..P4.*\"`) or UD `upos`/`feats` conditions. Prefer this tool to writing "+
					"positional tags manually, as the positions are easy to get wrong."),
			mcp.WithString("corpus_id", mcp.Required(), mcp.Description("An ID of a corpus the condition is for")),
			mcp.WithString("upos", mcp.Description("UD part(s) of speech separated by `|`, e.g. `NOUN` or `NOUN|PROPN`")),
			mcp.WithString("feats", mcp.Description(
				"UD features `Name=Value` separated by `|`, alternative values separated by commas, "+
					"e.g. `Case=Acc,Gen|Number=Plur` (accusative or genitive plural)")),
			mcp.WithString("attr", mcp.Description(
				"Optional target attribute (e.g. `verbtag` for whole verb forms incl. compound ones, `p_tag` "+
					"for a parent token); by default, the main morphological attribute of the corpus is used")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			corpusID := request.GetString("corpus_id", "")
			reg, err := corpInfoProv.GetRegistry(corpusID)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("failed to find corpus", err), nil
			}
			morph, err := tagsets.ParseMorphology(request.GetString("upos", ""), request.GetString("feats", ""))
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid input", err), nil
			}
			res, err := tagsets.BuildCondition(
				morph, request.GetString("attr", ""), newCorpusContext(corpusPropsOf(conf, corpusID), reg))
			if err != nil {
				return mcp.NewToolResultErrorFromErr("failed to build condition", err), nil
			}
			if res.CQL == "" {
				return mcp.NewToolResultError(formatTagsetIssues("No condition could be built:", res.Issues)), nil
			}
			heading := fmt.Sprintf("CQL condition (use it inside a position, e.g. `[%s]`): `%s`", res.CQL, res.CQL)
			if len(res.Issues) == 0 {
				return mcp.NewToolResultText(heading), nil
			}
			return mcp.NewToolResultText(formatTagsetIssues(heading+"\n\nNotes:", res.Issues)), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("tagset_reference",
			mcp.WithDescription(
				"Describe the tagset of a corpus attribute: positions and values of positional tags "+
					"(e.g. `tag`, `verbtag`), Universal Dependencies parts of speech, features or relations"),
			mcp.WithString("corpus_id", mcp.Required(), mcp.Description("An ID of a corpus")),
			mcp.WithString("attr", mcp.Required(), mcp.Description("An attribute, e.g. `tag`, `verbtag`, `upos`, `feats`, `deprel`")),
			mcp.WithNumber("position", mcp.Description("Optional position (1-based) to describe only a single position of a positional tag")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			corpusID := request.GetString("corpus_id", "")
			reg, err := corpInfoProv.GetRegistry(corpusID)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("failed to find corpus", err), nil
			}
			desc, err := tagsets.Describe(
				request.GetString("attr", ""),
				request.GetInt("position", 0),
				newCorpusContext(corpusPropsOf(conf, corpusID), reg),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("failed to describe tagset", err), nil
			}
			return mcp.NewToolResultText(desc), nil
		},
	)
}

// multiValueSep returns a separator of a multivalue attribute
// (registry MULTIVALUE + MULTISEP; Manatee's default separator is `,`)
func multiValueSep(entries regParser.KeyVals) (string, bool) {
	switch strings.ToLower(strings.Trim(entries.Get("MULTIVALUE").Value(), "\"")) {
	case "y", "yes", "1", "true":
	default:
		return "", false
	}
	sep := strings.Trim(entries.Get("MULTISEP").Value(), "\"")
	if sep == "" {
		sep = ","
	}
	return sep, true
}

func hasTagsetErrors(issues []tagsets.Issue) bool {
	for _, iss := range issues {
		if iss.Severity == tagsets.SeverityError {
			return true
		}
	}
	return false
}

func formatTagsetIssues(heading string, issues []tagsets.Issue) string {
	var buff strings.Builder
	buff.WriteString(heading)
	buff.WriteString("\n\n")
	for _, iss := range issues {
		buff.WriteString("- ")
		buff.WriteString(iss.String())
		buff.WriteString("\n")
	}
	return buff.String()
}
