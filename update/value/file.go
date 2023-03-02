package value

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// FileValuer is a valuer that returns the content of a specific file.
type FileValuer struct {
	Path string
}

func newFileValuer(params map[string]string) (*FileValuer, error) {
	valuer := &FileValuer{}

	valuer.Path = params["path"]
	if len(valuer.Path) == 0 {
		return nil, errors.New("missing path parameter")
	}

	return valuer, nil
}

// Value returns the value to replace while updating files in the given repository.
func (v FileValuer) Value(_ context.Context, repoPath string) (string, error) {
	filePath := v.Path
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(repoPath, v.Path)
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", v.Path, err)
	}
	return string(content), nil
}
