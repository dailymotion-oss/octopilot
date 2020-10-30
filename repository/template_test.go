package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTplExpandGitHubLinksToMarkdown(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		repo     Repository
		input    string
		expected string
	}{
		{
			name: "input with no url",
			repo: Repository{
				Owner: "owner",
				Name:  "name",
			},
			input:    "some text",
			expected: "some text",
		},
		{
			name: "basic PR link",
			repo: Repository{
				Owner: "owner",
				Name:  "name",
			},
			input:    "PR #42: some changes",
			expected: "PR [#42](https://github.com/owner/name/issues/42): some changes",
		},
		{
			name: "multiple PR links",
			repo: Repository{
				Owner: "owner",
				Name:  "name",
			},
			input:    "PR #42: some changes - see also #43 and #44",
			expected: "PR [#42](https://github.com/owner/name/issues/42): some changes - see also [#43](https://github.com/owner/name/issues/43) and [#44](https://github.com/owner/name/issues/44)",
		},
		{
			name: "already expanded PR link",
			repo: Repository{
				Owner: "owner",
				Name:  "name",
			},
			input:    "[#42](https://github.com/some/where/pull/42)",
			expected: "[#42](https://github.com/some/where/pull/42)",
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual := tplExpandGitHubLinksToMarkdown(test.repo)(test.input)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestTplExtractMarkdownURLs(t *testing.T) {
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
			actual := tplExtractMarkdownURLs()(test.input)
			assert.Equal(t, test.expected, actual)
		})
	}
}
