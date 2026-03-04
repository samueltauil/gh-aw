package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var autofixCodeScanningAlertLog = logger.New("workflow:autofix_code_scanning")

// AutofixCodeScanningAlertConfig holds configuration for adding autofixes to code scanning alerts
type AutofixCodeScanningAlertConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
}

// parseAutofixCodeScanningAlertConfig handles autofix-code-scanning-alert configuration
func (c *Compiler) parseAutofixCodeScanningAlertConfig(outputMap map[string]any) *AutofixCodeScanningAlertConfig {
	if configData, exists := outputMap["autofix-code-scanning-alert"]; exists {
		autofixCodeScanningAlertLog.Print("Parsing autofix-code-scanning-alert configuration")
		addCodeScanningAutofixConfig := &AutofixCodeScanningAlertConfig{}
		addCodeScanningAutofixConfig.Max = defaultIntStr(1) // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse common base fields with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &addCodeScanningAutofixConfig.BaseSafeOutputConfig, 1)
		}

		return addCodeScanningAutofixConfig
	}

	return nil
}
