//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
)

// TestSecretVerificationOutput tests that the activation job outputs include secret_verification_result
func TestSecretVerificationOutput(t *testing.T) {
	testDir := testutil.TempDir(t, "test-secret-verification-output-*")
	workflowFile := filepath.Join(testDir, "test-workflow.md")

	workflow := `---
on: workflow_dispatch
engine: copilot
---

Test workflow`

	if err := os.WriteFile(workflowFile, []byte(workflow), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := stringutil.MarkdownToLockFile(workflowFile)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Check that activation job has secret_verification_result output
	if !strings.Contains(lockStr, "secret_verification_result: ${{ steps.validate-secret.outputs.verification_result }}") {
		t.Error("Expected activation job to have secret_verification_result output")
	}

	// Check that validate-secret step has an id
	if !strings.Contains(lockStr, "id: validate-secret") {
		t.Error("Expected validate-secret step to have an id")
	}
}

// TestSecretVerificationOutputInConclusionJob tests that the conclusion job receives the secret verification result
func TestSecretVerificationOutputInConclusionJob(t *testing.T) {
	testDir := testutil.TempDir(t, "test-secret-verification-conclusion-*")
	workflowFile := filepath.Join(testDir, "test-workflow.md")

	workflow := `---
on: workflow_dispatch
engine: copilot
safe-outputs:
  add-comment:
    max: 5
---

Test workflow`

	if err := os.WriteFile(workflowFile, []byte(workflow), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := stringutil.MarkdownToLockFile(workflowFile)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Check that conclusion job receives secret verification result from activation job
	if !strings.Contains(lockStr, "GH_AW_SECRET_VERIFICATION_RESULT: ${{ needs.activation.outputs.secret_verification_result }}") {
		t.Error("Expected conclusion job to receive secret_verification_result from activation job")
	}
}

// TestSecretVerificationOutputSkippedWithGitHubApp tests that the validate-secret step is not
// generated when tools.github.app is configured (direct configuration).
func TestSecretVerificationOutputSkippedWithGitHubApp(t *testing.T) {
	testDir := testutil.TempDir(t, "test-secret-skip-github-app-*")
	workflowFile := filepath.Join(testDir, "test-workflow.md")

	workflow := `---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
tools:
  github:
    app:
      app-id: ${{ vars.APP_ID }}
      private-key: ${{ secrets.APP_PRIVATE_KEY }}
---

Test workflow with GitHub App authentication`

	if err := os.WriteFile(workflowFile, []byte(workflow), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := stringutil.MarkdownToLockFile(workflowFile)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Check that validate-secret step is NOT generated
	if strings.Contains(lockStr, "id: validate-secret") {
		t.Error("Expected validate-secret step to NOT be generated when tools.github.app is configured")
	}

	// Check that secret_verification_result output is NOT in activation job
	if strings.Contains(lockStr, "secret_verification_result: ${{ steps.validate-secret.outputs.verification_result }}") {
		t.Error("Expected secret_verification_result output to NOT be in activation job when tools.github.app is configured")
	}
}

// TestSecretVerificationOutputSkippedWithImportedGitHubApp tests that the validate-secret step is
// not generated when tools.github.app is configured via an imported shared workflow.
func TestSecretVerificationOutputSkippedWithImportedGitHubApp(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-secret-skip-imported-app-*")

	// Create shared directory structure
	sharedDir := filepath.Join(tmpDir, ".github", "workflows", "shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatalf("Failed to create shared directory: %v", err)
	}

	// Create a shared workflow file with GitHub App configuration
	sharedContent := `---
tools:
  github:
    app:
      app-id: ${{ vars.APP_ID }}
      private-key: ${{ secrets.APP_PRIVATE_KEY }}
---

# Shared GitHub App Configuration
`
	sharedFile := filepath.Join(sharedDir, "github-mcp-app.md")
	if err := os.WriteFile(sharedFile, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared workflow file: %v", err)
	}

	// Create main workflow that imports the shared app configuration
	mainContent := `---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
imports:
  - shared/github-mcp-app.md
tools:
  github:
    toolsets: [repos]
---

Test workflow with imported GitHub App authentication`

	workflowFile := filepath.Join(tmpDir, ".github", "workflows", "main.md")
	if err := os.WriteFile(workflowFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main workflow file: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := stringutil.MarkdownToLockFile(workflowFile)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Check that validate-secret step is NOT generated (app info was imported)
	if strings.Contains(lockStr, "id: validate-secret") {
		t.Error("Expected validate-secret step to NOT be generated when tools.github.app is configured via import")
	}

	// Check that secret_verification_result output is NOT in activation job
	if strings.Contains(lockStr, "secret_verification_result: ${{ steps.validate-secret.outputs.verification_result }}") {
		t.Error("Expected secret_verification_result output to NOT be in activation job when tools.github.app is imported")
	}

	// Verify the GitHub App token minting step IS still generated
	if !strings.Contains(lockStr, "id: github-mcp-app-token") {
		t.Error("Expected GitHub App token minting step to still be generated")
	}
}
