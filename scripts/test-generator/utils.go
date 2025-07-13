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
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
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
	// Use the coverage analyzer to extract functions
	analyzer := NewCoverageAnalyzer()
	functions, err := analyzer.ExtractModifiedFunctions(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract functions: %v", err)
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
	generator *KubeEdgeTestGenerator, maxRetries int) (string, bool) {
	
	var lastError error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("🤖 Attempt %d/%d: Generating tests for %s", attempt, maxRetries, filePath)
		
		testContent, err := generator.GenerateTests(ctx, filePath, functions, lastError)
		if err != nil {
			lastError = err
			log.Printf("❌ Attempt %d failed: %v", attempt, err)
			continue
		}

		// Basic validation of generated content
		if isValidGoTestContent(testContent) {
			log.Printf("✅ Test generation successful on attempt %d", attempt)
			return testContent, true
		}

		lastError = fmt.Errorf("generated content doesn't look like valid Go test code")
		log.Printf("⚠️ Attempt %d produced insufficient content", attempt)
	}

	log.Printf("❌ All %d attempts failed for %s", maxRetries, filePath)
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