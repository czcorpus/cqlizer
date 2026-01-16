// Copyright 2025 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2025 Department of Linguistics,
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

package apiserver

import (
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/czcorpus/cnc-gokit/unireq"
	"github.com/czcorpus/cnc-gokit/uniresp"
	"github.com/czcorpus/cqlizer/eval/feats"
	"github.com/gin-gonic/gin"
)

func (api *apiServer) handleVersion(ctx *gin.Context) {
	uniresp.WriteJSONResponse(ctx.Writer, api.version)
}

func (api *apiServer) handleEvalSimple(ctx *gin.Context) {
	q := ctx.Query("q")
	defaultAttr := ctx.QueryArray("defaultAttr")
	cqlChunks := make([]string, len(defaultAttr))
	for i, da := range defaultAttr {
		cqlChunks[i] = fmt.Sprintf("%s=\"%s\"", da, q)
	}
	q = strings.Join(cqlChunks, " | ")
	api.evaluateRawQuery(ctx, q)

}

func (api *apiServer) handleEvalCQL(ctx *gin.Context) {
	q := ctx.Query("q")
	api.evaluateRawQuery(ctx, q)
}

func (api *apiServer) evaluateRawQuery(ctx *gin.Context, q string) {
	corpname := ctx.Param("corpusId")
	//aligned := ctx.QueryArray("aligned")
	var corpusInfo feats.CorpusProps
	var ok bool
	if corpname != "" {
		corpusInfo, ok = api.conf.CorporaProps[corpname]

		if !ok {
			uniresp.RespondWithErrorJSON(
				ctx, fmt.Errorf("corpus not found"), http.StatusNotFound,
			)
			return
		}

		if ctx.Query("corpusSize") != "" {
			uniresp.RespondWithErrorJSON(
				ctx, fmt.Errorf("cannot specify corpusSize for a concrete corpus"), http.StatusBadRequest,
			)
			return
		}

	} else {
		corpusInfo.Size, ok = unireq.GetURLIntArgOrFail(ctx, "corpusSize", 1000000000)
		if !ok {
			return
		}
		corpusInfo.Lang = ctx.Query("lang")
	}
	charProb := feats.GetCharProbabilityProvider(corpusInfo.Lang)
	queryEval, err := feats.NewQueryEvaluation(q, float64(corpusInfo.Size), 0, 3, charProb)
	if err != nil {
		uniresp.RespondWithErrorJSON(ctx, err, http.StatusInternalServerError)
		return
	}
	predictions := make([]vote, 0, len(api.rfEnsemble))
	for _, md := range api.rfEnsemble {
		pr := md.Predict(queryEval)
		predictions = append(
			predictions,
			vote{
				Votes:  pr.Votes,
				Result: pr.PredictedClass,
			},
		)
	}

	var votesFor int
	for _, pred := range predictions {
		votesFor += pred.Result
	}
	resp := evaluation{
		CorpusSize:  corpusInfo.Size,
		Votes:       predictions,
		IsSlowQuery: votesFor > int(math.Floor(float64(len(api.rfEnsemble))/2)),
		AltCorpus:   corpusInfo.AltCorpus,
	}

	uniresp.WriteJSONResponse(ctx.Writer, resp)
}

type nlToCQLRequest struct {
	UserInput    string `json:"userInput"`
	SystemPrompt string `json:"systemPrompt"`
}

func (api *apiServer) TranslateNLQueryToCQL(ctx *gin.Context) {
	var req nlToCQLRequest
	if err := ctx.BindJSON(&req); err != nil {
		uniresp.RespondWithErrorJSON(ctx, fmt.Errorf("invalid request: %w", err), http.StatusBadRequest)
		return
	}

	// If system prompt is provided in the request, use it temporarily
	// Otherwise, use the default from the translator
	var resp string
	var err error
	if req.SystemPrompt != "" {
		resp, err = api.cqlTranslator.TranslateToCQLWithPrompt(ctx, req.UserInput, req.SystemPrompt)

	} else {
		resp, err = api.cqlTranslator.TranslateToCQL(ctx, req.UserInput)
	}

	if err != nil {
		uniresp.RespondWithErrorJSON(ctx, err, http.StatusInternalServerError)
		return
	}
	ans := map[string]any{"response": resp}
	uniresp.WriteJSONResponse(ctx.Writer, ans)
}

