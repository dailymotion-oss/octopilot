package yaml

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractLeadingContentForYQ(t *testing.T) {
	t.Parallel()
	const (
		yqDocSeparatorPrefix = "$yqDocSeperator$\n"
	)
	tests := []struct {
		name                   string
		input                  string
		expected               string
		expectedLeadingContent string
		expectedErrorMsg       string
	}{
		{
			name:  "empty input",
			input: "",
		},
		{
			name:     "no leading content",
			input:    "key: value",
			expected: "key: value",
		},
		{
			name: "top level comments",
			input: `# top level comment
# on multiple lines

key: value
object:
  key: value`,
			expected: `
key: value
object:
  key: value`,
			expectedLeadingContent: `# top level comment
# on multiple lines
`,
		},
		{
			name: "doc separator and top level comments",
			input: `---
# top level comment
# on multiple lines

key: value`,
			expected: `
key: value`,
			expectedLeadingContent: yqDocSeparatorPrefix + `# top level comment
# on multiple lines
`,
		},
		{
			name: "doc separator",
			input: `---
key: value`,
			expected:               `key: value`,
			expectedLeadingContent: yqDocSeparatorPrefix,
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual, actualLeadingContent, err := ExtractLeadingContentForYQ(strings.NewReader(test.input))
			if len(test.expectedErrorMsg) > 0 {
				require.EqualError(t, err, test.expectedErrorMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectedLeadingContent, actualLeadingContent)
				actualBytes, err := io.ReadAll(actual)
				require.NoError(t, err)
				assert.Equal(t, test.expected, string(actualBytes))
			}
		})
	}
}
