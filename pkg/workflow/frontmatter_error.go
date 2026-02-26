package workflow

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var frontmatterErrorLog = logger.New("workflow:frontmatter_error")

// Package-level compiled regex patterns for better performance
var (
	lineColPattern       = regexp.MustCompile(`\[(\d+):(\d+)\]\s*(.+)`)
	sourceContextPattern = regexp.MustCompile(`\n(\s+\d+\s*\|)`)
)

// yamlErrorTranslations maps raw goccy/go-yaml internal messages to user-friendly plain English.
// These messages are parser internals that are not helpful to end users.
// Patterns must match actual strings produced by goccy/go-yaml v1.19+; both singular and
// legacy plural forms are kept for broad compatibility.
var yamlErrorTranslations = []struct {
	pattern     string
	translation string
}{
	// Colon in wrong context (actual goccy v1.19 message uses singular "value")
	{
		"mapping value is not allowed in this context",
		"Invalid YAML syntax: unexpected ':' — check indentation or key syntax",
	},
	// Legacy plural form kept for tests and older goccy versions
	{
		"mapping values are not allowed",
		"Invalid YAML syntax: unexpected ':' — check your indentation",
	},
	// Bare key without colon OR list item in mapping context
	{
		"non-map value is specified",
		"Invalid YAML syntax: expected 'key: value' format (did you forget a colon after the key?)",
	},
	// Plain word without colon (e.g. "engine copilot")
	{
		"unexpected key name",
		"Invalid YAML syntax: expected 'key: value' format (did you forget a colon after the key?)",
	},
	// Generic "did not find expected" catch-all (kept for backward compatibility)
	{
		"did not find expected",
		"Invalid YAML syntax: check indentation or missing key",
	},
	// Tab character errors; goccy v1.19 uses an actual tab char (0x09) inside single quotes
	{
		"found a tab character where an indentation space is expected",
		"Invalid YAML syntax: use spaces for indentation, not tabs",
	},
	{
		"tab character cannot use as a map key",
		"Invalid YAML syntax: use spaces for indentation, not tabs",
	},
	// The full goccy message uses an actual tab character (0x09) inside single quotes
	{
		"found character '\t' that cannot start any token",
		"Invalid YAML syntax: use spaces for indentation, not tabs",
	},
	// List item '-' in wrong context
	{
		"block sequence entries are not allowed",
		"Invalid YAML syntax: unexpected list item '-' — check indentation",
	},
	// Unclosed sequences/brackets
	{
		"sequence end token ']' not found",
		"Invalid YAML syntax: unclosed bracket — add ']' to close the list",
	},
	// Unclosed string quotes
	{
		"could not find end character of double-quoted text",
		`Invalid YAML syntax: unclosed double quote — add '"' to close the string`,
	},
	{
		"could not find end character of single-quoted text",
		"Invalid YAML syntax: unclosed single quote — add \"'\" to close the string",
	},
}

// translateYAMLMessage converts raw YAML parser messages to user-friendly plain English.
// This prevents internal library jargon from reaching the end user.
func translateYAMLMessage(message string) string {
	for _, t := range yamlErrorTranslations {
		if strings.Contains(message, t.pattern) {
			return t.translation
		}
	}
	return message
}

// createFrontmatterError creates a detailed error for frontmatter parsing issues
// frontmatterLineOffset is the line number where the frontmatter content begins (1-based)
// Returns error in VSCode-compatible format: filename:line:column: error message
func (c *Compiler) createFrontmatterError(filePath, content string, err error, frontmatterLineOffset int) error {
	frontmatterErrorLog.Printf("Creating frontmatter error for file: %s, offset: %d", filePath, frontmatterLineOffset)

	errorStr := err.Error()

	// Check if error already contains formatted yaml.FormatError() output with source context
	// yaml.FormatError() produces output like "failed to parse frontmatter:\n[line:col] message\n>  line | content..."
	if strings.Contains(errorStr, "failed to parse frontmatter:\n[") && (strings.Contains(errorStr, "\n>") || strings.Contains(errorStr, "|")) {
		// Extract line and column from the formatted error for VSCode compatibility
		// Pattern: [line:col] message
		if matches := lineColPattern.FindStringSubmatch(errorStr); len(matches) >= 4 {
			line := matches[1]
			col := matches[2]
			message := matches[3]
			// Extract just the first line of the message (before newline)
			if idx := strings.Index(message, "\n"); idx != -1 {
				message = message[:idx]
			}
			// Translate raw YAML parser messages to user-friendly plain English
			message = translateYAMLMessage(message)

			// Format as: filename:line:column: error: message
			// This is compatible with VSCode's problem matcher
			vscodeFormat := fmt.Sprintf("%s:%s:%s: error: %s", filePath, line, col, message)

			// Extract just the source context lines (skip the [line:col] message line to avoid duplication)
			// Find the first line that starts with whitespace + digit + | (source context line)
			if loc := sourceContextPattern.FindStringIndex(errorStr); loc != nil {
				// Extract from the first source context line to the end
				context := errorStr[loc[0]+1:] // +1 to skip the leading newline
				// Return VSCode-compatible format on first line, followed by source context only
				frontmatterErrorLog.Print("Formatting error for VSCode compatibility")
				return fmt.Errorf("%s\n%s", vscodeFormat, context)
			}

			// If we can't extract source context, return just the VSCode format
			return fmt.Errorf("%s", vscodeFormat)
		}

		// Fallback if we can't parse the line/col
		frontmatterErrorLog.Print("Could not extract line/col from formatted error")
		return fmt.Errorf("%s: %w", filePath, err)
	}

	// Fallback: if not already formatted, return with filename prefix
	frontmatterErrorLog.Printf("Using fallback error message: %v", err)
	return fmt.Errorf("%s: failed to extract frontmatter: %w", filePath, err)
}
