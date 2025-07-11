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
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
)

// TestPrintProcessingSummary tests the printProcessingSummary function.
func TestPrintProcessingSummary(t *testing.T) {
	// Test cases
	testCases := []struct {
		name           string
		results        []ProcessingResult
		successCount   int
		failureCount   int
		prAuthor       string
		prNumber       string
		expectedOutput string //Simplified expected output for easier comparison
	}{
		{
			name:           "No results",
			results:        []ProcessingResult{},
			successCount:   0,
			failureCount:   0,
			prAuthor:       "",
			prNumber:       "",
			expectedOutput: "PROCESSING SUMMARY",
		},
		{
			name:         "One successful result",
			results: []ProcessingResult{{FilePath: "file1.go", Success: true, Coverage: 80.0, Duration: 10 * time.Second, TestsPRNumber: 1}},
			successCount: 1,
			failureCount: 0,
			prAuthor:     "author1",
			prNumber:     "123",
			expectedOutput: "PROCESSING SUMMARY\nStatistics:\nSuccessful: 1 files\nFailed: 0 files\nTotal processed: 1 files\n  Success Rate: 100.0%\n  Original PR: #123 by @author1\n  Timestamp:",
		},
		{
			name:         "One failed result",
			results: []ProcessingResult{{FilePath: "file2.go", Success: false, Coverage: 20.0, Duration: 5 * time.Second, Error: fmt.Errorf("test error")}},
			successCount: 0,
			failureCount: 1,
			prAuthor:     "",
			prNumber:     "",
			expectedOutput: "PROCESSING SUMMARY\nStatistics:\nSuccessful: 0 files\nFailed: 1 files\nTotal processed: 1 files\n  Success Rate: 0.0%\n  Timestamp:",
		},
		{
			name: "Multiple results",
			results: []ProcessingResult{
				{FilePath: "file1.go", Success: true, Coverage: 80.0, Duration: 10 * time.Second, TestsPRNumber: 1},
				{FilePath: "file2.go", Success: false, Coverage: 20.0, Duration: 5 * time.Second, Error: fmt.Errorf("test error")},
			},
			successCount: 1,
			failureCount: 1,
			prAuthor:     "author2",
			prNumber:     "456",
			expectedOutput: "PROCESSING SUMMARY\nStatistics:\nSuccessful: 1 files\nFailed: 1 files\nTotal processed: 2 files\n  Success Rate: 50.0%\n  Original PR: #456 by @author2\n  Timestamp:",
		},
	}

	// Mock log.Printf
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	var logOutput []string
	patches.ApplyFunc(log.Printf, func(format string, v ...interface{}) {
		logOutput = append(logOutput, fmt.Sprintf(format, v...))
	})

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			printProcessingSummary(tc.results, tc.successCount, tc.failureCount, tc.prAuthor, tc.prNumber)
			assert.True(t, strings.Contains(strings.Join(logOutput, "\n"), tc.expectedOutput))
			logOutput = []string{} //clear logOutput for next test case
		})
	}
}


