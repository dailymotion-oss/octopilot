package yaml

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/dailymotion/octopilot/update/value"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUpdater(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		params           map[string]string
		expected         *YamlUpdater
		expectedErrorMsg string
	}{
		{
			name: "valid params with single file",
			params: map[string]string{
				"file":   "values.yaml",
				"path":   "level1.level2",
				"create": "true",
			},
			expected: &YamlUpdater{
				FilePath:   "values.yaml",
				Path:       "level1.level2",
				AutoCreate: true,
			},
		},
		{
			name: "invalid create boolean value",
			params: map[string]string{
				"file":   "values.yaml",
				"path":   "level1.level2",
				"create": "maybe",
			},
			expected: &YamlUpdater{
				FilePath:   "values.yaml",
				Path:       "level1.level2",
				AutoCreate: false,
			},
		},
		{
			name: "valid params with multiple files using a glob pattern",
			params: map[string]string{
				"file": "**/values.yaml",
				"path": "level1.level2",
			},
			expected: &YamlUpdater{
				FilePath: "**/values.yaml",
				Path:     "level1.level2",
			},
		},
		{
			name:             "nil params",
			expectedErrorMsg: "missing file parameter",
		},
		{
			name: "missing mandatory file param",
			params: map[string]string{
				"path": "level1.level2",
			},
			expectedErrorMsg: "missing file parameter",
		},
		{
			name: "missing mandatory path param",
			params: map[string]string{
				"file": "values.yaml",
			},
			expectedErrorMsg: "missing path parameter",
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual, err := NewUpdater(test.params, nil)
			if len(test.expectedErrorMsg) > 0 {
				require.EqualError(t, err, test.expectedErrorMsg)
				assert.Nil(t, actual)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, actual)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		files            map[string]string
		updater          *YamlUpdater
		expected         bool
		expectedErrorMsg string
		expectedFiles    map[string]string
	}{
		{
			name: "update simple value in a single file",
			files: map[string]string{
				"basic-values.yaml": `
# a simple key
key: value
# an object
object:
  # with a key we want to update
  mykey: some-value
  # and another one we don't care about
  anotherkey: another-value
`,
			},
			updater: &YamlUpdater{
				FilePath: "basic-values.yaml",
				Path:     "object.mykey",
				Valuer:   value.StringValuer("updated-value"),
			},
			expected: true,
			expectedFiles: map[string]string{
				"basic-values.yaml": `# a simple key
key: value
# an object
object:
    # with a key we want to update
    mykey: updated-value
    # and another one we don't care about
    anotherkey: another-value
`,
			},
		},
		{
			name: "update complex value in a single file",
			files: map[string]string{
				"complex-values.yaml": `
array:
  - name: first entry
    key: first value
  - name: my entry
    key: old value
  - name: third entry
    key: third value
`,
			},
			updater: &YamlUpdater{
				FilePath: "complex-values.yaml",
				Path:     "array.(name==my entry).key",
				Valuer:   value.StringValuer("new value"),
			},
			expected: true,
			expectedFiles: map[string]string{
				"complex-values.yaml": `array:
  - name: first entry
    key: first value
  - name: my entry
    key: new value
  - name: third entry
    key: third value
`,
			},
		},
		{
			name: "update multiple values in a single file",
			files: map[string]string{
				"multiple-values.yaml": `
array:
  - name: first entry
    ref: abc123
    key: first value
  - name: second entry
    ref: xyz789
    key: second value
  - name: third entry
    ref: abc456
    key: third value
`,
			},
			updater: &YamlUpdater{
				FilePath: "multiple-values.yaml",
				Path:     "array.(ref==abc*).key",
				Valuer:   value.StringValuer("updated value"),
			},
			expected: true,
			expectedFiles: map[string]string{
				"multiple-values.yaml": `array:
  - name: first entry
    ref: abc123
    key: updated value
  - name: second entry
    ref: xyz789
    key: second value
  - name: third entry
    ref: abc456
    key: updated value
`,
			},
		},
		{
			name: "create missing key/value in a single file",
			files: map[string]string{
				"missing-key-values.yaml": `
# a simple key
key: value
`,
			},
			updater: &YamlUpdater{
				FilePath:   "missing-key-values.yaml",
				Path:       "object.mykey",
				AutoCreate: true,
				Valuer:     value.StringValuer("new-value"),
			},
			expected: true,
			expectedFiles: map[string]string{
				"missing-key-values.yaml": `# a simple key
key: value
object:
    mykey: new-value
`,
			},
		},
		{
			name: "no changes if new key but no auto-create",
			files: map[string]string{
				"no-changes-without-auto-create.yaml": `# a simple key
key: value
`,
			},
			updater: &YamlUpdater{
				FilePath: "no-changes-without-auto-create.yaml",
				Path:     "object.mykey",
				Valuer:   value.StringValuer("value"),
			},
			expected: false,
			expectedFiles: map[string]string{
				"no-changes-without-auto-create.yaml": `# a simple key
key: value
`,
			},
		},
		{
			name: "no changes",
			files: map[string]string{
				"no-changes.yaml": `# a simple key
key: value
`,
			},
			updater: &YamlUpdater{
				FilePath: "no-changes.yaml",
				Path:     "key",
				Valuer:   value.StringValuer("value"),
			},
			expected: false,
			expectedFiles: map[string]string{
				"no-changes.yaml": `# a simple key
key: value
`,
			},
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			{
				for filename, content := range test.files {
					err := os.MkdirAll(filepath.Dir(filepath.Join("testdata", filename)), 0755)
					require.NoErrorf(t, err, "can't create testdata directories for %s", filename)
					err = ioutil.WriteFile(filepath.Join("testdata", filename), []byte(content), 0644)
					require.NoErrorf(t, err, "can't write testdata file %s", filename)
				}
			}

			actual, err := test.updater.Update(context.Background(), "testdata")
			if len(test.expectedErrorMsg) > 0 {
				require.EqualError(t, err, test.expectedErrorMsg)
				assert.False(t, actual)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, actual)
				actualFilePaths, err := filepath.Glob(filepath.Join("testdata", test.updater.FilePath))
				require.NoError(t, err, "can't expand glob pattern for actual testdata file")
				for _, actualFilePath := range actualFilePaths {
					actualRelFilePath, err := filepath.Rel("testdata", actualFilePath)
					require.NoErrorf(t, err, "can't get relative path for actual testdata file %s", actualFilePath)
					actualFileContent, err := ioutil.ReadFile(actualFilePath)
					require.NoErrorf(t, err, "can't read actual testdata file %s", actualFilePath)
					expectedFileContent := test.expectedFiles[actualRelFilePath]
					assert.Equalf(t, expectedFileContent, string(actualFileContent), "testdata file %s doesn't match", actualFilePath)
				}
			}
		})
	}
}
