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

package templates

// StandardTestTemplate provides KubeEdge-specific standard Go test patterns
const StandardTestTemplate = `
// KubeEdge Standard Go Test Template
// This template shows common patterns for standard Go testing in KubeEdge

package {{.PackageName}}

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	
	// Add other imports as needed
)

// Example 1: Basic Function Testing
func TestBasicFunction(t *testing.T) {
	// Test successful case
	result, err := functionUnderTest("valid-input")
	assert.NoError(t, err)
	assert.Equal(t, "expected-result", result)

	// Test error case
	_, err = functionUnderTest("invalid-input")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid input")
}

// Example 2: Table-Driven Tests (Recommended for KubeEdge)
func TestFunctionWithTableDrivenTests(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid input",
			input:       "valid-input",
			expected:    "expected-output",
			expectError: false,
		},
		{
			name:        "Empty input",
			input:       "",
			expected:    "",
			expectError: true,
			errorMsg:    "input cannot be empty",
		},
		{
			name:        "Invalid format",
			input:       "invalid-format",
			expected:    "",
			expectError: true,
			errorMsg:    "invalid format",
		},
		{
			name:        "Special characters",
			input:       "special@#$",
			expected:    "processed-special",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := functionUnderTest(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// Example 3: Struct Method Testing
func TestStructMethods(t *testing.T) {
	// Setup
	config := &Config{
		Name:    "test-config",
		Version: "v1.0.0",
		Port:    8080,
	}
	component := NewComponent(config)
	require.NotNil(t, component)

	// Test getter methods
	assert.Equal(t, "test-config", component.GetName())
	assert.Equal(t, "v1.0.0", component.GetVersion())
	assert.Equal(t, 8080, component.GetPort())

	// Test setter methods
	component.SetName("updated-config")
	assert.Equal(t, "updated-config", component.GetName())

	// Test validation
	assert.True(t, component.IsValid())
	
	component.SetPort(-1)
	assert.False(t, component.IsValid())
}

// Example 4: Error Handling Tests
func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() error
		cleanup  func() error
		wantErr  bool
		errType  string
	}{
		{
			name: "Successful operation",
			setup: func() error {
				return setupTestEnvironment()
			},
			cleanup: func() error {
				return cleanupTestEnvironment()
			},
			wantErr: false,
		},
		{
			name: "Setup failure",
			setup: func() error {
				return errors.New("setup failed")
			},
			cleanup: func() error {
				return nil
			},
			wantErr: true,
			errType: "setup",
		},
		{
			name: "Cleanup failure",
			setup: func() error {
				return nil
			},
			cleanup: func() error {
				return errors.New("cleanup failed")
			},
			wantErr: true,
			errType: "cleanup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			setupErr := tt.setup()
			if setupErr != nil && tt.errType == "setup" {
				assert.Error(t, setupErr)
				return
			}
			require.NoError(t, setupErr)

			// Test the function
			err := functionUnderTest()
			
			// Cleanup
			cleanupErr := tt.cleanup()
			if tt.errType == "cleanup" {
				assert.Error(t, cleanupErr)
			}

			// Assertions
			if tt.wantErr && tt.errType != "cleanup" {
				assert.Error(t, err)
			} else if !tt.wantErr {
				assert.NoError(t, err)
			}
		})
	}
}

// Example 5: Configuration Validation Tests
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		valid  bool
		errors []string
	}{
		{
			name: "Valid configuration",
			config: &Config{
				Name:     "valid-config",
				Version:  "v1.0.0",
				Port:     8080,
				Endpoint: "https://api.example.com",
			},
			valid: true,
		},
		{
			name: "Missing name",
			config: &Config{
				Version:  "v1.0.0",
				Port:     8080,
				Endpoint: "https://api.example.com",
			},
			valid:  false,
			errors: []string{"name is required"},
		},
		{
			name: "Invalid port",
			config: &Config{
				Name:     "test-config",
				Version:  "v1.0.0",
				Port:     -1,
				Endpoint: "https://api.example.com",
			},
			valid:  false,
			errors: []string{"port must be positive"},
		},
		{
			name: "Invalid endpoint",
			config: &Config{
				Name:     "test-config",
				Version:  "v1.0.0",
				Port:     8080,
				Endpoint: "invalid-url",
			},
			valid:  false,
			errors: []string{"invalid endpoint URL"},
		},
		{
			name: "Multiple errors",
			config: &Config{
				Version:  "v1.0.0",
				Port:     -1,
				Endpoint: "invalid-url",
			},
			valid:  false,
			errors: []string{"name is required", "port must be positive", "invalid endpoint URL"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)

			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				for _, expectedError := range tt.errors {
					assert.Contains(t, err.Error(), expectedError)
				}
			}
		})
	}
}

// Example 6: String Manipulation and Parsing Tests
func TestStringOperations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Empty string", "", ""},
		{"Single word", "hello", "Hello"},
		{"Multiple words", "hello world", "Hello World"},
		{"Already capitalized", "Hello World", "Hello World"},
		{"Mixed case", "hELLo WoRLd", "Hello World"},
		{"With numbers", "hello123 world456", "Hello123 World456"},
		{"With special chars", "hello-world_test", "Hello-World_Test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CapitalizeWords(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Example 7: Slice and Map Operations Tests
func TestSliceOperations(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		filter   string
		expected []string
	}{
		{
			name:     "Filter existing items",
			input:    []string{"apple", "banana", "cherry", "apple"},
			filter:   "apple",
			expected: []string{"banana", "cherry"},
		},
		{
			name:     "Filter non-existing items",
			input:    []string{"apple", "banana", "cherry"},
			filter:   "orange",
			expected: []string{"apple", "banana", "cherry"},
		},
		{
			name:     "Empty slice",
			input:    []string{},
			filter:   "apple",
			expected: []string{},
		},
		{
			name:     "Single item slice - match",
			input:    []string{"apple"},
			filter:   "apple",
			expected: []string{},
		},
		{
			name:     "Single item slice - no match",
			input:    []string{"apple"},
			filter:   "banana",
			expected: []string{"apple"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterSlice(tt.input, tt.filter)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Example 8: Map Operations Tests
func TestMapOperations(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]int
		key      string
		value    int
		expected map[string]int
	}{
		{
			name:     "Add new key",
			input:    map[string]int{"a": 1, "b": 2},
			key:      "c",
			value:    3,
			expected: map[string]int{"a": 1, "b": 2, "c": 3},
		},
		{
			name:     "Update existing key",
			input:    map[string]int{"a": 1, "b": 2},
			key:      "a",
			value:    10,
			expected: map[string]int{"a": 10, "b": 2},
		},
		{
			name:     "Empty map",
			input:    map[string]int{},
			key:      "a",
			value:    1,
			expected: map[string]int{"a": 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UpdateMap(tt.input, tt.key, tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Example 9: Concurrent Operations Tests
func TestConcurrentOperations(t *testing.T) {
	// Test thread-safe operations
	counter := NewThreadSafeCounter()
	
	// Test concurrent increments
	numGoroutines := 100
	numIncrements := 10

	done := make(chan bool, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numIncrements; j++ {
				counter.Increment()
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	expected := numGoroutines * numIncrements
	assert.Equal(t, expected, counter.GetValue())
}

// Example 10: File Operations Tests (Common in KubeEdge)
func TestFileOperations(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()
	
	tests := []struct {
		name        string
		filename    string
		content     string
		expectError bool
	}{
		{
			name:        "Create and read file",
			filename:    "test1.txt",
			content:     "Hello, World!",
			expectError: false,
		},
		{
			name:        "Create file with special content",
			filename:    "test2.txt",
			content:     "Special chars: !@#$%^&*()",
			expectError: false,
		},
		{
			name:        "Create file with empty content",
			filename:    "test3.txt",
			content:     "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filepath := tempDir + "/" + tt.filename
			
			// Write file
			err := WriteFile(filepath, tt.content)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// Read file
			content, err := ReadFile(filepath)
			assert.NoError(t, err)
			assert.Equal(t, tt.content, content)

			// Check file exists
			exists := FileExists(filepath)
			assert.True(t, exists)
		})
	}
}

// Example 11: Benchmark Tests (Performance Testing)
func BenchmarkFunction(b *testing.B) {
	input := "test-input-string"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = functionUnderTest(input)
	}
}

func BenchmarkFunctionWithSetup(b *testing.B) {
	// Setup that shouldn't be measured
	setup := prepareTestData()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = functionWithSetup(setup)
	}
}

// Example 12: Helper Functions and Utilities
func TestHelperFunctions(t *testing.T) {
	// Test helper function
	assert.True(t, isValidEmail("user@example.com"))
	assert.False(t, isValidEmail("invalid-email"))
	
	// Test utility function
	result := normalizeString("  Hello World  ")
	assert.Equal(t, "hello world", result)
	
	// Test conversion function
	value, err := stringToInt("123")
	assert.NoError(t, err)
	assert.Equal(t, 123, value)
	
	_, err = stringToInt("not-a-number")
	assert.Error(t, err)
}

// Helper functions for tests
func setupTestEnvironment() error {
	// Setup logic here
	return nil
}

func cleanupTestEnvironment() error {
	// Cleanup logic here
	return nil
}

// Mock structs and functions for examples
type Config struct {
	Name     string
	Version  string
	Port     int
	Endpoint string
}

type Component struct {
	config *Config
}

func NewComponent(config *Config) *Component {
	return &Component{config: config}
}

func (c *Component) GetName() string    { return c.config.Name }
func (c *Component) GetVersion() string { return c.config.Version }
func (c *Component) GetPort() int       { return c.config.Port }
func (c *Component) SetName(name string) { c.config.Name = name }
func (c *Component) SetPort(port int)   { c.config.Port = port }
func (c *Component) IsValid() bool      { return c.config.Port > 0 }

type ThreadSafeCounter struct {
	mu    sync.Mutex
	value int
}

func NewThreadSafeCounter() *ThreadSafeCounter {
	return &ThreadSafeCounter{}
}

func (c *ThreadSafeCounter) Increment() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value++
}

func (c *ThreadSafeCounter) GetValue() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value
}

// Example functions to be tested
func functionUnderTest(input string) (string, error) {
	if input == "" {
		return "", errors.New("input cannot be empty")
	}
	if input == "invalid-input" {
		return "", errors.New("invalid input")
	}
	return "processed-" + input, nil
}

func ValidateConfig(config *Config) error {
	var errs []string
	
	if config.Name == "" {
		errs = append(errs, "name is required")
	}
	if config.Port <= 0 {
		errs = append(errs, "port must be positive")
	}
	if config.Endpoint != "" && !isValidURL(config.Endpoint) {
		errs = append(errs, "invalid endpoint URL")
	}
	
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func CapitalizeWords(input string) string {
	// Implementation here
	return strings.Title(strings.ToLower(input))
}

func FilterSlice(slice []string, filter string) []string {
	var result []string
	for _, item := range slice {
		if item != filter {
			result = append(result, item)
		}
	}
	return result
}

func UpdateMap(m map[string]int, key string, value int) map[string]int {
	result := make(map[string]int)
	for k, v := range m {
		result[k] = v
	}
	result[key] = value
	return result
}

func WriteFile(filepath, content string) error {
	return os.WriteFile(filepath, []byte(content), 0644)
}

func ReadFile(filepath string) (string, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func FileExists(filepath string) bool {
	_, err := os.Stat(filepath)
	return !os.IsNotExist(err)
}

func isValidEmail(email string) bool {
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

func normalizeString(input string) string {
	return strings.ToLower(strings.TrimSpace(input))
}

func stringToInt(s string) (int, error) {
	return strconv.Atoi(s)
}

func isValidURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

func prepareTestData() interface{} {
	return "test-data"
}

func functionWithSetup(setup interface{}) string {
	return "result"
}
`