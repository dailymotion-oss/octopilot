package repository

import (
	"github.com/dailymotion-oss/octopilot/update"
)

// RecreateStrategy is a strategy implementation that always creates a new Pull Request - even if an existing one for the same labels already exists.
func NewRecreateStrategy(repository Repository, repoPath string, updaters []update.Updater, options UpdateOptions) *Strategy {
	return &Strategy{
		Repository:              repository,
		RepoPath:                repoPath,
		Updaters:                updaters,
		Options:                 options,
		FindMatchingPullRequest: false,
		DefaultUpdateOperation:  "",
		ForcePush:               false,
		ForceBranchCreation:     false,
	}
}
