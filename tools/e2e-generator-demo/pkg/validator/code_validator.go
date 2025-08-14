package validator

import (
	"fmt"
	"go/parser"
	"go/token"
	"os/exec"
	"regexp"
	"strings"
)

// CodeValidator validates and fixes generated Go code
type CodeValidator struct {
	fset *token.FileSet
}

// ValidationResult contains validation results and fixes
type ValidationResult struct {
	IsValid       bool     `json:"is_valid"`
	Errors        []string `json:"errors"`
	Warnings      []string `json:"warnings"`
	FixedCode     string   `json:"fixed_code"`
	RequiredFixes []string `json:"required_fixes"`
}

// NewCodeValidator creates a new code validator
func NewCodeValidator() *CodeValidator {
	return &CodeValidator{
		fset: token.NewFileSet(),
	}
}

// ValidateGeneratedTest performs comprehensive validation on generated test code
func (cv *CodeValidator) ValidateGeneratedTest(code string, component string) (*ValidationResult, error) {
	result := &ValidationResult{
		IsValid:   true,
		Errors:    []string{},
		Warnings:  []string{},
		FixedCode: code,
	}

	// 1. Parse Go syntax
	if err := cv.validateGoSyntax(code, result); err != nil {
		return result, err
	}

	// 2. Check KubeEdge E2E patterns
	cv.validateKubeEdgePatterns(code, result)

	// 3. Fix common issues
	fixedCode := cv.fixCommonIssues(code, component)
	result.FixedCode = fixedCode

	// 4. Validate imports
	cv.validateImports(fixedCode, result)

	// 5. Check Ginkgo/Gomega usage
	cv.validateGinkgoUsage(fixedCode, result)

	// 6. Try compilation
	if err := cv.validateCompilation(fixedCode, result); err != nil {
		result.IsValid = false
	}

	return result, nil
}

// validateGoSyntax checks basic Go syntax
func (cv *CodeValidator) validateGoSyntax(code string, result *ValidationResult) error {
	_, err := parser.ParseFile(cv.fset, "test.go", code, parser.ParseComments)
	if err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Go syntax error: %v", err))
		return err
	}
	return nil
}

// validateKubeEdgePatterns checks for KubeEdge E2E test patterns
func (cv *CodeValidator) validateKubeEdgePatterns(code string, result *ValidationResult) {
	// Check for GroupDescribe (KubeEdge pattern)
	if !strings.Contains(code, "GroupDescribe") && strings.Contains(code, "Describe") {
		result.Warnings = append(result.Warnings, "Should use GroupDescribe instead of Describe for KubeEdge E2E tests")
		result.RequiredFixes = append(result.RequiredFixes, "Replace Describe with GroupDescribe")
	}

	// Check for E2E naming convention
	e2ePattern := regexp.MustCompile(`E2E_[A-Z]+_\d+`)
	if !e2ePattern.MatchString(code) {
		result.Warnings = append(result.Warnings, "Missing E2E naming convention (E2E_COMPONENT_N)")
	}

	// Check for test timer usage
	if !strings.Contains(code, "testTimer") {
		result.Warnings = append(result.Warnings, "Missing test timer for performance tracking")
	}

	// Check for proper cleanup
	if !strings.Contains(code, "AfterEach") {
		result.Warnings = append(result.Warnings, "Missing AfterEach for cleanup")
	}
}

