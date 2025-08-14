package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"e2e-generator-demo/pkg/types"
)

// SimpleAnalyzer provides basic component discovery and coverage analysis
type SimpleAnalyzer struct {
	kubeedgeRoot string
}

// NewSimpleAnalyzer creates a new simple analyzer
func NewSimpleAnalyzer(kubeedgeRoot string) *SimpleAnalyzer {
	return &SimpleAnalyzer{
		kubeedgeRoot: kubeedgeRoot,
	}
}

// DiscoverComponents finds major KubeEdge components
func (s *SimpleAnalyzer) DiscoverComponents() ([]types.ComponentInfo, error) {
	components := []types.ComponentInfo{
		{
			Name:        "cloudhub",
			Package:     "cloud/pkg/cloudhub",
			Path:        filepath.Join(s.kubeedgeRoot, "cloud/pkg/cloudhub"),
			Description: "WebSocket server for cloud-edge communication",
			Methods:     []string{"Start", "Stop", "HandleConnection", "RouteMessage"},
		},
		{
			Name:        "edgehub",
			Package:     "edge/pkg/edgehub",
			Path:        filepath.Join(s.kubeedgeRoot, "edge/pkg/edgehub"),
			Description: "WebSocket client for edge-cloud communication",
			Methods:     []string{"Start", "Stop", "Connect", "SendMessage"},
		},
		{
			Name:        "edgecontroller",
			Package:     "cloud/pkg/edgecontroller",
			Path:        filepath.Join(s.kubeedgeRoot, "cloud/pkg/edgecontroller"),
			Description: "Manages edge nodes and pod metadata synchronization",
			Methods:     []string{"SyncPods", "SyncConfigMaps", "SyncSecrets"},
		},
		{
			Name:        "metamanager",
			Package:     "edge/pkg/metamanager",
			Path:        filepath.Join(s.kubeedgeRoot, "edge/pkg/metamanager"),
			Description: "Manages metadata between edged and edgehub",
			Methods:     []string{"ProcessMessage", "QueryMetadata", "StoreMetadata"},
		},
	}

	// Verify components exist
	var existingComponents []types.ComponentInfo
	for _, comp := range components {
		if _, err := os.Stat(comp.Path); err == nil {
			existingComponents = append(existingComponents, comp)
		}
	}

	return existingComponents, nil
}

// AnalyzeCoverage identifies missing E2E test coverage
func (s *SimpleAnalyzer) AnalyzeCoverage() ([]types.CoverageGap, error) {
	// Scan existing E2E tests
	e2eTestsPath := filepath.Join(s.kubeedgeRoot, "tests", "e2e")
	existingTests, err := s.scanExistingTests(e2eTestsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan existing tests: %w", err)
	}

	// Define expected test coverage
	expectedTests := map[string][]string{
		"cloudhub": {
			"WebSocket connection establishment",
			"Message routing between cloud and edge",
			"Connection failure handling",
			"Session management",
		},
		"edgehub": {
			"Cloud connection establishment",
			"Message sending to cloud",
			"Reconnection on failure",
			"Certificate validation",
		},
		"edgecontroller": {
			"Pod synchronization to edge",
			"ConfigMap synchronization",
			"Secret synchronization",
			"Node status updates",
		},
		"metamanager": {
			"Metadata storage and retrieval",
			"Message processing",
			"SQLite database operations",
		},
	}

	// Identify gaps
	var gaps []types.CoverageGap
	for component, tests := range expectedTests {
		for _, test := range tests {
			if !s.hasTest(existingTests, component, test) {
				priority := "high"
				if component == "metamanager" || component == "edgecontroller" {
					priority = "medium"
				}
				
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

// scanExistingTests scans the E2E test directory for existing tests
func (s *SimpleAnalyzer) scanExistingTests(testPath string) ([]string, error) {
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

// hasTest checks if a test exists for the given component and scenario
func (s *SimpleAnalyzer) hasTest(existingTests []string, component, scenario string) bool {
	for _, test := range existingTests {
		if strings.Contains(strings.ToLower(test), strings.ToLower(component)) {
			return true
		}
	}
	return false
}

// GetHighPriorityGaps returns gaps that should be addressed first
func (s *SimpleAnalyzer) GetHighPriorityGaps(gaps []types.CoverageGap) []types.CoverageGap {
	var highPriority []types.CoverageGap
	for _, gap := range gaps {
		if gap.Priority == "high" {
			highPriority = append(highPriority, gap)
		}
	}
	return highPriority
}