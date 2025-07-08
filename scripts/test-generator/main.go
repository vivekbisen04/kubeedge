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
}

type ProcessingResult struct {
	FilePath      string
	Success       bool
	Error         error
	TestsPRNumber int
	Coverage      float64
	Duration      time.Duration
}

func main() {
	config := parseFlags()
	
	// Validate required configuration
	if err := validateConfig(config); err != nil {
		log.Fatalf("❌ Configuration validation failed: %v", err)
	}

	// Setup logging
	if err := setupLogging(config.Debug); err != nil {
		log.Printf("⚠️ Warning: Failed to setup logging: %v", err)
	}

	if config.ChangedFiles == "" {
		log.Println("✅ No changed files to process")
		return
	}

	ctx := context.Background()
	
	log.Printf("🤖 Starting KubeEdge Auto Test Generator")
	log.Printf("📊 Coverage threshold: %.1f%%", config.CoverageThreshold)
	log.Printf("🔄 Max retry attempts: %d", config.MaxRetryAttempts)
	log.Printf("🔧 Debug mode: %t", config.Debug)

	// Initialize services (without Docker for demo)
	coverageAnalyzer := NewCoverageAnalyzer(config.CoverageFile)
	testGenerator := NewKubeEdgeTestGenerator(config.GeminiAPIKey)
	prCreator := NewPRCreator(config.GithubToken, config.RepoOwner, config.RepoName)
	emailSender := NewEmailSender(config.RepoOwner, config.RepoName)

	// Check GitHub API rate limits
	if err := prCreator.CheckRateLimit(ctx); err != nil {
		log.Printf("⚠️ GitHub API rate limit warning: %v", err)
	}

	// Parse changed files
	changedFiles := strings.Split(config.ChangedFiles, "\n")
	successCount := 0
	failureCount := 0
	var results []ProcessingResult

	for _, file := range changedFiles {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}

		log.Printf("\n📁 Processing file: %s", file)
		
		result := processFile(ctx, file, config, coverageAnalyzer, testGenerator, prCreator)
		results = append(results, result)
		
		if result.Success {
			successCount++
			log.Printf("✅ Successfully processed %s", file)
		} else {
			failureCount++
			log.Printf("❌ Failed to process %s: %v", file, result.Error)
			
			// Send failure notification
			if err := emailSender.SendFailureNotification(ctx, file, config.PRAuthor, config.PRNumber); err != nil {
				log.Printf("⚠️ Failed to send failure notification: %v", err)
			}
		}
	}

	// Final summary
	log.Printf("\n📈 Generation Summary:")
	log.Printf("✅ Successful: %d files", successCount)
	log.Printf("❌ Failed: %d files", failureCount)
	log.Printf("📊 Total processed: %d files", successCount+failureCount)

	if successCount > 0 {
		// Send success summary
		if err := emailSender.SendSuccessSummary(ctx, successCount, failureCount, config.PRAuthor, config.PRNumber); err != nil {
			log.Printf("⚠️ Failed to send success summary: %v", err)
		}
	}
}

