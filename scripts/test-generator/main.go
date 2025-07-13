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

// Package main implements the KubeEdge Auto Test Generator
// This tool runs as part of KubeEdge's main module and uses existing dependencies
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	PRNumber          string
	ChangedFiles      string
	RepoOwner         string
	RepoName          string
	GithubToken       string
	GeminiAPIKey      string
	CoverageThreshold float64
	MaxRetryAttempts  int
	PRAuthor          string
	CoverageFile      string
	Debug             bool
	Mode              string // "full" or "generate-only"
}

type ProcessingResult struct {
	FilePath       string
	Success        bool
	Error          error
	TestsPRNumber  int
	TestFile       string
	BeforeCoverage float64
	AfterCoverage  float64
	Duration       time.Duration
}

func main() {
	config := parseFlags()

	// Validate required configuration
	if err := validateConfig(config); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	// Setup logging
	if err := setupLogging(config.Debug); err != nil {
		log.Printf("Warning: Failed to setup logging: %v", err)
	}

	if config.ChangedFiles == "" {
		log.Println("No changed files to process")
		return
	}

	ctx := context.Background()

	log.Printf("Starting KubeEdge Auto Test Generator - %s mode", config.Mode)
	log.Printf("Coverage threshold: %.1f%%", config.CoverageThreshold)
	log.Printf("Max retry attempts: %d", config.MaxRetryAttempts)
	log.Printf("Debug mode: %t", config.Debug)
	log.Printf("Working directory: %s", getWorkingDir())

	// Initialize services
	coverageAnalyzer := NewCoverageAnalyzer(config.CoverageFile)
	testGenerator := NewKubeEdgeTestGenerator(config.GeminiAPIKey)
	defer testGenerator.Close()

	var prCreator *PRCreator
	if config.Mode == "full" {
		prCreator = NewPRCreator(config.GithubToken, config.RepoOwner, config.RepoName)
		// Check GitHub API rate limits
		if err := prCreator.CheckRateLimit(ctx); err != nil {
			log.Printf("GitHub API rate limit warning: %v", err)
		}
	}

	// Parse changed files
	changedFiles := strings.Split(config.ChangedFiles, "\n")
	var results []ProcessingResult
	successCount := 0
	failureCount := 0

	// Process each changed file
	for _, filePath := range changedFiles {
		filePath = strings.TrimSpace(filePath)
		if filePath == "" {
			continue
		}

		log.Printf("Processing file: %s", filePath)
		
		var result ProcessingResult
		if config.Mode == "generate-only" {
			result = processFileForGeneration(ctx, filePath, config, coverageAnalyzer, testGenerator)
		} else {
			result = processFile(ctx, filePath, config, coverageAnalyzer, testGenerator, prCreator)
		}
		
		results = append(results, result)

		if result.Success {
			successCount++
			if config.Mode == "generate-only" {
				log.Printf("Successfully generated tests for %s (Duration: %v)", 
					result.FilePath, result.Duration)
			} else {
				log.Printf("Successfully processed %s (PR #%d, Duration: %v)",
					result.FilePath, result.TestsPRNumber, result.Duration)
			}
		} else {
			failureCount++
			log.Printf("Failed to process %s: %v (Duration: %v)",
				result.FilePath, result.Error, result.Duration)
		}
	}

	// Print summary
	if config.Mode == "generate-only" {
		printGenerationSummary(results, successCount, failureCount)
	} else {
		printProcessingSummary(results, successCount, failureCount, config.PRAuthor, config.PRNumber)
		logFailureDetails(results, config.PRAuthor, config.PRNumber)
	}

	// Print final summary
	log.Printf("KubeEdge Auto Test Generator completed!")
	log.Printf("Summary: %d successful, %d failed", successCount, failureCount)

	// Don't exit with error code for failures - just log them
	if failureCount > 0 {
		log.Printf("⚠️ Some files failed processing - details logged above")
	}
}

