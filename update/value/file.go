package value

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
)

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

func (v FileValuer) Value(ctx context.Context, repoPath string) (string, error) {
	filePath := filepath.Join(repoPath, v.Path)
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", v.Path, err)
	}
	return string(content), nil
}
