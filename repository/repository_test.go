package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		repos            []string
		expected         []Repository
		expectedErrorMsg string
	}{
		{
			name: "nil input",
		},
		{
			name:             "empty input",
			repos:            []string{""},
			expectedErrorMsg: "invalid syntax for : found 0 matches instead of 4: []",
		},
		{
			name:             "invalid input",
			repos:            []string{"whatever"},
			expectedErrorMsg: "invalid syntax for whatever: found 0 matches instead of 4: []",
		},
		{
			name:  "single repository without parameters",
			repos: []string{"dailymotion/octopilot"},
			expected: []Repository{
				{
					Owner:  "dailymotion",
					Name:   "octopilot",
					Params: map[string]string{},
				},
			},
		},
		{
			name:  "multiple repositories without parameters",
			repos: []string{"dailymotion/octopilot", "some-owner/MyGreatRepo"},
			expected: []Repository{
				{
					Owner:  "dailymotion",
					Name:   "octopilot",
					Params: map[string]string{},
				},
				{
					Owner:  "some-owner",
					Name:   "MyGreatRepo",
					Params: map[string]string{},
				},
			},
		},
		{
			name:  "single repository with a single parameter",
			repos: []string{"dailymotion/octopilot(draft=true)"},
			expected: []Repository{
				{
					Owner: "dailymotion",
					Name:  "octopilot",
					Params: map[string]string{
						"draft": "true",
					},
				},
			},
		},
		{
			name:  "multiple repositories with multiple parameters",
			repos: []string{"dailymotion/octopilot(draft=true,merge=true)", "some-owner/MyGreatRepo(merge=false)"},
			expected: []Repository{
				{
					Owner: "dailymotion",
					Name:  "octopilot",
					Params: map[string]string{
						"draft": "true",
						"merge": "true",
					},
				},
				{
					Owner: "some-owner",
					Name:  "MyGreatRepo",
					Params: map[string]string{
						"merge": "false",
					},
				},
			},
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual, err := Parse(test.repos)
			if len(test.expectedErrorMsg) > 0 {
				require.EqualError(t, err, test.expectedErrorMsg)
				assert.Empty(t, actual)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, actual)
			}
		})
	}
}
