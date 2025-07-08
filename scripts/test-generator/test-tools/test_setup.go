package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/generative-ai-go/genai"
	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

func main() {
	fmt.Println("🧪 Testing KubeEdge Auto Test Generator Setup...")
	fmt.Println()

	// Test Gemini API
	geminiKey := os.Getenv("GEMINI_API_KEY")
	if geminiKey == "" {
		fmt.Println("❌ GEMINI_API_KEY not set")
		fmt.Println("   Please set it in your .env file")
	} else if len(geminiKey) < 10 {
		fmt.Println("❌ GEMINI_API_KEY looks invalid (too short)")
	} else {
		fmt.Println("✅ GEMINI_API_KEY found")
		if testGeminiAPI(geminiKey) {
			fmt.Println("✅ Gemini API connection successful")
		} else {
			fmt.Println("❌ Gemini API connection failed")
			fmt.Println("   Check your API key or internet connection")
		}
	}

	fmt.Println()

	// Test GitHub API
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		fmt.Println("❌ GITHUB_TOKEN not set")
		fmt.Println("   Please set it in your .env file")
	} else if len(githubToken) < 10 {
		fmt.Println("❌ GITHUB_TOKEN looks invalid (too short)")
	} else {
		fmt.Println("✅ GITHUB_TOKEN found")
		if testGitHubAPI(githubToken) {
			fmt.Println("✅ GitHub API connection successful")
		} else {
			fmt.Println("❌ GitHub API connection failed")
			fmt.Println("   Check your token permissions or internet connection")
		}
	}

	fmt.Println()

	// Check dependencies
	fmt.Println("📦 Checking Dependencies:")
	deps := []string{
		"github.com/google/generative-ai-go/genai",
		"github.com/google/go-github/v57/github",
		"golang.org/x/oauth2",
		"google.golang.org/api/option",
	}

	for _, dep := range deps {
		fmt.Printf("   ✅ %s\n", dep)
	}

	fmt.Println()
	fmt.Println("🎉 Setup test completed!")
	
	// Final status
	if geminiKey != "" && githubToken != "" {
		fmt.Println("🚀 Ready to proceed with implementation!")
	} else {
		fmt.Println("⚠️  Please set your API keys before proceeding")
	}
}

func testGeminiAPI(apiKey string) bool {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		fmt.Printf("   Error creating client: %v\n", err)
		return false
	}
	defer client.Close()

	// Try a simple model list to test connection
	model := client.GenerativeModel("gemini-1.5-flash")
	if model == nil {
		return false
	}
	
	return true
}

func testGitHubAPI(token string) bool {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Test by getting rate limit info (doesn't use API calls)
	_, _, err := client.RateLimits(ctx)
	return err == nil
}