// processFileForGeneration processes a file in generate-only mode
func processFileForGeneration(ctx context.Context, filePath string, config *Config, 
	analyzer *CoverageAnalyzer, generator *KubeEdgeTestGenerator) ProcessingResult {
	
	startTime := time.Now()
	result := ProcessingResult{
		FilePath: filePath,
		Duration: 0,
	}

	defer func() {
		result.Duration = time.Since(startTime)
	}()

	// Validate file exists
	if !fileExists(filePath) {
		result.Error = fmt.Errorf("file does not exist: %s", filePath)
		return result
	}

	// Generate test file path
	fileDir := filepath.Dir(filePath)
	fileName := strings.TrimSuffix(filepath.Base(filePath), ".go")
	testFile := filepath.Join(fileDir, fileName+"_test.go")
	result.TestFile = testFile

	// Check if test file already exists
	if fileExists(testFile) {
		log.Printf("⚠️ Test file already exists: %s", testFile)
		result.Success = true
		return result
	}

	// Extract functions from source file
	functions, err := extractFunctionsFromFile(filePath)
	if err != nil {
		result.Error = fmt.Errorf("failed to extract functions: %v", err)
		return result
	}

	if len(functions) == 0 {
		log.Printf("⚠️ No testable functions found in %s", filePath)
		result.Success = true
		return result
	}

	log.Printf("🔍 Found %d functions to test in %s", len(functions), filePath)

	// Generate tests with retry logic
	testContent, success := generateTestsWithRetry(ctx, filePath, functions, generator, config.MaxRetryAttempts)
	if !success {
		result.Error = fmt.Errorf("test generation failed after %d attempts", config.MaxRetryAttempts)
		return result
	}

	// Write test file
	if err := writeTestFile(testFile, testContent); err != nil {
		result.Error = fmt.Errorf("failed to write test file: %v", err)
		return result
	}

	log.Printf("✅ Generated test file: %s", testFile)
	result.Success = true
	return result
}

// processFile processes a single Go file for test generation (full mode)
func processFile(ctx context.Context, filePath string, config *Config,
	analyzer *CoverageAnalyzer, generator *KubeEdgeTestGenerator, creator *PRCreator) ProcessingResult {

	startTime := time.Now()
	result := ProcessingResult{
		FilePath: filePath,
		Duration: 0,
	}

	defer func() {
		result.Duration = time.Since(startTime)
	}()

	// Step 1: Analyze coverage
	needsTests, coverage, err := analyzer.AnalyzeFile(ctx, filePath, config.CoverageThreshold)
	if err != nil {
		result.Error = fmt.Errorf("coverage analysis failed: %v", err)
		return result
	}

	result.BeforeCoverage = coverage
	log.Printf("Coverage for %s: %.2f%% (needs tests: %v)", filePath, coverage, needsTests)

	if !needsTests {
		log.Printf("%s has sufficient coverage (%.2f%%), skipping", filePath, coverage)
		result.Success = true
		return result
	}

	// Step 2: Extract functions
	functions, err := analyzer.ExtractModifiedFunctions(ctx, filePath)
	if err != nil {
		result.Error = fmt.Errorf("function extraction failed: %v", err)
		return result
	}

	if len(functions) == 0 {
		log.Printf("No testable functions found in %s", filePath)
		result.Success = true
		return result
	}

	log.Printf("🔍 Found %d functions to test in %s", len(functions), filePath)

	// Step 3: Generate tests with retry logic
	testContent, success := generateTestsWithRetry(ctx, filePath, functions, generator, config.MaxRetryAttempts)
	if !success {
		result.Error = fmt.Errorf("test generation failed after %d attempts", config.MaxRetryAttempts)
		return result
	}

	// Step 4: Create PR with generated tests
	prNumber, err := creator.CreateTestsPR(ctx, filePath, testContent, coverage)
	if err != nil {
		result.Error = fmt.Errorf("PR creation failed: %v", err)
		return result
	}

	result.TestsPRNumber = prNumber
	result.Success = true
	return result
}

// writeTestFile writes test content to a file
func writeTestFile(testFile, content string) error {
	// Ensure directory exists
	dir := filepath.Dir(testFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", dir, err)
	}

	// Write file
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %v", testFile, err)
	}

	return nil
}

