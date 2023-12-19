package repository

import (
	"context"

	"github.com/google/go-github/v57/github"
)

// Strategy defines how the pull request will be created or updated if one already exists.
type Strategy interface {
	// Run executes the strategy. It returns:
	// - a boolean indicating whether changes have been made to the repository
	// - a pull request if one has been created (or updated)
	Run(context.Context) (bool, *github.PullRequest, error)
}
