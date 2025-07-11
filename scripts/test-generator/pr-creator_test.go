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

package pr_creator_test

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/google/go-github/v57/github"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"kubeedge.io/kubeedge/scripts/test-generator/pr_creator"
)

// Mock functions for testing

func mockGetRepository(repo *github.Repository, err error) gomonkey.Output {
	return gomonkey.Output{Values: []interface{}{repo, nil, err}}
}

func mockGetRef(ref *github.Reference, err error) gomonkey.Output {
	return gomonkey.Output{Values: []interface{}{ref, nil, err}}
}

func mockCreateRef(err error) gomonkey.Output {
	return gomonkey.Output{Values: []interface{}{nil, nil, err}}
}

func mockCreateFile(err error) gomonkey.Output {
	return gomonkey.Output{Values: []interface{}{nil, nil, err}}
}

func mockCreatePullRequest(pr *github.PullRequest, err error) gomonkey.Output {
	return gomonkey.Output{Values: []interface{}{pr, nil, err}}
}

func mockAddLabelsToIssue(err error) gomonkey.Output {
	return gomonkey.Output{Values: []interface{}{nil, nil, err}}
}

func mockRequestReviewers(err error) gomonkey.Output {
	return gomonkey.Output{Values: []interface{}{nil, nil, err}}
}

func mockGetContents(content *github.RepositoryContent, err error) gomonkey.Output {
	return gomonkey.Output{Values: []interface{}{content, nil, nil, err}}
}

func mockGetContent(content string, err error) gomonkey.Output {
	return gomonkey.Output{Values: []interface{}{content, err}}
}

func mockListCollaborators(collaborators []*github.User, err error) gomonkey.Output {
	return gomonkey.Output{Values: []interface{}{collaborators, nil, err}}
}

func mockCreateComment(err error) gomonkey.Output {
	return gomonkey.Output{Values: []interface{}{nil, nil, err}}
}


func TestNewPRCreator(t *testing.T) {
	token := "test-token"
	repoOwner := "test-owner"
	repoName := "test-repo"

	pc := pr_creator.NewPRCreator(token, repoOwner, repoName)

	assert.NotNil(t, pc)
	assert.NotNil(t, pc.Client)
	assert.Equal(t, repoOwner, pc.RepoOwner)
	assert.Equal(t, repoName, pc.RepoName)
}

func TestCreateTestPR_FileExists(t *testing.T) {
	pc := &pr_creator.PRCreator{}
	ctx := context.Background()
	sourceFile := "test.go"
	testFile := "test_test.go"
	testContent := "test content"
	branchName := "test-branch"
	coverage := 80.0

	existingFile := &github.RepositoryContent{}
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(reflect.TypeOf(pc).MethodByName("getFileContent").Func, func(ctx context.Context, filePath string) (*github.RepositoryContent, error) {
		return existingFile, nil
	})
	patches.ApplyMethod(reflect.TypeOf(pc), "updateExistingTestFile", func(ctx context.Context, sourceFile, testFile, testContent, branchName string, coverage float64, existingFile *github.RepositoryContent) error {
		return nil
	})

	err := pc.CreateTestPR(ctx, sourceFile, testFile, testContent, branchName, coverage)
	assert.NoError(t, err)
}

func TestCreateTestPR_FileDoesNotExist(t *testing.T) {
	pc := &pr_creator.PRCreator{}
	ctx := context.Background()
	sourceFile := "test.go"
	testFile := "test_test.go"
	testContent := "test content"
	branchName := "test-branch"
	coverage := 80.0

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(reflect.TypeOf(pc).MethodByName("getFileContent").Func, func(ctx context.Context, filePath string) (*github.RepositoryContent, error) {
		return nil, nil
	})
	patches.ApplyMethod(reflect.TypeOf(pc), "createNewTestFile", func(ctx context.Context, sourceFile, testFile, testContent, branchName string, coverage float64) error {
		return nil
	})

	err := pc.CreateTestPR(ctx, sourceFile, testFile, testContent, branchName, coverage)
	assert.NoError(t, err)
}

