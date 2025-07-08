/*
Copyright 2025 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"time"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type KubeEdgeTestGenerator struct {
	client    *genai.Client
	templates map[string]string
}

func NewKubeEdgeTestGenerator(apiKey string) *KubeEdgeTestGenerator {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		panic(fmt.Sprintf("Failed to create Gemini client: %v", err))
	}

	templates := map[string]string{
		"gomonkey":  loadGoMonkeyTemplate(),
		"ginkgo":    loadGinkgoTemplate(),
		"standard":  loadStandardTemplate(),
	}

	return &KubeEdgeTestGenerator{
		client:    client,
		templates: templates,
	}
}

// GenerateTests generates KubeEdge-compliant tests using LLM
func (ktg *KubeEdgeTestGenerator) GenerateTests(ctx context.Context, filePath string, functions []FunctionInfo, previousError error) (string, error) {
	// Read the original file to understand context
	originalContent, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read original file: %v", err)
	}

	// Extract package name and imports
	packageName, imports := ktg.extractPackageInfo(string(originalContent))

	// Determine the appropriate test type based on KubeEdge patterns
	testType := ktg.determineKubeEdgeTestType(filePath, functions, string(originalContent))

	// Build KubeEdge-specific prompt
	prompt := ktg.buildKubeEdgePrompt(filePath, string(originalContent), functions, testType, previousError)

	// Generate with Gemini AI
	testContent, err := ktg.generateWithGemini(ctx, prompt, testType)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %v", err)
	}

	// Clean up and format the generated code
	finalTestContent := ktg.cleanupGeneratedCode(testContent, packageName, imports, testType)

	return finalTestContent, nil
}

// determineKubeEdgeTestType determines what kind of test pattern to use
func (ktg *KubeEdgeTestGenerator) determineKubeEdgeTestType(filePath string, functions []FunctionInfo, content string) string {
	// Check if we need gomonkey mocking
	if ktg.needsGoMonkey(functions, content) {
		return "gomonkey"
	}

	// Check if this is an e2e or integration test
	if strings.Contains(filePath, "e2e") || strings.Contains(filePath, "integration") || ktg.needsGinkgo(filePath) {
		return "ginkgo"
	}

	// Default to standard Go testing
	return "standard"
}

// needsGoMonkey checks if functions require mocking using gomonkey
func (ktg *KubeEdgeTestGenerator) needsGoMonkey(functions []FunctionInfo, content string) bool {
	// Check for common patterns that require mocking in KubeEdge
	mockPatterns := []string{
		// External function calls
		"exec.Command", "cmd.Exec", "cmd.CombinedOutput",
		// File operations
		"os.Stat", "os.Open", "os.ReadFile", "os.WriteFile", "os.Remove",
		// Kubernetes client operations
		"client.Get", "client.Create", "client.Update", "client.Delete",
		// Database operations (Beego ORM)
		"orm.RegisterDriver", "orm.RegisterDataBase", "orm.NewOrmUsingDB",
		// Network operations
		"http.Get", "http.Post", "net.Dial",
		// System operations
		"syscall", "runtime",
	}

	for _, pattern := range mockPatterns {
		if strings.Contains(content, pattern) {
			return true
		}
	}

	// Check function complexity - complex functions often need mocking
	for _, fn := range functions {
		if ktg.isFunctionComplex(fn) {
			return true
		}
	}

	return false
}

// needsGinkgo determines if Ginkgo framework should be used
func (ktg *KubeEdgeTestGenerator) needsGinkgo(filePath string) bool {
	ginkgoPatterns := []string{
		"e2e", "integration", "test/", 
		"suite", "behavior", "scenario",
	}

	for _, pattern := range ginkgoPatterns {
		if strings.Contains(strings.ToLower(filePath), pattern) {
			return true
		}
	}
	return false
}

// isFunctionComplex checks if a function is complex enough to need mocking
func (ktg *KubeEdgeTestGenerator) isFunctionComplex(fn FunctionInfo) bool {
	content := fn.Content
	
	// Count complexity indicators
	complexityIndicators := 0
	
	// Multiple if statements
	if strings.Count(content, "if ") > 2 {
		complexityIndicators++
	}
	
	// Error handling
	if strings.Contains(content, "error") && strings.Contains(content, "return") {
		complexityIndicators++
	}
	
	// External calls
	if strings.Contains(content, ".") && strings.Contains(content, "(") {
		complexityIndicators++
	}
	
	// Long functions (>10 lines)
	if strings.Count(content, "\n") > 10 {
		complexityIndicators++
	}

	return complexityIndicators >= 2
}

// buildKubeEdgePrompt creates a comprehensive prompt for test generation
func (ktg *KubeEdgeTestGenerator) buildKubeEdgePrompt(filePath string, originalContent string, functions []FunctionInfo, testType string, previousError error) string {
	var prompt strings.Builder

	prompt.WriteString("You are generating unit tests for KubeEdge, a Kubernetes edge computing framework.\n\n")
	
	// Add error context if this is a retry
	if previousError != nil {
		prompt.WriteString("IMPORTANT - PREVIOUS ATTEMPT FAILED:\n")
		prompt.WriteString(fmt.Sprintf("Previous error: %v\n", previousError))
		prompt.WriteString("Please fix the issues mentioned above.\n\n")
	}

	prompt.WriteString("CRITICAL KubeEdge Testing Requirements:\n")
	prompt.WriteString("1. Use Go's built-in testing package and github.com/stretchr/testify/assert\n")
	prompt.WriteString("2. For mocking, use ONLY gomonkey v2 patterns (NOT gomock)\n")
	prompt.WriteString("3. Follow KubeEdge's existing test patterns exactly\n")
	prompt.WriteString("4. Generate working, compilable Go code\n")
	prompt.WriteString("5. Include proper imports and package declaration\n")
	prompt.WriteString("6. Use table-driven tests where appropriate\n")
	prompt.WriteString("7. Test both success and error cases\n\n")

	// Add test type specific instructions
	switch testType {
	case "gomonkey":
		prompt.WriteString("GOMONKEY MOCKING REQUIREMENTS:\n")
		prompt.WriteString("- Use github.com/agiledragon/gomonkey/v2\n")
		prompt.WriteString("- Pattern: patches := gomonkey.NewPatches() / defer patches.Reset()\n")
		prompt.WriteString("- Mock functions: patches.ApplyFunc(originalFunc, mockFunc)\n")
		prompt.WriteString("- Mock methods: patches.ApplyMethod(reflect.TypeOf(obj), \"Method\", mockFunc)\n")
		prompt.WriteString("- Always clean up with defer patches.Reset()\n\n")
	case "ginkgo":
		prompt.WriteString("GINKGO BDD REQUIREMENTS:\n")
		prompt.WriteString("- Use github.com/onsi/ginkgo/v2 and github.com/onsi/gomega\n")
		prompt.WriteString("- Structure: Describe() -> Context() -> It()\n")
		prompt.WriteString("- Use BeforeEach/AfterEach for setup/cleanup\n")
		prompt.WriteString("- Use Expect() assertions from gomega\n\n")
	default:
		prompt.WriteString("STANDARD GO TESTING REQUIREMENTS:\n")
		prompt.WriteString("- Use testing.T and assert from testify\n")
		prompt.WriteString("- Create table-driven tests when testing multiple scenarios\n")
		prompt.WriteString("- Use t.Run() for subtests\n\n")
	}

	// Add KubeEdge context
	prompt.WriteString("KubeEdge Component Context:\n")
	component := ktg.identifyKubeEdgeComponent(filePath)
	switch component {
	case "keadm":
		prompt.WriteString("- This is keadm (KubeEdge admin CLI)\n")
		prompt.WriteString("- Focus on CLI command testing, flag validation, and command execution\n")
		prompt.WriteString("- Test configuration parsing and validation\n")
		prompt.WriteString("- Mock external command execution and file operations\n")
	case "cloud":
		prompt.WriteString("- This is a cloud component\n")
		prompt.WriteString("- Test cloud controller logic and Kubernetes API interactions\n")
		prompt.WriteString("- Mock Kubernetes client operations\n")
		prompt.WriteString("- Test webhook handlers and admission controllers\n")
	case "edge":
		prompt.WriteString("- This is an edge component\n")
		prompt.WriteString("- Test edge-specific functionality and device management\n")
		prompt.WriteString("- Mock hardware interactions and message routing\n")
		prompt.WriteString("- Test protocol handlers and device twin operations\n")
	case "pkg":
		prompt.WriteString("- This is a shared package component\n")
		prompt.WriteString("- Test utility functions and common libraries\n")
		prompt.WriteString("- Focus on unit testing with good edge case coverage\n")
	}
	prompt.WriteString("\n")

	// Add original file context
	prompt.WriteString(fmt.Sprintf("Original file: %s\n\n", filePath))
	prompt.WriteString("Original file content for context:\n")
	prompt.WriteString("```go\n")
	prompt.WriteString(originalContent)
	prompt.WriteString("\n```\n\n")

	// Add functions to test
	prompt.WriteString("Generate comprehensive unit tests for these functions:\n")
	for i, fn := range functions {
		prompt.WriteString(fmt.Sprintf("\n%d. Function: %s\n", i+1, fn.Name))
		prompt.WriteString(fmt.Sprintf("   Exported: %t\n", fn.IsExported))
		prompt.WriteString(fmt.Sprintf("   Has existing tests: %t\n", fn.HasTests))
		prompt.WriteString("   Function code:\n")
		prompt.WriteString("   ```go\n")
		prompt.WriteString("   " + strings.ReplaceAll(fn.Content, "\n", "\n   "))
		prompt.WriteString("\n   ```\n")
	}

	prompt.WriteString("\nIMPORTANT OUTPUT REQUIREMENTS:\n")
	prompt.WriteString("1. Generate ONLY the complete Go test file content\n")
	prompt.WriteString("2. Start with proper package declaration and imports\n")
	prompt.WriteString("3. Include all necessary test functions\n")
	prompt.WriteString("4. Ensure all code compiles and runs without errors\n")
	prompt.WriteString("5. Use descriptive test names that explain what is being tested\n")
	prompt.WriteString("6. Include both positive and negative test cases\n")
	prompt.WriteString("7. Add helpful comments explaining complex test scenarios\n")

	return prompt.String()
}

// generateWithGemini calls Gemini API to generate test content (with detailed logging)
func (ktg *KubeEdgeTestGenerator) generateWithGemini(ctx context.Context, prompt string, testType string) (string, error) {
	model := ktg.client.GenerativeModel("gemini-1.5-flash")
	
	// Configure model for code generation
	model.SetTemperature(0.3) // Lower temperature for more consistent code
	model.SetTopK(40)
	model.SetTopP(0.95)
	
	// Log request details
	ktg.logAPIRequest(prompt, testType)
	
	// Add timeout context (2 minutes instead of default)
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	
	startTime := time.Now()
	resp, err := model.GenerateContent(timeoutCtx, genai.Text(prompt))
	duration := time.Since(startTime)
	
	fmt.Printf("⏱️ API Call Duration: %v\n", duration)
	
	// Log response details
	ktg.logAPIResponse(resp, err)
	
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %v", err)
	}

	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no content generated")
	}

	generatedCode := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			generatedCode += string(text)
		}
	}

	if generatedCode == "" {
		return "", fmt.Errorf("empty content generated")
	}

	fmt.Printf("✅ Successfully extracted %d characters of generated code\n", len(generatedCode))
	return generatedCode, nil
}

// cleanupGeneratedCode cleans and formats the generated test code
func (ktg *KubeEdgeTestGenerator) cleanupGeneratedCode(content string, packageName string, imports string, testType string) string {
	// Remove markdown code blocks if present
	content = regexp.MustCompile("```go\n?").ReplaceAllString(content, "")
	content = regexp.MustCompile("```\n?").ReplaceAllString(content, "")
	
	// Remove any leading/trailing whitespace
	content = strings.TrimSpace(content)
	
	// Ensure proper package declaration
	if !strings.HasPrefix(content, "package ") {
		if packageName != "" {
			content = fmt.Sprintf("package %s\n\n%s", packageName, content)
		}
	}
	
	// Add KubeEdge copyright header
	header := ktg.generateKubeEdgeHeader()
	if !strings.Contains(content, "Copyright") {
		content = header + "\n\n" + content
	}
	
	// Ensure proper imports
	content = ktg.ensureProperImports(content, testType)
	
	return content
}

// extractPackageInfo extracts package name and imports from source file
func (ktg *KubeEdgeTestGenerator) extractPackageInfo(content string) (string, string) {
	lines := strings.Split(content, "\n")
	var packageName string
	var imports strings.Builder
	
	inImportBlock := false
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		if strings.HasPrefix(trimmed, "package ") {
			packageName = strings.TrimSpace(strings.TrimPrefix(trimmed, "package "))
		}
		
		if trimmed == "import (" {
			inImportBlock = true
			imports.WriteString(line + "\n")
			continue
		}
		
		if inImportBlock {
			imports.WriteString(line + "\n")
			if trimmed == ")" {
				break
			}
		}
		
		if strings.HasPrefix(trimmed, "import ") && !inImportBlock {
			imports.WriteString(line + "\n")
		}
	}
	
	return packageName, imports.String()
}

// identifyKubeEdgeComponent identifies which KubeEdge component a file belongs to
func (ktg *KubeEdgeTestGenerator) identifyKubeEdgeComponent(filePath string) string {
	if strings.Contains(filePath, "keadm/") {
		return "keadm"
	}
	if strings.Contains(filePath, "cloud/") {
		return "cloud"
	}
	if strings.Contains(filePath, "edge/") {
		return "edge"
	}
	if strings.Contains(filePath, "pkg/") {
		return "pkg"
	}
	return "unknown"
}

// generateKubeEdgeHeader generates the standard KubeEdge copyright header
func (ktg *KubeEdgeTestGenerator) generateKubeEdgeHeader() string {
	return `/*
Copyright 2025 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/`
}

// ensureProperImports ensures the test file has all necessary imports
func (ktg *KubeEdgeTestGenerator) ensureProperImports(content string, testType string) string {
	requiredImports := []string{
		"testing",
		"github.com/stretchr/testify/assert",
	}
	
	switch testType {
	case "gomonkey":
		requiredImports = append(requiredImports, 
			"github.com/agiledragon/gomonkey/v2",
			"reflect",
		)
	case "ginkgo":
		requiredImports = append(requiredImports,
			"github.com/onsi/ginkgo/v2",
			"github.com/onsi/gomega",
		)
	}
	
	// Check which imports are missing and add them
	for _, imp := range requiredImports {
		if !strings.Contains(content, imp) {
			content = ktg.addImport(content, imp)
		}
	}
	
	return content
}

// addImport adds an import to the test file
func (ktg *KubeEdgeTestGenerator) addImport(content string, importPath string) string {
	lines := strings.Split(content, "\n")
	
	for i, line := range lines {
		if strings.Contains(line, "import (") {
			// Add to existing import block
			lines = append(lines[:i+1], append([]string{fmt.Sprintf("\t\"%s\"", importPath)}, lines[i+1:]...)...)
			return strings.Join(lines, "\n")
		}
		
		if strings.HasPrefix(strings.TrimSpace(line), "import ") && !strings.Contains(line, "(") {
			// Convert single import to block and add
			lines[i] = "import ("
			lines = append(lines[:i+1], append([]string{
				fmt.Sprintf("\t\"%s\"", strings.Trim(strings.TrimPrefix(strings.TrimSpace(line), "import "), "\"")),
				fmt.Sprintf("\t\"%s\"", importPath),
				")",
			}, lines[i+1:]...)...)
			return strings.Join(lines, "\n")
		}
	}
	
	// Add new import block after package declaration
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "package ") {
			lines = append(lines[:i+1], append([]string{
				"",
				"import (",
				fmt.Sprintf("\t\"%s\"", importPath),
				")",
				"",
			}, lines[i+1:]...)...)
			return strings.Join(lines, "\n")
		}
	}
	
	return content
}

// logAPIRequest logs the API request details
func (ktg *KubeEdgeTestGenerator) logAPIRequest(prompt string, testType string) {
	fmt.Printf("🔍 API Request Details:\n")
	fmt.Printf("   Model: gemini-1.5-flash\n")
	fmt.Printf("   Test Type: %s\n", testType)
	fmt.Printf("   Prompt Length: %d characters\n", len(prompt))
	fmt.Printf("   Prompt Preview (first 500 chars):\n")
	if len(prompt) > 500 {
		fmt.Printf("   %s...\n", prompt[:500])
	} else {
		fmt.Printf("   %s\n", prompt)
	}
	fmt.Printf("   Temperature: 0.3\n")
	fmt.Printf("   TopK: 40\n")
	fmt.Printf("   TopP: 0.95\n")
	fmt.Printf("🚀 Sending request to Gemini API...\n")
}

// logAPIResponse logs the API response details
func (ktg *KubeEdgeTestGenerator) logAPIResponse(resp *genai.GenerateContentResponse, err error) {
	if err != nil {
		fmt.Printf("❌ API Error Details:\n")
		fmt.Printf("   Error Type: %T\n", err)
		fmt.Printf("   Error Message: %v\n", err)
		return
	}

	fmt.Printf("✅ API Response Details:\n")
	fmt.Printf("   Candidates Count: %d\n", len(resp.Candidates))
	
	if len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]
		fmt.Printf("   Content Parts: %d\n", len(candidate.Content.Parts))
		
		totalLength := 0
		for i, part := range candidate.Content.Parts {
			if text, ok := part.(genai.Text); ok {
				partLength := len(string(text))
				totalLength += partLength
				fmt.Printf("   Part %d Length: %d characters\n", i+1, partLength)
			}
		}
		fmt.Printf("   Total Response Length: %d characters\n", totalLength)
		
		// Show first 200 characters of response
		if totalLength > 0 {
			var firstText string
			for _, part := range candidate.Content.Parts {
				if text, ok := part.(genai.Text); ok {
					firstText = string(text)
					break
				}
			}
			fmt.Printf("   Response Preview (first 200 chars):\n")
			if len(firstText) > 200 {
				fmt.Printf("   %s...\n", firstText[:200])
			} else {
				fmt.Printf("   %s\n", firstText)
			}
		}
	}
}

// Close closes the Gemini client
func (ktg *KubeEdgeTestGenerator) Close() {
	if ktg.client != nil {
		ktg.client.Close()
	}
}
