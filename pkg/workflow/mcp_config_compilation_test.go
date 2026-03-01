//go:build !integration

package workflow

import (
	"os"
	"strings"
	"testing"
)

// TestMCPServersCompilation verifies that mcp-servers configuration is properly compiled into workflows
// TestMCPEnvVarsAlphabeticallySorted verifies that env vars in MCP configs are sorted alphabetically
func TestMCPEnvVarsAlphabeticallySorted(t *testing.T) {
	// Create a temporary markdown file with mcp-servers configuration containing env vars
	workflowContent := `---
on:
  workflow_dispatch:
strict: false
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
mcp-servers:
  test-server:
    container: example/test:latest
    env:
      ZEBRA_VAR: "z"
      ALPHA_VAR: "a"
      BETA_VAR: "b"
---

# Test MCP Env Var Sorting

This workflow tests that MCP server env vars are sorted alphabetically.
`

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "test-env-sort-*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write content to file
	if _, err := tmpFile.WriteString(workflowContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Create compiler and compile workflow
	compiler := NewCompiler()
	compiler.SetSkipValidation(true)

	// Generate YAML
	workflowData, err := compiler.ParseWorkflowFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	yamlContent, err := compiler.generateYAML(workflowData, tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Find the test-server env section in the generated YAML
	// Look for "test-server" first, then find the env section after it
	testServerIndex := strings.Index(yamlContent, `"test-server"`)
	if testServerIndex == -1 {
		t.Fatalf("Could not find test-server section in generated YAML")
	}

	// Find env section after test-server
	envIndex := strings.Index(yamlContent[testServerIndex:], `"env": {`)
	if envIndex == -1 {
		t.Fatalf("Could not find env section for test-server in generated YAML")
	}

	// Adjust envIndex to be relative to the full yamlContent
	envIndex += testServerIndex

	// Extract a portion of YAML starting from env section (next 300 chars should be enough)
	envSection := yamlContent[envIndex : envIndex+300]

	// Verify that ALPHA_VAR appears before BETA_VAR and ZEBRA_VAR
	alphaIndex := strings.Index(envSection, `"ALPHA_VAR"`)
	betaIndex := strings.Index(envSection, `"BETA_VAR"`)
	zebraIndex := strings.Index(envSection, `"ZEBRA_VAR"`)

	if alphaIndex == -1 || betaIndex == -1 || zebraIndex == -1 {
		t.Fatalf("Could not find all env vars in generated YAML. Section: %s", envSection)
	}

	// Verify alphabetical order
	if alphaIndex >= betaIndex {
		t.Errorf("Expected ALPHA_VAR to appear before BETA_VAR, but ALPHA_VAR is at %d and BETA_VAR is at %d", alphaIndex, betaIndex)
	}
	if betaIndex >= zebraIndex {
		t.Errorf("Expected BETA_VAR to appear before ZEBRA_VAR, but BETA_VAR is at %d and ZEBRA_VAR is at %d", betaIndex, zebraIndex)
	}
	if alphaIndex >= zebraIndex {
		t.Errorf("Expected ALPHA_VAR to appear before ZEBRA_VAR, but ALPHA_VAR is at %d and ZEBRA_VAR is at %d", alphaIndex, zebraIndex)
	}
}

// TestHasMCPConfigDetection verifies that hasMCPConfig properly detects MCP configurations
func TestHasMCPConfigDetection(t *testing.T) {
	testCases := []struct {
		name     string
		config   map[string]any
		expected bool
		mcpType  string
	}{
		{
			name: "explicit stdio type",
			config: map[string]any{
				"type":    "stdio",
				"command": "npx",
			},
			expected: true,
			mcpType:  "stdio",
		},
		{
			name: "explicit http type",
			config: map[string]any{
				"type": "http",
				"url":  "https://example.com",
			},
			expected: true,
			mcpType:  "http",
		},
		{
			name: "inferred stdio from command",
			config: map[string]any{
				"command": "npx",
				"args":    []any{"-y", "@microsoft/markitdown"},
			},
			expected: true,
			mcpType:  "stdio",
		},
		{
			name: "inferred http from url",
			config: map[string]any{
				"url": "https://example.com/mcp",
			},
			expected: true,
			mcpType:  "http",
		},
		{
			name: "inferred stdio from container",
			config: map[string]any{
				"container": "example/mcp:latest",
			},
			expected: true,
			mcpType:  "stdio",
		},
		{
			name: "not MCP config",
			config: map[string]any{
				"allowed": []any{"some_tool"},
			},
			expected: false,
			mcpType:  "",
		},
		{
			name: "markitdown-like config",
			config: map[string]any{
				"registry": "https://api.mcp.github.com/v0/servers/microsoft/markitdown",
				"command":  "npx",
				"args":     []any{"-y", "@microsoft/markitdown"},
			},
			expected: true,
			mcpType:  "stdio",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hasMcp, mcpType := hasMCPConfig(tc.config)
			if hasMcp != tc.expected {
				t.Errorf("Expected hasMCPConfig to return %v, got %v", tc.expected, hasMcp)
			}
			if mcpType != tc.mcpType {
				t.Errorf("Expected MCP type %q, got %q", tc.mcpType, mcpType)
			}
		})
	}
}

// TestDevModeAgenticWorkflowsContainer verifies that the agentic-workflows MCP server
// uses the locally built Docker image in dev mode instead of alpine:latest
func TestDevModeAgenticWorkflowsContainer(t *testing.T) {
	tests := []struct {
		name              string
		actionMode        ActionMode
		expectedContainer string
	}{
		{
			name:              "dev mode uses local image",
			actionMode:        ActionModeDev,
			expectedContainer: "localhost/gh-aw:dev",
		},
		{
			name:              "release mode uses alpine",
			actionMode:        ActionModeRelease,
			expectedContainer: "alpine:latest",
		},
		{
			name:              "script mode uses alpine",
			actionMode:        ActionModeScript,
			expectedContainer: "alpine:latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a workflow with agentic-workflows tool
			workflowContent := `---
on:
  workflow_dispatch:
strict: false
permissions:
  contents: read
  actions: read
engine: copilot
tools:
  agentic-workflows:
---

# Test Agentic Workflows Dev Mode

This workflow tests that agentic-workflows uses the correct container in dev mode.
`

			// Create temporary file
			tmpFile, err := os.CreateTemp("", "test-dev-mode-*.md")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			// Write content to file
			if _, err := tmpFile.WriteString(workflowContent); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			tmpFile.Close()

			// Compile the workflow with the specified action mode
			compiler := NewCompiler()
			compiler.SetActionMode(tt.actionMode)

			if err := compiler.CompileWorkflow(tmpFile.Name()); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the compiled lock file
			lockFile := strings.TrimSuffix(tmpFile.Name(), ".md") + ".lock.yml"
			defer os.Remove(lockFile)

			lockContent, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			// Check that the container image is correct
			if !strings.Contains(string(lockContent), `"container": "`+tt.expectedContainer+`"`) {
				t.Errorf("Expected container %q in lock file, but not found. Lock file content:\n%s",
					tt.expectedContainer, string(lockContent))
			}

			// In dev mode, verify dev-specific configuration
			if tt.actionMode == ActionModeDev {
				// Check for build steps
				requiredSteps := []string{
					"Setup Go for CLI build",
					"Build gh-aw CLI",
					"Setup Docker Buildx",
					"Build gh-aw Docker image",
					"tags: localhost/gh-aw:dev",
				}

				for _, step := range requiredSteps {
					if !strings.Contains(string(lockContent), step) {
						t.Errorf("Expected build step %q in dev mode lock file, but not found", step)
					}
				}

				// Verify binary copy step for user-defined steps that execute ./gh-aw
				if !strings.Contains(string(lockContent), "cp dist/gh-aw-linux-amd64 ./gh-aw") {
					t.Error("Expected 'cp dist/gh-aw-linux-amd64 ./gh-aw' in dev mode build step")
				}
				if !strings.Contains(string(lockContent), "chmod +x ./gh-aw") {
					t.Error("Expected 'chmod +x ./gh-aw' in dev mode build step")
				}

				// Verify NO entrypoint field (uses container's default ENTRYPOINT)
				if strings.Contains(string(lockContent), `"entrypoint"`) {
					t.Error("Did not expect entrypoint field in dev mode (uses container's ENTRYPOINT)")
				}

				// Verify NO entrypointArgs field (uses container's default CMD)
				if strings.Contains(string(lockContent), `"entrypointArgs"`) {
					t.Error("Did not expect entrypointArgs field in dev mode (uses container's CMD)")
				}

				// Verify no --cmd argument
				if strings.Contains(string(lockContent), `"--cmd"`) {
					t.Error("Did not expect --cmd argument in dev mode")
				}

				// Verify binary mounts are NOT present in dev mode
				if strings.Contains(string(lockContent), `/opt/gh-aw:/opt/gh-aw:ro`) {
					t.Error("Did not expect /opt/gh-aw mount in dev mode (binary is in image)")
				}
				if strings.Contains(string(lockContent), `/usr/bin/gh:/usr/bin/gh:ro`) {
					t.Error("Did not expect /usr/bin/gh mount in dev mode (gh CLI is in image)")
				}

				// Verify DEBUG and GITHUB_TOKEN are present
				if !strings.Contains(string(lockContent), `"DEBUG": "*"`) {
					t.Error("Expected DEBUG set to literal '*' in dev mode env vars")
				}
				if !strings.Contains(string(lockContent), `"GITHUB_TOKEN"`) {
					t.Error("Expected GITHUB_TOKEN in dev mode env vars")
				}

				// Verify working directory args are present
				if !strings.Contains(string(lockContent), `"args": ["--network", "host", "-w", "\${GITHUB_WORKSPACE}"]`) {
					t.Error("Expected args with network access and working directory in dev mode")
				}
			}
		})
	}
}

