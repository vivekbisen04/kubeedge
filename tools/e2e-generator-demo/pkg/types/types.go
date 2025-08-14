package types

// ComponentInfo represents a KubeEdge component
type ComponentInfo struct {
	Name        string   `json:"name"`
	Package     string   `json:"package"`
	Path        string   `json:"path"`
	Description string   `json:"description"`
	Methods     []string `json:"methods"`
}

// CoverageGap represents missing test coverage
type CoverageGap struct {
	Component   string   `json:"component"`
	MissingTest string   `json:"missing_test"`
	Description string   `json:"description"`
	Priority    string   `json:"priority"`
}

// LLMContext contains information for LLM prompt
type LLMContext struct {
	Component        ComponentInfo  `json:"component"`
	ExistingPatterns []string       `json:"existing_patterns"`
	CoverageGaps     []CoverageGap  `json:"coverage_gaps"`
	TestFramework    string         `json:"test_framework"`
}

// GeneratedTest represents the output from LLM
type GeneratedTest struct {
	Component string `json:"component"`
	FileName  string `json:"file_name"`
	Content   string `json:"content"`
}

// LLMRequest for DeepSeek API
type LLMRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

// Message for LLM conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMResponse from DeepSeek
type LLMResponse struct {
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice from LLM response
type Choice struct {
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage statistics
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}