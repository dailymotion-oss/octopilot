package glob

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/mattn/go-zglob"
)

// ExpandGlobPattern expands a glob pattern in the specified repository path.
// It reproduces similar error characteristics as filepath.Glob().
// Supports recursive glob patterns using **
func ExpandGlobPattern(repoPath, filePathPattern string) ([]string, error) {
	filePaths, err := zglob.Glob(filepath.Join(repoPath, filePathPattern))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{}, nil
		}
		return nil, err
	}

	return filePaths, nil
}