// fixCommonIssues automatically fixes common problems in generated code
func (cv *CodeValidator) fixCommonIssues(code string, component string) string {
	fixed := code

	// Fix 1: Replace Describe with GroupDescribe for KubeEdge
	if strings.Contains(fixed, `Describe("`) && !strings.Contains(fixed, "GroupDescribe") {
		fixed = strings.Replace(fixed, 
			`var _ = Describe("`+strings.Title(component)+` E2E Tests"`, 
			`var _ = GroupDescribe("`+strings.Title(component)+` E2E Tests"`, 1)
	}

	// Fix 2: Fix GinkgoTestDescription -> SpecReport
	fixed = strings.Replace(fixed, "GinkgoTestDescription", "ginkgo.SpecReport", -1)
	fixed = strings.Replace(fixed, "CurrentSpecReport()", "ginkgo.CurrentSpecReport()", -1)

	// Fix 3: Fix missing imports for GroupDescribe
	if strings.Contains(fixed, "GroupDescribe") && !strings.Contains(fixed, "tests/e2e/"+component) {
		// Add missing import after other imports
		importBlock := `import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"

	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)`

		// Replace the import block
		importPattern := regexp.MustCompile(`import \([^)]+\)`)
		fixed = importPattern.ReplaceAllString(fixed, importBlock)
	}

	// Fix 4: Add GroupDescribe function if missing
	if strings.Contains(fixed, "GroupDescribe") && !strings.Contains(fixed, "func GroupDescribe") {
		// Add the GroupDescribe function
		groupDescribeFunc := `
// GroupDescribe annotates the test with the group label.
func GroupDescribe(text string, body func()) bool {
	return ginkgo.Describe("[KubeEdge-` + strings.ToUpper(component) + `] "+text, body)
}
`
		// Insert before the test suite
		testSuitePattern := regexp.MustCompile(`var _ = GroupDescribe`)
		if testSuitePattern.MatchString(fixed) {
			fixed = testSuitePattern.ReplaceAllString(fixed, groupDescribeFunc+"\nvar _ = GroupDescribe")
		}
	}

	// Fix 5: Fix BeNil() -> BeNil()
	fixed = strings.Replace(fixed, "To(BeNil(),", "To(BeNil(),", -1)
	fixed = strings.Replace(fixed, "ToNot(BeNil(),", "ToNot(BeNil()),", -1)

	// Fix 6: Fix missing context import usage
	if strings.Contains(fixed, "context.") && !strings.Contains(fixed, `"context"`) {
		fixed = strings.Replace(fixed, 
			`"fmt"`, 
			`"context"`+"\n\t"+`"fmt"`, 1)
	}

	return fixed
}

// validateImports checks for required imports
func (cv *CodeValidator) validateImports(code string, result *ValidationResult) {
	requiredImports := map[string]string{
		"ginkgo":    "github.com/onsi/ginkgo/v2",
		"gomega":    "github.com/onsi/gomega", 
		"utils":     "github.com/kubeedge/kubeedge/tests/e2e/utils",
		"framework": "k8s.io/kubernetes/test/e2e/framework",
	}

	for name, importPath := range requiredImports {
		if !strings.Contains(code, importPath) {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Missing required import: %s (%s)", name, importPath))
		}
	}
}

// validateGinkgoUsage checks proper Ginkgo/Gomega patterns
func (cv *CodeValidator) validateGinkgoUsage(code string, result *ValidationResult) {
	// Check for proper It() usage
	if !strings.Contains(code, `It("E2E_`) {
		result.Warnings = append(result.Warnings, "Tests should use It() with E2E naming convention")
	}

	// Check for BeforeEach/AfterEach
	if strings.Contains(code, "BeforeEach") && !strings.Contains(code, "AfterEach") {
		result.Warnings = append(result.Warnings, "BeforeEach found but missing corresponding AfterEach")
	}

	// Check for proper expectations
	expectationPatterns := []string{"Expect(", "Eventually(", "Consistently("}
	hasExpectations := false
	for _, pattern := range expectationPatterns {
		if strings.Contains(code, pattern) {
			hasExpectations = true
			break
		}
	}
	if !hasExpectations {
		result.Warnings = append(result.Warnings, "No Gomega expectations found in tests")
	}
}

// validateCompilation attempts to compile the code
func (cv *CodeValidator) validateCompilation(code string, result *ValidationResult) error {
	// Write code to temporary file and try to compile
	tmpFile := "/tmp/test_validation.go"
	if err := writeToFile(tmpFile, code); err != nil {
		return err
	}

	// Try to compile
	cmd := exec.Command("go", "build", "-o", "/dev/null", tmpFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Compilation error: %s", string(output)))
		return err
	}

	return nil
}

// writeToFile writes code to a file
func writeToFile(filename, content string) error {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("echo '%s' > %s", strings.Replace(content, "'", "'\"'\"'", -1), filename))
	return cmd.Run()
}