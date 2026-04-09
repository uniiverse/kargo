package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseProjectLabelPrefixes(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single prefix",
			input:    "universe.engineer/",
			expected: []string{"universe.engineer/"},
		},
		{
			name:     "multiple prefixes",
			input:    "universe.engineer/,team.example.com/",
			expected: []string{"universe.engineer/", "team.example.com/"},
		},
		{
			name:     "trims whitespace and skips empty entries",
			input:    " universe.engineer/ , , team.example.com/ ",
			expected: []string{"universe.engineer/", "team.example.com/"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseProjectLabelPrefixes(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestDecodeArgoCDURLs(t *testing.T) {

	testCases := []struct {
		name     string
		input    string
		expected ArgoCDURLMap
		errMsg   string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: ArgoCDURLMap(map[string]string{}),
		},
		{
			name:  "empty shard",
			input: `=https://argocd.com`,
			expected: ArgoCDURLMap(map[string]string{
				"": "https://argocd.com",
			}),
		},
		{
			name:   "invalid input",
			input:  `https://argocd.com`,
			errMsg: "expected <shard>=<URL>",
		},
		{
			name:  "multiple shards",
			input: `foo=https://argocd.com,bar=https://argocd.org`,
			expected: ArgoCDURLMap(map[string]string{
				"foo": "https://argocd.com",
				"bar": "https://argocd.org",
			}),
		},
		{
			name:  "ignore empty items",
			input: `foo=https://argocd.com,,bar=https://argocd.org,`,
			expected: ArgoCDURLMap(map[string]string{
				"foo": "https://argocd.com",
				"bar": "https://argocd.org",
			}),
		},
		{
			name:  "trim leading and trailing whitespace",
			input: ` = https://argocd.com , , bar = https://argocd.org`,
			expected: ArgoCDURLMap(map[string]string{
				"":    "https://argocd.com",
				"bar": "https://argocd.org",
			}),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var acdMap ArgoCDURLMap
			err := acdMap.Decode(tc.input)
			if tc.errMsg != "" {
				require.ErrorContains(t, err, tc.errMsg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, acdMap)
		})
	}
}
