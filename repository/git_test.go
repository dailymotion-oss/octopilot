package repository

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/mholt/archiver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/src-d/go-git.v4"
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
				head, err := repo.Head()
				require.NoError(t, err)
				assert.Equal(t, "refs/heads/new-branch", head.Name().String())

				workTree, err := repo.Worktree()
				require.NoError(t, err)
				f, err := workTree.Filesystem.Open("master-branch.txt")
				require.NoError(t, err)
				data, err := ioutil.ReadAll(f)
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
				head, err := repo.Head()
				require.NoError(t, err)
				assert.Equal(t, "refs/heads/update", head.Name().String())

				workTree, err := repo.Worktree()
				require.NoError(t, err)
				f, err := workTree.Filesystem.Open("update-branch.txt")
				require.NoError(t, err)
				data, err := ioutil.ReadAll(f)
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
