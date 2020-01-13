package sops

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mozilla.org/sops/v3"
	"go.mozilla.org/sops/v3/aes"
	"go.mozilla.org/sops/v3/cmd/sops/common"
	"go.mozilla.org/sops/v3/cmd/sops/formats"
	"go.mozilla.org/sops/v3/decrypt"
	"go.mozilla.org/sops/v3/keys"
	"go.mozilla.org/sops/v3/pgp"

	"github.com/dailymotion/octopilot/update/value"
)

func TestNewUpdater(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		params           map[string]string
		expected         *SopsUpdater
		expectedErrorMsg string
	}{
		{
			name: "valid params",
			params: map[string]string{
				"file": "secrets.yaml",
				"key":  "path.to.key",
			},
			expected: &SopsUpdater{
				FilePath: "secrets.yaml",
				Key:      "path.to.key",
				Format:   formats.Yaml,
				Store:    common.StoreForFormat(formats.Yaml),
			},
		},
		{
			name: "custom format",
			params: map[string]string{
				"file":   "file.json.secrets",
				"key":    "path.to.key",
				"format": "json",
			},
			expected: &SopsUpdater{
				FilePath: "file.json.secrets",
				Key:      "path.to.key",
				Format:   formats.Json,
				Store:    common.StoreForFormat(formats.Json),
			},
		},
		{
			name:             "nil params",
			expectedErrorMsg: "missing file parameter",
		},
		{
			name: "missing mandatory file param",
			params: map[string]string{
				"key": "path.to.key",
			},
			expectedErrorMsg: "missing file parameter",
		},
		{
			name: "missing mandatory key param",
			params: map[string]string{
				"file": "secrets.yaml",
			},
			expectedErrorMsg: "missing key parameter",
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
		skipped          bool
		files            map[string]string
		updater          *SopsUpdater
		expected         bool
		expectedErrorMsg string
		expectedFiles    map[string]string
	}{
		{
			name: "update an existing secret value",
			files: map[string]string{
				"existing-secrets.yaml": `app:
    token: old-token
`,
			},
			updater: &SopsUpdater{
				FilePath: "existing-secrets.yaml",
				Key:      "app.token",
				Format:   formats.Yaml,
				Store:    common.StoreForFormat(formats.Yaml),
				Valuer:   value.StringValuer("new-token"),
			},
			expected: true,
			expectedFiles: map[string]string{
				"existing-secrets.yaml": `app:
    token: new-token
`,
			},
		},
		{
			name: "no change to an existing secret value",
			files: map[string]string{
				"no-changes-secrets.yaml": `app:
    token: good-token
`,
			},
			updater: &SopsUpdater{
				FilePath: "no-changes-secrets.yaml",
				Key:      "app.token",
				Format:   formats.Yaml,
				Store:    common.StoreForFormat(formats.Yaml),
				Valuer:   value.StringValuer("good-token"),
			},
			expected: false,
			expectedFiles: map[string]string{
				"no-changes-secrets.yaml": `app:
    token: good-token
`,
			},
		},
		{
			name:    "add a new secret value",
			skipped: true,
			files: map[string]string{
				"new-secrets.yaml": `first-app:
    token: some-token
`,
			},
			updater: &SopsUpdater{
				FilePath: "new-secrets.yaml",
				Key:      "second-app.token",
				Format:   formats.Yaml,
				Store:    common.StoreForFormat(formats.Yaml),
				Valuer:   value.StringValuer("new-token"),
			},
			expected: true,
			expectedFiles: map[string]string{
				"new-secrets.yaml": `first-app:
    token: some-token
second-app:
    token: new-token
`,
			},
		},
	}

	// we use GPG to encrypt/descrypt - a master key has already been generated in the following directory
	// see the testdata/README.md file for how to regenerate it if needed, and how to retrieve the fingerprint
	os.Setenv("GNUPGHOME", "testdata/.gnupg")
	masterKey := pgp.NewMasterKeyFromFingerprint("F7D394865A2FE709")

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.skipped {
				t.Skip()
			}

			{
				for filename, content := range test.files {
					branches, err := test.updater.Store.LoadPlainFile([]byte(content))
					require.NoErrorf(t, err, "can't parse data for file %s", filename)
					tree := sops.Tree{
						FilePath: filename,
						Metadata: sops.Metadata{
							KeyGroups: []sops.KeyGroup{
								[]keys.MasterKey{masterKey},
							},
							Version: "3.5.0",
						},
						Branches: branches,
					}
					dataKey, errs := tree.GenerateDataKey()
					require.Len(t, errs, 0)
					tree.Metadata.DataKey = dataKey
					err = common.EncryptTree(common.EncryptTreeOpts{
						Cipher:  aes.NewCipher(),
						DataKey: dataKey,
						Tree:    &tree,
					})
					require.NoErrorf(t, err, "failed to encrypt file %s", filename)
					encryptedData, err := test.updater.Store.EmitEncryptedFile(tree)
					require.NoErrorf(t, err, "failed to generate encrypted file %s", filename)
					err = ioutil.WriteFile(filepath.Join("testdata", filename), encryptedData, 0644)
					require.NoErrorf(t, err, "failed to write encrypted data to file %s", filename)
				}
			}

			actual, err := test.updater.Update(context.Background(), "testdata")
			if len(test.expectedErrorMsg) > 0 {
				require.EqualError(t, err, test.expectedErrorMsg)
				assert.False(t, actual)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, actual)

				actualEncryptedData, err := ioutil.ReadFile(filepath.Join("testdata", test.updater.FilePath))
				require.NoError(t, err, "can't read actual encrypted file")
				actualCleartextData, err := decrypt.DataWithFormat(actualEncryptedData, test.updater.Format)
				require.NoError(t, err, "can't decrypt actual encrypted content")
				expectedFileContent := test.expectedFiles[test.updater.FilePath]
				assert.Equal(t, expectedFileContent, string(actualCleartextData))
			}
		})
	}
}
