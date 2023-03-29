package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// note that this test can't be run in parallel, because it needs to change the working directory
func TestFindGitDirectory(t *testing.T) {
	baseDir, err := os.Getwd()
	require.NoError(t, err, "failed to get the current working directory")
	defer os.Chdir(baseDir) //nolint: errcheck // don't care in the tests...

	// copy the "git" dir to ".git", because otherwise it's a pain to commit a .git dir ;-)
	err = copy.Copy(
		filepath.Join(baseDir, "testdata", "dir-with-git-config", "git"),
		filepath.Join(baseDir, "testdata", "dir-with-git-config", ".git"),
	)
	require.NoError(t, err, "failed to copy the .git directory")

	// create the empty directory
	emptyDirectoryPath := filepath.Join(baseDir, "testdata", "empty-dir")
	err = os.MkdirAll(emptyDirectoryPath, 0755)
	require.NoErrorf(t, err, "failed to create an empty directory at %s", emptyDirectoryPath)

	tests := []struct {
		name       string
		workingDir string
		gitDir     string
	}{
		{
			name:       "dir with git config",
			workingDir: filepath.Join(baseDir, "testdata", "dir-with-git-config"),
			gitDir:     filepath.Join(baseDir, "testdata", "dir-with-git-config", ".git"),
		},
		{
			name:       "empty dir walks up to project's git directory",
			workingDir: emptyDirectoryPath,
			gitDir:     filepath.Join(baseDir, "..", "..", ".git"),
		},
	}

	for _, test := range tests {
		err = os.Chdir(test.workingDir)
		require.NoErrorf(t, err, "failed to switch current directory to %s", test.workingDir)

		t.Run(test.name, func(t *testing.T) {
			gitDir, err := findGitDirectory()
			require.NoError(t, err)
			assert.Equal(t, test.gitDir, gitDir)
		})
	}
}

// note that this test can't be run in parallel, because it needs to change the working directory
func TestCurrentRepositoryURL(t *testing.T) {
	baseDir, err := os.Getwd()
	require.NoError(t, err, "failed to get the current working directory")
	defer os.Chdir(baseDir) //nolint: errcheck // don't care in the tests...

	gitDir := filepath.Join(baseDir, "testdata", "dir-with-git-config")
	err = os.Chdir(gitDir)
	require.NoErrorf(t, err, "failed to switch current directory to %s", gitDir)

	repoURL := CurrentRepositoryURL()
	assert.Equal(t, "https://github.com/dailymotion-oss/octopilot", repoURL)
}
