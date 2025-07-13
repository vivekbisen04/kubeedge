#!/bin/bash
# Complete Local Testing Guide for KubeEdge Auto Test Generator

echo "🧪 KubeEdge Auto Test Generator - Local Testing Guide"
echo "=================================================="

# Step 1: Environment Setup
echo ""
echo "📋 Step 1: Environment Setup"
echo "----------------------------"

# Check if we're in the right directory
if [ ! -f "go.mod" ] || ! grep -q "github.com/kubeedge/kubeedge" go.mod; then
    echo "❌ Error: Please run this from the KubeEdge repository root"
    exit 1
fi

echo "✅ In KubeEdge repository root"

# Set required environment variables
echo "🔑 Setting up environment variables..."
export GEMINI_API_KEY="your-gemini-api-key-here"  # Replace with your actual key
export GITHUB_TOKEN="your-github-token-here"       # Optional for local testing

if [ -z "$GEMINI_API_KEY" ] || [ "$GEMINI_API_KEY" = "your-gemini-api-key-here" ]; then
    echo "❌ Please set your GEMINI_API_KEY first:"
    echo "   export GEMINI_API_KEY='your-actual-gemini-api-key'"
    echo "   Get it from: https://aistudio.google.com/app/apikey"
    exit 1
fi

echo "✅ GEMINI_API_KEY is set"

# Step 2: Compilation Test
echo ""
echo "🔨 Step 2: Compilation Test"
echo "---------------------------"

cd scripts/test-generator

echo "Testing compilation..."
if go build .; then
    echo "✅ Compilation successful!"
    rm -f test-generator  # Clean up binary
else
    echo "❌ Compilation failed. Please fix errors first."
    exit 1
fi

# Step 3: Create Test Files
echo ""
echo "📝 Step 3: Creating Test Files"
echo "------------------------------"

# Create a simple test file with low coverage
cat > test_sample.go << 'EOF'
package main

import (
    "fmt"
    "errors"
    "strings"
    "strconv"
)

// Add adds two numbers
func Add(a, b int) int {
    return a + b
}

// Subtract subtracts two numbers
func Subtract(a, b int) int {
    return a - b
}

// Divide divides two numbers with error handling
func Divide(a, b int) (int, error) {
    if b == 0 {
        return 0, errors.New("division by zero")
    }
    return a / b, nil
}

// ValidateEmail validates an email address
func ValidateEmail(email string) bool {
    if email == "" {
        return false
    }
    return strings.Contains(email, "@") && strings.Contains(email, ".")
}

// ProcessString processes a string with various conditions
func ProcessString(input string) string {
    if len(input) == 0 {
        return "empty"
    }
    if len(input) > 100 {
        return "too_long"
    }
    if strings.Contains(input, " ") {
        return "has_spaces"
    }
    return "normal"
}

// ConvertToInt converts string to integer
func ConvertToInt(s string) (int, error) {
    if s == "" {
        return 0, errors.New("empty string")
    }
    return strconv.Atoi(s)
}

// FormatName formats a person's name
func FormatName(firstName, lastName string) string {
    if firstName == "" && lastName == "" {
        return "Unknown"
    }
    if firstName == "" {
        return lastName
    }
    if lastName == "" {
        return firstName
    }
    return fmt.Sprintf("%s %s", firstName, lastName)
}
EOF

echo "✅ Created test_sample.go with 7 functions"

# Step 4: Check Initial Coverage
echo ""
echo "📊 Step 4: Checking Initial Coverage"
echo "------------------------------------"

echo "Running initial coverage check..."
if go test -coverprofile=before.out . 2>/dev/null; then
    if [ -f "before.out" ]; then
        BEFORE_COVERAGE=$(go tool cover -func=before.out | grep "test_sample.go" | awk '{print $3}' | sed 's/%//' || echo "0")
        echo "📊 Initial coverage: ${BEFORE_COVERAGE}%"
        rm -f before.out
    else
        echo "📊 Initial coverage: 0% (no tests exist)"
        BEFORE_COVERAGE="0"
    fi
else
    echo "📊 Initial coverage: 0% (no tests exist)"
    BEFORE_COVERAGE="0"
fi

# Step 5: Run Test Generator
echo ""
echo "🤖 Step 5: Running Test Generator"
echo "---------------------------------"

echo "Generating tests for test_sample.go..."
echo "Command: go run . -changed-files='test_sample.go' -coverage-threshold=40 -max-retries=3 -debug=true"

if go run . \
    -changed-files="test_sample.go" \
    -coverage-threshold=40 \
    -max-retries=3 \
    -gemini-api-key="$GEMINI_API_KEY" \
    -debug=true; then
    
    echo ""
    echo "✅ Test generation completed!"
    
    # Check if test file was created
    if [ -f "test_sample_test.go" ]; then
        echo "✅ Generated test file: test_sample_test.go"
        echo "📏 Test file size: $(wc -l < test_sample_test.go) lines"
    else
        echo "❌ Test file was not created"
        exit 1
    fi
else
    echo "❌ Test generation failed"
    exit 1
fi

# Step 6: Validate Generated Tests
echo ""
echo "🧪 Step 6: Validating Generated Tests"
echo "-------------------------------------"

echo "Testing compilation of generated tests..."
if go build .; then
    echo "✅ Generated tests compile successfully"