func TestCreateTestPR_Error(t *testing.T) {
	pc := &pr_creator.PRCreator{}
	ctx := context.Background()
	sourceFile := "test.go"
	testFile := "test_test.go"
	testContent := "test content"
	branchName := "test-branch"
	coverage := 80.0

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(reflect.TypeOf(pc).MethodByName("getFileContent").Func, func(ctx context.Context, filePath string) (*github.RepositoryContent, error) {
		return nil, fmt.Errorf("test error")
	})
	patches.ApplyMethod(reflect.TypeOf(pc), "createNewTestFile", func(ctx context.Context, sourceFile, testFile, testContent, branchName string, coverage float64) error {
		return fmt.Errorf("test error")
	})

	err := pc.CreateTestPR(ctx, sourceFile, testFile, testContent, branchName, coverage)
	assert.Error(t, err)
}


func TestcreateNewTestFile(t *testing.T) {
	pc := &pr_creator.PRCreator{client: &github.Client{}}
	ctx := context.Background()
	sourceFile := "test.go"
	testFile := "test_test.go"
	testContent := "test content"
	branchName := "test-branch"
	coverage := 80.0
	defaultBranch := "main"

	repo := &github.Repository{DefaultBranch: &defaultBranch}
	ref := &github.Reference{Object: &github.GitObject{SHA: github.String("test-sha")}}
	pr := &github.PullRequest{Number: github.Int(1)}

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(reflect.TypeOf(pc.Client).MethodByName("Repositories").MethodByName("Get").Func, mockGetRepository(repo, nil))
	patches.ApplyFunc(reflect.TypeOf(pc.Client).MethodByName("Git").MethodByName("GetRef").Func, mockGetRef(ref, nil))
	patches.ApplyFunc(reflect.TypeOf(pc.Client).MethodByName("Git").MethodByName("CreateRef").Func, mockCreateRef(nil))
	patches.ApplyFunc(reflect.TypeOf(pc.Client).MethodByName("Repositories").MethodByName("CreateFile").Func, mockCreateFile(nil))
	patches.ApplyFunc(reflect.TypeOf(pc.Client).MethodByName("PullRequests").MethodByName("Create").Func, mockCreatePullRequest(pr, nil))
	patches.ApplyFunc(reflect.TypeOf(pc.Client).MethodByName("Issues").MethodByName("AddLabelsToIssue").Func, mockAddLabelsToIssue(nil))
	patches.ApplyFunc(reflect.TypeOf(pc).MethodByName("getReviewersFromOWNERS").Func, func(ctx context.Context, sourceFile string) []string { return []string{} })
	patches.ApplyFunc(reflect.TypeOf(pc.Client).MethodByName("PullRequests").MethodByName("RequestReviewers").Func, mockRequestReviewers(nil))


	err := pc.createNewTestFile(ctx, sourceFile, testFile, testContent, branchName, coverage)
	assert.NoError(t, err)
}

func TestcreateNewTestFile_Error(t *testing.T) {
	pc := &pr_creator.PRCreator{client: &github.Client{}}
	ctx := context.Background()
	sourceFile := "test.go"
	testFile := "test_test.go"
	testContent := "test content"
	branchName := "test-branch"
	coverage := 80.0

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(reflect.TypeOf(pc.Client).MethodByName("Repositories").MethodByName("Get").Func, mockGetRepository(nil, fmt.Errorf("test error")))

	err := pc.createNewTestFile(ctx, sourceFile, testFile, testContent, branchName, coverage)
	assert.Error(t, err)
}


