package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/mholt/archiver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadLocalRepository(repository string, destination string) (*git.Repository, error) {
	var err error

	gitRepoPath := filepath.Join("testdata", "head-resolution", repository)
	err = os.RemoveAll(gitRepoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to delete %s", gitRepoPath)
	}

	err = archiver.Unarchive(filepath.Join("testdata", "head-resolution", fmt.Sprintf("%s.tar.gz", repository)), gitRepoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress git repository at %s", gitRepoPath)
	}

	err = os.RemoveAll(filepath.Join("testdata", "head-resolution", destination))
	if err != nil {
		return nil, err
	}

	repo, err := git.PlainClone(filepath.Join("testdata", "head-resolution", destination), false, &git.CloneOptions{
		URL: gitRepoPath,
	})
	return repo, err
}

func TestAdjustOptionsFromGitRepository(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                  string
		pullRequestBaseBranch string
		repositoryLoader      func(string) (*git.Repository, error)
		validateFunc          func(*testing.T, error, GitHubOptions)
	}{
		{
			name:                  "explicit",
			pullRequestBaseBranch: "master",
			repositoryLoader: func(testName string) (*git.Repository, error) {
				return loadLocalRepository("resolution-successful", testName)
			},
			validateFunc: func(t *testing.T, result error, options GitHubOptions) {
				t.Helper()

				require.NoError(t, result)
				assert.Equal(t, "master", options.PullRequest.BaseBranch)
			},
		},
		{
			name:                  "resolved",
			pullRequestBaseBranch: "",
			repositoryLoader: func(testName string) (*git.Repository, error) {
				return loadLocalRepository("resolution-successful", testName)
			},
			validateFunc: func(t *testing.T, result error, options GitHubOptions) {
				t.Helper()

				require.NoError(t, result)
				assert.Equal(t, "main", options.PullRequest.BaseBranch)
			},
		},
		{
			name:                  "resolved-null-repository",
			pullRequestBaseBranch: "",
			repositoryLoader: func(testName string) (*git.Repository, error) {
				return nil, nil
			},
			validateFunc: func(t *testing.T, result error, options GitHubOptions) {
				t.Helper()

				require.Error(t, result)
				assert.Equal(t, "", options.PullRequest.BaseBranch)
			},
		},
		{
			name:                  "resolved-head-error",
			pullRequestBaseBranch: "",
			repositoryLoader: func(testName string) (*git.Repository, error) {
				return git.Init(memory.NewStorage(), nil)
			},
			validateFunc: func(t *testing.T, result error, options GitHubOptions) {
				t.Helper()

				require.Error(t, result)
				assert.Equal(t, "", options.PullRequest.BaseBranch)
			},
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			repo, err := test.repositoryLoader(test.name)
			require.NoError(t, err, "failed to load repository: %w", err)

			options := GitHubOptions{
				PullRequest: PullRequestOptions{
					BaseBranch: test.pullRequestBaseBranch,
				},
			}

			err = options.adjustOptionsFromGitRepository(repo)

			test.validateFunc(t, err, options)
		})
	}
}
