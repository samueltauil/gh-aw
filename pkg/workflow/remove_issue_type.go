package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var removeIssueTypeLog = logger.New("workflow:remove_issue_type")

// RemoveIssueTypeConfig holds configuration for removing the issue type from issues from agent output
type RemoveIssueTypeConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
}

// parseRemoveIssueTypeConfig handles remove-issue-type configuration
func (c *Compiler) parseRemoveIssueTypeConfig(outputMap map[string]any) *RemoveIssueTypeConfig {
	// Check if the key exists
	if _, exists := outputMap["remove-issue-type"]; !exists {
		return nil
	}

	removeIssueTypeLog.Print("Parsing remove-issue-type configuration")

	// Unmarshal into typed config struct
	var config RemoveIssueTypeConfig
	if err := unmarshalConfig(outputMap, "remove-issue-type", &config, removeIssueTypeLog); err != nil {
		removeIssueTypeLog.Printf("Failed to unmarshal config: %v", err)
		// Handle null case: create empty config
		removeIssueTypeLog.Print("Using empty configuration")
		return &RemoveIssueTypeConfig{}
	}

	removeIssueTypeLog.Printf("Parsed configuration: target=%s", config.Target)

	return &config
}
