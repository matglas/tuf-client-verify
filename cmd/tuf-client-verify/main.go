package main

import (
	"log"
	"net/http"
	"os"

	"github.com/matglas/tuf-client-verify/internal/tuf"
)

const (
	DefaultPort     = "8080"
	DefaultRepoPath = "testdata/repository"
)

var tufClient *tuf.Client

// authHandler handles nginx auth_request calls with TUF verification
func authHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the original URI from nginx headers
	originalURI := r.Header.Get("X-Original-URI")
	if originalURI == "" {
		originalURI = r.URL.Path
	}

	log.Printf("Auth request for: %s (Method: %s)", originalURI, r.Method)

	// Verify path against TUF metadata
	allowed, err := tufClient.VerifyPath(originalURI)
	if err != nil {
		log.Printf("TUF verification error for %s: %v", originalURI, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
		return
	}

	if allowed {
		log.Printf("✅ ALLOWED: %s", originalURI)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	} else {
		log.Printf("❌ DENIED: %s", originalURI)
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Forbidden"))
	}
}

// healthHandler provides a simple health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("healthy"))
}

// debugHandler provides debug information about allowed paths
func debugHandler(w http.ResponseWriter, r *http.Request) {
	paths, err := tufClient.GetAllowedPaths()
	if err != nil {
		log.Printf("Error getting allowed paths: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := `{"allowed_paths": [`
	for i, path := range paths {
		if i > 0 {
			response += ","
		}
		response += `"` + path + `"`
	}
	response += `], "delegations": {`

	delegations := tufClient.GetDelegationInfo()
	first := true
	for role, patterns := range delegations {
		if !first {
			response += ","
		}
		first = false
		response += `"` + role + `": [`
		for j, pattern := range patterns {
			if j > 0 {
				response += ","
			}
			response += `"` + pattern + `"`
		}
		response += `]`
	}
	response += `}}`

	w.Write([]byte(response))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultPort
	}

	// Initialize TUF client
	repoPath := os.Getenv("TUF_REPO_PATH")
	if repoPath == "" {
		repoPath = DefaultRepoPath
	}

	var err error
	tufClient, err = tuf.NewClient(tuf.Config{RepoPath: repoPath})
	if err != nil {
		log.Fatalf("Failed to initialize TUF client: %v", err)
	}

	log.Printf("TUF client initialized with repository: %s", repoPath)

	// Set up routes
	http.HandleFunc("/auth", authHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/debug", debugHandler)

	// Root handler for basic info
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("TUF Client Verify Service - Phase 2 with TUF"))
	})

	log.Printf("TUF Client Verify service starting on port %s", port)
	log.Printf("Auth endpoint: http://localhost:%s/auth", port)
	log.Printf("Health endpoint: http://localhost:%s/health", port)
	log.Printf("Debug endpoint: http://localhost:%s/debug", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
