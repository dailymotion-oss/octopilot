package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTplExpandGitHubLinksToMarkdownFunc(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		fullRepoName string
		input        string
		expected     string
	}{
		{
			name:         "input with no url",
			fullRepoName: "owner/name",
			input:        "some text",
			expected:     "some text",
		},
		{
			name:         "basic PR link",
			fullRepoName: "owner/name",
			input:        "PR #42: some changes",
			expected:     "PR [#42](https://github.com/owner/name/issues/42): some changes",
		},
		{
			name:         "multiple PR links",
			fullRepoName: "owner/name",
			input:        "PR #42: some changes - see also #43 and #44",
			expected:     "PR [#42](https://github.com/owner/name/issues/42): some changes - see also [#43](https://github.com/owner/name/issues/43) and [#44](https://github.com/owner/name/issues/44)",
		},
		{
			name:         "already expanded PR link",
			fullRepoName: "owner/name",
			input:        "[#42](https://github.com/some/where/pull/42)",
			expected:     "[#42](https://github.com/some/where/pull/42)",
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual := tplExpandGitHubLinksToMarkdownFunc()(test.fullRepoName, test.input)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestTplExtractMarkdownURLsFunc(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "input with no url",
			input:    "some text",
			expected: "some text",
		},
		{
			name:     "basic link",
			input:    "PR [#42](https://github.com/owner/name/issues/42): some changes",
			expected: "PR https://github.com/owner/name/issues/42: some changes",
		},
		{
			name:     "multiple PR links",
			input:    "PR [#42](https://github.com/owner/name/issues/42): some changes - see also [#43](https://github.com/owner/name/issues/43) and [#44](https://github.com/owner/name/issues/44)",
			expected: "PR https://github.com/owner/name/issues/42: some changes - see also https://github.com/owner/name/issues/43 and https://github.com/owner/name/issues/44",
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual := tplExtractMarkdownURLsFunc()(test.input)
			assert.Equal(t, test.expected, actual)
		})
	}
}
