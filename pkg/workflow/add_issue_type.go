package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var addIssueTypeLog = logger.New("workflow:add_issue_type")

// AddIssueTypeConfig holds configuration for setting the issue type on issues from agent output
type AddIssueTypeConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	Allowed                []string `yaml:"allowed,omitempty"` // Optional list of allowed issue type names. If omitted, any issue type is allowed.
}

// parseAddIssueTypeConfig handles add-issue-type configuration
func (c *Compiler) parseAddIssueTypeConfig(outputMap map[string]any) *AddIssueTypeConfig {
	// Check if the key exists
	if _, exists := outputMap["add-issue-type"]; !exists {
		return nil
	}

	addIssueTypeLog.Print("Parsing add-issue-type configuration")

	// Unmarshal into typed config struct
	var config AddIssueTypeConfig
	if err := unmarshalConfig(outputMap, "add-issue-type", &config, addIssueTypeLog); err != nil {
		addIssueTypeLog.Printf("Failed to unmarshal config: %v", err)
		// Handle null case: create empty config (allows any issue type)
		addIssueTypeLog.Print("Using empty configuration (allows any issue type)")
		return &AddIssueTypeConfig{}
	}

	addIssueTypeLog.Printf("Parsed configuration: allowed_count=%d, target=%s", len(config.Allowed), config.Target)

	return &config
}
