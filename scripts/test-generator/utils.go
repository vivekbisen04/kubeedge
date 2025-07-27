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
	"os/exec"
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

// hasTestableContent checks if the file has content worth testing
func hasTestableContent(content string) bool {
	// Basic checks for Go file with functions
	if !strings.Contains(content, "package ") {
		log.Printf("❌ No package declaration found")
		return false
	}
	
	if !strings.Contains(content, "func ") {
		log.Printf("❌ No functions found")
		return false
	}
	
	// Count exportable functions (not main, init, or test functions)
	funcCount := 0
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "func ") && 
		   !strings.Contains(line, "func main") && 
		   !strings.Contains(line, "func init") && 
		   !strings.Contains(line, "func Test") &&
		   !strings.Contains(line, "func Benchmark") &&
		   !strings.Contains(line, "func Example") {
			funcCount++
		}
	}
	
	log.Printf("📊 Found %d potential functions to test", funcCount)
	return funcCount > 0
}

// generateTestsWithLLMDecision - let LLM decide everything
func generateTestsWithLLMDecision(ctx context.Context, filePath string, sourceContent string, 
	generator *KubeEdgeTestGenerator, maxRetries int, workingDir string) (string, bool) {
	
	var lastError error
	var lastGeneratedContent string

	// Check if existing test file exists
	testFilePath := generateTestFilePath(filePath)
	absTestFilePath := filepath.Join(workingDir, testFilePath)
	
	var existingTestContent string
	if fileExists(absTestFilePath) {
		log.Printf("📖 Found existing test file: %s", testFilePath)
		content, err := os.ReadFile(absTestFilePath)
		if err != nil {
			log.Printf("⚠️ Warning: Could not read existing test file: %v", err)
			existingTestContent = ""
		} else {
			existingTestContent = string(content)
			log.Printf("📊 Existing test file size: %d bytes", len(existingTestContent))
		}
	} else {
		log.Printf("📝 No existing test file found, generating new one")
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if existingTestContent != "" {
			log.Printf("🤖 Attempt %d/%d: Enhancing existing test file for %s", attempt, maxRetries, filePath)
		} else {
			log.Printf("🤖 Attempt %d/%d: Generating new test file for %s", attempt, maxRetries, filePath)
		}
		
		testContent, err := generator.GenerateTestsFromWholeFile(ctx, filePath, sourceContent, existingTestContent, lastError)
		if err != nil {
			lastError = err
			log.Printf("❌ Attempt %d failed: %v", attempt, err)
			continue
		}

		// Save the generated content for debugging
		lastGeneratedContent = testContent

		// Basic validation - check if it looks like Go test code
		if isValidGoTestContent(testContent) {
			log.Printf("✅ Test generation successful on attempt %d", attempt)
			return testContent, true
		}

		lastError = fmt.Errorf("generated content doesn't look like valid Go test code")
		log.Printf("⚠️ Attempt %d produced insufficient content", attempt)
	}

	log.Printf("❌ All %d attempts failed for %s", maxRetries, filePath)
	// Return the last generated content even if it failed, for debugging
	return lastGeneratedContent, false
}

// generateTestFilePath generates the test file path for a source file
func generateTestFilePath(sourceFile string) string {
	dir := filepath.Dir(sourceFile)
	base := filepath.Base(sourceFile)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return filepath.Join(dir, name+"_test.go")
}

// isValidGoTestContent - simplified validation
func isValidGoTestContent(content string) bool {
	required := []string{"package ", "import", "func Test", "testing"}
	for _, req := range required {
		if !strings.Contains(content, req) {
			log.Printf("⚠️ Generated content missing: %s", req)
			return false
		}
	}
	return true
}

// cleanupGoCode removes unused imports and formats the Go code
func cleanupGoCode(filePath string) error {
	log.Printf("🧹 Cleaning up unused imports in: %s", filePath)
	
	// Try goimports first (preferred - handles imports automatically)
	if err := runGoImports(filePath); err == nil {
		log.Printf("✅ Successfully cleaned with goimports")
		return nil
	}
	
	// Fallback to gofmt if goimports not available
	if err := runGoFmt(filePath); err == nil {
		log.Printf("✅ Successfully formatted with gofmt")
		return nil
	}
	
	log.Printf("⚠️ Could not clean up imports, but file may still be valid")
	return nil
}

// runGoImports runs goimports to fix imports and format code
func runGoImports(filePath string) error {
	// Try goimports from various locations
	goimportsPaths := []string{
		"goimports",                    // if it's in PATH
		os.ExpandEnv("$HOME/go/bin/goimports"), // default Go bin location
		"/usr/local/go/bin/goimports",          // system Go installation
	}
	
	for _, goimportsPath := range goimportsPaths {
		cmd := exec.Command(goimportsPath, "-w", filePath)
		_, err := cmd.CombinedOutput()
		if err == nil {
			return nil
		}
		// Continue to next path if this one failed
	}
	
	return fmt.Errorf("goimports not found in any expected location")
}

// runGoFmt runs gofmt to format code (doesn't fix imports but formats)
func runGoFmt(filePath string) error {
	cmd := exec.Command("gofmt", "-w", filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("🔍 gofmt output: %s", string(output))
		return fmt.Errorf("gofmt failed: %v", err)
	}
	return nil
}