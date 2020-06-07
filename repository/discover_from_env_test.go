package repository

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverRepositoriesFromEnvironment(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		envVarValue      string
		params           map[string]string
		expected         []Repository
		expectedErrorMsg string
	}{
		{
			name: "empty input",
		},
		{
			name:             "invalid input",
			envVarValue:      "whatever",
			expectedErrorMsg: "failed to parse [whatever]: invalid syntax for whatever: found 0 matches instead of 4: []",
		},
		{
			name:        "single repository without parameters",
			envVarValue: "dailymotion/octopilot",
			expected: []Repository{
				{
					Owner:  "dailymotion",
					Name:   "octopilot",
					Params: map[string]string{},
				},
			},
		},
		{
			name:        "multiple repositories without parameters",
			envVarValue: "dailymotion/octopilot some-owner/MyGreatRepo",
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
			name:        "multiple repositories with custom separator",
			envVarValue: "dailymotion/octopilot|some-owner/MyGreatRepo",
			params: map[string]string{
				"sep": "|",
			},
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
			name:        "single repository with a single parameter",
			envVarValue: "dailymotion/octopilot(draft=true)",
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
			name:        "multiple repositories with multiple parameters",
			envVarValue: "dailymotion/octopilot(draft=true,merge=true) some-owner/MyGreatRepo(merge=false)",
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
		{
			name:        "multiple repositories with multiple parameters with overrides",
			envVarValue: "dailymotion/octopilot(draft=true,merge=false) some-owner/MyGreatRepo(draft=true)",
			params: map[string]string{
				"merge": "true",
			},
			expected: []Repository{
				{
					Owner: "dailymotion",
					Name:  "octopilot",
					Params: map[string]string{
						"draft": "true",
						"merge": "false",
					},
				},
				{
					Owner: "some-owner",
					Name:  "MyGreatRepo",
					Params: map[string]string{
						"draft": "true",
						"merge": "true",
					},
				},
			},
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			envVar := fmt.Sprintf("DISCOVER_FROM_ENV_%v", i)
			os.Setenv(envVar, test.envVarValue)
			defer os.Unsetenv(envVar)

			actual, err := discoverRepositoriesFromEnvironment(context.Background(), envVar, test.params, "")
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