// TestReleaseModeGhAwInstallStepWithCustomSteps verifies that in release mode, when the
// agentic-workflows tool is enabled and user-defined steps are present, a "Install gh-aw CLI"
// step is generated before user steps to make ./gh-aw available.
func TestReleaseModeGhAwInstallStepWithCustomSteps(t *testing.T) {
	tests := []struct {
		name               string
		actionMode         ActionMode
		hasCustomSteps     bool
		expectInstallStep  bool
		expectDevBuildStep bool
	}{
		{
			name:               "release mode with custom steps generates install step",
			actionMode:         ActionModeRelease,
			hasCustomSteps:     true,
			expectInstallStep:  true,
			expectDevBuildStep: false,
		},
		{
			name:               "release mode without custom steps does not generate install step",
			actionMode:         ActionModeRelease,
			hasCustomSteps:     false,
			expectInstallStep:  false,
			expectDevBuildStep: false,
		},
		{
			name:               "dev mode with custom steps generates dev build steps not install step",
			actionMode:         ActionModeDev,
			hasCustomSteps:     true,
			expectInstallStep:  false,
			expectDevBuildStep: true,
		},
		{
			name:               "script mode with custom steps generates install step",
			actionMode:         ActionModeScript,
			hasCustomSteps:     true,
			expectInstallStep:  true,
			expectDevBuildStep: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stepsSection := ""
			if tt.hasCustomSteps {
				stepsSection = `steps:
  - name: Download logs
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      ./gh-aw logs --start-date -30d -o /tmp/logs --json > /tmp/logs/summary.json
`
			}

			workflowContent := `---
on:
  workflow_dispatch:
strict: false
permissions:
  contents: read
  actions: read
engine: copilot
tools:
  agentic-workflows:
` + stepsSection + `---

# Test Release Mode Install Step

This workflow tests the release mode gh-aw install step.
`

			tmpFile, err := os.CreateTemp("", "test-release-install-*.md")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(workflowContent); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			tmpFile.Close()

			compiler := NewCompiler()
			compiler.SetActionMode(tt.actionMode)

			if err := compiler.CompileWorkflow(tmpFile.Name()); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			lockFile := strings.TrimSuffix(tmpFile.Name(), ".md") + ".lock.yml"
			defer os.Remove(lockFile)

			lockContent, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			content := string(lockContent)

			if tt.expectInstallStep {
				if !strings.Contains(content, "name: Install gh-aw CLI") {
					t.Error("Expected 'Install gh-aw CLI' step in release mode with custom steps, but not found")
				}
				if !strings.Contains(content, "gh extension install github/gh-aw") {
					t.Error("Expected 'gh extension install github/gh-aw' in install step")
				}
				if !strings.Contains(content, "cp \"$GH_AW_BIN\" ./gh-aw") {
					t.Error("Expected binary copy to ./gh-aw in install step")
				}
				// Verify install step appears before user steps
				installIdx := strings.Index(content, "name: Install gh-aw CLI")
				userStepIdx := strings.Index(content, "name: Download logs")
				if installIdx < 0 {
					t.Error("Expected 'Install gh-aw CLI' step in lock file, but not found")
				} else if userStepIdx > 0 && installIdx > userStepIdx {
					t.Error("Expected 'Install gh-aw CLI' step to appear before user-defined steps")
				}
			} else {
				if strings.Contains(content, "name: Install gh-aw CLI") {
					t.Error("Did not expect 'Install gh-aw CLI' step, but found it")
				}
			}

			if tt.expectDevBuildStep {
				if !strings.Contains(content, "name: Build gh-aw CLI") {
					t.Error("Expected 'Build gh-aw CLI' dev mode step, but not found")
				}
			} else {
				if strings.Contains(content, "name: Build gh-aw CLI") {
					t.Error("Did not expect 'Build gh-aw CLI' dev mode step, but found it")
				}
			}
		})
	}
}
