package value

import (
	"context"
)

// StringValuer is a valuer to replace a string.
type StringValuer string

// Value returns the value to replace while updating files in the given repository.
func (v StringValuer) Value(_ context.Context, _ string) (string, error) {
	return string(v), nil
}
