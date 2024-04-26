package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		title    string
		body     string
		footer   string
		expected string
	}{
		{
			name:     "No content",
			expected: "",
		},
		{
			name:     "Only title",
			title:    "title",
			expected: "title",
		},
		{
			name:     "Title and body",
			title:    "title",
			body:     "body\nbody",
			expected: "title\n\nbody\nbody",
		},
		{
			name:     "Title and footer",
			title:    "title",
			footer:   "footer\nfooter",
			expected: "title\n\n-- \nfooter\nfooter",
		},
		{
			name:     "Title body and footer",
			title:    "title",
			body:     "body\nbody",
			footer:   "footer\nfooter",
			expected: "title\n\nbody\nbody\n\n-- \nfooter\nfooter",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			commitMessage := NewCommitMessage(test.title, test.body, test.footer)
			actual := commitMessage.String()
			assert.Equal(t, test.expected, actual)
		})
	}
}
