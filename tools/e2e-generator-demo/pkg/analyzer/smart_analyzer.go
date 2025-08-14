package analyzer

import (
	"fmt"
	"go/ast"
	"go/parser" 
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"e2e-generator-demo/pkg/types"
)

// SmartAnalyzer uses AST parsing and heuristics for intelligent analysis
type SmartAnalyzer struct {
	kubeedgeRoot string
	fset         *token.FileSet
}

// NewSmartAnalyzer creates an intelligent analyzer
func NewSmartAnalyzer(kubeedgeRoot string) *SmartAnalyzer {
	return &SmartAnalyzer{
		kubeedgeRoot: kubeedgeRoot,
		fset:         token.NewFileSet(),
	}
}

// DiscoverComponentsDynamically scans the codebase to find components
func (s *SmartAnalyzer) DiscoverComponentsDynamically() ([]types.ComponentInfo, error) {
	var components []types.ComponentInfo
	
	// Scan cloud and edge package directories
	scanDirs := []string{
		filepath.Join(s.kubeedgeRoot, "cloud/pkg"),
		filepath.Join(s.kubeedgeRoot, "edge/pkg"),
		filepath.Join(s.kubeedgeRoot, "pkg"),
	}
	
	for _, dir := range scanDirs {
		comps, err := s.scanDirectory(dir)
		if err != nil {
			continue // Skip directories that don't exist
		}
		components = append(components, comps...)
	}
	
	return components, nil
}

// scanDirectory recursively scans for Go packages that look like components
func (s *SmartAnalyzer) scanDirectory(dir string) ([]types.ComponentInfo, error) {
	var components []types.ComponentInfo
	
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		
		if !info.IsDir() || strings.Contains(path, "test") {
			return nil
		}
		
		// Check if this looks like a component directory
		if s.looksLikeComponent(path) {
			comp, err := s.analyzeComponent(path)
			if err == nil {
				components = append(components, comp)
			}
		}
		
		return nil
	})
	
	return components, err
}

// looksLikeComponent uses heuristics to identify component directories
func (s *SmartAnalyzer) looksLikeComponent(path string) bool {
	name := filepath.Base(path)
	
	// Heuristics for component identification
	componentPatterns := []string{
		"hub", "controller", "manager", "agent", "proxy", 
		"server", "client", "daemon", "service",
	}
	
	for _, pattern := range componentPatterns {
		if strings.Contains(strings.ToLower(name), pattern) {
			return true
		}
	}
	
	// Check if directory has main.go or manager.go or similar
	importantFiles := []string{"main.go", "manager.go", "server.go", "client.go"}
	for _, file := range importantFiles {
		if _, err := os.Stat(filepath.Join(path, file)); err == nil {
			return true
		}
	}
	
	return false
}

// analyzeComponent uses AST parsing to extract component details
func (s *SmartAnalyzer) analyzeComponent(componentPath string) (types.ComponentInfo, error) {
	name := filepath.Base(componentPath)
	relPath, _ := filepath.Rel(s.kubeedgeRoot, componentPath)
	
	component := types.ComponentInfo{
		Name:    name,
		Package: relPath,
		Path:    componentPath,
	}
	
	// Parse Go files to extract methods and description
	methods, description := s.extractComponentDetails(componentPath)
	component.Methods = methods
	component.Description = description
	
	return component, nil
}

// extractComponentDetails parses Go files to find methods and generate description
func (s *SmartAnalyzer) extractComponentDetails(componentPath string) ([]string, string) {
	var methods []string
	var description string
	
	err := filepath.Walk(componentPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(path, ".go") || strings.Contains(path, "_test.go") {
			return nil
		}
		
		// Parse the Go file
		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		
		file, err := parser.ParseFile(s.fset, path, src, parser.ParseComments)
		if err != nil {
			return nil
		}
		
		// Extract methods and comments
		fileMethods, fileDesc := s.parseASTForDetails(file)
		methods = append(methods, fileMethods...)
		if description == "" && fileDesc != "" {
			description = fileDesc
		}
		
		return nil
	})
	
	if err != nil {
		return methods, description
	}
	
	// Remove duplicates
	methods = s.removeDuplicates(methods)
	
	// Generate description if not found
	if description == "" {
		description = s.generateDescription(filepath.Base(componentPath), methods)
	}
	
	return methods, description
}

// parseASTForDetails extracts methods and comments from AST
func (s *SmartAnalyzer) parseASTForDetails(file *ast.File) ([]string, string) {
	var methods []string
	var description string
	
	// Extract package comment as description
	if file.Doc != nil && len(file.Doc.List) > 0 {
		description = strings.TrimPrefix(file.Doc.List[0].Text, "// ")
		description = strings.TrimPrefix(description, "Package ")
	}
	
	// Extract public methods
	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			// Only include public methods (start with capital letter)
			if funcDecl.Name.IsExported() {
				methods = append(methods, funcDecl.Name.Name)
			}
		}
	}
	
	return methods, description
}

// generateDescription creates description based on component name and methods
func (s *SmartAnalyzer) generateDescription(name string, methods []string) string {
	name = strings.ToLower(name)
	
	// Smart description based on name patterns
	if strings.Contains(name, "hub") {
		if strings.Contains(name, "cloud") {
			return "WebSocket server for cloud-edge communication"
		}
		if strings.Contains(name, "edge") {
			return "WebSocket client for edge-cloud communication" 
		}
		return "Communication hub for message routing"
	}
	
	if strings.Contains(name, "controller") {
		return "Controller for managing resources and synchronization"
	}
	
	if strings.Contains(name, "manager") {
		return "Manager for handling component lifecycle and operations"
	}
	
	// Fallback based on common methods
	if s.containsAny(methods, []string{"Start", "Stop", "Run"}) {
		return fmt.Sprintf("Service component for %s operations", name)
	}
	
	if s.containsAny(methods, []string{"Sync", "Update", "Watch"}) {
		return fmt.Sprintf("Synchronization component for %s", name)
	}
	
	return fmt.Sprintf("Component providing %s functionality", name)
}

