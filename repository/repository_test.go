package repository

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		repos            []string
		preTestHook      func()
		expected         []Repository
		expectedErrorMsg string
	}{
		{
			name: "nil input",
		},
		{
			name:             "empty input",
			repos:            []string{""},
			expectedErrorMsg: "invalid syntax for : missing repo type or name",
		},
		{
			name:             "invalid input",
			repos:            []string{"whatever"},
			expectedErrorMsg: "invalid syntax for whatever: found 0 matches instead of 4: []",
		},
		{
			name:  "single repository without parameters",
			repos: []string{"dailymotion-oss/octopilot"},
			expected: []Repository{
				{
					Owner:  "dailymotion-oss",
					Name:   "octopilot",
					Params: map[string]string{},
				},
			},
		},
		{
			name:  "single repository with dot inside",
			repos: []string{"dailymotion-oss/octopilot.test"},
			expected: []Repository{
				{
					Owner:  "dailymotion-oss",
					Name:   "octopilot.test",
					Params: map[string]string{},
				},
			},
		},
		{
			name:  "multiple repositories without parameters",
			repos: []string{"dailymotion-oss/octopilot", "some-owner/MyGreatRepo"},
			expected: []Repository{
				{
					Owner:  "dailymotion-oss",
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
			repos: []string{"dailymotion-oss/octopilot(draft=true)"},
			expected: []Repository{
				{
					Owner: "dailymotion-oss",
					Name:  "octopilot",
					Params: map[string]string{
						"draft": "true",
					},
				},
			},
		},
		{
			name:  "single repository with dot inside and a single parameter",
			repos: []string{"dailymotion-oss/octopilot.test(draft=true)"},
			expected: []Repository{
				{
					Owner: "dailymotion-oss",
					Name:  "octopilot.test",
					Params: map[string]string{
						"draft": "true",
					},
				},
			},
		},
		{
			name:  "multiple repositories with multiple parameters",
			repos: []string{"dailymotion-oss/octopilot(draft=true,merge=true)", "some-owner/MyGreatRepo(merge=false)"},
			expected: []Repository{
				{
					Owner: "dailymotion-oss",
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
		{
			name:  "discover from environment",
			repos: []string{"discover-from(env=OCTOPILOT_TEST_DISCOVER_FROM,sep=;,merge=true)"},
			preTestHook: func() {
				os.Setenv("OCTOPILOT_TEST_DISCOVER_FROM", "dailymotion-oss/octopilot;some-owner/MyGreatRepo(merge=false)")
			},
			expected: []Repository{
				{
					Owner: "dailymotion-oss",
					Name:  "octopilot",
					Params: map[string]string{
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
			if test.preTestHook != nil {
				test.preTestHook()
			}
			actual, err := Parse(context.Background(), test.repos, GitHubOptions{})
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
