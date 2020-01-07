package value

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileValuerValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		path             string
		expected         string
		expectedErrorMsg string
	}{
		{
			name:             "file does not exists",
			path:             "does-not-exists",
			expectedErrorMsg: "failed to read file does-not-exists: open testdata/does-not-exists: no such file or directory",
		},
		{
			name:     "regular file",
			path:     "test.txt",
			expected: "some content",
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			valuer := FileValuer{
				Path: test.path,
			}
			actual, err := valuer.Value(context.Background(), filepath.Join(".", "testdata"))
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
