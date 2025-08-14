package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"e2e-generator-demo/pkg/analyzer"
	"e2e-generator-demo/pkg/generator"
	"e2e-generator-demo/pkg/types"
)

func main() {
	var (
		kubeedgeRoot = flag.String("kubeedge-root", ".", "Path to KubeEdge repository root")
		component    = flag.String("component", "", "Component to generate tests for (cloudhub, edgehub, etc.)")
		apiKey       = flag.String("api-key", "", "Google Gemini API key")
		output       = flag.String("output", "tests/e2e/generated", "Output directory for generated tests")
		dryRun       = flag.Bool("dry-run", false, "Print generated tests without writing files")
		analyze      = flag.Bool("analyze", false, "Only analyze coverage gaps without generating tests")
		fix          = flag.Bool("fix", false, "Fix existing generated test files")
		validate     = flag.Bool("validate", false, "Validate existing generated test files")
		testFile     = flag.String("file", "", "Specific test file to fix (for --fix command)")
		backup       = flag.Bool("backup", true, "Create backup of original files (for --fix command)")
	)
	flag.Parse()

	// Handle fix command
	if *fix {
		FixCommand(*kubeedgeRoot, *testFile, *component, *backup, *dryRun)
		return
	}

	// Handle validate command  
	if *validate {
		ValidateCommand(*kubeedgeRoot, *component)
		return
	}

	// Check API key
	if *apiKey == "" {
		*apiKey = os.Getenv("GEMINI_API_KEY")
		if *apiKey == "" && !*analyze {
			log.Fatal("❌ Gemini API key is required. Set via --api-key or GEMINI_API_KEY env var")
		}
	}

	fmt.Println("🚀 KubeEdge E2E Test Generator Demo")
	fmt.Println("=====================================")

	// Initialize analyzer
	simpleAnalyzer := analyzer.NewSimpleAnalyzer(*kubeedgeRoot)

	// Discover components
	fmt.Println("📋 Discovering KubeEdge components...")
	components, err := simpleAnalyzer.DiscoverComponents()
	if err != nil {
		log.Fatalf("❌ Failed to discover components: %v", err)
	}

	fmt.Printf("✅ Found %d components:\n", len(components))
	for _, comp := range components {
		fmt.Printf("   - %s (%s)\n", comp.Name, comp.Description)
	}

	// Analyze coverage
	fmt.Println("\n🔍 Analyzing E2E test coverage...")
	gaps, err := simpleAnalyzer.AnalyzeCoverage()
	if err != nil {
		log.Fatalf("❌ Failed to analyze coverage: %v", err)
	}

	fmt.Printf("✅ Found %d coverage gaps:\n", len(gaps))
	for _, gap := range gaps {
		fmt.Printf("   - %s: %s (%s priority)\n", gap.Component, gap.MissingTest, gap.Priority)
	}

	// If only analyzing, stop here
	if *analyze {
		fmt.Println("\n📊 Analysis complete!")
		return
	}

	// Filter gaps by component if specified
	var targetGaps []types.CoverageGap
	if *component != "" {
		for _, gap := range gaps {
			if gap.Component == *component {
				targetGaps = append(targetGaps, gap)
			}
		}
		if len(targetGaps) == 0 {
			log.Fatalf("❌ No coverage gaps found for component: %s", *component)
		}
	} else {
		// Use high priority gaps only
		targetGaps = simpleAnalyzer.GetHighPriorityGaps(gaps)
		if len(targetGaps) == 0 {
			fmt.Println("✅ No high priority coverage gaps found!")
			return
		}
	}

	// Group gaps by component
	componentGaps := make(map[string][]types.CoverageGap)
	for _, gap := range targetGaps {
		componentGaps[gap.Component] = append(componentGaps[gap.Component], gap)
	}

	// Initialize Gemini client
	geminiClient := generator.NewGeminiClient(*apiKey)

	// Generate tests for each component
	fmt.Printf("\n🤖 Generating E2E tests using Gemini 1.5 Flash...\n")
	for compName, compGaps := range componentGaps {
		fmt.Printf("\n📝 Generating tests for %s...\n", compName)

		// Find component info
		var compInfo types.ComponentInfo
		for _, comp := range components {
			if comp.Name == compName {
				compInfo = comp
				break
			}
		}

		// Build context for LLM
		context := &types.LLMContext{
			Component:        compInfo,
			ExistingPatterns: []string{"Ginkgo/Gomega", "KubeEdge E2E patterns"},
			CoverageGaps:     compGaps,
			TestFramework:    "Ginkgo/Gomega",
		}

		// Generate tests
		generatedTest, err := geminiClient.GenerateE2ETest(context)
		if err != nil {
			log.Printf("❌ Failed to generate tests for %s: %v", compName, err)
			continue
		}

		if *dryRun {
			fmt.Printf("\n📄 Generated test for %s:\n", compName)
			fmt.Println("=====================================")
			fmt.Println(generatedTest.Content)
			fmt.Println("=====================================")
		} else {
			// Write to file
			outputPath := filepath.Join(*output, generatedTest.FileName)
			if err := writeTestFile(generatedTest, outputPath); err != nil {
				log.Printf("❌ Failed to write test file for %s: %v", compName, err)
			} else {
				fmt.Printf("✅ Generated test written to: %s\n", outputPath)
			}
		}
	}

	fmt.Println("\n🎉 Test generation complete!")
}

// writeTestFile writes the generated test to a file
func writeTestFile(test *types.GeneratedTest, outputPath string) error {
	// Create output directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", outputPath, err)
	}
	defer file.Close()

	if _, err := file.WriteString(test.Content); err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	return nil
}