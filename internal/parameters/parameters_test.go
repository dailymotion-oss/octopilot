package parameters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		paramsStr string
		expected  map[string]string
	}{
		{
			name:      "empty input",
			paramsStr: "",
			expected:  map[string]string{},
		},
		{
			name:      "invalid input",
			paramsStr: "whatever",
			expected:  map[string]string{},
		},
		{
			name:      "single simple param",
			paramsStr: "key=value",
			expected: map[string]string{
				"key": "value",
			},
		},
		{
			name:      "multiple simple params",
			paramsStr: "key=value,other-key=some-value,another-key=whatever value",
			expected: map[string]string{
				"key":         "value",
				"other-key":   "some-value",
				"another-key": "whatever value",
			},
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual := Parse(test.paramsStr)
			assert.Equal(t, test.expected, actual)
		})
	}
}
