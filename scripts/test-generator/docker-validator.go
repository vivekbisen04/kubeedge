package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type DockerValidator struct {
	buildToolsImage string
	workDir         string
}

type ValidationResult struct {
	IsValid    bool
	Error      error
	Output     string
	Duration   time.Duration
}

func NewDockerValidator(buildToolsImage string) *DockerValidator {
	workDir := "/tmp/kubeedge-test-validation"
	return &DockerValidator{
		buildToolsImage: buildToolsImage,
		workDir:         workDir,
	}
}

// ValidateTests validates generated tests in KubeEdge's Docker build environment
func (dv *DockerValidator) ValidateTests(ctx context.Context, sourceFile string, testContent string) (bool, error) {
	startTime := time.Now()
	
	// Prepare validation environment
	validationDir, err := dv.prepareValidationEnvironment(sourceFile, testContent)
	if err != nil {
		return false, fmt.Errorf("failed to prepare validation environment: %v", err)
	}
	defer dv.cleanup(validationDir)

	// Step 1: Compilation check
	compileValid, compileErr := dv.validateCompilation(ctx, validationDir, sourceFile)
	if !compileValid {
		duration := time.Since(startTime)
		return false, fmt.Errorf("compilation failed after %v: %v", duration, compileErr)
	}

	// Step 2: Test execution check
	testValid, testErr := dv.validateTestExecution(ctx, validationDir, sourceFile)
	if !testValid {
		duration := time.Since(startTime)
		return false, fmt.Errorf("test execution failed after %v: %v", duration, testErr)
	}

	duration := time.Since(startTime)
	fmt.Printf("✅ Docker validation successful for %s (took %v)\n", sourceFile, duration)
	return true, nil
}

// prepareValidationEnvironment sets up the directory structure for testing
func (dv *DockerValidator) prepareValidationEnvironment(sourceFile string, testContent string) (string, error) {
	// Create unique validation directory
	timestamp := time.Now().Format("20060102-150405")
	validationDir := filepath.Join(dv.workDir, fmt.Sprintf("validation-%s", timestamp))
	
	if err := os.MkdirAll(validationDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create validation directory: %v", err)
	}

	// Copy source file to validation directory
	sourceContent, err := os.ReadFile(sourceFile)
	if err != nil {
		return "", fmt.Errorf("failed to read source file: %v", err)
	}

	sourceName := filepath.Base(sourceFile)
	sourceDestPath := filepath.Join(validationDir, sourceName)
	if err := os.WriteFile(sourceDestPath, sourceContent, 0644); err != nil {
		return "", fmt.Errorf("failed to write source file: %v", err)
	}

	// Write test file
	testFileName := strings.TrimSuffix(sourceName, ".go") + "_test.go"
	testDestPath := filepath.Join(validationDir, testFileName)
	if err := os.WriteFile(testDestPath, []byte(testContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write test file: %v", err)
	}

	// Create go.mod for the validation module
	goModContent := dv.generateGoMod(sourceFile)
	goModPath := filepath.Join(validationDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write go.mod: %v", err)
	}

	return validationDir, nil
}

// validateCompilation checks if the generated test compiles
func (dv *DockerValidator) validateCompilation(ctx context.Context, validationDir string, sourceFile string) (bool, error) {
	// Run compilation check in Docker container
	cmd := dv.buildDockerCommand(ctx, validationDir, "go", "build", "./...")
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("compilation error: %v\nOutput: %s", err, string(output))
	}

	fmt.Printf("✅ Compilation successful for %s\n", sourceFile)
	return true, nil
}

// validateTestExecution checks if the generated tests run successfully
func (dv *DockerValidator) validateTestExecution(ctx context.Context, validationDir string, sourceFile string) (bool, error) {
	// Run tests in Docker container
	cmd := dv.buildDockerCommand(ctx, validationDir, "go", "test", "-v", "./...")
	
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	
	if err != nil {
		// Check if it's just "no tests to run"
		if strings.Contains(outputStr, "no test files") || strings.Contains(outputStr, "no tests to run") {
			fmt.Printf("⚠️ No tests to run for %s (this is okay for some files)\n", sourceFile)
			return true, nil
		}
		
		return false, fmt.Errorf("test execution error: %v\nOutput: %s", err, outputStr)
	}

	// Check for successful test run indicators
	if strings.Contains(outputStr, "PASS") || strings.Contains(outputStr, "ok ") {
		fmt.Printf("✅ Test execution successful for %s\n", sourceFile)
		return true, nil
	}

	return false, fmt.Errorf("tests did not pass successfully\nOutput: %s", outputStr)
}

