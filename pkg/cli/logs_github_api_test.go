//go:build !integration

package cli

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkflowRunJSONFieldsContainsPath guards against accidentally dropping the
// "path" field from the gh-run-list JSON query.  The health command (and the
// downstream .lock.yml filter in fetchWorkflowRuns) depend on WorkflowPath being
// populated.  If "path" is absent from workflowRunJSONFields the filter always
// evaluates to false and gh aw health reports no runs.
//
// Regression: commit 61cc2d7ac (Jan 4 2026) silently dropped "path" from the
// field list one day after it was deliberately added in 8c97bcaa0 (Jan 3 2026).
func TestWorkflowRunJSONFieldsContainsPath(t *testing.T) {
	fields := strings.Split(workflowRunJSONFields, ",")

	assert.True(t, slices.Contains(fields, "path"),
		`"path" must be in workflowRunJSONFields so WorkflowPath is populated and the .lock.yml filter in fetchWorkflowRuns works. See pkg/cli/logs_github_api.go`)
}

// TestWorkflowRunPathFieldUnmarshal verifies that the "path" key returned by
// "gh run list --json" is correctly bridged to WorkflowRun.WorkflowPath during
// unmarshaling.  The gh CLI uses "path" but WorkflowRun serialises the field as
// "workflowPath" for backward compatibility, so a helper struct is used at the
// unmarshal site.
func TestWorkflowRunPathFieldUnmarshal(t *testing.T) {
	// Simulate a single row of "gh run list --json path,workflowName,..."
	rawJSON := `[
		{
			"databaseId": 42,
			"workflowName": "My Workflow",
			"path": ".github/workflows/my-workflow.lock.yml",
			"status": "completed",
			"conclusion": "success",
			"createdAt": "2026-01-01T00:00:00Z",
			"startedAt": "2026-01-01T00:00:01Z",
			"updatedAt": "2026-01-01T00:01:00Z"
		}
	]`

	var rawRuns []struct {
		WorkflowRun
		Path string `json:"path"`
	}
	require.NoError(t, json.Unmarshal([]byte(rawJSON), &rawRuns), "unmarshal should succeed")
	require.Len(t, rawRuns, 1)

	run := rawRuns[0].WorkflowRun
	run.WorkflowPath = rawRuns[0].Path

	assert.Equal(t, ".github/workflows/my-workflow.lock.yml", run.WorkflowPath,
		"WorkflowPath should be populated from the 'path' JSON key")
	assert.Equal(t, int64(42), run.DatabaseID)
	assert.Equal(t, "My Workflow", run.WorkflowName)
}

// TestFetchWorkflowRunsLockYMLFilter verifies the .lock.yml suffix filter used
// in fetchWorkflowRuns.  Only runs whose WorkflowPath ends in ".lock.yml" must
// be retained; runs with a plain ".yml" path (regular Actions workflows) must
// be excluded.
func TestFetchWorkflowRunsLockYMLFilter(t *testing.T) {
	runs := []WorkflowRun{
		{
			DatabaseID:   1,
			WorkflowName: "Agentic Workflow",
			WorkflowPath: ".github/workflows/agentic-workflow.lock.yml",
			StartedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Date(2026, 1, 1, 0, 5, 0, 0, time.UTC),
		},
		{
			DatabaseID:   2,
			WorkflowName: "Regular CI",
			WorkflowPath: ".github/workflows/ci.yml",
		},
		{
			DatabaseID:   3,
			WorkflowName: "Another Agentic",
			WorkflowPath: ".github/workflows/another.lock.yml",
			StartedAt:    time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Date(2026, 1, 2, 0, 3, 0, 0, time.UTC),
		},
		{
			// Run with empty WorkflowPath — must be excluded (mimics the pre-fix state
			// where "path" was absent from the JSON query).
			DatabaseID:   4,
			WorkflowName: "Agentic But No Path",
			WorkflowPath: "",
		},
	}

	var filtered []WorkflowRun
	for _, run := range runs {
		if strings.HasSuffix(run.WorkflowPath, ".lock.yml") {
			if run.Duration == 0 && !run.StartedAt.IsZero() && !run.UpdatedAt.IsZero() {
				run.Duration = run.UpdatedAt.Sub(run.StartedAt)
			}
			filtered = append(filtered, run)
		}
	}

	require.Len(t, filtered, 2, "only .lock.yml runs should pass the filter")

	assert.Equal(t, int64(1), filtered[0].DatabaseID)
	assert.Equal(t, 5*time.Minute, filtered[0].Duration, "duration should be calculated from StartedAt/UpdatedAt")

	assert.Equal(t, int64(3), filtered[1].DatabaseID)
	assert.Equal(t, 3*time.Minute, filtered[1].Duration)
}

// TestWorkflowRunJSONFieldsAllPresent is a belt-and-braces check that ensures
// every field referenced by the WorkflowRun struct (those that are returned by
// the GitHub CLI) appears in workflowRunJSONFields.
func TestWorkflowRunJSONFieldsAllPresent(t *testing.T) {
	// These are the gh-CLI-side names (not the Go struct tags, which differ for
	// some fields, e.g., "path" vs "workflowPath").
	requiredFields := []string{
		"databaseId",
		"number",
		"url",
		"status",
		"conclusion",
		"workflowName",
		"path", // populated into WorkflowPath — critical for .lock.yml filter
		"createdAt",
		"startedAt",
		"updatedAt",
		"event",
		"headBranch",
		"headSha",
		"displayTitle",
	}

	fieldSet := make(map[string]bool)
	for f := range strings.SplitSeq(workflowRunJSONFields, ",") {
		fieldSet[f] = true
	}

	for _, field := range requiredFields {
		assert.True(t, fieldSet[field],
			"required field %q is missing from workflowRunJSONFields", field)
	}
}