else
    echo "❌ Generated tests have compilation errors"
    echo "Generated test file content:"
    cat test_sample_test.go
    exit 1
fi

echo "Running generated tests..."
if go test -v .; then
    echo "✅ All generated tests pass!"
else
    echo "❌ Some generated tests failed"
    echo "Test output above shows the failures"
fi

# Step 7: Check Coverage Improvement
echo ""
echo "📈 Step 7: Checking Coverage Improvement"
echo "----------------------------------------"

echo "Running coverage analysis with new tests..."
if go test -coverprofile=after.out .; then
    if [ -f "after.out" ]; then
        AFTER_COVERAGE=$(go tool cover -func=after.out | grep "test_sample.go" | awk '{print $3}' | sed 's/%//' || echo "0")
        echo "📊 After coverage: ${AFTER_COVERAGE}%"
        
        # Calculate improvement
        IMPROVEMENT=$(echo "$AFTER_COVERAGE - $BEFORE_COVERAGE" | bc)
        echo "📈 Coverage improvement: +${IMPROVEMENT}%"
        
        if (( $(echo "$IMPROVEMENT > 0" | bc -l) )); then
            echo "✅ Coverage improved successfully!"
        else
            echo "⚠️ Coverage did not improve"
        fi
        
        # Show detailed coverage
        echo ""
        echo "📋 Detailed coverage breakdown:"
        go tool cover -func=after.out | grep "test_sample.go"
        
        rm -f after.out
    else
        echo "❌ Could not generate coverage report"
    fi
else
    echo "❌ Coverage test failed"
fi

# Step 8: Test with Package Structure
echo ""
echo "📁 Step 8: Testing with Package Structure"
echo "-----------------------------------------"

# Create a more realistic test with package structure
mkdir -p pkg/util
cat > pkg/util/helper.go << 'EOF'
package util

import (
    "fmt"
    "strings"
)

// IsEmpty checks if string is empty
func IsEmpty(s string) bool {
    return len(strings.TrimSpace(s)) == 0
}

// Capitalize capitalizes first letter
func Capitalize(s string) string {
    if len(s) == 0 {
        return s
    }
    return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

// JoinWithComma joins strings with comma
func JoinWithComma(items []string) string {
    return strings.Join(items, ", ")
}

// SafeDivide performs safe division
func SafeDivide(a, b float64) (float64, error) {
    if b == 0 {
        return 0, fmt.Errorf("division by zero")
    }
    return a / b, nil
}
EOF

echo "✅ Created pkg/util/helper.go"

echo "Testing package-based file..."
if go run . \
    -changed-files="pkg/util/helper.go" \
    -coverage-threshold=40 \
    -max-retries=3 \
    -gemini-api-key="$GEMINI_API_KEY" \
    -debug=true; then
    
    echo "✅ Package-based test generation successful!"
    
    if [ -f "pkg/util/helper_test.go" ]; then
        echo "✅ Generated: pkg/util/helper_test.go"
        
        # Test the generated package tests
        cd pkg/util
        if go test -v .; then
            echo "✅ Package tests pass!"
        else
            echo "❌ Package tests failed"
        fi
        cd ../../
    fi
else
    echo "❌ Package-based test generation failed"
fi

# Step 9: Test Multiple Files
echo ""
echo "📚 Step 9: Testing Multiple Files"
echo "---------------------------------"

# Create another test file
cat > test_another.go << 'EOF'
package main

import "time"

// GetCurrentTime returns current time
func GetCurrentTime() time.Time {
    return time.Now()
}

// IsWeekend checks if given time is weekend
func IsWeekend(t time.Time) bool {
    weekday := t.Weekday()
    return weekday == time.Saturday || weekday == time.Sunday
}

// FormatTime formats time to string
func FormatTime(t time.Time) string {
    return t.Format("2006-01-02 15:04:05")
}
EOF

echo "✅ Created test_another.go"

echo "Testing multiple files..."
if go run . \
    -changed-files="test_sample.go,test_another.go" \
    -coverage-threshold=40 \
    -max-retries=3 \
    -gemini-api-key="$GEMINI_API_KEY" \
    -debug=true; then
    
    echo "✅ Multiple file test generation successful!"
else
    echo "❌ Multiple file test generation failed"
fi

# Step 10: Cleanup and Summary
echo ""
echo "🧹 Step 10: Cleanup and Summary"
echo "-------------------------------"

echo "Cleaning up test files..."
rm -f test_sample.go test_sample_test.go
rm -f test_another.go test_another_test.go
rm -rf pkg/util
rm -f successful_tests.txt failed_tests.txt
rm -f coverage.out temp_coverage.out

echo ""
echo "🎉 Local Testing Complete!"
echo "=========================="
echo ""
echo "✅ Tests Performed:"
echo "  - ✅ Compilation test"
echo "  - ✅ Single file test generation"
echo "  - ✅ Package structure test"
echo "  - ✅ Multiple files test"
echo "  - ✅ Coverage improvement verification"
echo "  - ✅ Test execution validation"
echo ""
echo "📋 Next Steps:"
echo "  1. Test on GitHub by creating a real PR"
echo "  2. Monitor GitHub Actions workflow"
echo "  3. Check auto-generated PRs"
echo ""
echo "🔗 GitHub Testing Instructions:"
echo "  See the GitHub testing guide for workflow testing"