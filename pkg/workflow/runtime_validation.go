// This file provides runtime validation for packages, containers, and expressions.
//
// # Runtime Validation
//
// This file validates runtime dependencies and configuration for agentic workflows.
// It ensures that:
//   - Container images exist and are accessible
//   - Runtime packages (npm, pip, uv) are available
//   - Expression sizes don't exceed GitHub Actions limits
//
// # Validation Functions
//
//   - validateExpressionSizes() - Validates expression size limits (21KB max)
//   - validateContainerImages() - Validates Docker images exist
//   - validateRuntimePackages() - Validates npm, pip, uv packages
//
// # Validation Patterns
//
// This file uses several patterns:
//   - External resource validation: Docker images, npm/pip packages
//   - Size limit validation: Expression sizes, file sizes
//   - Collection and deduplication: Package extraction
//
// # Size Limits
//
// GitHub Actions has a 21KB limit for expression values including environment variables.
// This validation prevents compilation of workflows that will fail at runtime.
//
// # When to Add Validation Here
//
// Add validation to this file when:
//   - It validates runtime dependencies (packages, containers)
//   - It checks expression or content size limits
//   - It requires external resource checking
//
// For general validation, see validation.go.
// For detailed documentation, see scratchpad/validation-architecture.md

package workflow

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
)

var runtimeValidationLog = logger.New("workflow:runtime_validation")

// validateExpressionSizes validates that no expression values in the generated YAML exceed GitHub Actions limits
func (c *Compiler) validateExpressionSizes(yamlContent string) error {
	lines := strings.Split(yamlContent, "\n")
	runtimeValidationLog.Printf("Validating expression sizes: yaml_lines=%d, max_size=%d", len(lines), MaxExpressionSize)
	maxSize := MaxExpressionSize

	for lineNum, line := range lines {
		// Check the line length (actual content that will be in the YAML)
		if len(line) > maxSize {
			// Extract the key/value for better error message
			trimmed := strings.TrimSpace(line)
			key := ""
			if colonIdx := strings.Index(trimmed, ":"); colonIdx > 0 {
				key = strings.TrimSpace(trimmed[:colonIdx])
			}

			// Format sizes for display
			actualSize := console.FormatFileSize(int64(len(line)))
			maxSizeFormatted := console.FormatFileSize(int64(maxSize))

			var errorMsg string
			if key != "" {
				errorMsg = fmt.Sprintf("expression value for %q (%s) exceeds maximum allowed size (%s) at line %d. GitHub Actions has a 21KB limit for expression values including environment variables. Consider chunking the content or using artifacts instead.",
					key, actualSize, maxSizeFormatted, lineNum+1)
			} else {
				errorMsg = fmt.Sprintf("line %d (%s) exceeds maximum allowed expression size (%s). GitHub Actions has a 21KB limit for expression values.",
					lineNum+1, actualSize, maxSizeFormatted)
			}

			return errors.New(errorMsg)
		}
	}

	return nil
}

// validateContainerImages validates that container images specified in MCP configs exist and are accessible
func (c *Compiler) validateContainerImages(workflowData *WorkflowData) error {
	if workflowData.Tools == nil {
		runtimeValidationLog.Print("No tools configured, skipping container validation")
		return nil
	}

	runtimeValidationLog.Printf("Validating container images for %d tools", len(workflowData.Tools))
	collector := NewErrorCollector(false)
	for toolName, toolConfig := range workflowData.Tools {
		if config, ok := toolConfig.(map[string]any); ok {
			// Get the MCP configuration to extract container info
			mcpConfig, err := getMCPConfig(config, toolName)
			if err != nil {
				// If we can't parse the MCP config, skip validation (will be caught elsewhere)
				continue
			}

			// Check if this tool originally had a container field (before transformation)
			if containerName, hasContainer := config["container"]; hasContainer && mcpConfig.Type == "stdio" {
				// Build the full container image name with version
				containerStr, ok := containerName.(string)
				if !ok {
					continue
				}

				containerImage := containerStr
				if version, hasVersion := config["version"]; hasVersion {
					if versionStr, ok := version.(string); ok && versionStr != "" {
						containerImage = containerImage + ":" + versionStr
					}
				}

				// Validate the container image exists using docker
				if err := validateDockerImage(containerImage, c.verbose); err != nil {
					_ = collector.Add(fmt.Errorf("tool '%s': %w", toolName, err))
				} else if c.verbose {
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage("✓ Container image validated: "+containerImage))
				}
			}
		}
	}

	if collector.HasErrors() {
		return NewValidationError(
			"container.images",
			fmt.Sprintf("%d images failed validation", collector.Count()),
			"container image validation failed",
			fmt.Sprintf("Fix the following container image issues:\n\n%s\n\nEnsure:\n1. Container images exist and are accessible\n2. Registry URLs are correct\n3. Image tags are specified\n4. You have pull permissions for private images", collector.Error().Error()),
		)
	}

	runtimeValidationLog.Print("Container image validation passed")
	return nil
}

// validateRuntimePackages validates that packages required by npx, pip, and uv are available
func (c *Compiler) validateRuntimePackages(workflowData *WorkflowData) error {
	// Detect runtime requirements
	requirements := DetectRuntimeRequirements(workflowData)
	runtimeValidationLog.Printf("Validating runtime packages: found %d runtime requirements", len(requirements))

	collector := NewErrorCollector(false)
	for _, req := range requirements {
		switch req.Runtime.ID {
		case "node":
			// Validate npx packages used in the workflow
			runtimeValidationLog.Print("Validating npx packages")
			if err := c.validateNpxPackages(workflowData); err != nil {
				runtimeValidationLog.Printf("Npx package validation failed: %v", err)
				_ = collector.Add(err)
			}
		case "python":
			// Validate pip packages used in the workflow
			runtimeValidationLog.Print("Validating pip packages")
			if err := c.validatePipPackages(workflowData); err != nil {
				runtimeValidationLog.Printf("Pip package validation failed: %v", err)
				_ = collector.Add(err)
			}
		case "uv":
			// Validate uv packages used in the workflow
			runtimeValidationLog.Print("Validating uv packages")
			if err := c.validateUvPackages(workflowData); err != nil {
				runtimeValidationLog.Printf("Uv package validation failed: %v", err)
				_ = collector.Add(err)
			}
		}
	}

	if collector.HasErrors() {
		runtimeValidationLog.Printf("Runtime package validation completed with %d errors", collector.Count())
		return NewValidationError(
			"runtime.packages",
			fmt.Sprintf("%d package validation errors", collector.Count()),
			"runtime package validation failed",
			fmt.Sprintf("Fix the following package issues:\n\n%s\n\nEnsure:\n1. Package names are spelled correctly\n2. Packages exist in their respective registries (npm, PyPI)\n3. Package managers (npm, pip, uv) are installed\n4. Network access is available for registry checks", collector.Error().Error()),
		)
	}

	runtimeValidationLog.Print("Runtime package validation passed")
	return nil
}