// TestLogFailureDetails tests the logFailureDetails function.
func TestLogFailureDetails(t *testing.T) {
	testCases := []struct {
		name           string
		results        []ProcessingResult
		prAuthor       string
		prNumber       string
		expectedOutput string //Simplified expected output for easier comparison

	}{
		{
			name:    "No failures",
			results: []ProcessingResult{{FilePath: "file1.go", Success: true}},
			expectedOutput: "",
		},
		{
			name: "One failure",
			results: []ProcessingResult{
				{FilePath: "file1.go", Success: false, Coverage: 50, Duration: 1 * time.Second, Error: fmt.Errorf("test error")},
			},
			prAuthor: "test",
			prNumber: "123",
			expectedOutput: "FAILURE ANALYSIS\nFAILURE #1:\n  File: file1.go\n  Coverage: 50.0%\n  Duration: 1s\n  Error: test error\n  Likely Causes:\n    - File has complex dependencies difficult to mock\n    - Code structure doesn't follow standard Go testing patterns\n    - Import or dependency issues\n    - File requires manual test setup or custom mocking\n  Recommended Actions:\n    1. Manual Test Creation: Consider creating tests manually\n    2. Code Review: Review file structure for testability\n    3. Refactoring: Consider refactoring complex functions\n    4. Dependencies: Check if external deps need custom mocking\nKubeEdge Testing Guidelines:\n  - Use gomonkey v2 for mocking external functions\n  - Follow table-driven test patterns\n  - Use github.com/stretchr/testify/assert for assertions\n  - Ensure tests are independent and repeatable\nResources:\n  - KubeEdge Testing Documentation: https://github.com/kubeedge/kubeedge/blob/master/docs/testing.md\n  - Go Testing Best Practices: https://golang.org/doc/tutorial/add-a-test\n  - gomonkey Documentation: https://github.com/agiledragon/gomonkey\nPR Author: @test\nOriginal PR: #123",
		},
		{
			name: "Multiple failures",
			results: []ProcessingResult{
				{FilePath: "file1.go", Success: false, Coverage: 50, Duration: 1 * time.Second, Error: fmt.Errorf("test error 1")},
				{FilePath: "file2.go", Success: false, Coverage: 30, Duration: 2 * time.Second, Error: fmt.Errorf("test error 2")},
			},
			prAuthor:       "test2",
			prNumber:       "456",
			expectedOutput: "FAILURE ANALYSIS\nFAILURE #1:\n  File: file1.go\n  Coverage: 50.0%\n  Duration: 1s\n  Error: test error 1\n  Likely Causes:\n    - File has complex dependencies difficult to mock\n    - Code structure doesn't follow standard Go testing patterns\n    - Import or dependency issues\n    - File requires manual test setup or custom mocking\n  Recommended Actions:\n    1. Manual Test Creation: Consider creating tests manually\n    2. Code Review: Review file structure for testability\n    3. Refactoring: Consider refactoring complex functions\n    4. Dependencies: Check if external deps need custom mocking\nFAILURE #2:\n  File: file2.go\n  Coverage: 30.0%\n  Duration: 2s\n  Error: test error 2\n  Likely Causes:\n    - File has complex dependencies difficult to mock\n    - Code structure doesn't follow standard Go testing patterns\n    - Import or dependency issues\n    - File requires manual test setup or custom mocking\n  Recommended Actions:\n    1. Manual Test Creation: Consider creating tests manually\n    2. Code Review: Review file structure for testability\n    3. Refactoring: Consider refactoring complex functions\n    4. Dependencies: Check if external deps need custom mocking\nKubeEdge Testing Guidelines:\n  - Use gomonkey v2 for mocking external functions\n  - Follow table-driven test patterns\n  - Use github.com/stretchr/testify/assert for assertions\n  - Ensure tests are independent and repeatable\nResources:\n  - KubeEdge Testing Documentation: https://github.com/kubeedge/kubeedge/blob/master/docs/testing.md\n  - Go Testing Best Practices: https://golang.org/doc/tutorial/add-a-test\n  - gomonkey Documentation: https://github.com/agiledragon/gomonkey\nPR Author: @test2\nOriginal PR: #456",
		},
	}

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	var logOutput []string
	patches.ApplyFunc(log.Printf, func(format string, v ...interface{}) {
		logOutput = append(logOutput, fmt.Sprintf(format, v...))
	})

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logFailureDetails(tc.results, tc.prAuthor, tc.prNumber)
			assert.Equal(t, tc.expectedOutput, strings.ReplaceAll(strings.Join(logOutput, "\n"), "\n=", "")) //removing the = from the output for easier comparison
			logOutput = []string{}
		})
	}
}

// TestGetWorkingDir tests the getWorkingDir function.  This test is inherently dependent on the runtime environment.
func TestGetWorkingDir(t *testing.T) {
	expectedDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Could not get working directory: %v", err)
	}
	assert.Equal(t, expectedDir, getWorkingDir())
}


