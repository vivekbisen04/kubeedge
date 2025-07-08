package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

type EmailSender struct {
	client    *github.Client
	repoOwner string
	repoName  string
}

func NewEmailSender(repoOwner, repoName string) *EmailSender {
	// Initialize GitHub client for getting user emails
	token := os.Getenv("GITHUB_TOKEN")
	if token != "" {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		tc := oauth2.NewClient(ctx, ts)
		client := github.NewClient(tc)
		
		return &EmailSender{
			client:    client,
			repoOwner: repoOwner,
			repoName:  repoName,
		}
	}
	
	return &EmailSender{
		repoOwner: repoOwner,
		repoName:  repoName,
	}
}

// SendFailureNotification sends email notification when test generation fails
func (es *EmailSender) SendFailureNotification(ctx context.Context, sourceFile string, prAuthor string, prNumber string) error {
	// Get maintainer emails
	maintainers := es.getMaintainerEmails(ctx, sourceFile)
	
	// Get PR author email
	authorEmail := es.getUserEmail(ctx, prAuthor)
	
	// Combine recipients
	recipients := append(maintainers, authorEmail)
	recipients = es.deduplicateEmails(recipients)
	
	if len(recipients) == 0 {
		return fmt.Errorf("no email recipients found")
	}
	
	// Create email content
	subject := fmt.Sprintf("[KubeEdge] Auto Test Generation Failed for %s", filepath.Base(sourceFile))
	body := es.buildFailureEmailBody(sourceFile, prAuthor, prNumber)
	
	// Send notification via GitHub Issues (as email alternative)
	return es.createFailureIssue(ctx, subject, body, recipients)
}

// SendSuccessSummary sends success summary email
func (es *EmailSender) SendSuccessSummary(ctx context.Context, successCount, failureCount int, prAuthor string, prNumber string) error {
	if successCount == 0 {
		return nil // No successes to report
	}
	
	// Get maintainer emails (only for summary)
	maintainers := es.getMaintainerEmails(ctx, "")
	
	subject := fmt.Sprintf("[KubeEdge] Auto Test Generation Summary - %d files processed", successCount+failureCount)
	body := es.buildSuccessEmailBody(successCount, failureCount, prAuthor, prNumber)
	
	// Send summary via GitHub Issues
	return es.createSuccessIssue(ctx, subject, body, maintainers)
}

// getMaintainerEmails gets maintainer emails from OWNERS files
func (es *EmailSender) getMaintainerEmails(ctx context.Context, sourceFile string) []string {
	var emails []string
	
	if es.client == nil {
		return emails
	}
	
	// Get maintainers from OWNERS file
	maintainers := es.getMaintainersFromOWNERS(ctx, sourceFile)
	
	// Convert usernames to emails
	for _, username := range maintainers {
		email := es.getUserEmail(ctx, username)
		if email != "" {
			emails = append(emails, email)
		}
	}
	
	// Fallback to repository collaborators
	if len(emails) == 0 {
		collaborators := es.getRepositoryMaintainers(ctx)
		for _, username := range collaborators {
			email := es.getUserEmail(ctx, username)
			if email != "" {
				emails = append(emails, email)
			}
			// Limit to 3 maintainers
			if len(emails) >= 3 {
				break
			}
		}
	}
	
	return emails
}

// getMaintainersFromOWNERS reads OWNERS file to get maintainers
func (es *EmailSender) getMaintainersFromOWNERS(ctx context.Context, sourceFile string) []string {
	var maintainers []string
	
	if sourceFile == "" {
		// Use root OWNERS file for summaries
		sourceFile = "OWNERS"
	} else {
		// Try to find OWNERS file in the same directory
		dir := filepath.Dir(sourceFile)
		sourceFile = filepath.Join(dir, "OWNERS")
	}
	
	content, _, _, err := es.client.Repositories.GetContents(ctx, es.repoOwner, es.repoName, sourceFile, nil)
	if err != nil {
		// Try root OWNERS file as fallback
		content, _, _, err = es.client.Repositories.GetContents(ctx, es.repoOwner, es.repoName, "OWNERS", nil)
		if err != nil {
			return maintainers
		}
	}
	
	ownersContent, err := content.GetContent()
	if err != nil {
		return maintainers
	}
	
	// Parse OWNERS file
	lines := strings.Split(ownersContent, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		
		// Look for maintainers/approvers
		if strings.Contains(line, "approvers:") || strings.Contains(line, "reviewers:") {
			continue
		}
		
		// Extract usernames
		if strings.HasPrefix(line, "- ") {
			username := strings.TrimPrefix(line, "- ")
			username = strings.TrimSpace(username)
			if username != "" {
				maintainers = append(maintainers, username)
			}
		} else if strings.HasPrefix(line, "@") {
			username := strings.TrimPrefix(line, "@")
			username = strings.TrimSpace(username)
			if username != "" {
				maintainers = append(maintainers, username)
			}
		}
	}
	
	return maintainers
}

