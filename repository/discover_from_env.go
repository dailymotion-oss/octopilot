package repository

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/imdario/mergo"
)

func discoverRepositoriesFromEnvironment(ctx context.Context, envVar string, params map[string]string, githubToken string) ([]Repository, error) {
	separator := params["sep"]
	if len(separator) == 0 {
		separator = " "
	}
	delete(params, "sep")

	envValue := os.Getenv(envVar)
	repoNames := strings.Split(envValue, separator)

	if len(repoNames) == 0 {
		return nil, nil
	}
	if len(repoNames) == 1 && len(repoNames[0]) == 0 {
		return nil, nil
	}

	repos, err := Parse(ctx, repoNames, githubToken)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %v: %w", repoNames, err)
	}

	for i := range repos {
		err = mergo.Merge(&repos[i].Params, params)
		if err != nil {
			return nil, fmt.Errorf("failed to merge params for repo %v: %w", repos[i], err)
		}
	}

	return repos, nil
}