func TestupdateExistingTestFile(t *testing.T) {
	pc := &pr_creator.PRCreator{client: &github.Client{}}
	ctx := context.Background()
	sourceFile := "test.go"
	testFile := "test_test.go"
	testContent := "test content"
	branchName := "test-branch"
	coverage := 80.0
	defaultBranch := "main"
	existingContent := "existing content"
	mergedContent := "merged content"
	existingFile := &github.RepositoryContent{SHA: github.String("test-sha"), Content: &existingContent}
	repo := &github.Repository{DefaultBranch: &defaultBranch}
	ref := &github.Reference{Object: &github.GitObject{SHA: github.String("test-sha")}}
	pr := &github.PullRequest{Number: github.Int(1)}

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyMethod(reflect.TypeOf(existingFile), "GetContent", mockGetContent(existingContent, nil))
	patches.ApplyMethod(reflect.TypeOf(pc), "mergeTestContent", func(existingContent, newContent string) string { return mergedContent })
	patches.ApplyFunc(reflect.TypeOf(pc.Client).MethodByName("Repositories").MethodByName("Get").Func, mockGetRepository(repo, nil))
	patches.ApplyFunc(reflect.TypeOf(pc.Client).MethodByName("Git").MethodByName("GetRef").Func, mockGetRef(ref, nil))
	patches.ApplyFunc(reflect.TypeOf(pc.Client).MethodByName("Git").MethodByName("CreateRef").Func, mockCreateRef(nil))
	patches.ApplyFunc(reflect.TypeOf(pc.Client).MethodByName("Repositories").MethodByName("UpdateFile").Func, mockCreateFile(nil))
	patches.ApplyFunc(reflect.TypeOf(pc.Client).MethodByName("PullRequests").MethodByName("Create").Func, mockCreatePullRequest(pr, nil))

	err := pc.updateExistingTestFile(ctx, sourceFile, testFile, testContent, branchName, coverage, existingFile)
	assert.NoError(t, err)
}

func TestgetFileContent(t *testing.T) {
	pc := &pr_creator.PRCreator{client: &github.Client{}}
	ctx := context.Background()
	filePath := "test.go"
	content := "test content"
	fileContent := &github.RepositoryContent{Content: &content}

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(reflect.TypeOf(pc.Client).MethodByName("Repositories").MethodByName("GetContents").Func, mockGetContents(fileContent, nil))

	result, err := pc.getFileContent(ctx, filePath)
	assert.NoError(t, err)
	assert.Equal(t, fileContent, result)
}

func TestmergeTestContent(t *testing.T) {
	pc := &pr_creator.PRCreator{}
	existingContent := "package main\nfunc TestExisting(){}\n"
	newContent := "package main\nfunc TestNew(){}\n"
	mergedContent := pc.mergeTestContent(existingContent, newContent)
	assert.Contains(t, mergedContent, "TestExisting")
	assert.Contains(t, mergedContent, "TestNew")
}

func TestbuildPRDescription(t *testing.T) {
	pc := &pr_creator.PRCreator{}
	description := pc.buildPRDescription("test.go", 80.0, false)
	assert.Contains(t, description, "Auto-Generated Unit Tests")
	assert.Contains(t, description, "80.00%")
	description = pc.buildPRDescription("test.go", 30.0, true)
	assert.Contains(t, description, "Auto-Generated Test Updates")
	assert.Contains(t, description, "30.00%")

}

func TestgetReviewersFromOWNERS(t *testing.T) {
	pc := &pr_creator.PRCreator{client: &github.Client{}}
	ctx := context.Background()
	sourceFile := "test.go"
	ownersContent := "@user1\n@user2\n#comment\nuser3"
	fileContent := &github.RepositoryContent{Content: &ownersContent}

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(reflect.TypeOf(pc).MethodByName("getFileContent").Func, func(ctx context.Context, filePath string) (*github.RepositoryContent, error) {
		return fileContent, nil
	})

	reviewers := pc.getReviewersFromOWNERS(ctx, sourceFile)
	assert.Len(t, reviewers, 3)
	assert.Contains(t, reviewers, "user1")
	assert.Contains(t, reviewers, "user2")
	assert.Contains(t, reviewers, "user3")
}

func TestgetRepositoryCollaborators(t *testing.T) {
	pc := &pr_creator.PRCreator{client: &github.Client{}}
	ctx := context.Background()
	collaborators := []*github.User{{Login: github.String("user1")}, {Login: github.String("user2")}}

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(reflect.TypeOf(pc.Client).MethodByName("Repositories").MethodByName("ListCollaborators").Func, mockListCollaborators(collaborators, nil))

	reviewers := pc.getRepositoryCollaborators(ctx)
	assert.Len(t, reviewers, 2)
	assert.Contains(t, reviewers, "user1")
	assert.Contains(t, reviewers, "user2")
}