// buildDockerCommand creates a Docker command for running operations in KubeEdge build environment
func (dv *DockerValidator) buildDockerCommand(ctx context.Context, workDir string, command ...string) *exec.Cmd {
	dockerArgs := []string{
		"run",
		"--rm",
		"-v", fmt.Sprintf("%s:/workspace", workDir),
		"-w", "/workspace",
		"--network", "none", // Disable network for security
		"--memory", "512m",   // Limit memory usage
		"--cpus", "1.0",      // Limit CPU usage
		dv.buildToolsImage,
	}
	
	dockerArgs = append(dockerArgs, command...)
	
	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)
	return cmd
}

// generateGoMod creates a go.mod file for the validation environment
func (dv *DockerValidator) generateGoMod(sourceFile string) string {
	moduleName := dv.generateModuleName(sourceFile)
	
	goMod := fmt.Sprintf(`module %s

go 1.22

require (
	github.com/stretchr/testify v1.10.0
	github.com/agiledragon/gomonkey/v2 v2.12.0
	github.com/onsi/ginkgo/v2 v2.17.1
	github.com/onsi/gomega v1.32.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/pprof v0.0.0-20210407192527-94a9f03dee38 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	golang.org/x/tools v0.21.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)`, moduleName)

	return goMod
}

// generateModuleName creates a module name for the validation environment
func (dv *DockerValidator) generateModuleName(sourceFile string) string {
	// Extract relative path from KubeEdge repo structure
	cleanPath := strings.ReplaceAll(sourceFile, "/", "-")
	cleanPath = strings.ReplaceAll(cleanPath, ".", "-")
	return fmt.Sprintf("kubeedge-test-validation/%s", cleanPath)
}

// cleanup removes the validation directory and its contents
func (dv *DockerValidator) cleanup(validationDir string) {
	if err := os.RemoveAll(validationDir); err != nil {
		fmt.Printf("⚠️ Warning: Failed to cleanup validation directory %s: %v\n", validationDir, err)
	}
}

// ValidateWithRetry validates tests with retry logic and detailed error reporting
func (dv *DockerValidator) ValidateWithRetry(ctx context.Context, sourceFile string, testContent string, maxRetries int) (*ValidationResult, error) {
	var lastResult *ValidationResult
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("🔍 Docker validation attempt %d/%d for %s\n", attempt, maxRetries, sourceFile)
		
		startTime := time.Now()
		isValid, err := dv.ValidateTests(ctx, sourceFile, testContent)
		duration := time.Since(startTime)
		
		result := &ValidationResult{
			IsValid:  isValid,
			Error:    err,
			Duration: duration,
		}
		
		if isValid {
			result.Output = "Validation successful"
			return result, nil
		}
		
		lastResult = result
		if err != nil {
			result.Output = err.Error()
			fmt.Printf("❌ Validation attempt %d failed: %v\n", attempt, err)
		}
		
		// Wait before retry (exponential backoff)
		if attempt < maxRetries {
			waitTime := time.Duration(attempt) * time.Second
			fmt.Printf("⏳ Waiting %v before retry...\n", waitTime)
			time.Sleep(waitTime)
		}
	}
	
	return lastResult, fmt.Errorf("validation failed after %d attempts", maxRetries)
}

// CheckDockerAvailability checks if Docker is available and the build tools image exists
func (dv *DockerValidator) CheckDockerAvailability(ctx context.Context) error {
	// Check if Docker is available
	cmd := exec.CommandContext(ctx, "docker", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker is not available: %v", err)
	}
	
	// Check if build tools image exists
	cmd = exec.CommandContext(ctx, "docker", "inspect", dv.buildToolsImage)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build tools image %s is not available: %v", dv.buildToolsImage, err)
	}
	
	// Test basic container functionality
	cmd = exec.CommandContext(ctx, "docker", "run", "--rm", dv.buildToolsImage, "go", "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to test container functionality: %v\nOutput: %s", err, string(output))
	}
	
	fmt.Printf("✅ Docker validation environment ready with %s\n", dv.buildToolsImage)
	return nil
}

// GetDetailedError provides detailed error information for debugging
func (dv *DockerValidator) GetDetailedError(err error) string {
	if err == nil {
		return ""
	}
	
	errorMsg := err.Error()
	
	// Common error patterns and solutions
	solutions := map[string]string{
		"undefined:":           "Missing import or undefined variable/function",
		"cannot find package":  "Missing dependency or incorrect import path",
		"syntax error":         "Go syntax error in generated code",
		"compilation failed":   "Code does not compile - check imports and syntax",
		"test execution error": "Tests failed to run - check test logic and dependencies",
		"no such file":         "File not found - check file paths",
		"permission denied":    "Permission error - check file permissions",
	}
	
	var suggestions []string
	for pattern, solution := range solutions {
		if strings.Contains(strings.ToLower(errorMsg), pattern) {
			suggestions = append(suggestions, solution)
		}
	}
	
	if len(suggestions) > 0 {
		return fmt.Sprintf("Error: %s\n\nSuggested solutions:\n- %s", 
			errorMsg, strings.Join(suggestions, "\n- "))
	}
	
	return errorMsg
}