package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"e2e-generator-demo/pkg/types"
	"e2e-generator-demo/pkg/validator"
)

// GeminiClient handles interaction with Google Gemini API
type GeminiClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// GeminiRequest represents the request format for Gemini API
type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
	GenerationConfig GeminiGenerationConfig `json:"generationConfig"`
}

// GeminiContent represents content in Gemini request
type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

// GeminiPart represents a part of content
type GeminiPart struct {
	Text string `json:"text"`
}

// GeminiGenerationConfig configures text generation
type GeminiGenerationConfig struct {
	Temperature     float64 `json:"temperature"`
	TopK           int     `json:"topK"`
	TopP           float64 `json:"topP"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}

// GeminiResponse represents the response from Gemini API
type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
}

// GeminiCandidate represents a response candidate
type GeminiCandidate struct {
	Content GeminiContent `json:"content"`
}

// NewGeminiClient creates a new Gemini API client
func NewGeminiClient(apiKey string) *GeminiClient {
	return &GeminiClient{
		apiKey:  apiKey,
		baseURL: "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent",
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

// GenerateE2ETest generates E2E test using Gemini
func (g *GeminiClient) GenerateE2ETest(context *types.LLMContext) (*types.GeneratedTest, error) {
	// Build prompt
	prompt := g.buildPrompt(context)

	// Create request
	request := &GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: GeminiGenerationConfig{
			Temperature:     0.3,
			TopK:           40,
			TopP:           0.95,
			MaxOutputTokens: 3000,
		},
	}

	// Call API
	response, err := g.callAPI(request)
	if err != nil {
		return nil, fmt.Errorf("failed to call Gemini API: %w", err)
	}

	// Process response
	return g.processResponse(response, context)
}

// buildPrompt creates the prompt for Gemini
func (g *GeminiClient) buildPrompt(context *types.LLMContext) string {
	prompt := `You are an expert Go developer specializing in Kubernetes, edge computing, and test automation with deep knowledge of KubeEdge architecture and Ginkgo/Gomega testing framework.

TASK: Generate comprehensive E2E tests for KubeEdge component.

COMPONENT INFORMATION:
` + fmt.Sprintf(`- Name: %s
- Package: %s  
- Description: %s
- Methods: %v

MISSING TEST COVERAGE:
`, context.Component.Name, context.Component.Package, 
   context.Component.Description, context.Component.Methods)

	for _, gap := range context.CoverageGaps {
		prompt += fmt.Sprintf("- %s (%s priority)\n", gap.MissingTest, gap.Priority)
	}

	prompt += `
REQUIREMENTS:
1. Use Ginkgo/Gomega testing framework
2. Follow KubeEdge E2E test naming conventions (E2E_COMPONENTNAME_*)
3. Include proper setup/teardown in BeforeEach/AfterEach
4. Test both success and failure scenarios
5. Use utilities from tests/e2e/utils/ package
6. Follow the pattern from existing KubeEdge E2E tests

EXAMPLE PATTERN:
` + g.getExamplePattern() + `

OUTPUT: Generate a complete Go test file with comprehensive test cases for the missing coverage areas. 
Include all necessary imports, proper package declaration, and follow Go best practices.
Ensure the code can be compiled and executed successfully.
`

	return prompt
}

// getExamplePattern provides an example test pattern
func (g *GeminiClient) getExamplePattern() string {
	return `
package componentname

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

var _ = GroupDescribe("Component E2E Tests", func() {
	var testTimer *utils.TestTimer
	var testSpecReport ginkgo.SpecReport
	var clientSet clientset.Interface

	ginkgo.BeforeEach(func() {
		clientSet = utils.NewKubeClient(framework.TestContext.KubeConfig)
		testSpecReport = ginkgo.CurrentSpecReport()
		testTimer = utils.CRDTestTimerGroup.NewTestTimer(testSpecReport.LeafNodeText)
	})

	ginkgo.AfterEach(func() {
		testTimer.End()
		testTimer.PrintResult()
		// Cleanup code here
		utils.PrintTestcaseNameandStatus()
	})

	ginkgo.It("E2E_COMPONENT_1: Test description", func() {
		// Test implementation
		gomega.Expect(err).To(gomega.BeNil())
	})
})`
}

// callAPI makes the actual API call to Gemini
func (g *GeminiClient) callAPI(request *GeminiRequest) (*GeminiResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Construct URL with API key
	url := fmt.Sprintf("%s?key=%s", g.baseURL, g.apiKey)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var response GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// processResponse processes the Gemini response and creates GeneratedTest
func (g *GeminiClient) processResponse(response *GeminiResponse, context *types.LLMContext) (*types.GeneratedTest, error) {
	if len(response.Candidates) == 0 {
		return nil, fmt.Errorf("no response candidates received")
	}

	if len(response.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content parts in response")
	}

	content := response.Candidates[0].Content.Parts[0].Text
	
	// Extract Go code from markdown code blocks if present
	content = g.extractCode(content)

	// Validate and fix the generated code
	codeValidator := validator.NewCodeValidator()
	validationResult, err := codeValidator.ValidateGeneratedTest(content, context.Component.Name)
	if err != nil {
		fmt.Printf("⚠️  Validation error: %v\n", err)
		// Continue with original code if validation fails
	} else {
		if !validationResult.IsValid {
			fmt.Printf("⚠️  Generated code has issues, applying fixes...\n")
			for _, err := range validationResult.Errors {
				fmt.Printf("   - Error: %s\n", err)
			}
			for _, warning := range validationResult.Warnings {
				fmt.Printf("   - Warning: %s\n", warning)
			}
			
			// Apply fixes
			fixer := validator.NewTestFixer(context.Component.Name)
			fixedContent, appliedFixes := fixer.FixGeneratedTest(content)
			
			fmt.Printf("✅ Applied fixes:\n")
			for _, fix := range appliedFixes {
				fmt.Printf("   - %s\n", fix)
			}
			
			content = fixedContent
		} else {
			fmt.Printf("✅ Generated code validation passed!\n")
		}
	}

	// Generate filename
	fileName := fmt.Sprintf("%s_generated_test.go", context.Component.Name)

	return &types.GeneratedTest{
		Component: context.Component.Name,
		FileName:  fileName,
		Content:   content,
	}, nil
}

// extractCode extracts Go code from markdown code blocks
func (g *GeminiClient) extractCode(content string) string {
	// Look for ```go or ``` blocks
	lines := strings.Split(content, "\n")
	var result []string
	inCodeBlock := false
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```go") || strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			result = append(result, line)
		}
	}
	
	if len(result) > 0 {
		return strings.Join(result, "\n")
	}
	
	// If no code blocks found, return original content
	return content
}