// TestProcessFile tests the processFile function.  This requires extensive mocking due to its dependencies.
func TestProcessFile(t *testing.T) {
	//Extensive mocking would be needed here for CoverageAnalyzer, KubeEdgeTestGenerator, and PRCreator.  This is omitted for brevity but would follow the gomonkey patterns specified.  The example below shows a skeletal structure.

	//Mock objects
	mockAnalyzer := &mockCoverageAnalyzer{}
	mockGenerator := &mockKubeEdgeTestGenerator{}
	mockCreator := &mockPRCreator{}

	//Test cases (simplified due to mocking complexity)
	testCases := []struct {
		name      string
		needsTest bool
		functions []FunctionInfo
		err       error
		expSuccess bool
	}{
		{name: "Success", needsTest: true, functions: []FunctionInfo{{Name: "TestFunc"}}, err: nil, expSuccess: true},
		{name: "No functions", needsTest: true, functions: []FunctionInfo{}, err: nil, expSuccess: true},
		{name: "Coverage sufficient", needsTest: false, functions: []FunctionInfo{{Name: "TestFunc"}}, err: nil, expSuccess: true},
		{name: "Analysis error", needsTest: true, functions: []FunctionInfo{{Name: "TestFunc"}}, err: fmt.Errorf("mock error"), expSuccess: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockAnalyzer.On("AnalyzeFile", gomonkey.Any(), gomonkey.Any(), gomonkey.Any()).Return(tc.needsTest, 80.0, tc.err)
			mockAnalyzer.On("ExtractModifiedFunctions", gomonkey.Any(), gomonkey.Any()).Return(tc.functions, nil)
			mockGenerator.On("GenerateTests", gomonkey.Any(), gomonkey.Any(), gomonkey.Any(), gomonkey.Any()).Return("test content", nil)
			mockCreator.On("CreateTestsPR", gomonkey.Any(), gomonkey.Any(), gomonkey.Any(), gomonkey.Any()).Return(1, nil)

			result := processFile(context.Background(), "testfile.go", &Config{}, mockAnalyzer, mockGenerator, mockCreator)
			assert.Equal(t, tc.expSuccess, result.Success)
		})
	}
}

// TestGenerateTestsWithRetry tests the generateTestsWithRetry function.
func TestGenerateTestsWithRetry(t *testing.T) {
	//Mock Generator
	mockGenerator := &mockKubeEdgeTestGenerator{}

	//Test cases
	testCases := []struct {
		name            string
		maxAttempts     int
		mockGeneratorErr []error
		isValid         bool
		expSuccess      bool
	}{
		{name: "Success on first attempt", maxAttempts: 1, mockGeneratorErr: []error{nil}, isValid: true, expSuccess: true},
		{name: "Success after retry", maxAttempts: 3, mockGeneratorErr: []error{fmt.Errorf("mock error"), nil}, isValid: true, expSuccess: true},
		{name: "All attempts fail", maxAttempts: 2, mockGeneratorErr: []error{fmt.Errorf("mock error"), fmt.Errorf("mock error")}, isValid: false, expSuccess: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			i := 0
			mockGenerator.On("GenerateTests", gomonkey.Any(), gomonkey.Any(), gomonkey.Any(), gomonkey.Any()).Return("test content", tc.mockGeneratorErr[i]).Run(func(args gomonkey.Arguments) {
				i++
			})
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFunc(isValidGoTestContent, func(content string) bool {
				return tc.isValid
			})
			testContent, success := generateTestsWithRetry(context.Background(), "testfile.go", []FunctionInfo{}, mockGenerator, tc.maxAttempts)
			assert.Equal(t, tc.expSuccess, success)
			if tc.expSuccess {
				assert.Equal(t, "test content", testContent)
			} else {
				assert.Equal(t, "", testContent)
			}
		})
	}
}

// TestIsValidGoTestContent tests the isValidGoTestContent function.
func TestIsValidGoTestContent(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected bool
	}{
		{name: "Valid content", content: "package test\nimport \"testing\"\nfunc TestExample(t *testing.T){}", expected: true},
		{name: "Missing package", content: "import \"testing\"\nfunc TestExample(t *testing.T){}", expected: false},
		{name: "Missing import", content: "package test\nfunc TestExample(t *testing.T){}", expected: false},
		{name: "Missing func Test", content: "package test\nimport \"testing\"\nfunc Example(t *testing.T){}", expected: false},
		{name: "Missing testing", content: "package test\nimport \"fmt\"\nfunc TestExample(t *testing.T){}", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, isValidGoTestContent(tc.content))
		})
	}
}