type savePromptRequest struct {
	Content string `json:"content"`
	Name    string `json:"name"`
}

func (api *apiServer) handleSaveSystemPrompt(ctx *gin.Context) {
	var req savePromptRequest
	if err := ctx.BindJSON(&req); err != nil {
		uniresp.RespondWithErrorJSON(ctx, fmt.Errorf("invalid request: %w", err), http.StatusBadRequest)
		return
	}

	if len(req.Content) == 0 {
		uniresp.RespondWithErrorJSON(ctx, fmt.Errorf("system prompt cannot be empty"), http.StatusBadRequest)
		return
	}

	// Create versioned filename with date
	dateStamp := time.Now().Format("2006-01-02")
	var filename string
	if req.Name != "" {
		// Custom name format: custom-name.yyyy-mm-dd.txt
		filename = fmt.Sprintf("%s.%s.txt", req.Name, dateStamp)
	} else {
		// Default format: update.yyyy-mm-dd.txt
		filename = fmt.Sprintf("update.%s.txt", dateStamp)
	}

	promptsDir := api.cqlTranslator.GetCustomSystemPromptsDir()
	filepath := filepath.Join(promptsDir, filename)
	if err := os.WriteFile(filepath, []byte(req.Content), 0644); err != nil {
		uniresp.RespondWithErrorJSON(ctx, fmt.Errorf("failed to save system prompt: %w", err), http.StatusInternalServerError)
		return
	}

	resp := map[string]any{
		"filename": filename,
		"path":     filepath,
		"size":     len(req.Content),
	}
	uniresp.WriteJSONResponse(ctx.Writer, resp)
}

func (api *apiServer) handleLoadSystemPrompt(ctx *gin.Context) {
	filename := ctx.Query("file")

	var prompt string
	var source string

	if filename != "" {
		promptsDir := api.cqlTranslator.GetCustomSystemPromptsDir()
		filepath := filepath.Join(promptsDir, filename)
		content, err := os.ReadFile(filepath)
		if err != nil {
			uniresp.RespondWithErrorJSON(ctx, fmt.Errorf("failed to load prompt file: %w", err), http.StatusInternalServerError)
			return
		}
		prompt = string(content)
		source = filename
	} else {
		// Return the current system prompt from the translator
		prompt = api.cqlTranslator.GetSystemPrompt()
		source = "current translator"
	}

	resp := map[string]any{
		"systemPrompt": prompt,
		"source":       source,
	}
	uniresp.WriteJSONResponse(ctx.Writer, resp)
}

func (api *apiServer) handleLoadDefaultPrompt(ctx *gin.Context) {
	promptsDir := api.cqlTranslator.GetCustomSystemPromptsDir()
	content, err := os.ReadFile(filepath.Join(promptsDir, "sysprompt.txt"))
	if err != nil {
		uniresp.RespondWithErrorJSON(ctx, fmt.Errorf("failed to load default prompt: %w", err), http.StatusInternalServerError)
		return
	}

	resp := map[string]any{
		"systemPrompt": string(content),
		"source":       "sysprompt.txt",
	}
	uniresp.WriteJSONResponse(ctx.Writer, resp)
}

func (api *apiServer) handleListPrompts(ctx *gin.Context) {
	promptsDir := api.cqlTranslator.GetCustomSystemPromptsDir()
	entries, err := os.ReadDir(promptsDir)
	if err != nil {
		uniresp.RespondWithErrorJSON(ctx, fmt.Errorf("failed to read prompts directory: %w", err), http.StatusInternalServerError)
		return
	}

	files := make([]map[string]any, 0)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".txt") {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			files = append(files, map[string]any{
				"name":    entry.Name(),
				"size":    info.Size(),
				"modTime": info.ModTime().Format(time.RFC3339),
			})
		}
	}

	resp := map[string]any{
		"files": files,
	}
	uniresp.WriteJSONResponse(ctx.Writer, resp)
}

func (api *apiServer) handleGetTools(ctx *gin.Context) {
	toolsJSON, err := api.cqlTranslator.GetToolsJSON()
	if err != nil {
		uniresp.RespondWithErrorJSON(ctx, fmt.Errorf("failed to get tools: %w", err), http.StatusInternalServerError)
		return
	}

	resp := map[string]any{
		"tools": toolsJSON,
	}
	uniresp.WriteJSONResponse(ctx.Writer, resp)
}
