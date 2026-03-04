//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInlinedImports_FrontmatterHashInline_SameBodySameHash verifies determinism.
func TestInlinedImports_FrontmatterHashInline_SameBodySameHash(t *testing.T) {
	tmpDir := t.TempDir()
	content := `---
name: test
on:
  workflow_dispatch:
inlined-imports: true
engine: copilot
---

# Same body content
`
	file1 := filepath.Join(tmpDir, "a.md")
	file2 := filepath.Join(tmpDir, "b.md")
	require.NoError(t, os.WriteFile(file1, []byte(content), 0o644))
	require.NoError(t, os.WriteFile(file2, []byte(content), 0o644))

	cache := parser.NewImportCache(tmpDir)
	hash1, err := parser.ComputeFrontmatterHashFromFile(file1, cache)
	require.NoError(t, err)
	hash2, err := parser.ComputeFrontmatterHashFromFile(file2, cache)
	require.NoError(t, err)

	assert.Equal(t, hash1, hash2, "same content should produce the same hash")
}

// TestInlinedImports_AgentFileLocalWorks verifies that when inlined-imports: true and a local
// agent file is imported, ParseWorkflowFile succeeds. Local agent imports are treated like
// snippets (runtime-import path) and their content is inlined at compile time.
func TestInlinedImports_AgentFileLocalWorks(t *testing.T) {
	tmpDir := t.TempDir()

	// Create the .github/agents directory and agent file
	agentsDir := filepath.Join(tmpDir, ".github", "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0o755))
	agentFile := filepath.Join(agentsDir, "my-agent.md")
	require.NoError(t, os.WriteFile(agentFile, []byte("# Agent\nDo things.\n"), 0o644))

	// Create the workflow file with inlined-imports: true importing the agent file
	workflowDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowDir, 0o755))
	workflowFile := filepath.Join(workflowDir, "test-workflow.md")
	workflowContent := `---
name: inlined-agent-test
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
inlined-imports: true
imports:
  - ../../.github/agents/my-agent.md
---

# Main Workflow

Do something.
`
	require.NoError(t, os.WriteFile(workflowFile, []byte(workflowContent), 0o644))

	compiler := NewCompiler(
		WithNoEmit(true),
		WithSkipValidation(true),
	)

	// Local agent import + inlined-imports: true should succeed now
	// (local agents are treated like snippets, not the special AGENT_CONTENT path)
	_, err := compiler.ParseWorkflowFile(workflowFile)
	require.NoError(t, err, "local agent import with inlined-imports should succeed")
}

// TestInlinedImports_AgentFileCleared verifies that buildInitialWorkflowData clears the AgentFile
// field when inlined-imports is true. This simulates a remote agent import scenario
// (local imports no longer set AgentFile at all). For remote agent imports, ParseWorkflowFile
// would error before this state is used in production (the inlined-imports + remote agent check
// at compiler_orchestrator_workflow.go fires first).
func TestInlinedImports_AgentFileCleared(t *testing.T) {
	compiler := NewCompiler()

	frontmatterResult := &parser.FrontmatterResult{
		Frontmatter: map[string]any{
			"name":            "agent-test",
			"engine":          "copilot",
			"inlined-imports": true,
		},
		FrontmatterLines: []string{
			"name: agent-test",
			"engine: copilot",
			"inlined-imports: true",
		},
	}

	toolsResult := &toolsProcessingResult{
		workflowName:         "agent-test",
		frontmatterName:      "agent-test",
		parsedFrontmatter:    &FrontmatterConfig{Name: "agent-test", Engine: "copilot", InlinedImports: true},
		tools:                map[string]any{},
		importPaths:          []string{".github/agents/my-agent.md"},
		mainWorkflowMarkdown: "# Main",
	}

	engineSetup := &engineSetupResult{
		engineSetting: "copilot",
		engineConfig:  &EngineConfig{ID: "copilot"},
		sandboxConfig: &SandboxConfig{},
	}

	importsResult := &parser.ImportsResult{
		AgentFile:       ".github/agents/my-agent.md",
		AgentImportSpec: ".github/agents/my-agent.md",
	}

	wd := compiler.buildInitialWorkflowData(frontmatterResult, toolsResult, engineSetup, importsResult)

	// InlinedImports should be true in WorkflowData
	assert.True(t, wd.InlinedImports, "InlinedImports should be true in WorkflowData")

	// AgentFile should be cleared (content inlined via ImportPaths instead)
	assert.Empty(t, wd.AgentFile, "AgentFile should be cleared when inlined-imports is true")
	assert.Empty(t, wd.AgentImportSpec, "AgentImportSpec should be cleared when inlined-imports is true")
}
