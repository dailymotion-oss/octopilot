package helm

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
		expected         *HelmUpdater
		expectedErrorMsg string
	}{
		{
			name: "valid params",
			params: map[string]string{
				"dependency": "some-chart",
			},
			expected: &HelmUpdater{
				Dependency: "some-chart",
			},
		},
		{
			name:             "nil params",
			expectedErrorMsg: "missing dependency parameter",
		},
		{
			name: "missing mandatory dependency param",
			params: map[string]string{
				"whatever": "here too",
			},
			expectedErrorMsg: "missing dependency parameter",
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
		updater          *HelmUpdater
		expected         bool
		expectedErrorMsg string
		expectedFiles    map[string]string
	}{
		{
			name: "update a single version in a single chart in helm 2 format",
			files: map[string]string{
				filepath.Join("helm2", "Chart.yaml"): "",
				filepath.Join("helm2", "requirements.yaml"): `
# some comment
dependencies:
# this is a comment for my chart
- name: my-chart
  # and a version I like
  version: 1.0.0
  # our enterprise repo
  repository: example.com/charts
  # an unknown field
  whatever: something else
`,
			},
			updater: &HelmUpdater{
				Dependency: "my-chart",
				Valuer:     value.StringValuer("2.0.0"),
			},
			expected: true,
			expectedFiles: map[string]string{
				filepath.Join("helm2", "requirements.yaml"): `# some comment
dependencies:
  - # this is a comment for my chart
    name: my-chart
    # and a version I like
    version: 2.0.0
    # our enterprise repo
    repository: example.com/charts
    # an unknown field
    whatever: something else
`,
			},
		},
		{
			name: "update a single version in a single chart in helm 2 format",
			files: map[string]string{
				filepath.Join("helm3", "Chart.yaml"): `
# metadata fields
name: some-chart
version: 1.2.3
# all dependencies
dependencies:
# first dependency
- name: first-chart
  version: 1.0.0
# second dependency
- name: second-chart
  version: 1.0.0
# third dependency
- name: third-chart
  version: 1.0.0
`,
			},
			updater: &HelmUpdater{
				Dependency: "second-chart",
				Valuer:     value.StringValuer("2.0.0"),
			},
			expected: true,
			expectedFiles: map[string]string{
				filepath.Join("helm3", "Chart.yaml"): `# metadata fields
name: some-chart
version: 1.2.3
# all dependencies
dependencies:
  - # first dependency
    name: first-chart
    version: 1.0.0
  - # second dependency
    name: second-chart
    version: 2.0.0
  - # third dependency
    name: third-chart
    version: 1.0.0
`,
			},
		},
		{
			name: "no matching dependency to update",
			files: map[string]string{
				filepath.Join("no-update", "Chart.yaml"): `
dependencies:
- name: some-chart
  version: 1.0.0
`,
			},
			updater: &HelmUpdater{
				Dependency: "not-a-dependency-chart",
				Valuer:     value.StringValuer("2.0.0"),
			},
			expected: false,
			expectedFiles: map[string]string{
				filepath.Join("no-update", "Chart.yaml"): `
dependencies:
- name: some-chart
  version: 1.0.0
`,
			},
		},
		{
			name: "no version change",
			files: map[string]string{
				filepath.Join("same-version", "Chart.yaml"): `
dependencies:
- name: already-uptodate-chart
  version: 1.0.0
`,
			},
			updater: &HelmUpdater{
				Dependency: "already-uptodate-chart",
				Valuer:     value.StringValuer("1.0.0"),
			},
			expected: false,
			expectedFiles: map[string]string{
				filepath.Join("same-version", "Chart.yaml"): `
dependencies:
- name: already-uptodate-chart
  version: 1.0.0
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
				for expectedFilePath, expectedFileContent := range test.expectedFiles {
					actualFilePath := filepath.Join("testdata", expectedFilePath)
					actualFileContent, err := ioutil.ReadFile(actualFilePath)
					require.NoErrorf(t, err, "can't read actual testdata file %s", actualFilePath)
					assert.Equalf(t, expectedFileContent, string(actualFileContent), "testdata file %s doesn't match", actualFilePath)
				}
			}
		})
	}
}
