#!/bin/bash

echo "🧪 Running KubeEdge Auto Test Generator Tests..."

# Load environment variables if .env exists
if [ -f ../.env ]; then
    export $(cat ../.env | xargs)
fi

# Test setup
echo "📋 Testing setup..."
cd ../
go run test-tools/test_setup.go

# Test compilation
echo "📋 Testing compilation..."
go run . -help

echo "✅ Test completed!"
