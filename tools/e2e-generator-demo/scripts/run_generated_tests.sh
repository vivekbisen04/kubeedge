#!/bin/bash

# Script to run generated E2E tests in KubeEdge environment
set -e

KUBEEDGE_ROOT=${1:-"../.."}
COMPONENT=${2:-""}

echo "🚀 Running Generated E2E Tests"
echo "==============================="

cd "$KUBEEDGE_ROOT"

# Check if generated tests exist
if [ ! -d "tests/e2e/generated" ]; then
    echo "❌ No generated tests found. Run the generator first:"
    echo "   cd tools/e2e-generator-demo"
    echo "   ./e2e-generator --kubeedge-root ../.. --component cloudhub --output ../../tests/e2e/generated"
    exit 1
fi

echo "📋 Available generated tests:"
ls -la tests/e2e/generated/

# Install ginkgo if not available
if ! command -v ginkgo &> /dev/null; then
    echo "📦 Installing Ginkgo..."
    go install github.com/onsi/ginkgo/v2/ginkgo@latest
fi

echo ""
echo "🧪 Testing compilation of generated tests..."

cd tests/e2e

# Try to compile generated tests
if [ -n "$COMPONENT" ]; then
    TEST_FILE="generated/${COMPONENT}_generated_test.go"
    if [ -f "$TEST_FILE" ]; then
        echo "🔧 Compiling $TEST_FILE..."
        if ginkgo build generated/ 2>/dev/null; then
            echo "✅ Compilation successful!"
        else
            echo "❌ Compilation failed. Showing errors:"
            ginkgo build generated/
        fi
    else
        echo "❌ Test file not found: $TEST_FILE"
        exit 1
    fi
else
    echo "🔧 Compiling all generated tests..."
    if ginkgo build generated/ 2>/dev/null; then
        echo "✅ All generated tests compile successfully!"
    else
        echo "❌ Some tests failed compilation:"
        ginkgo build generated/
    fi
fi

echo ""
echo "💡 To run generated tests in full KubeEdge environment:"
echo "   1. Ensure KubeEdge cluster is running (make e2e sets this up)"
echo "   2. Run: ginkgo tests/e2e/generated/"
echo ""
echo "⚠️  Note: Generated tests may need KubeEdge cluster to be running"
echo "   Use 'make e2e' to set up full test environment"