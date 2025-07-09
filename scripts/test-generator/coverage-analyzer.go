package main

import (
	"bufio"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type CoverageAnalyzer struct {
	fileSet      *token.FileSet
	coverageFile string
}

type FunctionInfo struct {
	Name       string
	Content    string
	StartLine  int
	EndLine    int
	IsExported bool
	HasTests   bool
}

func NewCoverageAnalyzer(coverageFile string) *CoverageAnalyzer {
	return &CoverageAnalyzer{
		fileSet:      token.NewFileSet(),
		coverageFile: coverageFile,
	}
}
// resolveFilePath converts relative paths to absolute paths from repo root
func (ca *CoverageAnalyzer) resolveFilePath(filePath string) string {
	// If it's already an absolute path, return as-is
	if filepath.IsAbs(filePath) {
		return filePath
	}
	
	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		return filePath
	}
	
	// First, try the file path as-is from current directory
	if fileExists(filePath) {
		absPath, _ := filepath.Abs(filePath)
		return absPath
	}
	
	// Find the repository root by looking for go.mod
	repoRoot := ca.findRepoRoot(wd)
	if repoRoot == "" {
		return filePath
	}
	
	// Try resolving from repo root
	resolvedPath := filepath.Join(repoRoot, filePath)
	if fileExists(resolvedPath) {
		return filepath.Clean(resolvedPath)
	}
	
	// If still not found, return the resolved path anyway
	// (the error will be caught later in AnalyzeFile)
	return filepath.Clean(resolvedPath)
}

// findRepoRoot finds the KubeEdge repository root by looking for go.mod
func (ca *CoverageAnalyzer) findRepoRoot(startDir string) string {
	currentDir := startDir
	
	for i := 0; i < 10; i++ { // Limit search to prevent infinite loop
		// Check if go.mod exists and contains kubeedge
		goModPath := filepath.Join(currentDir, "go.mod")
		if fileExists(goModPath) {
			// Read go.mod to verify it's the KubeEdge repository
			content, err := os.ReadFile(goModPath)
			if err == nil && strings.Contains(string(content), "github.com/kubeedge/kubeedge") {
				return currentDir
			}
		}
		
		// Go up one level
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// Reached filesystem root
			break
		}
		currentDir = parentDir
	}
	
	return ""
}

