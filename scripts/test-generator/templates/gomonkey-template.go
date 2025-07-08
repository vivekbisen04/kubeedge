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

// GoMonkeyTestTemplate provides KubeEdge-specific gomonkey test patterns
const GoMonkeyTestTemplate = `
// KubeEdge gomonkey Test Template
// This template shows common patterns for mocking in KubeEdge tests

package {{.PackageName}}

import (
	"errors"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	
	// Add other imports as needed
)

// Example 1: Basic Function Mocking
func TestFunctionWithExternalCall(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	// Mock external function call
	patches.ApplyFunc(externalFunction, func(param string) error {
		return nil
	})

	// Test your function
	err := functionUnderTest("test-param")
	assert.NoError(t, err)
}

// Example 2: Method Mocking (Common in KubeEdge)
func TestFunctionWithMethodCall(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	// Create mock object
	mockObj := &MockStruct{}
	
	// Mock method calls
	patches.ApplyMethod(reflect.TypeOf(mockObj), "Method", 
		func(_ *MockStruct, param string) (string, error) {
			return "mocked-result", nil
		})

	// Test your function
	result, err := functionUnderTest(mockObj, "test-param")
	assert.NoError(t, err)
	assert.Equal(t, "expected-result", result)
}

// Example 3: File Operations Mocking (Common in keadm)
func TestFunctionWithFileOperations(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	// Mock os.Stat
	patches.ApplyFunc(os.Stat, func(filename string) (os.FileInfo, error) {
		return nil, nil // File exists
	})

	// Mock os.ReadFile
	patches.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
		return []byte("mocked file content"), nil
	})

	// Mock os.WriteFile
	patches.ApplyFunc(os.WriteFile, func(filename string, data []byte, perm os.FileMode) error {
		return nil
	})

	// Test your function
	err := functionUnderTest("test-file.txt")
	assert.NoError(t, err)
}

// Example 4: Command Execution Mocking (Common in keadm)
func TestFunctionWithCommandExecution(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	// Mock command execution
	mockCmd := &MockCommand{}
	patches.ApplyFunc(exec.Command, func(name string, args ...string) *exec.Cmd {
		return &exec.Cmd{} // Return mock command
	})

	// Mock command methods
	patches.ApplyMethod(reflect.TypeOf(&exec.Cmd{}), "CombinedOutput", 
		func(_ *exec.Cmd) ([]byte, error) {
			return []byte("command output"), nil
		})

	patches.ApplyMethod(reflect.TypeOf(&exec.Cmd{}), "Run", 
		func(_ *exec.Cmd) error {
			return nil
		})

	// Test your function
	output, err := functionUnderTest("test-command")
	assert.NoError(t, err)
	assert.Equal(t, "expected-output", output)
}

// Example 5: Kubernetes Client Mocking (Common in cloud components)
func TestFunctionWithKubernetesClient(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	// Mock Kubernetes client creation
	mockClient := &MockKubernetesClient{}
	patches.ApplyFunc(kubernetes.NewForConfig, func(config *rest.Config) (*kubernetes.Clientset, error) {
		return &kubernetes.Clientset{}, nil
	})

	// Mock client operations
	patches.ApplyMethod(reflect.TypeOf(mockClient), "Get", 
		func(_ *MockKubernetesClient, ctx context.Context, name string, opts metav1.GetOptions) (*v1.Pod, error) {
			return &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: name},
			}, nil
		})

	// Test your function
	pod, err := functionUnderTest("test-pod")
	assert.NoError(t, err)
	assert.Equal(t, "test-pod", pod.Name)
}

// Example 6: Database Operations Mocking (Beego ORM - Common in edge components)
func TestFunctionWithDatabaseOperations(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	// Mock ORM operations
	mockOrmer := &MockOrmer{}
	patches.ApplyFunc(orm.NewOrmUsingDB, func(aliasName string) orm.Ormer {
		return mockOrmer
	})

	// Mock database operations
	patches.ApplyMethod(reflect.TypeOf(mockOrmer), "Insert", 
		func(_ *MockOrmer, md interface{}) (int64, error) {
			return 1, nil
		})

	patches.ApplyMethod(reflect.TypeOf(mockOrmer), "Read", 
		func(_ *MockOrmer, md interface{}, cols ...string) error {
			return nil
		})

	// Test your function
	id, err := functionUnderTest(&TestModel{Name: "test"})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), id)
}

// Example 7: Error Case Testing with gomonkey
func TestFunctionErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		mockSetup   func(*gomonkey.Patches)
		expectError bool
		errorMsg    string
	}{
		{
			name: "File not found error",
			mockSetup: func(patches *gomonkey.Patches) {
				patches.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
					return nil, os.ErrNotExist
				})
			},
			expectError: true,
			errorMsg:    "file not found",
		},
		{
			name: "Command execution error",
			mockSetup: func(patches *gomonkey.Patches) {
				patches.ApplyMethod(reflect.TypeOf(&exec.Cmd{}), "Run", 
					func(_ *exec.Cmd) error {
						return errors.New("command failed")
					})
			},
			expectError: true,
			errorMsg:    "command failed",
		},
		{
			name: "Success case",
			mockSetup: func(patches *gomonkey.Patches) {
				patches.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
					return []byte("success"), nil
				})
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			tt.mockSetup(patches)

			err := functionUnderTest("test-param")

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Example 8: Complex Scenario with Multiple Mocks
func TestComplexScenarioWithMultipleMocks(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	// Setup multiple mocks for complex scenario
	patches.ApplyFunc(os.Stat, func(filename string) (os.FileInfo, error) {
		return &MockFileInfo{name: "test-file", size: 100}, nil
	})

	patches.ApplyFunc(exec.Command, func(name string, args ...string) *exec.Cmd {
		return &exec.Cmd{}
	})

	patches.ApplyMethod(reflect.TypeOf(&exec.Cmd{}), "CombinedOutput", 
		func(_ *exec.Cmd) ([]byte, error) {
			return []byte("success output"), nil
		})

	patches.ApplyFunc(kubernetes.NewForConfig, func(config *rest.Config) (*kubernetes.Clientset, error) {
		return &kubernetes.Clientset{}, nil
	})

	// Test complex function
	result, err := complexFunctionUnderTest("test-param")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// Mock structs for testing
type MockStruct struct{}
func (m *MockStruct) Method(param string) (string, error) { return "", nil }

type MockCommand struct{}
func (m *MockCommand) Run() error { return nil }
func (m *MockCommand) CombinedOutput() ([]byte, error) { return nil, nil }

type MockKubernetesClient struct{}
func (m *MockKubernetesClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Pod, error) {
	return nil, nil
}

type MockOrmer struct{}
func (m *MockOrmer) Insert(md interface{}) (int64, error) { return 0, nil }
func (m *MockOrmer) Read(md interface{}, cols ...string) error { return nil }

type MockFileInfo struct {
	name string
	size int64
}
func (m *MockFileInfo) Name() string       { return m.name }
func (m *MockFileInfo) Size() int64        { return m.size }
func (m *MockFileInfo) Mode() os.FileMode  { return 0644 }
func (m *MockFileInfo) ModTime() time.Time { return time.Now() }
func (m *MockFileInfo) IsDir() bool        { return false }
func (m *MockFileInfo) Sys() interface{}   { return nil }

type TestModel struct {
	Name string
}
`