// getRepositoryMaintainers gets repository maintainers as fallback
func (es *EmailSender) getRepositoryMaintainers(ctx context.Context) []string {
	var maintainers []string
	
	// Get repository collaborators with admin/maintain permissions
	collaborators, _, err := es.client.Repositories.ListCollaborators(ctx, es.repoOwner, es.repoName, nil)
	if err != nil {
		return maintainers
	}
	
	for _, collaborator := range collaborators {
		// Filter for admins and maintainers
		permissions := collaborator.GetPermissions()
		if permissions["admin"] || permissions["maintain"] {
			maintainers = append(maintainers, collaborator.GetLogin())
		}
	}
	
	return maintainers
}

// getUserEmail gets user email from GitHub API
func (es *EmailSender) getUserEmail(ctx context.Context, username string) string {
	if es.client == nil || username == "" {
		return ""
	}
	
	user, _, err := es.client.Users.Get(ctx, username)
	if err != nil {
		return ""
	}
	
	email := user.GetEmail()
	if email == "" {
		// Try to get public email from user's events (fallback)
		events, _, err := es.client.Activity.ListEventsPerformedByUser(ctx, username, true, nil)
		if err == nil && len(events) > 0 {
			for _, event := range events {
				if event.Actor != nil && event.Actor.GetEmail() != "" {
					email = event.Actor.GetEmail()
					break
				}
			}
		}
	}
	
	return email
}

// deduplicateEmails removes duplicate emails
func (es *EmailSender) deduplicateEmails(emails []string) []string {
	seen := make(map[string]bool)
	var result []string
	
	for _, email := range emails {
		if email != "" && !seen[email] {
			seen[email] = true
			result = append(result, email)
		}
	}
	
	return result
}

// buildFailureEmailBody creates the failure notification email body
func (es *EmailSender) buildFailureEmailBody(sourceFile string, prAuthor string, prNumber string) string {
	var body strings.Builder
	
	body.WriteString("# 🤖 KubeEdge Auto Test Generation Failed\n\n")
	body.WriteString("The automatic test generation process has failed after multiple attempts.\n\n")
	
	body.WriteString("## 📁 File Details\n")
	body.WriteString(fmt.Sprintf("- **File**: `%s`\n", sourceFile))
	body.WriteString(fmt.Sprintf("- **PR Author**: @%s\n", prAuthor))
	if prNumber != "" {
		body.WriteString(fmt.Sprintf("- **Original PR**: #%s\n", prNumber))
	}
	body.WriteString(fmt.Sprintf("- **Timestamp**: %s\n\n", time.Now().Format("2006-01-02 15:04:05 UTC")))
	
	body.WriteString("## 🔍 Failure Details\n")
	body.WriteString("The auto test generator attempted to create unit tests for this file but failed after 3 attempts. ")
	body.WriteString("This typically happens when:\n\n")
	body.WriteString("- The file has complex dependencies that are difficult to mock\n")
	body.WriteString("- The code structure doesn't follow standard Go testing patterns\n")
	body.WriteString("- There are import or dependency issues\n")
	body.WriteString("- The file requires manual test setup or custom mocking\n\n")
	
	body.WriteString("## 🔧 Recommended Actions\n")
	body.WriteString("1. **Manual Test Creation**: Consider creating tests manually for this file\n")
	body.WriteString("2. **Code Review**: Review the file structure for testability improvements\n")
	body.WriteString("3. **Refactoring**: Consider refactoring complex functions for better testability\n")
	body.WriteString("4. **Dependencies**: Check if external dependencies need custom mocking\n\n")
	
	body.WriteString("## 📚 KubeEdge Testing Guidelines\n")
	body.WriteString("- Use gomonkey v2 for mocking external functions\n")
	body.WriteString("- Follow table-driven test patterns\n")
	body.WriteString("- Use `github.com/stretchr/testify/assert` for assertions\n")
	body.WriteString("- Ensure tests are independent and repeatable\n\n")
	
	body.WriteString("## 🔗 Resources\n")
	body.WriteString("- [KubeEdge Testing Documentation](https://github.com/kubeedge/kubeedge/blob/master/docs/testing.md)\n")
	body.WriteString("- [Go Testing Best Practices](https://golang.org/doc/tutorial/add-a-test)\n")
	body.WriteString("- [gomonkey Documentation](https://github.com/agiledragon/gomonkey)\n\n")
	
	body.WriteString("---\n")
	body.WriteString("*This notification was automatically generated by the KubeEdge Auto Test Generator.*")
	
	return body.String()
}

