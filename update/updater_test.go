package update

import (
	"regexp"
	"testing"

	"github.com/dailymotion-oss/octopilot/update/exec"
	"github.com/dailymotion-oss/octopilot/update/helm"
	"github.com/dailymotion-oss/octopilot/update/regex"
	"github.com/dailymotion-oss/octopilot/update/sops"
	"github.com/dailymotion-oss/octopilot/update/value"
	"github.com/dailymotion-oss/octopilot/update/yaml"
	"github.com/dailymotion-oss/octopilot/update/yq"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mozilla.org/sops/v3/cmd/sops/common"
	"go.mozilla.org/sops/v3/cmd/sops/formats"
)

func TestParse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		updates          []string
		expected         []Updater
		expectedErrorMsg string
	}{
		{
			name: "nil input",
		},
		{
			name:    "empty input",
			updates: []string{""},
		},
		{
			name:             "invalid input",
			updates:          []string{"whatever"},
			expectedErrorMsg: "invalid syntax for whatever: missing updater name",
		},
		{
			name:             "invalid updater syntax input",
			updates:          []string{"sops(key=value)"},
			expectedErrorMsg: "invalid syntax for sops(key=value): found 0 matches instead of 4: []",
		},
		{
			name:             "invalid regex updater syntax input",
			updates:          []string{"regex(key=value)=value"},
			expectedErrorMsg: "failed to create an updater instance for regex: missing file parameter",
		},
		{
			name:             "invalid valuer syntax input",
			updates:          []string{"helm(dependency=my-chart)=whatever(key=value)"},
			expectedErrorMsg: "failed to parse value whatever(key=value) for helm: unknown valuer whatever",
		},
		{
			name:             "unknown updater",
			updates:          []string{"whatever(key=value)=value"},
			expectedErrorMsg: "unknown updater whatever",
		},
		{
			name:    "single exec updater",
			updates: []string{"exec(cmd=something)"},
			expected: []Updater{
				&exec.ExecUpdater{
					Command: "something",
				},
			},
		},
		{
			name:    "single helm updater",
			updates: []string{"helm(dependency=my-chart)=1.2.3"},
			expected: []Updater{
				&helm.HelmUpdater{
					Dependency: "my-chart",
					Valuer:     value.StringValuer("1.2.3"),
					Indent:     2,
				},
			},
		},
		{
			name:    "single yq updater",
			updates: []string{`yq(file=values.yaml,expression='.path.to.subkey = "value"')`},
			expected: []Updater{
				&yq.YQUpdater{
					FilePath:     "values.yaml",
					Expression:   `.path.to.subkey = "value"`,
					Output:       "",
					OutputToJSON: false,
					Trim:         false,
					UnwrapScalar: true,
					Indent:       2,
				},
			},
		},
		{
			name:    "complex yaml updater",
			updates: []string{"yaml(file=values.yaml,path='array.(key==prefix*).**.subkey',trim=true,indent=4)=1.2.3"},
			expected: []Updater{
				&yaml.YamlUpdater{
					FilePath: "values.yaml",
					Path:     "array.(key==prefix*).**.subkey",
					Valuer:   value.StringValuer("1.2.3"),
					Trim:     true,
					Indent:   4,
				},
			},
		},
		{
			name: "regex and sops updaters",
			updates: []string{
				`regex(file=helmfile.yaml,pattern='chart: example/my-chart\s+version: \"(.*)\"')=file(path=VERSION)`,
				`sops(file=certificates/secrets.yaml,key=certificates.b64encKey)=e30k`,
			},
			expected: []Updater{
				&regex.RegexUpdater{
					FilePath: "helmfile.yaml",
					Pattern:  `chart: example/my-chart\s+version: \"(.*)\"`,
					Regexp:   regexp.MustCompile(`chart: example/my-chart\s+version: \"(.*)\"`),
					Valuer: &value.FileValuer{
						Path: "VERSION",
					},
				},
				&sops.SopsUpdater{
					FilePath: "certificates/secrets.yaml",
					Format:   formats.Yaml,
					Key:      "certificates.b64encKey",
					Store:    common.StoreForFormat(formats.Yaml),
					Valuer:   value.StringValuer("e30k"),
				},
			},
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual, err := Parse(test.updates)
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