func processFile(ctx context.Context, filePath string, config *Config, 
	coverageAnalyzer *CoverageAnalyzer, testGenerator *KubeEdgeTestGenerator, 
	prCreator *PRCreator) ProcessingResult {
	
	startTime := time.Now()
	result := ProcessingResult{
		FilePath: filePath,
		Duration: 0,
	}

	// Check if file needs tests based on coverage
	needsTests, coverage, err := coverageAnalyzer.AnalyzeFile(ctx, filePath, config.CoverageThreshold)
	if err != nil {
		result.Error = fmt.Errorf("coverage analysis failed: %v", err)
		result.Duration = time.Since(startTime)
		return result
	}
	
	result.Coverage = coverage

	if !needsTests {
		log.Printf("✅ File %s has sufficient coverage (%.2f%%), skipping", filePath, coverage)
		result.Success = true
		result.Duration = time.Since(startTime)
		return result
	}

	log.Printf("🎯 File %s needs tests (coverage: %.2f%%)", filePath, coverage)

	// Extract functions that need testing
	functions, err := coverageAnalyzer.ExtractModifiedFunctions(ctx, filePath)
	if err != nil {
		result.Error = fmt.Errorf("function extraction failed: %v", err)
		result.Duration = time.Since(startTime)
		return result
	}

	if len(functions) == 0 {
		log.Printf("⚠️ No testable functions found in %s", filePath)
		result.Success = true
		result.Duration = time.Since(startTime)
		return result
	}

	log.Printf("🔍 Found %d functions to test in %s", len(functions), filePath)

	// Generate tests with retry logic (simplified without Docker validation)
	testContent, success := generateTestsWithRetry(
		ctx, testGenerator, filePath, functions, config.MaxRetryAttempts,
	)

	if !success {
		result.Error = fmt.Errorf("test generation failed after %d attempts", config.MaxRetryAttempts)
		result.Duration = time.Since(startTime)
		return result
	}

	// Create PR with generated tests
	testFileName := generateTestFileName(filePath)
	branchName := generateBranchName(filePath)
	
	if err := prCreator.CreateTestPR(ctx, filePath, testFileName, testContent, branchName, coverage); err != nil {
		result.Error = fmt.Errorf("PR creation failed: %v", err)
		result.Duration = time.Since(startTime)
		return result
	}

	result.Success = true
	result.Duration = time.Since(startTime)
	return result
}

func generateTestsWithRetry(
	ctx context.Context,
	testGenerator *KubeEdgeTestGenerator,
	filePath string,
	functions []FunctionInfo,
	maxAttempts int,
) (string, bool) {
	
	var lastError error
	
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		log.Printf("🔄 Attempt %d/%d for %s", attempt, maxAttempts, filePath)
		
		// Generate tests using LLM
		testContent, err := testGenerator.GenerateTests(ctx, filePath, functions, lastError)
		if err != nil {
			lastError = err
			log.Printf("⚠️ Generation failed (attempt %d): %v", attempt, err)
			continue
		}
		
		log.Printf("✅ Generated test content (attempt %d)", attempt)
		
		// Simple validation: check if generated content looks like Go code
		if isValidGoTestContent(testContent) {
			log.Printf("🎉 Tests generated successfully for %s", filePath)
			return testContent, true
		}
		
		lastError = fmt.Errorf("generated content doesn't look like valid Go test code")
		log.Printf("⚠️ Validation failed (attempt %d): %v", attempt, lastError)
	}
	
	log.Printf("❌ All %d attempts failed for %s", maxAttempts, filePath)
	return "", false
}

// Simple validation function to check if content looks like Go test code
func isValidGoTestContent(content string) bool {
	// Basic checks for Go test content
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
	
	flag.Parse()
	return config
}

func validateConfig(config *Config) error {
	if config.GeminiAPIKey == "" {
		return fmt.Errorf("gemini-api-key is required")
	}
	if config.GithubToken == "" {
		return fmt.Errorf("github-token is required")
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
	return nil
}

func setupLogging(debug bool) error {
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}
	
	logFile := filepath.Join(logDir, fmt.Sprintf("test-generator-%s.log", 
		time.Now().Format("2006-01-02-15-04-05")))
	
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	
	log.SetOutput(file)
	
	if debug {
		log.Printf("🔧 Debug logging enabled")
	}
	
	return nil
}

func generateTestFileName(sourceFile string) string {
	dir := filepath.Dir(sourceFile)
	base := filepath.Base(sourceFile)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return filepath.Join(dir, name+"_test.go")
}

func generateBranchName(sourceFile string) string {
	// Clean the file path for branch name
	cleanPath := strings.ReplaceAll(sourceFile, "/", "-")
	cleanPath = strings.ReplaceAll(cleanPath, ".", "-")
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("auto-test-generation-%s-%s", cleanPath, timestamp)
}