package repository

import (
	"context"

	"github.com/google/go-github/v36/github"
)

type Strategy interface {
	Run(context.Context) (bool, *github.PullRequest, error)
}