// ALSO UPDATE the AnalyzeFile function to have better error handling
func (ca *CoverageAnalyzer) AnalyzeFile(ctx context.Context, filePath string, threshold float64) (needsTests bool, coverage float64, err error) {
	// Resolve the file path first
	resolvedPath := ca.resolveFilePath(filePath)
	
	// Check if resolved file exists
	if !fileExists(resolvedPath) {
		return false, 0.0, fmt.Errorf("file not found: %s (resolved to: %s)", filePath, resolvedPath)
	}
	
	// Get absolute path
	absPath, err := filepath.Abs(resolvedPath)
	if err != nil {
		return false, 0.0, fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Rest of the existing function remains the same...
	// Check if test file already exists
	testFile := ca.getTestFileName(absPath)

	// Get package directory for coverage analysis
	packageDir := filepath.Dir(absPath)

	// If coverage file is provided, use it; otherwise run coverage analysis
	if ca.coverageFile != "" && fileExists(ca.coverageFile) {
		coverage, err = ca.parseCoverageFromFile(filePath)
		if err != nil {
			// Fallback to live analysis if parsing fails
			coverage, err = ca.runKubeEdgeCoverage(ctx, packageDir)
		}
	} else {
		coverage, err = ca.runKubeEdgeCoverage(ctx, packageDir)
	}

	if err != nil {
		// If coverage analysis fails, assume we need tests
		return true, 0.0, nil
	}

	// Check if test file exists
	if !fileExists(testFile) {
		return true, coverage, nil
	}

	needsTests = coverage < threshold
	return needsTests, coverage, nil
}

// ALSO UPDATE ExtractModifiedFunctions function
func (ca *CoverageAnalyzer) ExtractModifiedFunctions(ctx context.Context, filePath string) ([]FunctionInfo, error) {
	// Resolve the file path first
	resolvedPath := ca.resolveFilePath(filePath)
	
	// Check if resolved file exists
	if !fileExists(resolvedPath) {
		return nil, fmt.Errorf("file not found: %s (resolved to: %s)", filePath, resolvedPath)
	}
	
	// Get absolute path
	absPath, err := filepath.Abs(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Read the file
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Parse the Go file
	node, err := parser.ParseFile(ca.fileSet, absPath, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %v", err)
	}

	var functions []FunctionInfo
	lines := strings.Split(string(content), "\n")

	// Check if test file exists to determine which functions already have tests
	testFile := ca.getTestFileName(absPath)
	existingTests := ca.getExistingTestFunctions(testFile)

	// Extract all functions
	ast.Inspect(node, func(n ast.Node) bool {
		switch fn := n.(type) {
		case *ast.FuncDecl:
			if fn.Name != nil && ca.shouldIncludeFunction(fn) {
				funcInfo := ca.extractFunctionInfo(fn, lines)

				// Check if this function already has tests
				funcInfo.HasTests = ca.hasExistingTests(fn.Name.Name, existingTests)

				functions = append(functions, funcInfo)
			}
		}
		return true
	})

	return functions, nil
}


// runKubeEdgeCoverage runs KubeEdge's coverage analysis using make test
func (ca *CoverageAnalyzer) runKubeEdgeCoverage(ctx context.Context, packageDir string) (float64, error) {
	originalDir, err := os.Getwd()
	if err != nil {
		return 0.0, fmt.Errorf("failed to get current directory: %v", err)
	}

	// Change to repository root (assuming we're in scripts/test-generator)
	repoRoot := filepath.Join(originalDir, "..", "..")
	if err := os.Chdir(repoRoot); err != nil {
		return 0.0, fmt.Errorf("failed to change to repo root: %v", err)
	}
	defer os.Chdir(originalDir)

	// Determine component type for KubeEdge testing
	component := ca.determineKubeEdgeComponent(packageDir)

	// Run KubeEdge's coverage command
	var cmd *exec.Cmd
	if component != "" {
		cmd = exec.CommandContext(ctx, "make", "test", "PROFILE=y", fmt.Sprintf("WHAT=%s", component))
	} else {
		cmd = exec.CommandContext(ctx, "make", "test", "PROFILE=y")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try fallback with go test if make fails
		return ca.fallbackGoCoverage(ctx, packageDir)
	}

	return ca.parseKubeEdgeCoverageOutput(string(output))
}

// determineKubeEdgeComponent determines which KubeEdge component a package belongs to
func (ca *CoverageAnalyzer) determineKubeEdgeComponent(packageDir string) string {
	if strings.Contains(packageDir, "cloud/") {
		return "cloud"
	}
	if strings.Contains(packageDir, "edge/") {
		return "edge"
	}
	if strings.Contains(packageDir, "keadm/") {
		return "edge" // keadm is part of edge component in KubeEdge
	}
	if strings.Contains(packageDir, "pkg/") {
		return "" // Let make test handle pkg automatically
	}
	return ""
}

// parseKubeEdgeCoverageOutput parses coverage output from KubeEdge's make test
func (ca *CoverageAnalyzer) parseKubeEdgeCoverageOutput(output string) (float64, error) {
	// Look for coverage percentage in output
	// KubeEdge uses standard Go coverage format: "coverage: XX.X% of statements"
	re := regexp.MustCompile(`coverage:\s+(\d+\.?\d*)%\s+of\s+statements`)
	matches := re.FindStringSubmatch(output)

	if len(matches) >= 2 {
		coverage, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			return 0.0, fmt.Errorf("failed to parse coverage percentage: %v", err)
		}
		return coverage, nil
	}

	// Try alternative patterns
	re2 := regexp.MustCompile(`total:\s+\(statements\)\s+(\d+\.?\d*)%`)
	matches2 := re2.FindStringSubmatch(output)
	if len(matches2) >= 2 {
		coverage, err := strconv.ParseFloat(matches2[1], 64)
		if err != nil {
			return 0.0, fmt.Errorf("failed to parse coverage percentage: %v", err)
		}
		return coverage, nil
	}

	// No coverage found, assume 0%
	return 0.0, nil
}

// fallbackGoCoverage runs basic go test coverage as fallback
func (ca *CoverageAnalyzer) fallbackGoCoverage(ctx context.Context, packageDir string) (float64, error) {
	cmd := exec.CommandContext(ctx, "go", "test", "-cover", packageDir)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// If tests fail to run, might be because no tests exist
		if strings.Contains(string(output), "no test files") {
			return 0.0, nil
		}
		return 0.0, fmt.Errorf("failed to run fallback coverage: %v, output: %s", err, string(output))
	}

	return ca.parseKubeEdgeCoverageOutput(string(output))
}

