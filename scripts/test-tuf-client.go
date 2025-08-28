package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/matglas/tuf-client-verify/internal/tuf"
)

func main() {
	// Path to our test repository
	repoPath, err := filepath.Abs("testdata/repository")
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}

	fmt.Printf("Testing TUF client with repository at: %s\n", repoPath)

	// Create TUF client for local file repository
	client, err := tuf.NewLocalFileClient(repoPath)
	if err != nil {
		log.Fatalf("Failed to create TUF client: %v", err)
	}
	defer client.Close()

	// Test paths that should be allowed (defined in our delegation)
	allowedPaths := []string{
		"/v2/library/alpine/manifests/latest",
		"/v2/library/ubuntu/manifests/20.04",
		"/v2/library/nginx/manifests/latest",
	}

	// Test paths that should be denied
	deniedPaths := []string{
		"/v2/library/redis/manifests/latest",
		"/v1/library/alpine/manifests/latest",
		"/api/admin/panel",
		"/some/random/path",
	}

	fmt.Println("\n=== Testing Allowed Paths ===")
	for _, path := range allowedPaths {
		allowed, err := client.VerifyPath(path)
		if err != nil {
			fmt.Printf("❌ ERROR verifying %s: %v\n", path, err)
			continue
		}
		if allowed {
			fmt.Printf("✅ ALLOWED: %s\n", path)
		} else {
			fmt.Printf("❌ DENIED: %s (should be allowed)\n", path)
		}
	}

	fmt.Println("\n=== Testing Denied Paths ===")
	for _, path := range deniedPaths {
		allowed, err := client.VerifyPath(path)
		if err != nil {
			fmt.Printf("❌ ERROR verifying %s: %v\n", path, err)
			continue
		}
		if !allowed {
			fmt.Printf("✅ DENIED: %s\n", path)
		} else {
			fmt.Printf("❌ ALLOWED: %s (should be denied)\n", path)
		}
	}

	// Get all allowed paths from the TUF metadata
	fmt.Println("\n=== All Allowed Paths in TUF Metadata ===")
	allPaths, err := client.GetAllowedPaths()
	if err != nil {
		log.Fatalf("Failed to get allowed paths: %v", err)
	}

	for _, path := range allPaths {
		fmt.Printf("📝 %s\n", path)
	}

	fmt.Println("\n🎉 TUF client test completed!")
}