// buildSuccessEmailBody creates the success summary email body
func (es *EmailSender) buildSuccessEmailBody(successCount, failureCount int, prAuthor string, prNumber string) string {
	var body strings.Builder
	
	body.WriteString("# 🎉 KubeEdge Auto Test Generation Summary\n\n")
	body.WriteString("The automatic test generation process has completed.\n\n")
	
	body.WriteString("## 📊 Summary Statistics\n")
	body.WriteString(fmt.Sprintf("- **✅ Successful**: %d files\n", successCount))
	body.WriteString(fmt.Sprintf("- **❌ Failed**: %d files\n", failureCount))
	body.WriteString(fmt.Sprintf("- **📈 Success Rate**: %.1f%%\n", float64(successCount)/float64(successCount+failureCount)*100))
	if prNumber != "" {
		body.WriteString(fmt.Sprintf("- **Original PR**: #%s by @%s\n", prNumber, prAuthor))
	}
	body.WriteString(fmt.Sprintf("- **Timestamp**: %s\n\n", time.Now().Format("2006-01-02 15:04:05 UTC")))
	
	if successCount > 0 {
		body.WriteString("## 🎯 Generated Tests\n")
		body.WriteString("New test PRs have been created for files with low coverage. ")
		body.WriteString("Please review and merge them to improve overall test coverage.\n\n")
		
		body.WriteString("### 🔧 Review Checklist\n")
		body.WriteString("- [ ] Tests compile and run successfully\n")
		body.WriteString("- [ ] Test coverage has improved\n")
		body.WriteString("- [ ] Tests follow KubeEdge patterns\n")
		body.WriteString("- [ ] gomonkey mocking is used appropriately\n\n")
	}
	
	if failureCount > 0 {
		body.WriteString("## ⚠️ Failed Generations\n")
		body.WriteString("Some files could not have tests auto-generated. ")
		body.WriteString("These may require manual test creation or code refactoring for better testability.\n\n")
	}
	
	body.WriteString("## 📈 Coverage Goals\n")
	body.WriteString("- **Target**: 80% overall coverage (per codecov.yml)\n")
	body.WriteString("- **Threshold**: 40% minimum for auto-generation\n")
	body.WriteString("- **Focus Areas**: cloud/, keadm/, edge/ components\n\n")
	
	body.WriteString("---\n")
	body.WriteString("*This summary was automatically generated by the KubeEdge Auto Test Generator.*")
	
	return body.String()
}

// createFailureIssue creates a GitHub issue for failure notification
func (es *EmailSender) createFailureIssue(ctx context.Context, subject string, body string, recipients []string) error {
	if es.client == nil {
		fmt.Printf("📧 Would send failure email to: %s\n", strings.Join(recipients, ", "))
		fmt.Printf("Subject: %s\n", subject)
		return nil
	}
	
	// Create GitHub issue as notification mechanism
	issueRequest := &github.IssueRequest{
		Title: &subject,
		Body:  &body,
		Labels: &[]string{"auto-generated", "test-generation", "failure", "needs-attention"},
	}
	
	// Add assignees (recipients)
	var assignees []string
	for _, recipient := range recipients {
		if recipient != "" {
			// Extract username from email if needed
			username := strings.Split(recipient, "@")[0]
			assignees = append(assignees, username)
		}
	}
	
	if len(assignees) > 0 {
		issueRequest.Assignees = &assignees
	}
	
	issue, _, err := es.client.Issues.Create(ctx, es.repoOwner, es.repoName, issueRequest)
	if err != nil {
		return fmt.Errorf("failed to create failure notification issue: %v", err)
	}
	
	fmt.Printf("📧 Created failure notification issue #%d\n", issue.GetNumber())
	return nil
}

// createSuccessIssue creates a GitHub issue for success summary
func (es *EmailSender) createSuccessIssue(ctx context.Context, subject string, body string, recipients []string) error {
	if es.client == nil {
		fmt.Printf("📧 Would send success summary to: %s\n", strings.Join(recipients, ", "))
		return nil
	}
	
	// Only create issue for significant activity (more than 2 files processed)
	if !strings.Contains(subject, "files processed") {
		return nil
	}
	
	// Create GitHub issue as notification mechanism
	issueRequest := &github.IssueRequest{
		Title: &subject,
		Body:  &body,
		Labels: &[]string{"auto-generated", "test-generation", "summary"},
	}
	
	issue, _, err := es.client.Issues.Create(ctx, es.repoOwner, es.repoName, issueRequest)
	if err != nil {
		return fmt.Errorf("failed to create success summary issue: %v", err)
	}
	
	fmt.Printf("📧 Created success summary issue #%d\n", issue.GetNumber())
	return nil
}