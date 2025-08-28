package main

import (
	"log"
	"net/http"
	"os"
)

const (
	DefaultPort = "8080"
)

// authHandler handles nginx auth_request calls
func authHandler(w http.ResponseWriter, r *http.Request) {
	// For MVP Phase 1: Always return 200 (allow access)
	// Extract the original URI from nginx headers for logging
	originalURI := r.Header.Get("X-Original-URI")
	if originalURI == "" {
		originalURI = r.URL.Path
	}

	log.Printf("Auth request for: %s (Method: %s)", originalURI, r.Method)

	// For now, always allow access
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// healthHandler provides a simple health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("healthy"))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultPort
	}

	// Set up routes
	http.HandleFunc("/auth", authHandler)
	http.HandleFunc("/health", healthHandler)

	// Root handler for basic info
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("TUF Client Verify Service - Phase 1 MVP"))
	})

	log.Printf("TUF Client Verify service starting on port %s", port)
	log.Printf("Auth endpoint: http://localhost:%s/auth", port)
	log.Printf("Health endpoint: http://localhost:%s/health", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
