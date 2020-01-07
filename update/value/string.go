package value

import (
	"context"
)

type StringValuer string

func (v StringValuer) Value(ctx context.Context, repoPath string) (string, error) {
	return string(v), nil
}
