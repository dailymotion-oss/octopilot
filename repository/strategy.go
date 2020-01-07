package repository

import (
	"context"

	"github.com/google/go-github/v28/github"
)

type Strategy interface {
	Run(context.Context) (bool, *github.PullRequest, error)
}
