package value

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		value            string
		expected         Valuer
		expectedErrorMsg string
	}{
		{
			name:     "nil input",
			expected: StringValuer(""),
		},
		{
			name:     "empty input",
			value:    "",
			expected: StringValuer(""),
		},
		{
			name:             "unknown valuer",
			value:            "whatever(key=value)",
			expectedErrorMsg: "unknown valuer whatever",
		},
		{
			name:     "string value",
			value:    "1.2.3",
			expected: StringValuer("1.2.3"),
		},
		{
			name:  "file value",
			value: "file(path=/path/to/something)",
			expected: &FileValuer{
				Path: "/path/to/something",
			},
		},
		{
			name:             "file value without path",
			value:            "file(path=)",
			expectedErrorMsg: "failed to create a valuer instance for file: missing path parameter",
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual, err := ParseValuer(test.value)
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