// extractFunctionsFromFile extracts functions from a Go source file
func extractFunctionsFromFile(filePath string) ([]FunctionInfo, error) {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Use existing AST parsing logic from coverage-analyzer.go
	analyzer := &CoverageAnalyzer{}
	functions, err := analyzer.extractFunctionsFromContent(string(content), filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse functions: %v", err)
	}

	// Filter out functions that shouldn't be tested
	var testableFunctions []FunctionInfo
	for _, fn := range functions {
		if shouldTestFunction(fn) {
			testableFunctions = append(testableFunctions, fn)
		}
	}

	return testableFunctions, nil
}

// shouldTestFunction determines if a function should be tested
func shouldTestFunction(fn FunctionInfo) bool {
	// Skip unexported functions unless they're complex
	if !fn.IsExported && len(fn.Content) < 200 {
		return false
	}

	// Skip simple getters/setters
	if isSimpleGetterSetter(fn.Content) {
		return false
	}

	// Skip functions that are just variable assignments
	if isSimpleAssignment(fn.Content) {
		return false
	}

	return true
}

// isSimpleGetterSetter checks if a function is a simple getter or setter
func isSimpleGetterSetter(content string) bool {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) <= 3 {
		// Check if it's just return or assignment
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "return ") || 
			   strings.Contains(line, " = ") && !strings.Contains(line, "if ") {
				return true
			}
		}
	}
	return false
}

// isSimpleAssignment checks if a function is just a simple assignment
func isSimpleAssignment(content string) bool {
	return strings.Count(content, "\n") <= 2 && 
		   strings.Contains(content, " = ") && 
		   !strings.Contains(content, "if ") &&
		   !strings.Contains(content, "for ") &&
		   !strings.Contains(content, "switch ")
}

// generateTestsWithRetry attempts to generate tests with retry logic
func generateTestsWithRetry(ctx context.Context, filePath string, functions []FunctionInfo,
	generator *KubeEdgeTestGenerator, maxAttempts int) (string, bool) {

	var lastError error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		log.Printf("Generating tests for %s (attempt %d/%d)", filePath, attempt, maxAttempts)

		// Call LLM
		testContent, err := generator.GenerateTests(ctx, filePath, functions, lastError)
		if err != nil {
			lastError = err
			log.Printf("Generation failed (attempt %d): %v", attempt, err)
			continue
		}

		log.Printf("Generated test content (attempt %d)", attempt)

		// Validate generated content
		if isValidGoTestContent(testContent) {
			log.Printf("Tests generated successfully for %s", filePath)
			return testContent, true
		}

		lastError = fmt.Errorf("generated content doesn't look like valid Go test code")
		log.Printf("Validation failed (attempt %d): %v", attempt, lastError)
	}

	log.Printf("❌ All %d attempts failed for %s", maxAttempts, filePath)
	return "", false
}

// isValidGoTestContent validates if content looks like Go test code
func isValidGoTestContent(content string) bool {
	checks := []string{
		"package ",
		"import",
		"func Test",
		"testing",
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			return false
		}
	}
	return true
}

// printGenerationSummary prints summary for generate-only mode
func printGenerationSummary(results []ProcessingResult, successCount, failureCount int) {
	log.Printf("\n" + strings.Repeat("=", 60))
	log.Printf("TEST GENERATION SUMMARY")
	log.Print(strings.Repeat("=", 60))

	log.Printf("Statistics:")
	log.Printf("Successfully generated: %d files", successCount)
	log.Printf("Failed: %d files", failureCount)
	log.Printf("Total processed: %d files", len(results))

	if len(results) > 0 {
		successRate := float64(successCount) / float64(len(results)) * 100
		log.Printf("Success Rate: %.1f%%", successRate)
	}

	// Show detailed results
	if len(results) > 0 {
		log.Printf("\nDetailed Results:")
		for i, result := range results {
			status := "SUCCESS"
			if !result.Success {
				status = "FAILED"
			}

			log.Printf("  %d. %s %s", i+1, status, result.FilePath)
			if result.Success && result.TestFile != "" {
				log.Printf("     Generated: %s", result.TestFile)
			}
			if !result.Success && result.Error != nil {
				log.Printf("     Error: %v", result.Error)
			}
			log.Printf("     Duration: %v", result.Duration.Round(time.Second))
		}
	}

	log.Printf(strings.Repeat("=", 60) + "\n")
}

