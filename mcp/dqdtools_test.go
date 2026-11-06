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
	"maps"
	"os"
	"slices"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writtenLayerInfoJSON(t *testing.T) string {
	data, err := os.ReadFile("../dqd/testdata/layerinfo_written.json")
	require.NoError(t, err)
	return string(data)
}

func toolRequest(args map[string]any) mcp.CallToolRequest {
	var req mcp.CallToolRequest
	req.Params.Arguments = args
	return req
}

// callTool calls a handler and returns its text output and error flag
func callTool(t *testing.T, handler server.ToolHandlerFunc, args map[string]any) (string, bool) {
	res, err := handler(context.Background(), toolRequest(args))
	require.NoError(t, err)
	require.Len(t, res.Content, 1)
	text, ok := res.Content[0].(mcp.TextContent)
	require.True(t, ok)
	return text.Text, res.IsError
}

func TestAddDQDTools(t *testing.T) {
	registered := func(enabled bool) []string {
		srv := server.NewMCPServer("test", "0")
		addDQDTools(srv, enabled)
		return slices.Sorted(maps.Keys(srv.ListTools()))
	}
	assert.Empty(t, registered(false))
	assert.Equal(t,
		[]string{DQDCorpusStructureToolName, DQDTranslateCQLToolName, DQDValidateCQLToolName},
		registered(true))
}

func TestDQDCorpusStructure(t *testing.T) {
	out, isErr := callTool(t, handleDQDCorpusStructure, map[string]any{
		"corpus_id": "written", "layer_info": writtenLayerInfoJSON(t)})
	assert.False(t, isErr)
	assert.Contains(t, out, "# Structure of the corpus written\n")
	assert.Contains(t, out, "### word\n\nlabel: an alias of `form`\n")

	out, isErr = callTool(t, handleDQDCorpusStructure, map[string]any{"corpus_id": "written"})
	assert.True(t, isErr)
	assert.Contains(t, out, "missing layer_info")

	out, isErr = callTool(t, handleDQDCorpusStructure, map[string]any{
		"corpus_id": "written", "layer_info": `{"layer": `})
	assert.True(t, isErr)
	assert.Contains(t, out, "invalid layer_info")
}

func TestDQDValidateCQL(t *testing.T) {
	li := writtenLayerInfoJSON(t)
	out, isErr := callTool(t, handleDQDValidateCQL, map[string]any{
		"corpus_id": "written", "q": `[lemma="dog" & upos="NOUN"]`, "layer_info": li})
	assert.False(t, isErr)
	assert.Equal(t, "query OK", out)

	out, isErr = callTool(t, handleDQDValidateCQL, map[string]any{
		"corpus_id": "written", "q": `[upos="SUBST"]`, "layer_info": li})
	assert.True(t, isErr)
	assert.Contains(t, out, "`SUBST` is not a valid value of attribute `upos`")

	out, isErr = callTool(t, handleDQDValidateCQL, map[string]any{
		"corpus_id": "written", "q": `"dog"`})
	assert.True(t, isErr)
	assert.Contains(t, out, "missing layer_info")
}

func TestDQDTranslateCQL(t *testing.T) {
	expected := "Segment s\nToken@s t1\n    form = \"dog\"\n"

	// the corpus need not be configured anywhere
	out, isErr := callTool(t, handleDQDTranslateCQL, map[string]any{
		"corpus_id": "other", "q": `"dog"`, "layer_info": writtenLayerInfoJSON(t)})
	assert.False(t, isErr)
	assert.Contains(t, out, expected)

	// no lookup of a configured LCP corpus without layer_info
	out, isErr = callTool(t, handleDQDTranslateCQL, map[string]any{"corpus_id": "written", "q": `"dog"`})
	assert.True(t, isErr)
	assert.Contains(t, out, "missing layer_info")

	out, isErr = callTool(t, handleDQDTranslateCQL, map[string]any{
		"corpus_id": "written", "q": `"dog"`, "layer_info": `{"layer": `})
	assert.True(t, isErr)
	assert.Contains(t, out, "invalid layer_info")
}
