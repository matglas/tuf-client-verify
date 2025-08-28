// Copyright 2024 The Update Framework Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License
//
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"crypto"
	"crypto/ed25519"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/theupdateframework/go-tuf/v2/metadata"
	"github.com/theupdateframework/go-tuf/v2/metadata/repository"
)

// A TUF repository generator for tuf-client-verify testing.
// Creates a repository with delegation for /v2/library/* paths.

func main() {
	// Create testdata directory if it doesn't exist
	testdataDir := "testdata"
	if err := os.MkdirAll(testdataDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create testdata directory: %v", err))
	}

	// Create repository metadata output directory
	repoDir := filepath.Join(testdataDir, "repository")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create repository directory: %v", err))
	}

	// Create targets directory for actual target files
	targetsDir := filepath.Join(repoDir, "targets")
	if err := os.MkdirAll(targetsDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create targets directory: %v", err))
	}

	// Initialize repository and key storage
	roles := repository.New()
	keys := map[string]ed25519.PrivateKey{}

	// Helper function for expiration times
	expireIn := func(days int) time.Time {
		return time.Now().Add(time.Duration(days) * 24 * time.Hour)
	}

	// Create target files for testing
	createTestTargetFiles(targetsDir)

	// Create top-level targets role
	targets := metadata.Targets(expireIn(30))
	roles.SetTargets("targets", targets)

	// Create snapshot and timestamp metadata
	snapshot := metadata.Snapshot(expireIn(7))
	roles.SetSnapshot(snapshot)

	timestamp := metadata.Timestamp(expireIn(1))
	roles.SetTimestamp(timestamp)

	// Create root metadata and keys for all top-level roles
	root := metadata.Root(expireIn(365))
	roles.SetRoot(root)

	// Generate keys and register them with root
	for _, roleName := range []string{"root", "targets", "snapshot", "timestamp"} {
		_, privateKey, err := ed25519.GenerateKey(nil)
		if err != nil {
			panic(fmt.Sprintf("Failed to generate key for %s: %v", roleName, err))
		}
		keys[roleName] = privateKey

		// Convert to TUF key format and add to root
		key, err := metadata.KeyFromPublicKey(privateKey.Public())
		if err != nil {
			panic(fmt.Sprintf("Failed to convert key for %s: %v", roleName, err))
		}

		err = roles.Root().Signed.AddKey(key, roleName)
		if err != nil {
			panic(fmt.Sprintf("Failed to add key for %s: %v", roleName, err))
		}
	}

	// Create delegation for /v2/library/* paths
	delegateeName := "registry-library"
	_, delegateePrivateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to generate delegatee key: %v", err))
	}
	keys[delegateeName] = delegateePrivateKey

	// Create delegated targets metadata
	delegatee := metadata.Targets(expireIn(30))
	roles.SetTargets(delegateeName, delegatee)

	// Add target files to delegated role
	targetPaths := []string{
		"/v2/library/alpine/manifests/latest",
		"/v2/library/ubuntu/manifests/20.04",
		"/v2/library/nginx/manifests/latest",
	}

	for _, targetPath := range targetPaths {
		// Create a dummy target file for this path
		dummyFileName := filepath.Base(targetPath) + ".json"
		dummyFilePath := filepath.Join(targetsDir, dummyFileName)
		dummyContent := fmt.Sprintf(`{"path": "%s", "created": "%s"}`, targetPath, time.Now().Format(time.RFC3339))

		err := os.WriteFile(dummyFilePath, []byte(dummyContent), 0644)
		if err != nil {
			panic(fmt.Sprintf("Failed to create dummy target file %s: %v", dummyFilePath, err))
		}

		// Generate target file info from the actual file
		targetFileInfo, err := metadata.TargetFile().FromFile(dummyFilePath, "sha256")
		if err != nil {
			panic(fmt.Sprintf("Failed to generate target file info for %s: %v", targetPath, err))
		}

		delegatee.Signed.Targets[targetPath] = targetFileInfo
	}

	// Set up delegation in top-level targets
	delegateeKey, err := metadata.KeyFromPublicKey(delegateePrivateKey.Public())
	if err != nil {
		panic(fmt.Sprintf("Failed to convert delegatee key: %v", err))
	}

	roles.Targets("targets").Signed.Delegations = &metadata.Delegations{
		Keys: map[string]*metadata.Key{
			delegateeKey.ID(): delegateeKey,
		},
		Roles: []metadata.DelegatedRole{
			{
				Name:        delegateeName,
				KeyIDs:      []string{delegateeKey.ID()},
				Threshold:   1,
				Terminating: true,
				Paths:       []string{"/v2/library/*"},
			},
		},
	}

	// Update snapshot to include delegated targets
	roles.Snapshot().Signed.Meta["targets.json"] = metadata.MetaFile(1)
	roles.Snapshot().Signed.Meta[fmt.Sprintf("%s.json", delegateeName)] = metadata.MetaFile(1)

	// Update timestamp to reference snapshot
	roles.Timestamp().Signed.Meta["snapshot.json"] = metadata.MetaFile(1)

	// Sign all metadata
	for _, roleName := range []string{"root", "targets", "snapshot", "timestamp", delegateeName} {
		key := keys[roleName]
		signer, err := signature.LoadSigner(key, crypto.Hash(0))
		if err != nil {
			panic(fmt.Sprintf("Failed to load signer for %s: %v", roleName, err))
		}

		switch roleName {
		case "root":
			_, err = roles.Root().Sign(signer)
		case "targets":
			_, err = roles.Targets("targets").Sign(signer)
		case "snapshot":
			_, err = roles.Snapshot().Sign(signer)
		case "timestamp":
			_, err = roles.Timestamp().Sign(signer)
		case delegateeName:
			_, err = roles.Targets(delegateeName).Sign(signer)
		}

		if err != nil {
			panic(fmt.Sprintf("Failed to sign %s metadata: %v", roleName, err))
		}
	}

	// Write metadata files
	err = roles.Root().ToFile(filepath.Join(repoDir, "root.json"), true)
	if err != nil {
		panic(fmt.Sprintf("Failed to write root.json: %v", err))
	}
	fmt.Println("✓ Created root.json")

	err = roles.Targets("targets").ToFile(filepath.Join(repoDir, "targets.json"), true)
	if err != nil {
		panic(fmt.Sprintf("Failed to write targets.json: %v", err))
	}
	fmt.Println("✓ Created targets.json")

	err = roles.Snapshot().ToFile(filepath.Join(repoDir, "snapshot.json"), true)
	if err != nil {
		panic(fmt.Sprintf("Failed to write snapshot.json: %v", err))
	}
	fmt.Println("✓ Created snapshot.json")

	err = roles.Timestamp().ToFile(filepath.Join(repoDir, "timestamp.json"), true)
	if err != nil {
		panic(fmt.Sprintf("Failed to write timestamp.json: %v", err))
	}
	fmt.Println("✓ Created timestamp.json")

	err = roles.Targets(delegateeName).ToFile(filepath.Join(repoDir, delegateeName+".json"), true)
	if err != nil {
		panic(fmt.Sprintf("Failed to write %s.json: %v", delegateeName, err))
	}
	fmt.Printf("✓ Created %s.json\n", delegateeName)

	// Verify metadata signatures
	fmt.Println("\nVerifying metadata signatures...")

	err = roles.Root().VerifyDelegate("root", roles.Root())
	if err != nil {
		panic(fmt.Sprintf("Root verification failed: %v", err))
	}
	fmt.Println("✓ Root metadata verified")

	err = roles.Root().VerifyDelegate("targets", roles.Targets("targets"))
	if err != nil {
		panic(fmt.Sprintf("Targets verification failed: %v", err))
	}
	fmt.Println("✓ Targets metadata verified")

	err = roles.Root().VerifyDelegate("snapshot", roles.Snapshot())
	if err != nil {
		panic(fmt.Sprintf("Snapshot verification failed: %v", err))
	}
	fmt.Println("✓ Snapshot metadata verified")

	err = roles.Root().VerifyDelegate("timestamp", roles.Timestamp())
	if err != nil {
		panic(fmt.Sprintf("Timestamp verification failed: %v", err))
	}
	fmt.Println("✓ Timestamp metadata verified")

	err = roles.Targets("targets").VerifyDelegate(delegateeName, roles.Targets(delegateeName))
	if err != nil {
		panic(fmt.Sprintf("Delegated targets verification failed: %v", err))
	}
	fmt.Printf("✓ %s metadata verified\n", delegateeName)

	fmt.Printf("\n🎉 TUF repository created successfully in %s\n", repoDir)
	fmt.Println("\nDelegated paths for /v2/library/*:")
	for _, path := range targetPaths {
		fmt.Printf("  - %s\n", path)
	}
}

// createTestTargetFiles creates dummy target files for testing
func createTestTargetFiles(targetsDir string) {
	testFiles := []string{
		"alpine-manifest.json",
		"ubuntu-manifest.json",
		"nginx-manifest.json",
	}

	for _, filename := range testFiles {
		filepath := filepath.Join(targetsDir, filename)
		content := fmt.Sprintf(`{"name": "%s", "tag": "latest"}`, filename)

		err := os.WriteFile(filepath, []byte(content), 0644)
		if err != nil {
			panic(fmt.Sprintf("Failed to create test file %s: %v", filepath, err))
		}
	}
}