// printProcessingSummary prints a comprehensive summary of processing results (full mode)
func printProcessingSummary(results []ProcessingResult, successCount, failureCount int, prAuthor, prNumber string) {
	log.Printf("\n" + strings.Repeat("=", 60))
	log.Printf("PROCESSING SUMMARY")
	log.Print(strings.Repeat("=", 60))

	log.Printf("Statistics:")
	log.Printf("Successful: %d files", successCount)
	log.Printf("Failed: %d files", failureCount)
	log.Printf("Total processed: %d files", len(results))

	if len(results) > 0 {
		successRate := float64(successCount) / float64(len(results)) * 100
		log.Printf("Success Rate: %.1f%%", successRate)
	}

	if prNumber != "" {
		log.Printf("Original PR: #%s by @%s", prNumber, prAuthor)
	}

	log.Printf("Timestamp: %s", time.Now().Format("2006-01-02 15:04:05 UTC"))

	// Show detailed results
	if len(results) > 0 {
		log.Printf("\nDetailed Results:")
		for i, result := range results {
			status := "SUCCESS"
			if !result.Success {
				status = "FAILED"
			}

			log.Printf("  %d. %s %s", i+1, status, result.FilePath)
			log.Printf("     Coverage: %.1f%% | Duration: %v", result.BeforeCoverage, result.Duration.Round(time.Second))

			if result.Success && result.TestsPRNumber > 0 {
				log.Printf("     Created test PR #%d", result.TestsPRNumber)
			} else if !result.Success && result.Error != nil {
				log.Printf("     Error: %v", result.Error)
			}
		}
	}

	if successCount > 0 {
		log.Printf("\nGenerated Test PRs:")
		log.Printf("  - %d new test PRs created", successCount)
		log.Printf("  - Review and merge them to improve coverage")
		log.Printf("  - Target overall coverage: 80%% (per codecov.yml)")
	}

	if failureCount > 0 {
		log.Printf("\nFailed Generations:")
		log.Printf("  - %d files could not have tests auto-generated", failureCount)
		log.Printf("  - See failure details below")
		log.Printf("  - These may require manual test creation")
	}

	log.Printf(strings.Repeat("=", 60) + "\n")
}

// logFailureDetails logs detailed information about failed test generations
func logFailureDetails(results []ProcessingResult, prAuthor, prNumber string) {
	failedResults := make([]ProcessingResult, 0)
	for _, result := range results {
		if !result.Success {
			failedResults = append(failedResults, result)
		}
	}

	if len(failedResults) == 0 {
		return
	}

	log.Printf("\n" + strings.Repeat("=", 60))
	log.Printf("FAILURE ANALYSIS")
	log.Print(strings.Repeat("=", 60))

	for i, result := range failedResults {
		log.Printf("\nFAILURE #%d:", i+1)
		log.Printf("  File: %s", result.FilePath)
		log.Printf("  Coverage: %.1f%%", result.BeforeCoverage)
		log.Printf("  Duration: %v", result.Duration.Round(time.Second))
		log.Printf("  Error: %v", result.Error)

		log.Printf("\n  Likely Causes:")
		log.Printf("    - File has complex dependencies difficult to mock")
		log.Printf("    - Code structure doesn't follow standard Go testing patterns")
		log.Printf("    - Import or dependency issues")
		log.Printf("    - File requires manual test setup or custom mocking")

		log.Printf("\n  Recommended Actions:")
		log.Printf("    1. Manual Test Creation: Consider creating tests manually")
		log.Printf("    2. Code Review: Review file structure for testability")
		log.Printf("    3. Refactoring: Consider refactoring complex functions")
		log.Printf("    4. Dependencies: Check if external deps need custom mocking")
	}

	log.Printf("\nKubeEdge Testing Guidelines:")
	log.Printf("  - Use gomonkey v2 for mocking external functions")
	log.Printf("  - Follow table-driven test patterns")
	log.Printf("  - Use github.com/stretchr/testify/assert for assertions")
	log.Printf("  - Ensure tests are independent and repeatable")

	log.Printf("\nResources:")
	log.Printf("  - KubeEdge Testing Documentation: https://github.com/kubeedge/kubeedge/blob/master/docs/testing.md")
	log.Printf("  - Go Testing Best Practices: https://golang.org/doc/tutorial/add-a-test")
	log.Printf("  - gomonkey Documentation: https://github.com/agiledragon/gomonkey")

	if prAuthor != "" {
		log.Printf("\nPR Author: @%s", prAuthor)
	}
	if prNumber != "" {
		log.Printf("Original PR: #%s", prNumber)
	}

	log.Printf(strings.Repeat("=", 60) + "\n")
}

