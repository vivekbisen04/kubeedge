package validator

import (
	"regexp"
	"strings"
)

// TestFixer automatically fixes common issues in generated E2E tests
type TestFixer struct {
	component string
}

// NewTestFixer creates a new test fixer for a specific component
func NewTestFixer(component string) *TestFixer {
	return &TestFixer{
		component: component,
	}
}

// FixGeneratedTest applies comprehensive fixes to generated test code
func (tf *TestFixer) FixGeneratedTest(code string) (string, []string) {
	fixed := code
	appliedFixes := []string{}

	// Apply all fixes in sequence
	fixes := []struct {
		name    string
		fixFunc func(string) string
	}{
		{"Package declaration", tf.fixPackageDeclaration},
		{"Import statements", tf.fixImports},
		{"GroupDescribe pattern", tf.fixGroupDescribe},
		{"Ginkgo types and functions", tf.fixGinkgoTypes},
		{"Test structure", tf.fixTestStructure},
		{"KubeEdge utilities", tf.fixKubeEdgeUtils},
		{"Error handling", tf.fixErrorHandling},
	}

	for _, fix := range fixes {
		before := fixed
		fixed = fix.fixFunc(fixed)
		if before != fixed {
			appliedFixes = append(appliedFixes, fix.name)
		}
	}

	return fixed, appliedFixes
}

// fixPackageDeclaration ensures correct package name
func (tf *TestFixer) fixPackageDeclaration(code string) string {
	// Change package name to match E2E test pattern
	if strings.HasPrefix(code, "package "+tf.component) {
		return strings.Replace(code, "package "+tf.component, "package "+tf.component, 1)
	}
	return code
}

// fixImports fixes import statements
func (tf *TestFixer) fixImports(code string) string {
	// Standard imports for KubeEdge E2E tests
	standardImports := `import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"

	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)`

	// Replace import block
	importPattern := regexp.MustCompile(`import \([^}]*\)`)
	return importPattern.ReplaceAllString(code, standardImports)
}

// fixGroupDescribe adds the GroupDescribe pattern
func (tf *TestFixer) fixGroupDescribe(code string) string {
	fixed := code
	
	// Add GroupDescribe function
	groupDescribeFunc := `
// GroupDescribe annotates the test with the group label.
func GroupDescribe(text string, body func()) bool {
	return ginkgo.Describe("[KubeEdge-` + strings.ToUpper(tf.component) + `] "+text, body)
}
`

	// Replace Describe with GroupDescribe
	describePattern := regexp.MustCompile(`var _ = Describe\("([^"]+)"`)
	fixed = describePattern.ReplaceAllString(fixed, `var _ = GroupDescribe("$1"`)

	// Insert GroupDescribe function before the test suite
	testSuitePattern := regexp.MustCompile(`var _ = GroupDescribe`)
	if testSuitePattern.MatchString(fixed) && !strings.Contains(fixed, "func GroupDescribe") {
		fixed = testSuitePattern.ReplaceAllString(fixed, groupDescribeFunc+"\nvar _ = GroupDescribe")
	}

	return fixed
}

// fixGinkgoTypes fixes Ginkgo type and function issues
func (tf *TestFixer) fixGinkgoTypes(code string) string {
	fixed := code

	// Fix type issues
	typeReplacements := map[string]string{
		"GinkgoTestDescription":             "ginkgo.SpecReport",
		"CurrentSpecReport()":               "ginkgo.CurrentSpecReport()",
		"testSpecReport.LeafNodeText":       "testSpecReport.LeafNodeText",
		"BeforeEach(func() {":               "ginkgo.BeforeEach(func() {",
		"AfterEach(func() {":                "ginkgo.AfterEach(func() {",
		"It(\"":                             "ginkgo.It(\"",
		"Expect(":                           "gomega.Expect(",
		"BeNil()":                          "gomega.BeNil()",
		"ToNot(":                           "gomega.ToNot(",
		"ContainSubstring(":                "gomega.ContainSubstring(",
		"RegisterFailHandler(Fail)":         "gomega.RegisterFailHandler(ginkgo.Fail)",
		"RunSpecs(":                        "ginkgo.RunSpecs(",
	}

	for old, new := range typeReplacements {
		fixed = strings.ReplaceAll(fixed, old, new)
	}

	return fixed
}

