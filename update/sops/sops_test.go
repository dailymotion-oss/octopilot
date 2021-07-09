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
	"go.mozilla.org/sops/v3/age"
	"go.mozilla.org/sops/v3/cmd/sops/common"
	"go.mozilla.org/sops/v3/cmd/sops/formats"
	"go.mozilla.org/sops/v3/decrypt"
	"go.mozilla.org/sops/v3/keys"

	"github.com/dailymotion-oss/octopilot/update/value"
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
			name: "add a new secret value",
			files: map[string]string{
				"new-secrets.yaml": `first-app:
    token: some-token
`,
			},
			updater: &SopsUpdater{
				FilePath: "new-secrets.yaml",
				Key:      "second-app.token",
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
		{
			name: "add a new deeply nested secret value",
			files: map[string]string{
				"new-secrets-deeply-nested.yaml": `first-app:
    token: some-token
`,
			},
			updater: &SopsUpdater{
				FilePath: "new-secrets-deeply-nested.yaml",
				Key:      "second-app.path.to.my.token",
				Valuer:   value.StringValuer("new-token"),
			},
			expected: true,
			expectedFiles: map[string]string{
				"new-secrets-deeply-nested.yaml": `first-app:
    token: some-token
second-app:
    path:
        to:
            my:
                token: new-token
`,
			},
		},
		{
			name: "add a new root secret value",
			files: map[string]string{
				"new-secrets-root.yaml": `first-app:
    token: some-token
`,
			},
			updater: &SopsUpdater{
				FilePath: "new-secrets-root.yaml",
				Key:      "newtoken",
				Valuer:   value.StringValuer("new-token-value"),
			},
			expected: true,
			expectedFiles: map[string]string{
				"new-secrets-root.yaml": `first-app:
    token: some-token
newtoken: new-token-value
`,
			},
		},
	}

	// we use https://age-encryption.org to encrypt/decrypt
	// an age key for unit-tests purpose was created with the following command:
	// $ age-keygen -o testdata/age.key
	// if you need to regenerate it, you'll also need to update its public key here:
	const (
		ageKeyFile   = "testdata/age.key"
		agePublicKey = "age16fvu9n7dkhdkrrrtfwctfzf94zvh58ars22k2fv9rmhkr9rkfszsyw8zzq"
	)
	os.Setenv("SOPS_AGE_KEY_FILE", ageKeyFile)
	masterKey, err := age.MasterKeyFromRecipient(agePublicKey)
	require.NoErrorf(t, err, "can't get age master key from pubkey %s", agePublicKey)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup
			{
				for filename, content := range test.files {
					format := formats.FormatForPath(filename)
					store := common.StoreForFormat(format)
					branches, err := store.LoadPlainFile([]byte(content))
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
					encryptedData, err := store.EmitEncryptedFile(tree)
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
				actualCleartextData, err := decrypt.DataWithFormat(actualEncryptedData, formats.FormatForPath(test.updater.FilePath))
				require.NoError(t, err, "can't decrypt actual encrypted content")
				expectedFileContent := test.expectedFiles[test.updater.FilePath]
				assert.Equal(t, expectedFileContent, string(actualCleartextData))
			}
		})
	}
}