// TestParseFlags tests the parseFlags function.  This requires mocking os.Getenv.
func TestParseFlags(t *testing.T) {
	//Mock os.Getenv
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(os.Getenv, func(key string) string {
		switch key {
		case "GITHUB_TOKEN":
			return "mock-github-token"
		case "GEMINI_API_KEY":
			return "mock-gemini-api-key"
		case "DEBUG":
			return "true"
		default:
			return ""
		}
	})

	//Set command line arguments
	os.Args = []string{"test-generator", "-pr-number=123", "-changed-files=file1.go", "-repo-owner=test", "-repo-name=testrepo", "-coverage-threshold=50", "-max-retry-attempts=5", "-pr-author=testuser", "-coverage-file=coverage.out", "-debug=true"}

	config := parseFlags()
	assert.Equal(t, "123", config.PRNumber)
	assert.Equal(t, "file1.go", config.ChangedFiles)
	assert.Equal(t, "test", config.RepoOwner)
	assert.Equal(t, "testrepo", config.RepoName)
	assert.Equal(t, "mock-github-token", config.GithubToken)
	assert.Equal(t, "mock-gemini-api-key", config.GeminiAPIKey)
	assert.Equal(t, 50.0, config.CoverageThreshold)
	assert.Equal(t, 5, config.MaxRetryAttempts)
	assert.Equal(t, "testuser", config.PRAuthor)
	assert.Equal(t, "coverage.out", config.CoverageFile)
	assert.Equal(t, true, config.Debug)
}

// TestValidateConfig tests the validateConfig function.
func TestValidateConfig(t *testing.T) {
	testCases := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{name: "Valid config", config: Config{GeminiAPIKey: "key", GithubToken: "token", RepoOwner: "owner", RepoName: "repo", CoverageThreshold: 50, MaxRetryAttempts: 3}, wantErr: false},
		{name: "Missing GeminiAPIKey", config: Config{GithubToken: "token", RepoOwner: "owner", RepoName: "repo", CoverageThreshold: 50, MaxRetryAttempts: 3}, wantErr: true},
		{name: "Missing GithubToken", config: Config{GeminiAPIKey: "key", RepoOwner: "owner", RepoName: "repo", CoverageThreshold: 50, MaxRetryAttempts: 3}, wantErr: true},
		{name: "Missing RepoOwner", config: Config{GeminiAPIKey: "key", GithubToken: "token", RepoName: "repo", CoverageThreshold: 50, MaxRetryAttempts: 3}, wantErr: true},
		{name: "Missing RepoName", config: Config{GeminiAPIKey: "key", GithubToken: "token", RepoOwner: "owner", CoverageThreshold: 50, MaxRetryAttempts: 3}, wantErr: true},
		{name: "Invalid CoverageThreshold", config: Config{GeminiAPIKey: "key", GithubToken: "token", RepoOwner: "owner", RepoName: "repo", CoverageThreshold: 150, MaxRetryAttempts: 3}, wantErr: true},
		{name: "Invalid MaxRetryAttempts", config: Config{GeminiAPIKey: "key", GithubToken: "token", RepoOwner: "owner", RepoName: "repo", CoverageThreshold: 50, MaxRetryAttempts: 15}, wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateConfig(&tc.config)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSetupLogging tests the setupLogging function. This test is inherently dependent on the runtime environment and file system.  A more robust test would involve mocking os operations.
func TestSetupLogging(t *testing.T) {
	err := setupLogging(true)
	assert.NoError(t, err)
	//Cleanup - remove the log directory.  Error handling omitted for brevity.
	os.RemoveAll("logs")
}


// Mock objects for processFile test
type mockCoverageAnalyzer struct {
	mock.Mock
}

func (m *mockCoverageAnalyzer) AnalyzeFile(ctx context.Context, filePath string, threshold float64) (bool, float64, error) {
	args := m.Called(ctx, filePath, threshold)
	return args.Bool(0), args.Float64(1), args.Error(2)
}

func (m *mockCoverageAnalyzer) ExtractModifiedFunctions(ctx context.Context, filePath string) ([]FunctionInfo, error) {
	args := m.Called(ctx, filePath)
	return args.Get(0).([]FunctionInfo), args.Error(1)
}

type mockKubeEdgeTestGenerator struct {
	mock.Mock
}

func (m *mockKubeEdgeTestGenerator) GenerateTests(ctx context.Context, filePath string, functions []FunctionInfo, lastError error) (string, error) {
	args := m.Called(ctx, filePath, functions, lastError)
	return args.String(0), args.Error(1)
}

type mockPRCreator struct {
	mock.Mock
}

func (m *mockPRCreator) CreateTestsPR(ctx context.Context, filePath string, testContent string, coverage float64) (int, error) {
	args := m.Called(ctx, filePath, testContent, coverage)
	return args.Int(0), args.Error(1)
}

type FunctionInfo struct {
	Name string
}