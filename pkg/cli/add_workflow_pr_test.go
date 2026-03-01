//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple workflow name",
			input:    "my-workflow",
			expected: "my-workflow",
		},
		{
			name:     "workflow with .md extension",
			input:    "my-workflow.md",
			expected: "my-workflow",
		},
		{
			name:     "full path",
			input:    ".github/workflows/my-workflow.md",
			expected: "my-workflow",
		},
		{
			name:     "path with spaces",
			input:    "my workflow.md",
			expected: "my-workflow",
		},
		{
			name:     "path with special chars",
			input:    "my:workflow?.md",
			expected: "my-workflow",
		},
		{
			name:     "path with dots",
			input:    "my..workflow.md",
			expected: "my-workflow",
		},
		{
			name:     "path with backslashes",
			input:    "path\\to\\workflow.md",
			expected: "path-to-workflow", // On Linux, backslashes are not path separators
		},
		{
			name:     "path with tilde",
			input:    "~my~workflow.md",
			expected: "my-workflow",
		},
		{
			name:     "path with caret",
			input:    "my^workflow.md",
			expected: "my-workflow",
		},
		{
			name:     "path with asterisk",
			input:    "my*workflow.md",
			expected: "my-workflow",
		},
		{
			name:     "path with brackets",
			input:    "my[workflow].md",
			expected: "my-workflow",
		},
		{
			name:     "path with at-brace",
			input:    "my@{workflow}.md",
			expected: "my-workflow",
		},
		{
			name:     "consecutive special chars",
			input:    "my---workflow.md",
			expected: "my-workflow",
		},
		{
			name:     "leading special chars",
			input:    "---my-workflow.md",
			expected: "my-workflow",
		},
		{
			name:     "trailing special chars",
			input:    "my-workflow---.md",
			expected: "my-workflow",
		},
		{
			name:     "empty after sanitization",
			input:    "....md",
			expected: "workflow",
		},
		{
			name:     "underscores preserved",
			input:    "my_workflow.md",
			expected: "my_workflow",
		},
		{
			name:     "numbers preserved",
			input:    "workflow123.md",
			expected: "workflow123",
		},
		{
			name:     "mixed case preserved",
			input:    "MyWorkflow.md",
			expected: "MyWorkflow",
		},
		{
			name:     "unicode characters replaced",
			input:    "workflow-日本語.md",
			expected: "workflow",
		},
		{
			name:     "emoji replaced",
			input:    "workflow-🚀-test.md",
			expected: "workflow-test",
		},
		{
			name:     "only special characters",
			input:    "!@#$%^&*()+=",
			expected: "workflow",
		},
		{
			name:     "only dots",
			input:    "...",
			expected: "workflow",
		},
		{
			name:     "only hyphens",
			input:    "---",
			expected: "workflow",
		},
		{
			name:     "very long string truncation behavior",
			input:    "this-is-a-very-long-workflow-name-that-exceeds-typical-branch-name-lengths.md",
			expected: "this-is-a-very-long-workflow-name-that-exceeds-typical-branch-name-lengths",
		},
		{
			name:     "spaces only",
			input:    "     ",
			expected: "workflow",
		},
		{
			name:     "control characters",
			input:    "work\tflow\nname",
			expected: "work-flow-name",
		},
		{
			name:     "null bytes",
			input:    "work\x00flow",
			expected: "work-flow",
		},
		{
			name:     "mixed unicode and ascii",
			input:    "test-αβγ-workflow.md",
			expected: "test-workflow",
		},
		{
			name:     "accented characters",
			input:    "café-workflow.md",
			expected: "caf-workflow",
		},
		{
			name:     "cyrillic characters",
			input:    "workflow-работа.md",
			expected: "workflow",
		},
		{
			name:     "chinese characters only",
			input:    "工作流程.md",
			expected: "workflow",
		},
		{
			name:     "path separators extracts basename",
			input:    "a/b\\c/d.md",
			expected: "d", // normalizeWorkflowID extracts base name
		},
		{
			name:     "question mark and asterisk",
			input:    "test?file*.md",
			expected: "test-file",
		},
		{
			name:     "colon for windows paths",
			input:    "C:\\Users\\test.md",
			expected: "C-Users-test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeBranchName(tt.input)
			assert.Equal(t, tt.expected, result, "sanitizeBranchName(%q) should return %q", tt.input, tt.expected)
		})
	}
}

func TestAddWizardResult(t *testing.T) {
	tests := []struct {
		name                string
		prNumber            int
		prURL               string
		hasWorkflowDispatch bool
	}{
		{
			name:                "default values",
			prNumber:            0,
			prURL:               "",
			hasWorkflowDispatch: false,
		},
		{
			name:                "with PR number",
			prNumber:            123,
			prURL:               "",
			hasWorkflowDispatch: false,
		},
		{
			name:                "with PR URL",
			prNumber:            0,
			prURL:               "https://github.com/owner/repo/pull/123",
			hasWorkflowDispatch: false,
		},
		{
			name:                "with workflow dispatch",
			prNumber:            0,
			prURL:               "",
			hasWorkflowDispatch: true,
		},
		{
			name:                "all fields set",
			prNumber:            456,
			prURL:               "https://github.com/owner/repo/pull/456",
			hasWorkflowDispatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &AddWizardResult{
				PRNumber:            tt.prNumber,
				PRURL:               tt.prURL,
				HasWorkflowDispatch: tt.hasWorkflowDispatch,
			}

			assert.Equal(t, tt.prNumber, result.PRNumber, "PRNumber should match")
			assert.Equal(t, tt.prURL, result.PRURL, "PRURL should match")
			assert.Equal(t, tt.hasWorkflowDispatch, result.HasWorkflowDispatch, "HasWorkflowDispatch should match")
		})
	}
}
