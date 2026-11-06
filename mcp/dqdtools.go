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
	"strings"

	"github.com/czcorpus/cqlizer/dqd"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// DQD tools - CQL for LCP corpora, translated to DQD. Unlike the other
// tools, they don't look corpora up anywhere: the caller always passes the
// corpus layer info (LCP layer info JSON) along with the request, so the
// caller alone decides which corpora exist and which version of their
// layer info applies. The corpus ID is used only as a label in the output.
//
// The tools are meant to be called by an application (e.g. NLQL) rather
// than directly by a model, which is not expected to handle layer infos.

const (
	DQDCorpusStructureToolName = "dqd_corpus_structure"
	DQDValidateCQLToolName     = "dqd_validate_cql"
	DQDTranslateCQLToolName    = "dqd_translate_cql"
)

var (
	dqdCorpusIDArg = mcp.WithString(
		"corpus_id",
		mcp.Required(),
		mcp.Description("An ID of the LCP corpus (used only as a label in the output)"),
	)
	dqdQueryArg = mcp.WithString("q", mcp.Required(), mcp.Description("CQL query string"))
)

func dqdLayerInfoArg(required bool) mcp.ToolOption {
	opts := []mcp.PropertyOption{mcp.Description("LCP layer info (JSON) of the corpus")}
	if required {
		opts = append(opts, mcp.Required())
	}
	return mcp.WithString("layer_info", opts...)
}

// layerInfoFromRequest parses the layer_info argument. If the argument is
// missing, li is nil.
func layerInfoFromRequest(request mcp.CallToolRequest) (*dqd.LayerInfo, error) {
	raw := request.GetString("layer_info", "")
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	return dqd.ParseLayerInfo([]byte(raw))
}

// requireLayerInfo is layerInfoFromRequest for tools which cannot work
// without the layer info. A nil result means a failure described by errRes.
func requireLayerInfo(request mcp.CallToolRequest) (li *dqd.LayerInfo, errRes *mcp.CallToolResult) {
	li, err := layerInfoFromRequest(request)
	if err != nil {
		return nil, mcp.NewToolResultErrorFromErr("invalid layer_info", err)
	}
	if li == nil {
		return nil, mcp.NewToolResultError("missing layer_info")
	}
	return li, nil
}

func handleDQDCorpusStructure(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	li, errRes := requireLayerInfo(request)
	if li == nil {
		return errRes, nil
	}
	info, err := lcpCorpusInfo(request.GetString("corpus_id", ""), li)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to describe corpus", err), nil
	}
	return mcp.NewToolResultText(info.AsMarkdown()), nil
}

func handleDQDValidateCQL(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	li, errRes := requireLayerInfo(request)
	if li == nil {
		return errRes, nil
	}
	issues, err := validateLCPQuery(request.GetString("corpus_id", ""), li, request.GetString("q", ""))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to validate query", err), nil
	}
	if len(issues) > 0 {
		return mcp.NewToolResultError(strings.Join(issues, "\n")), nil
	}
	return mcp.NewToolResultText("query OK"), nil
}

func handleDQDTranslateCQL(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	li, errRes := requireLayerInfo(request)
	if li == nil {
		return errRes, nil
	}
	ans, err := translateLCPQuery(request.GetString("corpus_id", ""), li, request.GetString("q", ""))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to translate query", err), nil
	}
	return mcp.NewToolResultText(ans), nil
}

// addDQDTools registers the DQD tools (if dqdToolsEnabled)
func addDQDTools(srv *server.MCPServer, dqdToolsEnabled bool) {
	if !dqdToolsEnabled {
		return
	}
	srv.AddTool(
		mcp.NewTool(DQDCorpusStructureToolName,
			mcp.WithDescription(
				"Describe an LCP corpus in terms of CQL as translated to DQD (attributes and "+
					"structures usable in CQL, values of categorical attributes)"),
			dqdCorpusIDArg,
			dqdLayerInfoArg(true),
		),
		handleDQDCorpusStructure,
	)
	srv.AddTool(
		mcp.NewTool(DQDValidateCQLToolName,
			mcp.WithDescription(
				"Validate a CQL query for an LCP corpus - check it can be translated to DQD "+
					"and that plain values of categorical attributes are valid"),
			dqdCorpusIDArg,
			dqdQueryArg,
			dqdLayerInfoArg(true),
		),
		handleDQDValidateCQL,
	)
	srv.AddTool(
		mcp.NewTool(DQDTranslateCQLToolName,
			mcp.WithDescription("Translate a CQL query for an LCP corpus to DQD (the query language of LCP)"),
			dqdCorpusIDArg,
			dqdQueryArg,
			dqdLayerInfoArg(true),
		),
		handleDQDTranslateCQL,
	)
}
