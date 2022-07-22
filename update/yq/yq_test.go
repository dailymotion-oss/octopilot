package yq

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/mikefarah/yq/v4/pkg/yqlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUpdater(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		params           map[string]string
		expected         *YQUpdater
		expectedErrorMsg string
	}{
		{
			name: "valid params with single file",
			params: map[string]string{
				"file":         "values.yaml",
				"expression":   ".version = 1.2.3",
				"output":       "/tmp/output.yaml",
				"indent":       "4",
				"trim":         "true",
				"unwrapscalar": "false",
				"json":         "true",
			},
			expected: &YQUpdater{
				FilePath:     "values.yaml",
				Expression:   ".version = 1.2.3",
				Output:       "/tmp/output.yaml",
				Indent:       4,
				Trim:         true,
				UnwrapScalar: false,
				OutputFormat: yqlib.JSONOutputFormat,
			},
		},
		{
			name: "valid params with multiple files using a glob pattern",
			params: map[string]string{
				"file":       "**/values.yaml",
				"expression": "something",
			},
			expected: &YQUpdater{
				FilePath:     "**/values.yaml",
				Expression:   "something",
				Indent:       2,
				Trim:         false,
				UnwrapScalar: true,
				OutputFormat: yqlib.YamlOutputFormat,
			},
		},
		{
			name:             "nil params",
			expectedErrorMsg: "missing file parameter",
		},
		{
			name: "missing mandatory file param",
			params: map[string]string{
				"expression": "something",
			},
			expectedErrorMsg: "missing file parameter",
		},
		{
			name: "missing mandatory expression param",
			params: map[string]string{
				"file": "values.yaml",
			},
			expectedErrorMsg: "missing expression parameter",
		},
		{
			name: "invalid trim boolean value",
			params: map[string]string{
				"file":       "values.yaml",
				"expression": "something",
				"trim":       "maybe",
			},
			expected: &YQUpdater{
				FilePath:     "values.yaml",
				Expression:   "something",
				Trim:         false,
				UnwrapScalar: true,
				OutputFormat: yqlib.YamlOutputFormat,
				Indent:       2,
			},
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual, err := NewUpdater(test.params)
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
		updater          *YQUpdater
		expected         bool
		expectedErrorMsg string
		expectedFiles    map[string]string
	}{
		{
			name: "update simple value in a single file",
			files: map[string]string{
				"basic-values.yaml": `# top level comment

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
			updater: &YQUpdater{
				FilePath:     "basic-values.yaml",
				Expression:   `.object.mykey = "updated-value"`,
				OutputFormat: yqlib.YamlOutputFormat,
				Indent:       2,
			},
			expected: true,
			expectedFiles: map[string]string{
				"basic-values.yaml": `# top level comment

# a simple key
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
			name: "extract a sub object from a single file to a new file",
			files: map[string]string{
				"extract-source.yaml": `# a simple key
key: value
# an object
object:
  # with a key
  mykey: some-value
  # and another one
  anotherkey: another-value
`,
			},
			updater: &YQUpdater{
				FilePath:     "extract-source.yaml",
				Expression:   `.object`,
				Output:       "extract-output.yaml",
				OutputFormat: yqlib.YamlOutputFormat,
			},
			expected: true,
			expectedFiles: map[string]string{
				"extract-output.yaml": `# with a key
mykey: some-value
# and another one
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
			updater: &YQUpdater{
				FilePath:     "complex-values.yaml",
				Expression:   `(.array[] | select(.name == "my entry") | .key) = "new value"`,
				OutputFormat: yqlib.YamlOutputFormat,
				Indent:       2,
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
			updater: &YQUpdater{
				FilePath:     "multiple-values.yaml",
				Expression:   `(.array[] | select(.ref == "abc*") | .key) = "updated value"`,
				OutputFormat: yqlib.YamlOutputFormat,
				Indent:       2,
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
			name: "update with folded style",
			files: map[string]string{
				"folded-style.yaml": `# a simple key
key: value
anotherkey: another-value
`,
			},
			updater: &YQUpdater{
				FilePath:     "folded-style.yaml",
				Expression:   `.key = "updated-value" | .key style="folded"`,
				OutputFormat: yqlib.YamlOutputFormat,
			},
			expected: true,
			expectedFiles: map[string]string{
				"folded-style.yaml": `# a simple key
key: >-
    updated-value
anotherkey: another-value
`,
			},
		},
		{
			name: "trim file",
			files: map[string]string{
				"trim.yaml": `
# a simple key
key: value
`,
			},
			updater: &YQUpdater{
				FilePath:     "trim.yaml",
				Expression:   `.key = "updated-value"`,
				OutputFormat: yqlib.YamlOutputFormat,
				Trim:         true,
			},
			expected: true,
			expectedFiles: map[string]string{
				"trim.yaml": `# a simple key
key: updated-value`,
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
			updater: &YQUpdater{
				FilePath:     "missing-key-values.yaml",
				Expression:   `.object.mykey = "new-value"`,
				OutputFormat: yqlib.YamlOutputFormat,
			},
			expected: true,
			expectedFiles: map[string]string{
				"missing-key-values.yaml": `
# a simple key
key: value
object:
    mykey: new-value
`,
			},
		},
		{
			name: "unwrap scalar",
			files: map[string]string{
				"unwrap-scalar.yaml": `
object:
  key: value # comment
`,
			},
			updater: &YQUpdater{
				FilePath:     "unwrap-scalar.yaml",
				Expression:   `.object.key`,
				OutputFormat: yqlib.YamlOutputFormat,
				UnwrapScalar: true,
			},
			expected: true,
			expectedFiles: map[string]string{
				"unwrap-scalar.yaml": `value
`,
			},
		},
		{
			name: "do not unwrap scalar",
			files: map[string]string{
				"no-unwrap-scalar.yaml": `
object:
  key: value # comment
`,
			},
			updater: &YQUpdater{
				FilePath:     "no-unwrap-scalar.yaml",
				Expression:   `.object.key`,
				OutputFormat: yqlib.YamlOutputFormat,
				UnwrapScalar: false,
			},
			expected: true,
			expectedFiles: map[string]string{
				"no-unwrap-scalar.yaml": `value # comment
`,
			},
		},
		{
			name: "output to json",
			files: map[string]string{
				"output-to-json.yaml": `
object:
  key: value # comment
`,
			},
			updater: &YQUpdater{
				FilePath:     "output-to-json.yaml",
				Expression:   `.`,
				Output:       "output-to-json.json",
				OutputFormat: yqlib.JSONOutputFormat,
				Indent:       2,
			},
			expected: true,
			expectedFiles: map[string]string{
				"output-to-json.json": `{
  "object": {
    "key": "value"
  }
}
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
			updater: &YQUpdater{
				FilePath:     "no-changes.yaml",
				Expression:   `.key = "value"`,
				OutputFormat: yqlib.YamlOutputFormat,
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
				actualFilePath := test.updater.Output
				if actualFilePath == "" {
					actualFilePath = test.updater.FilePath
				}
				actualFilePaths, err := filepath.Glob(filepath.Join("testdata", actualFilePath))
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
