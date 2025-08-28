// Package tuf provides TUF client wrapper functionality for path verification
// against delegated TUF metadata.
package tuf

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/theupdateframework/go-tuf/v2/metadata"
)

// Client wraps TUF metadata for path verification
type Client struct {
	rootMeta      *metadata.Metadata[metadata.RootType]
	targetsMeta   *metadata.Metadata[metadata.TargetsType]
	delegatedMeta map[string]*metadata.Metadata[metadata.TargetsType]
}

// Config holds configuration for TUF client initialization
type Config struct {
	// RepoPath is the local path to the TUF repository
	RepoPath string
}

// NewLocalFileClient creates a TUF client that reads from local files
func NewLocalFileClient(repoPath string) (*Client, error) {
	cfg := Config{
		RepoPath: repoPath,
	}

	return NewClient(cfg)
}

// NewClient creates a new TUF client with the given configuration
func NewClient(cfg Config) (*Client, error) {
	// Load root metadata
	rootBytes, err := os.ReadFile(filepath.Join(cfg.RepoPath, "root.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read root metadata: %w", err)
	}

	rootMeta := &metadata.Metadata[metadata.RootType]{}
	if err := json.Unmarshal(rootBytes, rootMeta); err != nil {
		return nil, fmt.Errorf("failed to parse root metadata: %w", err)
	}

	// Load targets metadata
	targetsBytes, err := os.ReadFile(filepath.Join(cfg.RepoPath, "targets.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read targets metadata: %w", err)
	}

	targetsMeta := &metadata.Metadata[metadata.TargetsType]{}
	if err := json.Unmarshal(targetsBytes, targetsMeta); err != nil {
		return nil, fmt.Errorf("failed to parse targets metadata: %w", err)
	}

	client := &Client{
		rootMeta:      rootMeta,
		targetsMeta:   targetsMeta,
		delegatedMeta: make(map[string]*metadata.Metadata[metadata.TargetsType]),
	}

	// Load delegated targets metadata
	if err := client.loadDelegatedTargets(cfg.RepoPath); err != nil {
		return nil, fmt.Errorf("failed to load delegated targets: %w", err)
	}

	return client, nil
}

// loadDelegatedTargets loads all delegated targets metadata files
func (c *Client) loadDelegatedTargets(repoPath string) error {
	if c.targetsMeta.Signed.Delegations == nil {
		return nil // No delegations
	}

	for _, role := range c.targetsMeta.Signed.Delegations.Roles {
		metaFile := filepath.Join(repoPath, role.Name+".json")
		delegatedBytes, err := os.ReadFile(metaFile)
		if err != nil {
			// Don't fail if delegated metadata is missing - just log and continue
			continue
		}

		delegatedMeta := &metadata.Metadata[metadata.TargetsType]{}
		if err := json.Unmarshal(delegatedBytes, delegatedMeta); err != nil {
			return fmt.Errorf("failed to parse delegated metadata %s: %w", role.Name, err)
		}

		c.delegatedMeta[role.Name] = delegatedMeta
	}

	return nil
}

// VerifyPath checks if the given path is allowed according to TUF delegation
func (c *Client) VerifyPath(path string) (bool, error) {
	// Normalize path by ensuring it starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// First check top-level targets
	if c.hasTargetInMetadata(c.targetsMeta, path) {
		return true, nil
	}

	// Check delegated targets if they exist
	if c.targetsMeta.Signed.Delegations != nil {
		for _, role := range c.targetsMeta.Signed.Delegations.Roles {
			// Check if path matches delegation patterns
			if c.pathMatchesDelegation(path, role.Paths) {
				// Check if we have the delegated metadata
				if delegatedMeta, exists := c.delegatedMeta[role.Name]; exists {
					if c.hasTargetInMetadata(delegatedMeta, path) {
						return true, nil
					}
				}
			}
		}
	}

	return false, nil
}

// hasTargetInMetadata checks if a target path exists in the given metadata
func (c *Client) hasTargetInMetadata(meta *metadata.Metadata[metadata.TargetsType], path string) bool {
	_, exists := meta.Signed.Targets[path]
	return exists
}

// pathMatchesDelegation checks if a path matches any of the delegation patterns
func (c *Client) pathMatchesDelegation(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if c.matchPattern(path, pattern) {
			return true
		}
	}
	return false
}

// matchPattern performs simple pattern matching (supports * wildcard)
func (c *Client) matchPattern(path, pattern string) bool {
	// Simple wildcard matching - in production you'd want more sophisticated pattern matching
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(path, prefix)
	}

	return path == pattern
}

// GetAllowedPaths returns a list of all paths that are allowed by the TUF metadata
func (c *Client) GetAllowedPaths() ([]string, error) {
	var paths []string

	// Get paths from top-level targets
	for path := range c.targetsMeta.Signed.Targets {
		paths = append(paths, path)
	}

	// Get paths from delegated targets
	for _, delegatedMeta := range c.delegatedMeta {
		for path := range delegatedMeta.Signed.Targets {
			paths = append(paths, path)
		}
	}

	return paths, nil
}

// GetDelegationInfo returns information about the delegations
func (c *Client) GetDelegationInfo() map[string][]string {
	delegations := make(map[string][]string)

	if c.targetsMeta.Signed.Delegations != nil {
		for _, role := range c.targetsMeta.Signed.Delegations.Roles {
			delegations[role.Name] = role.Paths
		}
	}

	return delegations
}

// Close cleans up any resources used by the client
func (c *Client) Close() error {
	// Currently no cleanup needed, but this provides a clean interface
	// for future resource management
	return nil
}

// ValidateConfig checks if the provided configuration is valid
func ValidateConfig(cfg Config) error {
	if cfg.RepoPath == "" {
		return fmt.Errorf("RepoPath is required")
	}

	// Check if repository directory exists
	if _, err := os.Stat(cfg.RepoPath); os.IsNotExist(err) {
		return fmt.Errorf("repository directory does not exist: %s", cfg.RepoPath)
	}

	// Check if root metadata file exists
	rootPath := filepath.Join(cfg.RepoPath, "root.json")
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return fmt.Errorf("root metadata file does not exist: %s", rootPath)
	}

	return nil
}
