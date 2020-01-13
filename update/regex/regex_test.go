package regex

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
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
		expected         *RegexUpdater
		expectedErrorMsg string
	}{
		{
			name: "valid params with single file",
			params: map[string]string{
				"file":    "helmfile.yaml",
				"pattern": `\s+version: \"(.*)\"`,
			},
			expected: &RegexUpdater{
				FilePath: "helmfile.yaml",
				Pattern:  `\s+version: \"(.*)\"`,
				Regexp:   regexp.MustCompile(`\s+version: \"(.*)\"`),
			},
		},
		{
			name: "valid params with multiple files using a glob pattern",
			params: map[string]string{
				"file":    "**/helmfile.yaml",
				"pattern": `\s+version: \"(.*)\"`,
			},
			expected: &RegexUpdater{
				FilePath: "**/helmfile.yaml",
				Pattern:  `\s+version: \"(.*)\"`,
				Regexp:   regexp.MustCompile(`\s+version: \"(.*)\"`),
			},
		},
		{
			name:             "nil params",
			expectedErrorMsg: "missing file parameter",
		},
		{
			name: "missing mandatory file param",
			params: map[string]string{
				"pattern": `\s+version: \"(.*)\"`,
			},
			expectedErrorMsg: "missing file parameter",
		},
		{
			name: "missing mandatory pattern param",
			params: map[string]string{
				"file": "helmfile.yaml",
			},
			expectedErrorMsg: "missing pattern parameter",
		},
		{
			name: "invalid pattern",
			params: map[string]string{
				"file":    "helmfile.yaml",
				"pattern": `(.*`,
			},
			expectedErrorMsg: "invalid pattern (.*: error parsing regexp: missing closing ): `(.*`",
		},
		{
			name: "pattern with 0 parenthesized subexpression",
			params: map[string]string{
				"file":    "helmfile.yaml",
				"pattern": `.*`,
			},
			expectedErrorMsg: "invalid pattern .*: it must have a single parenthesized subexpression, but it has 0",
		},
		{
			name: "pattern with 2 parenthesized subexpressions",
			params: map[string]string{
				"file":    "helmfile.yaml",
				"pattern": `([0-9]+).*([a-z]+)`,
			},
			expectedErrorMsg: "invalid pattern ([0-9]+).*([a-z]+): it must have a single parenthesized subexpression, but it has 2",
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
		updater          *RegexUpdater
		expected         bool
		expectedErrorMsg string
		expectedFiles    map[string]string
	}{
		{
			name: "update multiple versions in a single file",
			files: map[string]string{
				"helmfile-multiple-versions.yaml": `
releases:
  - name: my-release
	chart: example/my-chart
	version: "1.0.0"
	namespace: default
  - name: another-release
	chart: example/another-chart
	version: "1.0.0"
	namespace: another-ns
`,
			},
			updater: &RegexUpdater{
				FilePath: "helmfile-multiple-versions.yaml",
				Pattern:  `\s+version: \"(.*)\"`,
				Regexp:   regexp.MustCompile(`\s+version: \"(.*)\"`),
				Valuer:   value.StringValuer("2.0.0"),
			},
			expected: true,
			expectedFiles: map[string]string{
				"helmfile-multiple-versions.yaml": `
releases:
  - name: my-release
	chart: example/my-chart
	version: "2.0.0"
	namespace: default
  - name: another-release
	chart: example/another-chart
	version: "2.0.0"
	namespace: another-ns
`,
			},
		},
		{
			name: "update a single version in a single file",
			files: map[string]string{
				"helmfile-single-version.yaml": `
releases:
  - name: my-release
	chart: example/my-chart
	version: "1.0.0"
	namespace: default
  - name: another-release
	chart: example/another-chart
	version: "1.0.0"
	namespace: another-ns
`,
			},
			updater: &RegexUpdater{
				FilePath: "helmfile-single-version.yaml",
				Pattern:  `chart: example/my-chart\s+version: \"(.*)\"`,
				Regexp:   regexp.MustCompile(`chart: example/my-chart\s+version: \"(.*)\"`),
				Valuer:   value.StringValuer("2.0.0"),
			},
			expected: true,
			expectedFiles: map[string]string{
				"helmfile-single-version.yaml": `
releases:
  - name: my-release
	chart: example/my-chart
	version: "2.0.0"
	namespace: default
  - name: another-release
	chart: example/another-chart
	version: "1.0.0"
	namespace: another-ns
`,
			},
		},
		{
			name: "update in multi-line mode in a single file",
			files: map[string]string{
				"readme.txt": `
whatever content
`,
			},
			updater: &RegexUpdater{
				FilePath: "readme.txt",
				Pattern:  `(?ms)(.*)`,
				Regexp:   regexp.MustCompile(`(?ms)(.*)`),
				Valuer:   value.StringValuer("new content"),
			},
			expected: true,
			expectedFiles: map[string]string{
				"readme.txt": `new content`,
			},
		},
		{
			name: "no update in a single file",
			files: map[string]string{
				"helmfile-no-update.yaml": `
releases:
  - name: my-release
	chart: example/my-chart
	version: "1.0.0"
	namespace: default
  - name: another-release
	chart: example/another-chart
	version: "1.0.0"
	namespace: another-ns
`,
			},
			updater: &RegexUpdater{
				FilePath: "helmfile-no-update.yaml",
				Pattern:  `chart: example/some-chart\s+version: \"(.*)\"`,
				Regexp:   regexp.MustCompile(`chart: example/some-chart\s+version: \"(.*)\"`),
				Valuer:   value.StringValuer("2.0.0"),
			},
			expected: false,
			expectedFiles: map[string]string{
				"helmfile-no-update.yaml": `
releases:
  - name: my-release
	chart: example/my-chart
	version: "1.0.0"
	namespace: default
  - name: another-release
	chart: example/another-chart
	version: "1.0.0"
	namespace: another-ns
`,
			},
		},
		{
			name: "updates in multiple files",
			files: map[string]string{
				"app1/helmfile.yaml": `
releases:
  - name: my-release
	chart: example/my-chart
	version: "1.0.0"
	namespace: default
`,
				"app2/helmfile.yaml": `
releases:
  - name: another-release
	chart: example/another-chart
	version: "1.0.0"
	namespace: another-ns
`,
			},
			updater: &RegexUpdater{
				FilePath: "app*/helmfile.yaml",
				Pattern:  `\s+version: \"(.*)\"`,
				Regexp:   regexp.MustCompile(`\s+version: \"(.*)\"`),
				Valuer:   value.StringValuer("2.0.0"),
			},
			expected: true,
			expectedFiles: map[string]string{
				"app1/helmfile.yaml": `
releases:
  - name: my-release
	chart: example/my-chart
	version: "2.0.0"
	namespace: default
`,
				"app2/helmfile.yaml": `
releases:
  - name: another-release
	chart: example/another-chart
	version: "2.0.0"
	namespace: another-ns
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
