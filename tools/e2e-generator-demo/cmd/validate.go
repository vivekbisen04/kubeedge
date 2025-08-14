package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"e2e-generator-demo/pkg/validator"
)

// ValidateCommand handles validating existing generated tests
func ValidateCommand(kubeedgeRoot, component string) {

	fmt.Println("🔍 KubeEdge E2E Test Validator")
	fmt.Println("==============================")

	generatedDir := filepath.Join(kubeedgeRoot, "tests", "e2e", "generated")

	// Check if generated directory exists
	if _, err := os.Stat(generatedDir); os.IsNotExist(err) {
		log.Fatalf("❌ Generated tests directory not found: %s", generatedDir)
	}

	var filesToValidate []string

	if component != "" {
		// Validate files for specific component
		pattern := filepath.Join(generatedDir, component+"*_test.go")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			log.Fatalf("❌ Error finding files for component %s: %v", component, err)
		}
		filesToValidate = matches
	} else {
		// Validate all generated test files
		pattern := filepath.Join(generatedDir, "*_test.go")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			log.Fatalf("❌ Error finding test files: %v", err)
		}
		filesToValidate = matches
	}

	if len(filesToValidate) == 0 {
		fmt.Println("✅ No test files found to validate")
		return
	}

	fmt.Printf("📁 Found %d test file(s) to validate:\n", len(filesToValidate))
	for _, file := range filesToValidate {
		fmt.Printf("   - %s\n", filepath.Base(file))
	}

	// Validation summary
	totalFiles := len(filesToValidate)
	validFiles := 0
	filesWithErrors := 0
	filesWithWarnings := 0

	// Process each file
	for _, filePath := range filesToValidate {
		fmt.Printf("\n🔍 Validating: %s\n", filepath.Base(filePath))
		
		result, err := validateTestFile(filePath, true)
		if err != nil {
			log.Printf("❌ Error validating %s: %v", filepath.Base(filePath), err)
			continue
		}

		if result.IsValid {
			validFiles++
			fmt.Printf("✅ Valid: %s\n", filepath.Base(filePath))
		} else {
			if len(result.Errors) > 0 {
				filesWithErrors++
			}
			if len(result.Warnings) > 0 {
				filesWithWarnings++
			}
		}
	}

	// Print summary
	fmt.Println("\n📊 Validation Summary:")
	fmt.Printf("   - Total files: %d\n", totalFiles)
	fmt.Printf("   - Valid files: %d\n", validFiles)
	fmt.Printf("   - Files with errors: %d\n", filesWithErrors)
	fmt.Printf("   - Files with warnings: %d\n", filesWithWarnings)

	if filesWithErrors > 0 {
		fmt.Println("\n💡 To fix errors automatically, run:")
		fmt.Printf("   ./e2e-generator --fix --kubeedge-root %s\n", kubeedgeRoot)
	}

	fmt.Println("\n🎉 Validation complete!")
}

// validateTestFile validates a single test file
func validateTestFile(filePath string, verbose bool) (*validator.ValidationResult, error) {
	// Read file
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	originalCode := string(content)
	
	// Determine component from filename
	filename := filepath.Base(filePath)
	component := strings.TrimSuffix(filename, "_generated_test.go")
	component = strings.TrimSuffix(component, "_test.go")

	// Validate the code
	codeValidator := validator.NewCodeValidator()
	result, err := codeValidator.ValidateGeneratedTest(originalCode, component)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Show results
	if verbose || !result.IsValid {
		fmt.Printf("   📊 Results:\n")
		fmt.Printf("      - Valid: %v\n", result.IsValid)
		fmt.Printf("      - Errors: %d\n", len(result.Errors))
		fmt.Printf("      - Warnings: %d\n", len(result.Warnings))

		// Show errors
		for _, err := range result.Errors {
			fmt.Printf("      ❌ Error: %s\n", err)
		}

		// Show warnings
		for _, warning := range result.Warnings {
			fmt.Printf("      ⚠️  Warning: %s\n", warning)
		}

		// Show required fixes
		if len(result.RequiredFixes) > 0 {
			fmt.Printf("   🔧 Suggested fixes:\n")
			for _, fix := range result.RequiredFixes {
				fmt.Printf("      - %s\n", fix)
			}
		}
	}

	return result, nil
}