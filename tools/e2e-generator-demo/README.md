# KubeEdge E2E Test Generator Demo

A demo implementation of LLM-powered E2E test generation for KubeEdge using Google Gemini 1.5 Flash API.

## Features

- ✅ **Component Discovery**: Automatically discovers KubeEdge components (cloudhub, edgehub, etc.)
- ✅ **Coverage Analysis**: Identifies missing E2E test coverage
- ✅ **LLM Integration**: Uses Google Gemini 1.5 Flash (free API) to generate tests
- ✅ **Ginkgo/Gomega**: Generates tests following KubeEdge patterns
- ✅ **CLI Interface**: Easy to use command-line tool

## Setup

### 1. Get Gemini API Key
1. Go to [Google AI Studio](https://aistudio.google.com/app/apikey)
2. Create a new API key
3. Set environment variable:
```bash
export GEMINI_API_KEY="your-api-key-here"
```

### 2. Build the Tool
```bash
cd tools/e2e-generator-demo
go mod tidy
go build -o e2e-generator ./cmd/main.go
```

## Usage

### Analyze Coverage Only
```bash
./e2e-generator --kubeedge-root ../.. --analyze
```

### Generate Tests (Dry Run)
```bash
./e2e-generator --kubeedge-root ../.. --dry-run
```

### Generate Tests for Specific Component
```bash
./e2e-generator --kubeedge-root ../.. --component cloudhub --output ../../tests/e2e/generated
```

### Generate All High Priority Tests
```bash
./e2e-generator --kubeedge-root ../.. --output ../../tests/e2e/generated
```

## Command Line Options

- `--kubeedge-root`: Path to KubeEdge repository root (default: ".")
- `--component`: Specific component to generate tests for (optional)
- `--api-key`: Gemini API key (or set GEMINI_API_KEY env var)
- `--output`: Output directory for generated tests (default: "tests/e2e/generated")
- `--dry-run`: Print generated tests without writing files
- `--analyze`: Only analyze coverage gaps without generating tests

## Example Output

```bash
🚀 KubeEdge E2E Test Generator Demo
=====================================
📋 Discovering KubeEdge components...
✅ Found 4 components:
   - cloudhub (WebSocket server for cloud-edge communication)
   - edgehub (WebSocket client for edge-cloud communication)
   - edgecontroller (Manages edge nodes and pod metadata synchronization)
   - metamanager (Manages metadata between edged and edgehub)

🔍 Analyzing E2E test coverage...
✅ Found 16 coverage gaps:
   - cloudhub: WebSocket connection establishment (high priority)
   - cloudhub: Message routing between cloud and edge (high priority)
   - edgehub: Cloud connection establishment (high priority)
   ...

🤖 Generating E2E tests using Gemini 1.5 Flash...

📝 Generating tests for cloudhub...
✅ Generated test written to: ../../tests/e2e/generated/cloudhub_generated_test.go

🎉 Test generation complete!
```

## Generated Test Structure

The tool generates E2E tests following KubeEdge patterns:

```go
package cloudhub

import (
    "github.com/onsi/ginkgo/v2"
    "github.com/onsi/gomega"
    // ... other imports
)

var _ = GroupDescribe("CloudHub E2E Tests", func() {
    var testTimer *utils.TestTimer
    var clientSet clientset.Interface

    ginkgo.BeforeEach(func() {
        // Setup code
    })

    ginkgo.AfterEach(func() {
        // Cleanup code
    })

    ginkgo.It("E2E_CLOUDHUB_1: WebSocket connection establishment", func() {
        // Test implementation
    })

    ginkgo.It("E2E_CLOUDHUB_2: Message routing between cloud and edge", func() {
        // Test implementation
    })
})
```

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Coverage        │    │ Component       │    │ Gemini 1.5      │
│ Analyzer        │───▶│ Discovery       │───▶│ Flash API       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                        │
                                                        ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Generated       │◀───│ Response        │◀───│ LLM Context     │
│ E2E Tests       │    │ Processor       │    │ Builder         │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Next Steps

This demo provides a foundation for:
1. Integration with CI/CD pipelines
2. Automated PR creation
3. More sophisticated coverage analysis
4. Component-specific prompt templates
5. Test validation and compilation checks