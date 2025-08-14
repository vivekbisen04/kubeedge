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

// FixCommand handles fixing existing generated tests
func FixCommand(kubeedgeRoot, testFile, component string, backup, dryRun bool) {

	fmt.Println("🔧 KubeEdge E2E Test Fixer")
	fmt.Println("==========================")

	generatedDir := filepath.Join(kubeedgeRoot, "tests", "e2e", "generated")

	// Check if generated directory exists
	if _, err := os.Stat(generatedDir); os.IsNotExist(err) {
		log.Fatalf("❌ Generated tests directory not found: %s", generatedDir)
	}

	var filesToFix []string

	if testFile != "" {
		// Fix specific file
		filePath := filepath.Join(generatedDir, testFile)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Fatalf("❌ Test file not found: %s", filePath)
		}
		filesToFix = append(filesToFix, filePath)
	} else if component != "" {
		// Fix files for specific component
		pattern := filepath.Join(generatedDir, component+"*_test.go")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			log.Fatalf("❌ Error finding files for component %s: %v", component, err)
		}
		filesToFix = matches
	} else {
		// Fix all generated test files
		pattern := filepath.Join(generatedDir, "*_test.go")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			log.Fatalf("❌ Error finding test files: %v", err)
		}
		filesToFix = matches
	}

	if len(filesToFix) == 0 {
		fmt.Println("✅ No test files found to fix")
		return
	}

	fmt.Printf("📁 Found %d test file(s) to fix:\n", len(filesToFix))
	for _, file := range filesToFix {
		fmt.Printf("   - %s\n", filepath.Base(file))
	}

	// Process each file
	for _, filePath := range filesToFix {
		fmt.Printf("\n🔍 Processing: %s\n", filepath.Base(filePath))
		
		if err := fixTestFile(filePath, backup, dryRun); err != nil {
			log.Printf("❌ Error fixing %s: %v", filepath.Base(filePath), err)
		} else {
			fmt.Printf("✅ Fixed: %s\n", filepath.Base(filePath))
		}
	}

	fmt.Println("\n🎉 Test fixing complete!")
}

// fixTestFile fixes a single test file
func fixTestFile(filePath string, backup bool, dryRun bool) error {
	// Read original file
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	originalCode := string(content)
	
	// Determine component from filename
	filename := filepath.Base(filePath)
	component := strings.TrimSuffix(filename, "_generated_test.go")
	component = strings.TrimSuffix(component, "_test.go")

	// Validate the current code
	codeValidator := validator.NewCodeValidator()
	validationResult, err := codeValidator.ValidateGeneratedTest(originalCode, component)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	fmt.Printf("   📊 Validation Results:\n")
	fmt.Printf("      - Valid: %v\n", validationResult.IsValid)
	fmt.Printf("      - Errors: %d\n", len(validationResult.Errors))
	fmt.Printf("      - Warnings: %d\n", len(validationResult.Warnings))

	// Show errors and warnings
	for _, err := range validationResult.Errors {
		fmt.Printf("      ❌ Error: %s\n", err)
	}
	for _, warning := range validationResult.Warnings {
		fmt.Printf("      ⚠️  Warning: %s\n", warning)
	}

	// Apply fixes
	fixer := validator.NewTestFixer(component)
	fixedCode, appliedFixes := fixer.FixGeneratedTest(originalCode)

	if len(appliedFixes) > 0 {
		fmt.Printf("   🔧 Applied fixes:\n")
		for _, fix := range appliedFixes {
			fmt.Printf("      - %s\n", fix)
		}
	} else {
		fmt.Printf("   ✅ No fixes needed\n")
		return nil
	}

	if dryRun {
		fmt.Printf("   📄 [DRY RUN] Would write fixed code to: %s\n", filePath)
		return nil
	}

	// Create backup if requested
	if backup {
		backupPath := filePath + ".backup"
		if err := ioutil.WriteFile(backupPath, content, 0644); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		fmt.Printf("   📦 Backup created: %s\n", filepath.Base(backupPath))
	}

	// Write fixed code
	if err := ioutil.WriteFile(filePath, []byte(fixedCode), 0644); err != nil {
		return fmt.Errorf("failed to write fixed code: %w", err)
	}

	return nil
}