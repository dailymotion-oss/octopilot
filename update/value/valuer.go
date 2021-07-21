package value

import (
	"context"
	"fmt"
	"regexp"

	"github.com/dailymotion-oss/octopilot/internal/parameters"
)

var (
	// name(params)
	valueRegexp = regexp.MustCompile(`(?P<name>[a-z]+)\((?P<params>.+)\)`)
)

// Valuer is the interface for retrieving a value to replace while updating files.
type Valuer interface {
	// Value returns the value to replace while updating files in the given repository.
	Value(ctx context.Context, repoPath string) (string, error)
}

// ParseValuer parses the valuer defined as string - from the CLI for example - and returns a properly formatted valuer.
func ParseValuer(valueStr string) (Valuer, error) {
	matches := valueRegexp.FindStringSubmatch(valueStr)
	if len(matches) == 0 {
		return StringValuer(valueStr), nil
	}

	if len(matches) < 3 {
		return nil, fmt.Errorf("invalid syntax for %s", valueStr)
	}
	valuerName := matches[1]
	paramsStr := matches[2]

	params := parameters.Parse(paramsStr)

	var (
		valuer Valuer
		err    error
	)
	switch valuerName {
	case "file":
		valuer, err = newFileValuer(params)
	default:
		return nil, fmt.Errorf("unknown valuer %s", valuerName)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create a valuer instance for %s: %w", valuerName, err)
	}

	return valuer, nil
}
