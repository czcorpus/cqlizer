package ai

type Conf struct {
	SystemPromptFile       string `json:"systemPromptFile"`
	CustomSystemPromptsDir string `json:"customSystemPromptsDir"`
	APIURL                 string `json:"apiUrl"`
	ModelName              string `json:"modelName"`
	CorporaRegistryDir     string `json:"corporaRegistryDir"`
}
