package repository

import (
	"github.com/dailymotion-oss/octopilot/update"
)

// AppendStrategy is a strategy that appends new commits to any existing Pull Request.
// So it will try to find a matching PR first, and use it (its branch). Then it will commit on this branch, and update the existing PR - or create a new one if there is no matching PR.
func NewAppendStrategy(repository Repository, repoPath string, updaters []update.Updater, options UpdateOptions) *Strategy {
	return &Strategy{
		Repository:              repository,
		RepoPath:                repoPath,
		Updaters:                updaters,
		Options:                 options,
		FindMatchingPullRequest: true,
		DefaultUpdateOperation:  IgnoreUpdateOperation,
		ResetFromBase:           false,
	}
}
