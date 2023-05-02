package repository

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/go-git/go-git/v5"
	"github.com/mholt/archiver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSwitchBranch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		options      switchBranchOptions
		validateFunc func(*testing.T, *git.Repository)
	}{
		{
			name: "create-new-branch",
			options: switchBranchOptions{
				BranchName:   "new-branch",
				CreateBranch: true,
			},
			validateFunc: func(t *testing.T, repo *git.Repository) {
				t.Helper()

				head, err := repo.Head()
				require.NoError(t, err)
				assert.Equal(t, "refs/heads/new-branch", head.Name().String())

				workTree, err := repo.Worktree()
				require.NoError(t, err)
				f, err := workTree.Filesystem.Open("master-branch.txt")
				require.NoError(t, err)
				data, err := io.ReadAll(f)
				require.NoError(t, err)
				assert.Equal(t, "this is a file from the master branch\n", string(data))
			},
		},
		{
			name: "existing-branch",
			options: switchBranchOptions{
				BranchName: "update",
			},
			validateFunc: func(t *testing.T, repo *git.Repository) {
				t.Helper()

				head, err := repo.Head()
				require.NoError(t, err)
				assert.Equal(t, "refs/heads/update", head.Name().String())

				workTree, err := repo.Worktree()
				require.NoError(t, err)
				f, err := workTree.Filesystem.Open("update-branch.txt")
				require.NoError(t, err)
				data, err := io.ReadAll(f)
				require.NoError(t, err)
				assert.Equal(t, "this is a file from the update branch\n", string(data))
			},
		},
	}

	// the git repo is stored as a tar.gz archive to make it easy to commit
	gitRepoPath := filepath.Join("testdata", "switch-branch", "git-repo")
	err := os.RemoveAll(gitRepoPath)
	require.NoErrorf(t, err, "failed to delete %s", gitRepoPath)
	err = archiver.Unarchive(filepath.Join("testdata", "switch-branch", "git-repo.tar.gz"), gitRepoPath)
	require.NoErrorf(t, err, "failed to decompress git repository at %s", gitRepoPath)

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			// remove any previous testing data
			err := os.RemoveAll(filepath.Join("testdata", "switch-branch", test.name))
			require.NoError(t, err)

			repo, err := git.PlainClone(filepath.Join("testdata", "switch-branch", test.name), false, &git.CloneOptions{
				URL: gitRepoPath,
			})
			require.NoError(t, err)

			head, err := repo.Head()
			require.NoError(t, err)
			assert.Equal(t, "refs/heads/master", head.Name().String())

			err = switchBranch(context.Background(), repo, test.options)
			require.NoError(t, err)

			test.validateFunc(t, repo)
		})
	}
}

func TestParseSigningKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                 string
		signingKeyPath       string
		signingKeyPassphrase string
		validateFunc         func(*testing.T, *openpgp.Entity, error)
	}{
		{
			name: "no-signing-key-path",
			validateFunc: func(t *testing.T, signingKey *openpgp.Entity, err error) {
				t.Helper()

				assert.Nil(t, signingKey)
				assert.NoError(t, err)
			},
		},
		{
			name:           "invalid-signing-key-path",
			signingKeyPath: "testdata/parse-signing-key/unknown-dir/private-key.pgp",
			validateFunc: func(t *testing.T, signingKey *openpgp.Entity, err error) {
				t.Helper()

				assert.Nil(t, signingKey)
				assert.Error(t, err)
			},
		},
		{
			name:           "invalid-signing-key-format",
			signingKeyPath: "testdata/parse-signing-key/invalid-format/private-key.pgp",
			validateFunc: func(t *testing.T, signingKey *openpgp.Entity, err error) {
				t.Helper()

				assert.Nil(t, signingKey)
				assert.Error(t, err)
			},
		},
		{
			name:           "valid-unencrypted",
			signingKeyPath: "testdata/parse-signing-key/valid-unencrypted/private-key.pgp",
			validateFunc: func(t *testing.T, signingKey *openpgp.Entity, err error) {
				t.Helper()

				assert.NotNil(t, signingKey)
				assert.NoError(t, err)
			},
		},
		{
			name:           "valid-encrypted-without-passphrase",
			signingKeyPath: "testdata/parse-signing-key/valid-encrypted/private-key.pgp",
			validateFunc: func(t *testing.T, signingKey *openpgp.Entity, err error) {
				t.Helper()

				assert.Nil(t, signingKey)
				assert.Error(t, err)
			},
		},
		{
			name:                 "valid-encrypted-with-passphrase",
			signingKeyPath:       "testdata/parse-signing-key/valid-encrypted/private-key.pgp",
			signingKeyPassphrase: "fake-passphrase",
			validateFunc: func(t *testing.T, signingKey *openpgp.Entity, err error) {
				t.Helper()

				assert.NotNil(t, signingKey)
				assert.NoError(t, err)
			},
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			signingKey, err := parseSigningKey(test.signingKeyPath, test.signingKeyPassphrase)
			test.validateFunc(t, signingKey, err)
		})
	}
}
