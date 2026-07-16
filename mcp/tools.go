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
	"math"
	"net/http"
	"strings"

	"github.com/czcorpus/cnc-gokit/mcptools"
	"github.com/czcorpus/cqlizer/ai"
	"github.com/czcorpus/cqlizer/apiserver"
	"github.com/czcorpus/cqlizer/cnf"
	"github.com/czcorpus/cqlizer/cql"
	"github.com/czcorpus/cqlizer/eval/feats"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog/log"

	regParser "github.com/czcorpus/rexplorer/parser"
)

type MCPMode string

const (
	ModeStdio MCPMode = "stdio"
	ModeHttp  MCPMode = "http"
)

func (m MCPMode) Validate() error {
	if m != ModeStdio && m != ModeHttp {
		return fmt.Errorf("invalid mcpMode: %s", m)
	}
	return nil
}

func (m MCPMode) String() string {
	return string(m)
}

func normVersionInfo(v string) string {
	return strings.TrimLeft(strings.Trim(v, "'"), "v")
}

func posattrExists(name string, reg *regParser.Document) bool {
	for _, p := range reg.PosAttrs {
		if p.Name == name {
			return true
		}
	}
	return false
}

func structExists(name string, reg *regParser.Document) bool {
	for _, s := range reg.Structures {
		if s.Name == name {
			return true
		}
	}
	return false
}

func structAttrExists(structName, name string, reg *regParser.Document) bool {
	for _, s := range reg.Structures {
		if s.Name == structName {
			for _, sa := range s.Attrs {
				if sa.Name == name {
					return true
				}
			}
		}
	}
	return false
}

