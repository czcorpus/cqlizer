package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/czcorpus/cqlizer/cql"
	"github.com/sashabaranov/go-openai"
)

type CQLTranslator struct {
	modelURL               string
	systemPrompt           string
	modelName              string
	corpinfo               *CorpInfoProvider
	customSystemPromptsDir string
	tools                  []openai.Tool
}

func NewCQLTRanslator(
	modelURL string,
	systemPrompt string,
	customSystemPromptsDir string,
	modelName string,
	corpusInfo *CorpInfoProvider,
) *CQLTranslator {
	tools := []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "validate_cql",
				Description: "Checks if the CQL syntax is valid. Returns 'valid' or an error message describing the problem.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"query": {"type": "string", "description": "The CQL query to validate"}
					},
					"required": ["query"]
				}`),
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_token_attrs",
				Description: "Provides a list of attributes applicable for token search (word, lemma, tag,...)",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"corpname": {"type": "string", "description": "A corpus to get attributes for"}
					},
					"required": ["corpname"]
				}`),
			},
		},
	}
	return &CQLTranslator{
		modelURL:               modelURL,
		systemPrompt:           systemPrompt,
		customSystemPromptsDir: customSystemPromptsDir,
		modelName:              modelName,
		corpinfo:               corpusInfo,
		tools:                  tools,
	}
}

type validateCQLArgs struct {
	Query string `json:"query"`
}

type corpinfoArgs struct {
	Corpname string `json:"corpname"`
}

func (ct *CQLTranslator) GetSystemPrompt() string {
	return ct.systemPrompt
}

func (ct *CQLTranslator) GetModelName() string {
	return ct.modelName
}

func (ct *CQLTranslator) GetCustomSystemPromptsDir() string {
	return ct.customSystemPromptsDir
}

func (ct *CQLTranslator) GetToolsJSON() (string, error) {
	data, err := json.MarshalIndent(ct.tools, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (ct *CQLTranslator) TranslateToCQLWithPrompt(ctx context.Context, userInput string, customPrompt string) (string, error) {
	return ct.translateToCQLInternal(ctx, userInput, customPrompt)
}

func (ct *CQLTranslator) TranslateToCQL(ctx context.Context, userInput string) (string, error) {
	return ct.translateToCQLInternal(ctx, userInput, ct.systemPrompt)
}

func (ct *CQLTranslator) translateToCQLInternal(ctx context.Context, userInput string, systemPrompt string) (string, error) {
	config := openai.DefaultConfig("")
	config.BaseURL = ct.modelURL
	client := openai.NewClientWithConfig(config)

	messages := []openai.ChatCompletionMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userInput},
	}

	maxIterations := 5 // prevent infinite loops
	for range maxIterations {
		resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:       ct.modelName,
			Messages:    messages,
			Tools:       ct.tools,
			Temperature: 0.1,
		})
		if err != nil {
			return "", fmt.Errorf("completion request failed: %w", err)
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("no choices in response")
		}

		msg := resp.Choices[0].Message

		if len(msg.ToolCalls) == 0 {
			return msg.Content, nil
		}

		messages = append(messages, msg)

		for _, call := range msg.ToolCalls {
			var result string

			switch call.Function.Name {
			case "validate_cql":
				var args validateCQLArgs
				if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
					result = fmt.Sprintf("error parsing arguments: %v", err)

				} else {
					_, err := cql.ParseCQL("", args.Query)
					if err != nil {
						result = fmt.Sprintf("invalid: %v", err)

					} else {
						result = "valid"
					}
				}
			case "get_token_attrs":
				var args corpinfoArgs
				if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
					result = fmt.Sprintf("error parsing arguments: %v", err)

				} else {
					tmp, err := ct.corpinfo.GetAttributes(args.Corpname)
					if err != nil {
						result = fmt.Sprintf("failed to get supported attributes: %s", err)
					}
					result = strings.Join(tmp, ", ")
				}
			default:
				result = fmt.Sprintf("unknown tool: %s", call.Function.Name)
			}

			messages = append(messages, openai.ChatCompletionMessage{
				Role:       "tool",
				Content:    result,
				ToolCallID: call.ID,
			})
		}
	}

	return "", fmt.Errorf("max iterations reached without final response")
}
