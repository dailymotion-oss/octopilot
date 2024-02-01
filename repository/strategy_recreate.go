package repository

import (
	"context"
	"fmt"

	"github.com/dailymotion-oss/octopilot/update"
	"github.com/google/go-github/v57/github"
	"github.com/sirupsen/logrus"
)

// RecreateStrategy is a strategy implementation that always creates a new Pull Request - even if an existing one for the same labels already exists.
type RecreateStrategy struct {
	Repository Repository
	RepoPath   string
	Updaters   []update.Updater
	Options    UpdateOptions
}

// Run executes the strategy, and returns true if the repo was updated, and the created PR.
func (s *RecreateStrategy) Run(ctx context.Context) (bool, *github.PullRequest, error) {
	gitRepo, err := cloneGitRepository(ctx, s.Repository, s.RepoPath, s.Options.GitHub)
	if err != nil {
		return false, nil, fmt.Errorf("failed to clone repository %s: %w", s.Repository.FullName(), err)
	}

	err = s.Options.GitHub.adjustOptionsFromGitRepository(gitRepo)
	if err != nil {
		return false, nil, fmt.Errorf("failed to adjust options for repository %s: %w", s.Repository.FullName(), err)
	}

	branchName := s.Repository.newBranchName(s.Options.Git.BranchPrefix)
	err = switchBranch(ctx, gitRepo, switchBranchOptions{
		BranchName:   branchName,
		CreateBranch: true,
	})
	if err != nil {
		return false, nil, fmt.Errorf("failed to switch to branch %s: %w", branchName, err)
	}

	repoUpdated, err := s.Repository.runUpdaters(ctx, s.Updaters, s.RepoPath)
	if err != nil {
		return false, nil, fmt.Errorf("failed to update repository %s: %w", s.Repository.FullName(), err)
	}
	if !repoUpdated {
		return false, nil, nil
	}

	if err = s.Options.Git.setDefaultValues(s.Updaters, templateExecutorFor(s.Options, s.Repository, s.RepoPath)); err != nil {
		return false, nil, fmt.Errorf("failed to set default git values: %w", err)
	}
	if err = s.Options.GitHub.setDefaultValues(s.Options.Git, templateExecutorFor(s.Options, s.Repository, s.RepoPath)); err != nil {
		return false, nil, fmt.Errorf("failed to set default github values: %w", err)
	}

	changesCommitted, err := commitChanges(ctx, gitRepo, s.Options)
	if err != nil {
		return false, nil, fmt.Errorf("failed to commit changes to git repository %s: %w", s.Repository.FullName(), err)
	}
	if !changesCommitted {
		logrus.WithField("repository", s.Repository.FullName()).Debug("No changes recorded, nothing to push")
		return false, nil, nil
	}
	if s.Options.DryRun {
		logrus.WithField("repository", s.Repository.FullName()).Warning("Running in dry-run mode, not pushing changes")
		return false, nil, nil
	}

	err = pushChanges(ctx, gitRepo, pushOptions{
		GitHubOpts: s.Options.GitHub,
		BranchName: branchName,
	})
	if err != nil {
		return false, nil, fmt.Errorf("failed to push changes to git repository %s: %w", s.Repository.FullName(), err)
	}

	pr, err := s.Repository.createPullRequest(ctx, s.Options.GitHub, branchName)
	if err != nil {
		return false, nil, fmt.Errorf("failed to create Pull Request: %w", err)
	}

	return true, pr, nil
}