func TestCommentOnPR(t *testing.T) {
	pc := &pr_creator.PRCreator{client: &github.Client{}}
	ctx := context.Background()
	prNumber := "1"
	message := "test message"

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(reflect.TypeOf(pc.Client).MethodByName("Issues").MethodByName("CreateComment").Func, mockCreateComment(nil))

	err := pc.CommentOnPR(ctx, prNumber, message)
	assert.NoError(t, err)
}

func TestAddSuccessComment(t *testing.T) {
	pc := &pr_creator.PRCreator{client: &github.Client{}}
	ctx := context.Background()
	prNumber := "1"
	sourceFile := "test.go"
	testPRNumber := 2

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyMethod(reflect.TypeOf(pc), "CommentOnPR", func(ctx context.Context, prNumber, message string) error { return nil })

	err := pc.AddSuccessComment(ctx, prNumber, sourceFile, testPRNumber)
	assert.NoError(t, err)
}

func TestAddFailureComment(t *testing.T) {
	pc := &pr_creator.PRCreator{client: &github.Client{}}
	ctx := context.Background()
	prNumber := "1"
	sourceFile := "test.go"
	attempts := 3
	lastError := fmt.Errorf("test error")

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyMethod(reflect.TypeOf(pc), "CommentOnPR", func(ctx context.Context, prNumber, message string) error { return nil })

	err := pc.AddFailureComment(ctx, prNumber, sourceFile, attempts, lastError)
	assert.NoError(t, err)
}

func TestCheckRateLimit(t *testing.T) {
	pc := &pr_creator.PRCreator{client: &github.Client{}}
	ctx := context.Background()
	rateLimit := &github.Rate{Rate: &github.RateLimits{Core: &github.RateLimit{Remaining: 15, Limit: 100}}}

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(reflect.TypeOf(pc.Client).MethodByName("RateLimits").Func, func(ctx context.Context) (*github.Rate, *github.Response, error) { return rateLimit, nil, nil })

	err := pc.CheckRateLimit(ctx)
	assert.NoError(t, err)
}

func TestGetRepository(t *testing.T) {
	pc := &pr_creator.PRCreator{client: &github.Client{}}
	ctx := context.Background()
	repo := &github.Repository{}

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(reflect.TypeOf(pc.Client).MethodByName("Repositories").MethodByName("Get").Func, mockGetRepository(repo, nil))

	result, err := pc.GetRepository(ctx)
	assert.NoError(t, err)
	assert.Equal(t, repo, result)
}

func TestCreateTestsPR(t *testing.T) {
	pc := &pr_creator.PRCreator{}
	ctx := context.Background()
	sourceFile := "test.go"
	testContent := "test content"
	coverage := 80.0

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyMethod(reflect.TypeOf(pc), "generateTestFileName", func(sourceFile string) string { return "test_test.go" })
	patches.ApplyMethod(reflect.TypeOf(pc), "generateBranchName", func(sourceFile string) string { return "test-branch" })
	patches.ApplyMethod(reflect.TypeOf(pc), "CreateTestPR", func(ctx context.Context, sourceFile, testFile, testContent, branchName string, coverage float64) error { return nil })

	prNumber, err := pc.CreateTestsPR(ctx, sourceFile, testContent, coverage)
	assert.NoError(t, err)
	assert.Equal(t, 0, prNumber)
}

func TestgenerateTestFileName(t *testing.T) {
	pc := &pr_creator.PRCreator{}
	sourceFile := "/path/to/test.go"
	testFileName := pc.generateTestFileName(sourceFile)
	assert.Equal(t, "/path/to/test_test.go", testFileName)
}

func TestgenerateBranchName(t *testing.T) {
	pc := &pr_creator.PRCreator{}
	sourceFile := "/path/to/test.go"
	timestamp := time.Now().Format("20060102-150405")
	branchName := pc.generateBranchName(sourceFile)
	assert.Equal(t, fmt.Sprintf("auto-test-generation-path-to-test-go-%s", timestamp), branchName)
}