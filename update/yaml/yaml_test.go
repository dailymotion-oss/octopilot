package yaml

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/dailymotion-oss/octopilot/update/value"

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
				"style":  "double",
				"trim":   "true",
				"indent": "4",
			},
			expected: &YamlUpdater{
				FilePath:   "values.yaml",
				Path:       "level1.level2",
				AutoCreate: true,
				Style:      "double",
				Trim:       true,
				Indent:     4,
			},
		},
		{
			name: "invalid create boolean value",
			params: map[string]string{
				"file":   "values.yaml",
				"path":   "level1.level2",
				"create": "maybe",
				"indent": "not-an-int",
			},
			expected: &YamlUpdater{
				FilePath:   "values.yaml",
				Path:       "level1.level2",
				AutoCreate: false,
				Trim:       false,
				Indent:     2,
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
				Indent:   2,
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
			updater: &YamlUpdater{
				FilePath: "basic-values.yaml",
				Path:     "object.mykey",
				Valuer:   value.StringValuer("updated-value"),
				Indent:   2,
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
				Indent:   2,
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
				Indent:   2,
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
			name: "update with custom style",
			files: map[string]string{
				"custom-style.yaml": `
# a simple key
key: value
`,
			},
			updater: &YamlUpdater{
				FilePath: "custom-style.yaml",
				Path:     "key",
				Style:    "double",
				Valuer:   value.StringValuer("updated-value"),
			},
			expected: true,
			expectedFiles: map[string]string{
				"custom-style.yaml": `
# a simple key
key: "updated-value"
`,
			},
		},
		{
			name: "update with folded style",
			files: map[string]string{
				"folded-style.yaml": `# a simple key
key: value
`,
			},
			updater: &YamlUpdater{
				FilePath: "folded-style.yaml",
				Path:     "key",
				Style:    "folded",
				Valuer:   value.StringValuer("updated-value"),
			},
			expected: true,
			expectedFiles: map[string]string{
				"folded-style.yaml": `# a simple key
key: >-
    updated-value
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
			updater: &YamlUpdater{
				FilePath: "trim.yaml",
				Path:     "key",
				Trim:     true,
				Valuer:   value.StringValuer("updated-value"),
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
				"missing-key-values.yaml": `# a simple key
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

func TestYqExpression(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		updater            YamlUpdater
		value              string
		expectedExpression string
		expectedErrorMsg   string
	}{
		{
			name: "simple v4 path",
			updater: YamlUpdater{
				Path: ".version",
			},
			value:              "1.2.3",
			expectedExpression: `(.version) ref $x | $x = "1.2.3"`,
		},
		{
			name: "v4 path with custom style",
			updater: YamlUpdater{
				Path:  ".path.to.version",
				Style: "double",
			},
			value:              "1.2.3",
			expectedExpression: `(.path.to.version) ref $x | $x = "1.2.3" | $x style="double"`,
		},
		{
			name: "v4 path with auto-create and custom style",
			updater: YamlUpdater{
				Path:       ".path.to.version",
				Style:      "folded",
				AutoCreate: true,
			},
			value:              "1.2.3",
			expectedExpression: `.path.to.version = "1.2.3" | (.path.to.version) ref $x | $x = "1.2.3" | $x style="folded"`,
		},
		{
			name: "complex v4 path",
			updater: YamlUpdater{
				Path: `.releases[] | select(.chart == "repo/chart") | .version`,
			},
			value:              "1.2.3",
			expectedExpression: `(.releases[] | select(.chart == "repo/chart") | .version) ref $x | $x = "1.2.3"`,
		},
		{
			name: "simple v3 path",
			updater: YamlUpdater{
				Path: "version",
			},
			value:              "1.2.3",
			expectedExpression: `(.version) ref $x | $x = "1.2.3"`,
		},
		{
			name: "v3 path with custom style",
			updater: YamlUpdater{
				Path:  "path.to.version",
				Style: "double",
			},
			value:              "1.2.3",
			expectedExpression: `(.path.to.version) ref $x | $x = "1.2.3" | $x style="double"`,
		},
		{
			name: "complex v3 path",
			updater: YamlUpdater{
				Path: `releases.(chart==repo/chart).version`,
			},
			value:              "1.2.3",
			expectedExpression: `(.releases[] | select(.chart == "repo/chart") | .version) ref $x | $x = "1.2.3"`,
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			expression, expressionNode, err := test.updater.yqExpression(test.value)
			if len(test.expectedErrorMsg) > 0 {
				require.EqualError(t, err, test.expectedErrorMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectedExpression, expression)
				assert.NotNil(t, expressionNode)
			}
		})
	}
}
