package value

import (
	"context"
	"fmt"
	"regexp"

	"github.com/dailymotion/scribe/internal/parameters"
)

var (
	// name(params)
	valueRegexp = regexp.MustCompile(`(?P<name>[a-z]+)\((?P<params>.+)\)`)
)

type Valuer interface {
	Value(ctx context.Context, repoPath string) (string, error)
}

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