// Dynamic Test Scenario Generation
func (s *SmartAnalyzer) GenerateExpectedTestsDynamically(components []types.ComponentInfo) map[string][]string {
	expectedTests := make(map[string][]string)
	
	for _, comp := range components {
		tests := s.generateTestScenariosForComponent(comp)
		expectedTests[comp.Name] = tests
	}
	
	return expectedTests
}

// generateTestScenariosForComponent creates test scenarios based on component analysis
func (s *SmartAnalyzer) generateTestScenariosForComponent(comp types.ComponentInfo) []string {
	var scenarios []string
	name := strings.ToLower(comp.Name)
	
	// Base scenarios for all components
	scenarios = append(scenarios, "Basic initialization and startup")
	scenarios = append(scenarios, "Graceful shutdown and cleanup")
	scenarios = append(scenarios, "Error handling and recovery")
	
	// Component-specific scenarios based on name and methods
	if strings.Contains(name, "hub") {
		scenarios = append(scenarios, "Connection establishment")
		scenarios = append(scenarios, "Message routing and delivery")
		scenarios = append(scenarios, "Connection failure handling")
		scenarios = append(scenarios, "Session management")
	}
	
	if strings.Contains(name, "controller") {
		scenarios = append(scenarios, "Resource synchronization")
		scenarios = append(scenarios, "Status updates and monitoring")
		scenarios = append(scenarios, "Event handling")
	}
	
	if strings.Contains(name, "manager") {
		scenarios = append(scenarios, "Data storage and retrieval")
		scenarios = append(scenarios, "Cache management")
		scenarios = append(scenarios, "Metadata operations")
	}
	
	// Method-based scenarios
	if s.containsAny(comp.Methods, []string{"Connect", "Dial"}) {
		scenarios = append(scenarios, "Connection establishment")
	}
	
	if s.containsAny(comp.Methods, []string{"Send", "Publish", "Write"}) {
		scenarios = append(scenarios, "Message sending and delivery")
	}
	
	if s.containsAny(comp.Methods, []string{"Receive", "Subscribe", "Read"}) {
		scenarios = append(scenarios, "Message receiving and processing")
	}
	
	if s.containsAny(comp.Methods, []string{"Sync", "Update"}) {
		scenarios = append(scenarios, "Data synchronization")
	}
	
	return s.removeDuplicates(scenarios)
}

// Helper functions
func (s *SmartAnalyzer) containsAny(slice []string, items []string) bool {
	for _, item := range slice {
		for _, target := range items {
			if strings.Contains(strings.ToLower(item), strings.ToLower(target)) {
				return true
			}
		}
	}
	return false
}

func (s *SmartAnalyzer) removeDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	var result []string
	
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	
	return result
}

// AnalyzeCoverageDynamically performs intelligent coverage analysis
func (s *SmartAnalyzer) AnalyzeCoverageDynamically() ([]types.CoverageGap, error) {
	// Discover components dynamically
	components, err := s.DiscoverComponentsDynamically()
	if err != nil {
		return nil, err
	}
	
	// Generate expected tests dynamically
	expectedTests := s.GenerateExpectedTestsDynamically(components)
	
	// Scan existing tests
	e2eTestsPath := filepath.Join(s.kubeedgeRoot, "tests", "e2e")
	existingTests, err := s.scanExistingTests(e2eTestsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan existing tests: %w", err)
	}
	
	// Identify gaps
	var gaps []types.CoverageGap
	for component, tests := range expectedTests {
		for _, test := range tests {
			if !s.hasTest(existingTests, component, test) {
				priority := s.calculatePriority(component, test)
				
				gaps = append(gaps, types.CoverageGap{
					Component:   component,
					MissingTest: test,
					Description: fmt.Sprintf("E2E test for %s %s", component, test),
					Priority:    priority,
				})
			}
		}
	}
	
	return gaps, nil
}

// calculatePriority dynamically determines priority based on component and test type
func (s *SmartAnalyzer) calculatePriority(component, test string) string {
	component = strings.ToLower(component)
	test = strings.ToLower(test)
	
	// Critical components get high priority
	if strings.Contains(component, "hub") {
		return "high"
	}
	
	// Critical test types get high priority
	if strings.Contains(test, "connection") || strings.Contains(test, "startup") || strings.Contains(test, "error") {
		return "high"
	}
	
	// Controller and manager operations are medium priority
	if strings.Contains(component, "controller") || strings.Contains(component, "manager") {
		return "medium"
	}
	
	return "low"
}

// Reuse existing helper methods from simple analyzer
func (s *SmartAnalyzer) scanExistingTests(testPath string) ([]string, error) {
	var tests []string
	
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		return tests, nil
	}
	
	err := filepath.Walk(testPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, ".go") {
			tests = append(tests, strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
		}
		
		return nil
	})
	
	return tests, err
}

func (s *SmartAnalyzer) hasTest(existingTests []string, component, scenario string) bool {
	for _, test := range existingTests {
		if strings.Contains(strings.ToLower(test), strings.ToLower(component)) {
			return true
		}
	}
	return false
}