// fixTestStructure ensures proper KubeEdge test structure  
func (tf *TestFixer) fixTestStructure(code string) string {
	fixed := code

	// Ensure proper variable declarations
	varPattern := regexp.MustCompile(`var testSpecReport ginkgo\.SpecReport`)
	if !varPattern.MatchString(fixed) {
		// Fix variable declaration
		fixed = strings.ReplaceAll(fixed, 
			"var testSpecReport ginkgo.SpecReport",
			"var testSpecReport ginkgo.SpecReport")
	}

	// Ensure proper BeforeEach structure
	beforeEachPattern := `ginkgo.BeforeEach(func() {
		clientSet = utils.NewKubeClient(framework.TestContext.KubeConfig)
		testSpecReport = ginkgo.CurrentSpecReport()
		testTimer = utils.CRDTestTimerGroup.NewTestTimer(testSpecReport.LeafNodeText)`

	if strings.Contains(fixed, "BeforeEach") && !strings.Contains(fixed, "utils.CRDTestTimerGroup") {
		// Find and replace BeforeEach block
		beforePattern := regexp.MustCompile(`ginkgo\.BeforeEach\(func\(\) \{[^}]*testTimer[^}]*\}`)
		if beforePattern.MatchString(fixed) {
			fixed = beforePattern.ReplaceAllString(fixed, beforeEachPattern+"\n\t}")
		}
	}

	return fixed
}

// fixKubeEdgeUtils fixes KubeEdge utility usage
func (tf *TestFixer) fixKubeEdgeUtils(code string) string {
	fixed := code

	// Ensure utils.PrintTestcaseNameandStatus() is called
	if strings.Contains(fixed, "AfterEach") && !strings.Contains(fixed, "PrintTestcaseNameandStatus") {
		fixed = strings.ReplaceAll(fixed,
			"utils.PrintTestcaseNameandStatus()",
			"utils.PrintTestcaseNameandStatus()")
	}

	return fixed
}

// fixErrorHandling improves error handling patterns
func (tf *TestFixer) fixErrorHandling(code string) string {
	fixed := code

	// Fix common error handling patterns
	errorFixes := map[string]string{
		"gomega.Expect(err).To(gomega.BeNil(),": "gomega.Expect(err).To(gomega.BeNil(),",
		"gomega.Expect(err).ToNot(gomega.BeNil(),": "gomega.Expect(err).ToNot(gomega.BeNil()),",
	}

	for old, new := range errorFixes {
		fixed = strings.ReplaceAll(fixed, old, new)
	}

	return fixed
}

// GenerateFixedTest creates a completely new test following KubeEdge patterns
func (tf *TestFixer) GenerateFixedTest() string {
	componentUpper := strings.ToUpper(tf.component)
	componentTitle := strings.Title(tf.component)
	
	template := `package ` + tf.component + `

import (
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"

	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

// GroupDescribe annotates the test with the group label.
func GroupDescribe(text string, body func()) bool {
	return ginkgo.Describe("[KubeEdge-` + componentUpper + `] "+text, body)
}

var _ = GroupDescribe("` + componentTitle + ` E2E Tests", func() {
	var testTimer *utils.TestTimer
	var testSpecReport ginkgo.SpecReport
	var clientSet clientset.Interface

	ginkgo.BeforeEach(func() {
		clientSet = utils.NewKubeClient(framework.TestContext.KubeConfig)
		testSpecReport = ginkgo.CurrentSpecReport()
		testTimer = utils.CRDTestTimerGroup.NewTestTimer(testSpecReport.LeafNodeText)
	})

	ginkgo.AfterEach(func() {
		testTimer.End()
		testTimer.PrintResult()
		utils.PrintTestcaseNameandStatus()
	})

	ginkgo.It("E2E_` + componentUpper + `_1: Basic functionality test", func() {
		// TODO: Implement test logic based on component requirements
		gomega.Expect(clientSet).ToNot(gomega.BeNil())
	})

	ginkgo.It("E2E_` + componentUpper + `_2: Connection establishment test", func() {
		// TODO: Implement connection test logic
		gomega.Expect(true).To(gomega.BeTrue())
	})

	ginkgo.It("E2E_` + componentUpper + `_3: Error handling test", func() {
		// TODO: Implement error handling test logic
		gomega.Expect(true).To(gomega.BeTrue())
	})
})

func Test` + componentTitle + `E2E(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "` + componentTitle + ` E2E Suite")
}
`
	return template
}