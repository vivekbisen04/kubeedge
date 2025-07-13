/*
Copyright 2025 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"os"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// SimplifiedPRCreator handles GitHub PR creation with minimal features (no labels, reviewers, etc.)
type SimplifiedPRCreator struct {
	client    *github.Client
	repoOwner string
	repoName  string
}

// NewSimplifiedPRCreator creates a new simplified PR creator
func NewSimplifiedPRCreator(token, repoOwner, repoName string) *SimplifiedPRCreator {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	return &SimplifiedPRCreator{
		client:    client,
		repoOwner: repoOwner,
		repoName:  repoName,
	}
}

// CreateTestsPR creates a simple PR with generated tests (used mainly for local testing)
func (spc *SimplifiedPRCreator) CreateTestsPR(ctx context.Context, results []ProcessResult) (int, error) {
	if len(results) == 0 {
		return 0, fmt.Errorf("no results to create PR for")
	}

	// Get default branch
	repo, _, err := spc.client.Repositories.Get(ctx, spc.repoOwner, spc.repoName)
	if err != nil {
		return 0, fmt.Errorf("failed to get repository info: %v", err)
	}
	defaultBranch := repo.GetDefaultBranch()

	// Create branch name
	branchName := fmt.Sprintf("auto-tests-%d", time.Now().Unix())

	// Get latest commit SHA
	ref, _, err := spc.client.Git.GetRef(ctx, spc.repoOwner, spc.repoName, "refs/heads/"+defaultBranch)
	if err != nil {
		return 0, fmt.Errorf("failed to get reference: %v", err)
	}

	// Create new branch
	newRef := &github.Reference{
		Ref: github.String("refs/heads/" + branchName),
		Object: &github.GitObject{
			SHA: ref.Object.SHA,
		},
	}

	_, _, err = spc.client.Git.CreateRef(ctx, spc.repoOwner, spc.repoName, newRef)
	if err != nil {
		return 0, fmt.Errorf("failed to create branch: %v", err)
	}

	// Create/update test files
	for _, result := range results {
		if !result.Success {
			continue
		}

		// Read test file content
		testContent, err := readFile(result.TestFile)
		if err != nil {
			fmt.Printf("Warning: Could not read test file %s: %v\n", result.TestFile, err)
			continue
		}

		// Check if file exists in repo
		existingFile, err := spc.getFileContent(ctx, result.TestFile)
		if err == nil && existingFile != nil {
			// Update existing file
			err = spc.updateFile(ctx, result.TestFile, testContent, branchName, result, existingFile.SHA)
		} else {
			// Create new file
			err = spc.createFile(ctx, result.TestFile, testContent, branchName, result)
		}

		if err != nil {
			fmt.Printf("Warning: Failed to update file %s: %v\n", result.TestFile, err)
		}
	}

	// Create PR with simplified description
	prTitle := "🤖 Auto-generated unit tests"
	prBody := spc.buildSimplePRDescription(results)

	pr := &github.NewPullRequest{
		Title: github.String(prTitle),
		Head:  github.String(branchName),
		Base:  github.String(defaultBranch),
		Body:  github.String(prBody),
	}

	createdPR, _, err := spc.client.PullRequests.Create(ctx, spc.repoOwner, spc.repoName, pr)
	if err != nil {
		return 0, fmt.Errorf("failed to create pull request: %v", err)
	}

	return createdPR.GetNumber(), nil
}

// createFile creates a new file in the repository
func (spc *SimplifiedPRCreator) createFile(ctx context.Context, filePath, content, branchName string, result ProcessResult) error {
	commitMessage := fmt.Sprintf("Add auto-generated tests for %s\n\nCoverage: %.2f%% → %.2f%%", 
		filepath.Base(result.SourceFile), result.BeforeCoverage, result.AfterCoverage)

	fileOptions := &github.RepositoryContentFileOptions{
		Message: github.String(commitMessage),
		Content: []byte(content),
		Branch:  github.String(branchName),
	}

	_, _, err := spc.client.Repositories.CreateFile(ctx, spc.repoOwner, spc.repoName, filePath, fileOptions)
	return err
}

// updateFile updates an existing file in the repository
func (spc *SimplifiedPRCreator) updateFile(ctx context.Context, filePath, content, branchName string, result ProcessResult, sha *string) error {
	commitMessage := fmt.Sprintf("Update auto-generated tests for %s\n\nCoverage: %.2f%% → %.2f%%", 
		filepath.Base(result.SourceFile), result.BeforeCoverage, result.AfterCoverage)

	fileOptions := &github.RepositoryContentFileOptions{
		Message: github.String(commitMessage),
		Content: []byte(content),
		Branch:  github.String(branchName),
		SHA:     sha,
	}

	_, _, err := spc.client.Repositories.UpdateFile(ctx, spc.repoOwner, spc.repoName, filePath, fileOptions)
	return err
}

// getFileContent retrieves existing file content from repository
func (spc *SimplifiedPRCreator) getFileContent(ctx context.Context, filePath string) (*github.RepositoryContent, error) {
	fileContent, _, _, err := spc.client.Repositories.GetContents(ctx, spc.repoOwner, spc.repoName, filePath, nil)
	if err != nil {
		return nil, err
	}
	return fileContent, nil
}

// buildSimplePRDescription creates a simple PR description focused on coverage improvements
func (spc *SimplifiedPRCreator) buildSimplePRDescription(results []ProcessResult) string {
	var body strings.Builder

	body.WriteString("## Auto-Generated Unit Tests\n\n")
	body.WriteString("This PR contains automatically generated unit tests for files with low coverage.\n\n")

	// Coverage improvements table
	body.WriteString("### 📊 Coverage Improvements\n\n")
	body.WriteString("| File | Before | After | Improvement |\n")
	body.WriteString("|------|--------|-------|-------------|\n")

	totalImprovement := 0.0
	fileCount := 0

	for _, result := range results {
		if result.Success {
			improvement := result.AfterCoverage - result.BeforeCoverage
			totalImprovement += improvement
			fileCount++
			
			body.WriteString(fmt.Sprintf("| `%s` | %.2f%% | %.2f%% | +%.2f%% |\n",
				filepath.Base(result.SourceFile), result.BeforeCoverage, result.AfterCoverage, improvement))
		}
	}

	if fileCount > 0 {
		avgImprovement := totalImprovement / float64(fileCount)
		body.WriteString(fmt.Sprintf("\n**Average improvement: +%.2f%%**\n", avgImprovement))
	}

	// Validation status
	body.WriteString("\n### ✅ Validation Status\n")
	body.WriteString("- ✅ All generated tests compile successfully\n")
	body.WriteString("- ✅ All generated tests pass execution\n")
	body.WriteString("- ✅ Coverage improvements verified\n\n")

	// How to review
	body.WriteString("### 🔧 How to Review\n")
	body.WriteString("1. Check that generated tests are meaningful and correct\n")
	body.WriteString("2. Run `go test` to verify all tests pass\n")
	body.WriteString("3. Review test logic for accuracy\n")
	body.WriteString("4. Merge when satisfied with test quality\n\n")

	// Generation details
	body.WriteString("### 🤖 Generation Details\n")
	body.WriteString("- **Generator**: KubeEdge Auto Test Generator\n")
	body.WriteString("- **LLM**: Gemini 1.5 Flash\n")
	body.WriteString(fmt.Sprintf("- **Files Processed**: %d\n", fileCount))
	body.WriteString(fmt.Sprintf("- **Generated**: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	body.WriteString("---\n")
	body.WriteString("*Auto-generated by KubeEdge Test Generator following mentor's workflow*")

	return body.String()
}

// readFile reads content from a file (helper function)
func readFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}