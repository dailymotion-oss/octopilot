package repository

import (
	"github.com/dailymotion-oss/octopilot/update"
)

// ResetStrategy is a strategy implementation that resets any existing Pull Request from the base branch.
// So it will try to find a matching PR first, and use it (its branch) - but it will "reset" the branch from the base branch. And it will update the existing PR - or create a new one.
func NewResetStrategy(repository Repository, repoPath string, updaters []update.Updater, options UpdateOptions) *Strategy {
	return &Strategy{
		Repository:              repository,
		RepoPath:                repoPath,
		Updaters:                updaters,
		Options:                 options,
		FindMatchingPullRequest: true,
		DefaultUpdateOperation:  ReplaceUpdateOperation,
		ResetFromBase:           true,
	}
}
