//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTranslateYAMLMessage tests that raw goccy/go-yaml error messages are translated to plain English
func TestTranslateYAMLMessage(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantNot []string // substrings that must NOT appear in output
		wantAny []string // at least one of these must appear in output
	}{
		{
			name:    "non-map value translated to user-friendly message",
			input:   "non-map value is specified",
			wantNot: []string{"non-map value is specified"},
			wantAny: []string{"Invalid YAML syntax", "key: value", "colon"},
		},
		{
			name:    "mapping values not allowed translated",
			input:   "mapping values are not allowed in this context",
			wantNot: []string{"mapping values are not allowed"},
			wantAny: []string{"Invalid YAML syntax", "indentation"},
		},
		{
			// Actual goccy v1.19 singular form
			name:    "mapping value (singular) not allowed translated",
			input:   "mapping value is not allowed in this context",
			wantNot: []string{"mapping value is not allowed"},
			wantAny: []string{"Invalid YAML syntax", "indentation"},
		},
		{
			// goccy "unexpected key name" for bare keys without colon
			name:    "unexpected key name translated",
			input:   "unexpected key name",
			wantNot: []string{"unexpected key name"},
			wantAny: []string{"Invalid YAML syntax", "key: value"},
		},
		{
			name:    "did not find expected translated",
			input:   "did not find expected key",
			wantNot: []string{"did not find expected"},
			wantAny: []string{"Invalid YAML syntax"},
		},
		{
			// Tab indentation error from goccy
			name:    "cannot start any token translated",
			input:   "found character '	' that cannot start any token",
			wantNot: []string{"cannot start any token"},
			wantAny: []string{"Invalid YAML syntax", "spaces", "tabs"},
		},
		{
			// Block sequence in wrong place
			name:    "block sequence entries not allowed translated",
			input:   "block sequence entries are not allowed in this context",
			wantNot: []string{"block sequence entries are not allowed"},
			wantAny: []string{"Invalid YAML syntax"},
		},
		{
			// Unclosed bracket
			name:    "sequence end token not found translated",
			input:   "sequence end token ']' not found",
			wantNot: []string{"sequence end token"},
			wantAny: []string{"Invalid YAML syntax", "unclosed"},
		},
		{
			name:    "unrecognized message returned unchanged",
			input:   "found unknown escape character 'z'",
			wantNot: []string{},
			wantAny: []string{"found unknown escape character 'z'"},
		},
		{
			name:    "empty message returned unchanged",
			input:   "",
			wantNot: []string{},
			wantAny: []string{""},
		},
		{
			name:    "partially matching message translated",
			input:   "[3:1] non-map value is specified as a key\n   2 | foo: bar\n>  3 | baz qux\n       ^",
			wantNot: []string{"non-map value is specified"},
			wantAny: []string{"Invalid YAML syntax"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := translateYAMLMessage(tt.input)

			for _, unwanted := range tt.wantNot {
				assert.NotContains(t, result, unwanted,
					"Result should not contain %q\nResult: %s", unwanted, result)
			}

			for _, wanted := range tt.wantAny {
				if wanted == "" {
					continue
				}
				assert.Contains(t, result, wanted,
					"Result should contain %q\nResult: %s", wanted, result)
			}
		})
	}
}