func Init(
	mode MCPMode,
	listenAddress string,
	version apiserver.VersionInfo,
	corpInfoProv *ai.CorpInfoProvider,
	rfEnsemble []apiserver.EnsembleModel,
	corpusStructToolEnabled bool,
	conf *cnf.Conf,
) {

	srv := server.NewMCPServer("cqlizer-mcp", version.Version, server.WithHooks(mcptools.DefaultLoggingHooks()))

	srv.AddTool(
		mcp.NewTool("cql_validate",
			mcp.WithDescription("Validate CQL syntax, check attributes support"),
			mcp.WithString("corpus_id", mcp.Required(), mcp.Description("An ID of a corpus to apply CQL to")),
			mcp.WithString("q", mcp.Required(), mcp.Description("CQL query string")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

			corpusID := request.GetString("corpus_id", "")
			corpInfo, err := corpInfoProv.GetRegistry(corpusID)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("failed to find corpus", err), nil
			}

			query, err := cql.ParseCQL("", request.GetString("q", ""))
			if err != nil {
				return mcp.NewToolResultErrorFromErr("failed to parse query", err), nil
			}

			qProps := query.ExtractProps()
			for _, prop := range qProps {
				if prop.IsPosattr() && !posattrExists(prop.Name, corpInfo) {
					return mcp.NewToolResultErrorf("unsupported positional attribute `%s`", prop.Name), nil
				}
				if prop.IsStructure() && !structExists(prop.Structure, corpInfo) {
					return mcp.NewToolResultErrorf("unsupported structure <%s>", prop.Structure), nil
				}
				if prop.IsStructAttr() && !structAttrExists(prop.Structure, prop.Name, corpInfo) {
					return mcp.NewToolResultErrorf("unsupported structural attribute <%s %s=\"...\">", prop.Structure, prop.Name), nil
				}
			}

			return mcp.NewToolResultText("query OK"), nil
		},
	)

	if corpusStructToolEnabled {
		srv.AddTool(
			mcp.NewTool("get_corpus_structure",
				mcp.WithDescription("Get all attributes and structures of a specified corpus"),
				mcp.WithString("corpus_id", mcp.Required(), mcp.Description("An ID of a corpus to apply CQL to")),
			),
			func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				corpusID := request.GetString("corpus_id", "")
				registry, err := corpInfoProv.GetRegistry(corpusID)
				if err != nil {
					return mcp.NewToolResultErrorFromErr("failed to find corpus", err), nil
				}

				ans := corpusInfo{
					Name:                 string(registry.GetProperty("NAME").Value()),
					Info:                 string(registry.GetProperty("INFO").Value()),
					TagsetDoc:            string(registry.GetProperty("TAGSETDOC").Value()),
					PositionalAttributes: make([]attr, 0, 20),
					Structures:           make([]structure, 0, 20),
				}
				if ans.Name == "" {
					ans.Name = "??"
				}
				for _, pa := range registry.PosAttrs {
					ans.PositionalAttributes = append(
						ans.PositionalAttributes,
						attr{
							Name:    pa.Name,
							Label:   string(pa.Entries.Get("LABEL").Value()),
							AttrDoc: string(pa.Entries.Get("ATTRDOC").Value()),
						},
					)
				}
				for _, s := range registry.Structures {
					strct := structure{
						Name:       s.Name,
						Attributes: make([]attr, 0, 10),
					}
					for _, sa := range s.Attrs {
						strct.Attributes = append(
							strct.Attributes,
							attr{
								Name:    sa.Name,
								Label:   string(sa.GetProperty("LABEL")),
								AttrDoc: string(sa.GetProperty("ATTRDOC")),
							},
						)
					}
					ans.Structures = append(ans.Structures, strct)
				}

				return mcp.NewToolResultText(ans.AsMarkdown()), nil
			},
		)
	}

	if len(rfEnsemble) > 0 {
		srv.AddTool(
			mcp.NewTool("cql_eval_complexity",
				mcp.WithDescription("Evaluate query as either fast or slow."),
				mcp.WithString("corpus_id", mcp.Required(), mcp.Description("An ID of a corpus to apply CQL to")),
				mcp.WithInteger("corpus_size", mcp.Description("Corpus size (in tokens), if not set, the tool will try to find the size")),
				mcp.WithString("q", mcp.Required(), mcp.Description("CQL query string")),
			),
			func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

				corpusID := request.GetString("corpus_id", "")
				var corpusProps feats.CorpusProps
				if len(conf.CorporaProps) > 0 {
					corpusProps = conf.CorporaProps[corpusID]
				}
				corpusSize := request.GetInt("corpus_size", corpusProps.Size)

				if corpusSize <= 0 {
					return mcp.NewToolResultError("failed to find corpus size"), nil
				}

				charProb := feats.GetCharProbabilityProvider(corpusProps.Lang)
				queryEval, err := feats.NewQueryEvaluation(request.GetString("q", ""), float64(corpusSize), 0, 3, charProb)
				if err != nil {
					return mcp.NewToolResultErrorFromErr("failed to extract CQL query features", err), nil
				}
				predictions := make(apiserver.VoteList, 0, len(rfEnsemble))
				for _, md := range rfEnsemble {
					pr := md.Predict(queryEval)
					predictions = append(
						predictions,
						apiserver.Vote{
							Votes:  pr.Votes,
							Result: pr.PredictedClass,
						},
					)
				}

				var votesFor int
				for _, pred := range predictions {
					votesFor += pred.Result
				}

				isSlowQuery := votesFor > int(math.Floor(float64(len(rfEnsemble))/2))
				if isSlowQuery {
					if corpusProps.AltCorpus != "" {
						return mcp.NewToolResultText(
							fmt.Sprintf("The query is possibly slow. It is suggested to use corpus %s which is smaller but similar in terms of data.", corpusProps.AltCorpus),
						), nil
					}
					return mcp.NewToolResultText("The query is possibly slow."), nil
				}
				return mcp.NewToolResultText("The query should run fast."), nil
			},
		)
	}

	switch mode {
	case ModeStdio:
		log.Info().Msg("running in stdio mode")
		// Start the stdio server
		if err := server.ServeStdio(srv); err != nil {
			fmt.Printf("Server error: %v\n", err)
		}

	case ModeHttp:
		log.Info().Msg("running in HTTP mode")
		httpServer := server.NewStreamableHTTPServer(
			srv,
			server.WithEndpointPath("/mcp"),
			server.WithHTTPContextFunc(mcptools.WithClientIPContext),
		)

		mux := http.NewServeMux()
		mux.Handle("/mcp", httpServer)

		log.Info().Msgf("MQuery MCP (Streamable HTTP) listening on %s", listenAddress)
		if err := http.ListenAndServe(fmt.Sprintf("%s", listenAddress), mux); err != nil {
			log.Fatal().Err(err).Send()
		}
	default:
		log.Fatal().Str("mode", mode.String()).Msg("invalid mode")
	}

}
