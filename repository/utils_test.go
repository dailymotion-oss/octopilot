package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSearchType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected SearchType
	}{
		{
			name:     "Valid search type (repositories)",
			input:    "repositories",
			expected: Repositories,
		},
		{
			name:     "Valid search type (code)",
			input:    "code",
			expected: Code,
		},
		{
			name:     "Empty search type",
			input:    "",
			expected: Repositories,
		},
		{
			name:     "Invalid search type",
			input:    "invalid_type",
			expected: Repositories, // Defaults to Repositories
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := parseSearchType(test.input)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestRemoveDuplicate(t *testing.T) {
	t.Parallel()

	repositories := []Repository{
		{Owner: "owner1", Name: "repo1"},
		{Owner: "owner1", Name: "repo1"},
		{Owner: "owner1", Name: "repo2"},
		{Owner: "owner2", Name: "repo2"},
		{Owner: "owner3", Name: "repo3"},
	}

	result := removeDuplicate(repositories)

	expected := []Repository{
		{Owner: "owner1", Name: "repo1"},
		{Owner: "owner1", Name: "repo2"},
		{Owner: "owner2", Name: "repo2"},
		{Owner: "owner3", Name: "repo3"},
	}

	assert.ElementsMatch(t, expected, result)
}
