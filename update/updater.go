package update

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/dailymotion-oss/octopilot/internal/parameters"
	"github.com/dailymotion-oss/octopilot/update/exec"
	"github.com/dailymotion-oss/octopilot/update/helm"
	"github.com/dailymotion-oss/octopilot/update/regex"
	"github.com/dailymotion-oss/octopilot/update/sops"
	"github.com/dailymotion-oss/octopilot/update/value"
	"github.com/dailymotion-oss/octopilot/update/yaml"
	"github.com/dailymotion-oss/octopilot/update/yq"
)

var (
	// name(params)
	updaterRegexp = regexp.MustCompile(`^(?P<name>[a-z]+)\((?P<params>.+)\)`)

	// name(params)=value
	updaterWithValueRegexp = regexp.MustCompile(`^(?P<name>[a-z]+)\((?P<params>.+)\)=(?P<value>.*)$`)
)

// Updater updates a git repository
type Updater interface {
	// Update updates the repository cloned at the given path, and returns true if changes have been made
	Update(ctx context.Context, repoPath string) (bool, error)
	// Message returns the default title and body that should be used in the commits / pull requests
	Message() (title, body string)
	// String returns a string representation of the updater
	String() string
}

// Parse parses a set of updates defined as string - from the CLI for example - and returns properly formatted Updaters.
// expected syntax is documented in the user documentation: docs/current-version/content/updaters/
func Parse(updates []string) ([]Updater, error) {
	var updaters []Updater

	for _, update := range updates {
		if len(strings.TrimSpace(update)) == 0 {
			continue
		}

		matches := updaterRegexp.FindStringSubmatch(update)
		if len(matches) < 2 {
			return nil, fmt.Errorf("invalid syntax for %s: missing updater name", update)
		}
		updaterName := matches[1]
		var paramsStr, valueStr string

		switch updaterName {
		case "exec", "yq":
			if len(matches) < 3 {
				return nil, fmt.Errorf("invalid syntax for %s: found %d matches instead of 3: %v", update, len(matches), matches)
			}
			paramsStr = matches[2]
		default:
			matches = updaterWithValueRegexp.FindStringSubmatch(update)
			if len(matches) < 4 {
				return nil, fmt.Errorf("invalid syntax for %s: found %d matches instead of 4: %v", update, len(matches), matches)
			}
			paramsStr = matches[2]
			valueStr = matches[3]
		}

		// hack to fix the value if coming from shell expansion, which adds quotes around it
		if strings.HasPrefix(valueStr, "\"") && strings.HasSuffix(valueStr, "\"") {
			valueStr = strings.TrimPrefix(valueStr, "\"")
			valueStr = strings.TrimSuffix(valueStr, "\"")
		}

		params := parameters.Parse(paramsStr)
		valuer, err := value.ParseValuer(valueStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse value %s for %s: %w", valueStr, updaterName, err)
		}

		var updater Updater
		switch updaterName {
		case "regex":
			updater, err = regex.NewUpdater(params, valuer)
		case "sops":
			updater, err = sops.NewUpdater(params, valuer)
		case "helm":
			updater, err = helm.NewUpdater(params, valuer)
		case "yaml":
			updater, err = yaml.NewUpdater(params, valuer)
		case "yq":
			updater, err = yq.NewUpdater(params)
		case "exec":
			updater, err = exec.NewUpdater(params)
		default:
			return nil, fmt.Errorf("unknown updater %s", updaterName)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to create an updater instance for %s: %w", updaterName, err)
		}

		updaters = append(updaters, updater)
	}

	return updaters, nil
}
