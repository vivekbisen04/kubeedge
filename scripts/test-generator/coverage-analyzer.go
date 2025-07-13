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
	Name        string
	Content     string
	StartLine   int
	EndLine     int
	IsExported  bool
	HasTests    bool
	Signature   string
	Parameters  []string
	ReturnTypes []string
	Complexity  int
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

// AnalyzeFile analyzes coverage for a specific file
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

// ExtractModifiedFunctions extracts functions from a modified file
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

	// Use the extractFunctionsFromContent method
	return ca.extractFunctionsFromContent(string(content), absPath)
}

// extractFunctionsFromContent extracts functions from Go source code content
func (ca *CoverageAnalyzer) extractFunctionsFromContent(content, filePath string) ([]FunctionInfo, error) {
	// Parse the Go source code
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go source: %v", err)
	}

	var functions []FunctionInfo
	lines := strings.Split(content, "\n")

	// Check if test file exists to determine which functions already have tests
	testFile := ca.getTestFileName(filePath)
	existingTests := ca.getExistingTestFunctions(testFile)

	// Walk the AST to find function declarations
	ast.Inspect(node, func(n ast.Node) bool {
		switch fn := n.(type) {
		case *ast.FuncDecl:
			if fn.Name != nil && ca.shouldIncludeFunction(fn) {
				funcInfo := ca.extractFunctionInfo(fn, lines, fset)
				funcInfo.HasTests = ca.hasExistingTests(fn.Name.Name, existingTests)
				funcInfo.Complexity = ca.calculateComplexity(fn)
				functions = append(functions, funcInfo)
			}
		}
		return true
	})

	return functions, nil
}

// shouldIncludeFunction determines if a function should be included for testing
func (ca *CoverageAnalyzer) shouldIncludeFunction(fn *ast.FuncDecl) bool {
	if fn.Name == nil {
		return false
	}

	funcName := fn.Name.Name

	// Skip init functions
	if funcName == "init" {
		return false
	}

	// Skip main function
	if funcName == "main" {
		return false
	}

	// Skip test functions
	if strings.HasPrefix(funcName, "Test") || 
	   strings.HasPrefix(funcName, "Benchmark") || 
	   strings.HasPrefix(funcName, "Example") {
		return false
	}

	// Skip functions with build tags or special comments that indicate they shouldn't be tested
	if fn.Doc != nil {
		for _, comment := range fn.Doc.List {
			if strings.Contains(comment.Text, "// +build") ||
			   strings.Contains(comment.Text, "//go:build") ||
			   strings.Contains(comment.Text, "// TODO") ||
			   strings.Contains(comment.Text, "// FIXME") ||
			   strings.Contains(comment.Text, "// Deprecated") {
				return false
			}
		}
	}

	return true
}

// extractFunctionInfo extracts detailed information about a function
func (ca *CoverageAnalyzer) extractFunctionInfo(fn *ast.FuncDecl, lines []string, fset *token.FileSet) FunctionInfo {
	funcName := fn.Name.Name
	
	// Determine if function is exported
	isExported := ast.IsExported(funcName)
	
	// Extract function content
	startPos := fset.Position(fn.Pos())
	endPos := fset.Position(fn.End())
	
	var content strings.Builder
	for i := startPos.Line - 1; i < endPos.Line && i < len(lines); i++ {
		content.WriteString(lines[i])
		if i < endPos.Line-1 {
			content.WriteString("\n")
		}
	}

	// Extract function signature for better context
	signature := ca.extractFunctionSignature(fn)

	// Extract parameters
	var parameters []string
	if fn.Type.Params != nil {
		for _, param := range fn.Type.Params.List {
			for _, name := range param.Names {
				parameters = append(parameters, name.Name)
			}
		}
	}

	// Extract return types
	var returnTypes []string
	if fn.Type.Results != nil {
		for _, result := range fn.Type.Results.List {
			returnTypes = append(returnTypes, ca.typeToString(result.Type))
		}
	}

	return FunctionInfo{
		Name:        funcName,
		IsExported:  isExported,
		Content:     content.String(),
		Signature:   signature,
		Parameters:  parameters,
		ReturnTypes: returnTypes,
		StartLine:   startPos.Line,
		EndLine:     endPos.Line,
		HasTests:    false, // Will be determined later if needed
	}
}

// extractFunctionSignature creates a readable function signature
func (ca *CoverageAnalyzer) extractFunctionSignature(fn *ast.FuncDecl) string {
	var sig strings.Builder
	
	sig.WriteString("func ")
	
	// Add receiver if it's a method
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		sig.WriteString("(")
		for i, recv := range fn.Recv.List {
			if i > 0 {
				sig.WriteString(", ")
			}
			if len(recv.Names) > 0 {
				sig.WriteString(recv.Names[0].Name + " ")
			}
			sig.WriteString(ca.typeToString(recv.Type))
		}
		sig.WriteString(") ")
	}
	
	sig.WriteString(fn.Name.Name)
	
	// Add parameters
	sig.WriteString("(")
	if fn.Type.Params != nil {
		for i, param := range fn.Type.Params.List {
			if i > 0 {
				sig.WriteString(", ")
			}
			
			for j, name := range param.Names {
				if j > 0 {
					sig.WriteString(", ")
				}
				sig.WriteString(name.Name)
			}
			
			if len(param.Names) > 0 {
				sig.WriteString(" ")
			}
			sig.WriteString(ca.typeToString(param.Type))
		}
	}
	sig.WriteString(")")
	
	// Add return types
	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		sig.WriteString(" ")
		if len(fn.Type.Results.List) > 1 {
			sig.WriteString("(")
		}
		
		for i, result := range fn.Type.Results.List {
			if i > 0 {
				sig.WriteString(", ")
			}
			sig.WriteString(ca.typeToString(result.Type))
		}
		
		if len(fn.Type.Results.List) > 1 {
			sig.WriteString(")")
		}
	}
	
	return sig.String()
}

// typeToString converts an AST type to a string representation
func (ca *CoverageAnalyzer) typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return ca.typeToString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + ca.typeToString(t.X)
	case *ast.ArrayType:
		return "[]" + ca.typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + ca.typeToString(t.Key) + "]" + ca.typeToString(t.Value)
	case *ast.ChanType:
		prefix := "chan"
		if t.Dir == ast.SEND {
			prefix = "chan<-"
		} else if t.Dir == ast.RECV {
			prefix = "<-chan"
		}
		return prefix + " " + ca.typeToString(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.FuncType:
		return "func(...)"
	default:
		return "unknown"
	}
}

// calculateComplexity calculates cyclomatic complexity of a function
func (ca *CoverageAnalyzer) calculateComplexity(fn *ast.FuncDecl) int {
	complexity := 1 // Base complexity

	ast.Inspect(fn, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, 
		     *ast.TypeSwitchStmt, *ast.SelectStmt:
			complexity++
		case *ast.CaseClause:
			complexity++
		}
		return true
	})

	return complexity
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
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, testFile, content, parser.ParseComments)
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