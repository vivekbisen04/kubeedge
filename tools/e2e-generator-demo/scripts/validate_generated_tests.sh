#!/bin/bash

# Validation script for generated E2E tests
set -e

KUBEEDGE_ROOT=${1:-"../.."}
GENERATED_DIR="${KUBEEDGE_ROOT}/tests/e2e/generated"

echo "🔍 Validating Generated E2E Tests"
echo "=================================="

if [ ! -d "$GENERATED_DIR" ]; then
    echo "❌ Generated tests directory not found: $GENERATED_DIR"
    exit 1
fi

echo "📁 Found generated tests directory: $GENERATED_DIR"

# Check for generated test files
TEST_FILES=$(find "$GENERATED_DIR" -name "*_test.go" 2>/dev/null || echo "")

if [ -z "$TEST_FILES" ]; then
    echo "❌ No generated test files found in $GENERATED_DIR"
    exit 1
fi

echo "📄 Found test files:"
for file in $TEST_FILES; do
    echo "   - $(basename $file)"
done

echo ""
echo "🔧 Validating Go syntax..."

# Validate each test file
for file in $TEST_FILES; do
    echo "Checking: $(basename $file)"
    
    # Check Go syntax
    if ! go fmt "$file" > /dev/null 2>&1; then
        echo "❌ Syntax error in: $(basename $file)"
        go fmt "$file"
        continue
    fi
    
    # Check for required imports
    if ! grep -q "github.com/onsi/ginkgo" "$file"; then
        echo "⚠️  Missing Ginkgo import in: $(basename $file)"
    fi
    
    if ! grep -q "github.com/onsi/gomega" "$file"; then
        echo "⚠️  Missing Gomega import in: $(basename $file)"
    fi
    
    # Check for KubeEdge test patterns
    if grep -q "GroupDescribe" "$file"; then
        echo "✅ Found GroupDescribe pattern in: $(basename $file)"
    else
        echo "⚠️  Missing GroupDescribe pattern in: $(basename $file)"
    fi
    
    if grep -q "E2E_.*:" "$file"; then
        echo "✅ Found E2E test naming convention in: $(basename $file)"
    else
        echo "⚠️  Missing E2E naming convention in: $(basename $file)"
    fi
    
    echo "✅ Syntax valid: $(basename $file)"
    echo ""
done

echo "🧪 Testing compilation with KubeEdge dependencies..."

# Try to compile in the context of KubeEdge E2E tests
cd "${KUBEEDGE_ROOT}/tests/e2e"

for file in $(find generated -name "*_test.go" 2>/dev/null || echo ""); do
    echo "Compiling: $file"
    if go build "$file" 2>/dev/null; then
        echo "✅ Compilation successful: $file"
        rm -f generated/$(basename $file .go)  # Remove compiled binary
    else
        echo "❌ Compilation failed: $file"
        echo "Error details:"
        go build "$file"
        echo ""
    fi
done

echo ""
echo "🎯 Validation Summary:"
echo "- Generated test files: $(echo $TEST_FILES | wc -w)"
echo "- Location: $GENERATED_DIR"
echo "- Next step: Run 'make e2e' to test in full KubeEdge environment"
echo ""
echo "📚 To run specific generated tests:"
echo "   cd $KUBEEDGE_ROOT"
echo "   ginkgo tests/e2e/generated/"