// getWorkingDir returns the current working directory for debugging
func getWorkingDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return "unknown"
	}
	return wd
}

// parseFlags parses command line flags
func parseFlags() *Config {
	config := &Config{}

	flag.StringVar(&config.PRNumber, "pr-number", "", "Pull request number")
	flag.StringVar(&config.ChangedFiles, "changed-files", "", "Newline-separated list of changed files")
	flag.StringVar(&config.RepoOwner, "repo-owner", "kubeedge", "Repository owner")
	flag.StringVar(&config.RepoName, "repo-name", "kubeedge", "Repository name")
	flag.StringVar(&config.GithubToken, "github-token", os.Getenv("GITHUB_TOKEN"), "GitHub token")
	flag.StringVar(&config.GeminiAPIKey, "gemini-api-key", os.Getenv("GEMINI_API_KEY"), "Gemini API key")
	flag.Float64Var(&config.CoverageThreshold, "coverage-threshold", 40.0, "Coverage threshold percentage")
	flag.IntVar(&config.MaxRetryAttempts, "max-retry-attempts", 3, "Maximum retry attempts for test generation")
	flag.StringVar(&config.PRAuthor, "pr-author", "", "PR author username")
	flag.StringVar(&config.CoverageFile, "coverage-file", "", "Path to coverage file (optional)")
	flag.BoolVar(&config.Debug, "debug", strings.ToLower(os.Getenv("DEBUG")) == "true", "Enable debug logging")
	flag.StringVar(&config.Mode, "mode", "full", "Operation mode: 'full' or 'generate-only'")

	flag.Parse()
	return config
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	if config.GeminiAPIKey == "" {
		return fmt.Errorf("gemini-api-key is required")
	}
	if config.RepoOwner == "" {
		return fmt.Errorf("repo-owner is required")
	}
	if config.RepoName == "" {
		return fmt.Errorf("repo-name is required")
	}
	if config.CoverageThreshold < 0 || config.CoverageThreshold > 100 {
		return fmt.Errorf("coverage-threshold must be between 0 and 100")
	}
	if config.MaxRetryAttempts < 1 || config.MaxRetryAttempts > 10 {
		return fmt.Errorf("max-retry-attempts must be between 1 and 10")
	}
	if config.Mode != "full" && config.Mode != "generate-only" {
		return fmt.Errorf("mode must be 'full' or 'generate-only'")
	}
	
	// For full mode, GitHub token is required
	if config.Mode == "full" && config.GithubToken == "" {
		return fmt.Errorf("github-token is required for full mode")
	}
	
	return nil
}

// setupLogging sets up logging configuration
func setupLogging(debug bool) error {
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	logFile := filepath.Join(logDir, fmt.Sprintf("test-generator-%s.log", time.Now().Format("2006-01-02-15-04-05")))
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	log.SetOutput(file)
	if debug {
		log.Printf("Debug logging enabled")
	}

	return nil
}
