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
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// TestValidationResult represents validation results for a test file
type TestValidationResult struct {
	SourceFile       string
	TestFile         string
	Success          bool
	Error            error
	BeforeCoverage   float64
	AfterCoverage    float64
	CoverageImproved bool
	TestsPass        bool
	CompileSuccess   bool
	Duration         time.Duration
}

// TestValidator handles all validation and compilation logic
type TestValidator struct {
	workingDir        string
	coverageThreshold float64
}

// NewTestValidator creates a new test validator
func NewTestValidator(workingDir string, coverageThreshold float64) *TestValidator {
	return &TestValidator{
		workingDir:        workingDir,
		coverageThreshold: coverageThreshold,
	}
}

// Run `go test -coverprofile=coverage.out file_path` → If Tests Pass
func (tv *TestValidator) ValidateGeneratedTest(ctx context.Context, sourceFile string, testFile string) TestValidationResult {
	startTime := time.Now()
	result := TestValidationResult{
		SourceFile: sourceFile,
		TestFile:   testFile,
	}

	defer func() {
		result.Duration = time.Since(startTime)
	}()

	fmt.Printf("🔍 Validating test file: %s\n", testFile)

	// Step 1: Get coverage before test generation
	beforeCoverage, err := tv.GetFileCoverage(ctx, sourceFile)
	if err != nil {
		result.Error = fmt.Errorf("failed to get before coverage: %v", err)
		return result
	}
	result.BeforeCoverage = beforeCoverage
	fmt.Printf("📊 Before coverage: %.2f%%\n", beforeCoverage)

	// Step 2: Check if test file exists
	if !fileExists(testFile) {
		result.Error = fmt.Errorf("test file does not exist: %s", testFile)
		return result
	}

	// Step 3: Run `go test -coverprofile=coverage.out file_path` as specified by mentor
	testsPass, compileSuccess, testErr := tv.RunGoTestWithCoverage(ctx, sourceFile)
	result.CompileSuccess = compileSuccess
	result.TestsPass = testsPass
	
	if !compileSuccess {
		result.Error = fmt.Errorf("test compilation failed: %v", testErr)
		return result
	}
	fmt.Printf("✅ Test compilation: SUCCESS\n")

	if !testsPass {
		result.Error = fmt.Errorf("tests execution failed: %v", testErr)
		return result
	}
	fmt.Printf("✅ Test execution: SUCCESS\n")

	// Step 4: Get coverage after test generation
	afterCoverage, err := tv.GetFileCoverage(ctx, sourceFile)
	if err != nil {
		result.Error = fmt.Errorf("failed to get after coverage: %v", err)
		return result
	}
	result.AfterCoverage = afterCoverage
	fmt.Printf("📊 After coverage: %.2f%%\n", afterCoverage)

	// Step 5: Verify coverage improvement
	result.CoverageImproved = afterCoverage > beforeCoverage
	if !result.CoverageImproved {
		result.Error = fmt.Errorf("coverage did not improve: %.2f%% → %.2f%%", beforeCoverage, afterCoverage)
		return result
	}

	improvement := afterCoverage - beforeCoverage
	fmt.Printf("📈 Coverage improvement: +%.2f%%\n", improvement)

	result.Success = true
	return result
}

// `go test -coverprofile=coverage.out file_path`
func (tv *TestValidator) RunGoTestWithCoverage(ctx context.Context, sourceFile string) (testsPass bool, compileSuccess bool, err error) {
	packageDir := filepath.Dir(sourceFile)
	
	// Run go test -coverprofile=coverage.out file_path
	cmd := exec.CommandContext(ctx, "go", "test", "-coverprofile=coverage.out", "./"+packageDir)
	cmd.Dir = tv.workingDir

	output, execErr := cmd.CombinedOutput()
	outputStr := string(output)

	// Check compilation success
	if execErr != nil && strings.Contains(outputStr, "build failed") {
		return false, false, fmt.Errorf("compilation failed: %v, output: %s", execErr, outputStr)
	}
	compileSuccess = true

	// Check test execution success
	if execErr != nil {
		return false, true, fmt.Errorf("test execution failed: %v, output: %s", execErr, outputStr)
	}

	// Check for test failures in output
	if strings.Contains(outputStr, "FAIL") {
		return false, true, fmt.Errorf("some tests failed: %s", outputStr)
	}

	// Ensure tests actually ran
	if !strings.Contains(outputStr, "PASS") && !strings.Contains(outputStr, "ok") {
		return false, true, fmt.Errorf("no tests were executed")
	}

	return true, true, nil
}