// parseCoverageFromFile parses coverage from existing coverage.out file
func (ca *CoverageAnalyzer) parseCoverageFromFile(filePath string) (float64, error) {
	file, err := os.Open(ca.coverageFile)
	if err != nil {
		return 0.0, fmt.Errorf("failed to open coverage file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var totalStatements, coveredStatements int

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "mode:") {
			continue
		}

		// Parse coverage line: file.go:startLine.startCol,endLine.endCol numStmt count
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			// Check if this line is for our file
			if strings.Contains(parts[0], filePath) {
				numStmt, err := strconv.Atoi(parts[1])
				if err != nil {
					continue
				}
				count, err := strconv.Atoi(parts[2])
				if err != nil {
					continue
				}

				totalStatements += numStmt
				if count > 0 {
					coveredStatements += numStmt
				}
			}
		}
	}

	if totalStatements == 0 {
		return 0.0, nil
	}

	coverage := (float64(coveredStatements) / float64(totalStatements)) * 100
	return coverage, nil
}

// shouldIncludeFunction determines if a function should be included for testing
func (ca *CoverageAnalyzer) shouldIncludeFunction(fn *ast.FuncDecl) bool {
	if fn.Name == nil {
		return false
	}

	funcName := fn.Name.Name

	// Skip main functions
	if funcName == "main" {
		return false
	}

	// Skip init functions
	if funcName == "init" {
		return false
	}

	// Skip test functions
	if strings.HasPrefix(funcName, "Test") || strings.HasPrefix(funcName, "Benchmark") || strings.HasPrefix(funcName, "Example") {
		return false
	}

	// Include exported functions
	if fn.Name.IsExported() {
		return true
	}

	// Include unexported functions that have significant logic
	if fn.Body != nil && len(fn.Body.List) > 1 {
		return true
	}

	return false
}

// extractFunctionInfo extracts detailed information about a function
func (ca *CoverageAnalyzer) extractFunctionInfo(fn *ast.FuncDecl, lines []string) FunctionInfo {
	startPos := ca.fileSet.Position(fn.Pos())
	endPos := ca.fileSet.Position(fn.End())

	// Extract function content
	var funcContent strings.Builder
	for i := startPos.Line - 1; i < endPos.Line && i < len(lines); i++ {
		funcContent.WriteString(lines[i])
		funcContent.WriteString("\n")
	}

	return FunctionInfo{
		Name:       fn.Name.Name,
		Content:    funcContent.String(),
		StartLine:  startPos.Line,
		EndLine:    endPos.Line,
		IsExported: fn.Name.IsExported(),
		HasTests:   false, // Will be set by caller
	}
}

// getTestFileName returns the corresponding test file name
func (ca *CoverageAnalyzer) getTestFileName(sourceFile string) string {
	dir := filepath.Dir(sourceFile)
	base := filepath.Base(sourceFile)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return filepath.Join(dir, name+"_test.go")
}

// getExistingTestFunctions reads existing test file and returns test function names
func (ca *CoverageAnalyzer) getExistingTestFunctions(testFile string) map[string]bool {
	tests := make(map[string]bool)

	if !fileExists(testFile) {
		return tests
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		return tests
	}

	// Parse test file to find existing test functions
	node, err := parser.ParseFile(ca.fileSet, testFile, content, parser.ParseComments)
	if err != nil {
		return tests
	}

	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok && fn.Name != nil {
			funcName := fn.Name.Name
			if strings.HasPrefix(funcName, "Test") {
				// Extract the function being tested
				testedFunc := strings.TrimPrefix(funcName, "Test")
				tests[testedFunc] = true
			}
		}
		return true
	})

	return tests
}

// hasExistingTests checks if a function already has test coverage
func (ca *CoverageAnalyzer) hasExistingTests(funcName string, existingTests map[string]bool) bool {
	// Check direct match
	if existingTests[funcName] {
		return true
	}

	// Check variations (TestFuncName, TestNewFuncName, etc.)
	for testFunc := range existingTests {
		if strings.Contains(testFunc, funcName) {
			return true
		}
	}

	return false
}

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}