// GetFileCoverage gets coverage percentage for a specific file
func (tv *TestValidator) GetFileCoverage(ctx context.Context, sourceFile string) (float64, error) {
	packageDir := filepath.Dir(sourceFile)
	coverageFile := "temp_coverage.out"

	// Run go test with coverage
	cmd := exec.CommandContext(ctx, "go", "test", "-coverprofile="+coverageFile, "./"+packageDir)
	cmd.Dir = tv.workingDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		// If no tests exist, coverage is 0
		if strings.Contains(string(output), "no test files") || 
		   strings.Contains(string(output), "no Go files") {
			return 0.0, nil
		}
		return 0.0, fmt.Errorf("coverage test failed: %v, output: %s", err, string(output))
	}

	// Parse coverage from file
	coverage := tv.ParseCoverageFromFile(coverageFile, filepath.Base(sourceFile))

	// Clean up
	os.Remove(coverageFile)

	return coverage, nil
}

// ParseCoverageFromFile parses coverage percentage from coverage output file
func (tv *TestValidator) ParseCoverageFromFile(coverageFile string, targetFile string) float64 {
	if !fileExists(coverageFile) {
		return 0.0
	}

	// Use go tool cover to get function-level coverage
	cmd := exec.Command("go", "tool", "cover", "-func", coverageFile)
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Warning: Failed to parse coverage file: %v\n", err)
		return 0.0
	}

	// Look for the specific file in coverage output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, targetFile) && !strings.Contains(line, "total:") {
			// Extract percentage from the line (last column)
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				percentStr := fields[len(fields)-1]
				percentStr = strings.TrimSuffix(percentStr, "%")
				if coverage, err := strconv.ParseFloat(percentStr, 64); err == nil {
					return coverage
				}
			}
		}
	}

	return 0.0
}

// GetModifiedFilesFromGit gets modified Go files from the last commit
func (tv *TestValidator) GetModifiedFilesFromGit(ctx context.Context) ([]string, error) {
	// Get files modified in the last commit (merged PR)
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only", "HEAD~1", "HEAD")
	cmd.Dir = tv.workingDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get modified files: %v", err)
	}

	var goFiles []string
	files := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, file := range files {
		file = strings.TrimSpace(file)
		// Include only .go files, exclude test files
		if strings.HasSuffix(file, ".go") && !strings.HasSuffix(file, "_test.go") && file != "" {
			if fileExists(file) {
				goFiles = append(goFiles, file)
			}
		}
	}

	return goFiles, nil
}

// FilterLowCoverageFiles filters files that need test generation based on coverage threshold
// Check Coverage of Modified Files → If < 40%
func (tv *TestValidator) FilterLowCoverageFiles(ctx context.Context, files []string) ([]string, []CoverageInfo, error) {
	var lowCoverageFiles []string
	var coverageInfos []CoverageInfo

	fmt.Printf("🔍 Checking coverage for %d files with threshold %.0f%%\n", len(files), tv.coverageThreshold)

	for _, file := range files {
		coverage, err := tv.GetFileCoverage(ctx, file)
		if err != nil {
			fmt.Printf("⚠️ Warning: Could not get coverage for %s: %v\n", file, err)
			continue
		}

		info := CoverageInfo{
			FilePath: file,
			Coverage: coverage,
		}
		coverageInfos = append(coverageInfos, info)

		if coverage < tv.coverageThreshold {
			lowCoverageFiles = append(lowCoverageFiles, file)
			fmt.Printf("🎯 %s: %.2f%% (needs tests)\n", file, coverage)
		} else {
			fmt.Printf("✅ %s: %.2f%% (sufficient)\n", file, coverage)
		}
	}

	return lowCoverageFiles, coverageInfos, nil
}

// CoverageInfo holds coverage information for a file
type CoverageInfo struct {
	FilePath string
	Coverage float64
}

// GenerateTestFilePath generates the test file path for a source file
// Add filename_test.go
func (tv *TestValidator) GenerateTestFilePath(sourceFile string) string {
	dir := filepath.Dir(sourceFile)
	base := filepath.Base(sourceFile)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return filepath.Join(dir, name+"_test.go")
}

// CleanupTempFiles removes temporary coverage files
func (tv *TestValidator) CleanupTempFiles() {
	tempFiles := []string{
		"temp_coverage.out",
		"coverage.out",
		"before_coverage.out", 
		"after_coverage.out",
	}
	
	for _, file := range tempFiles {
		os.Remove(file)
	}
}