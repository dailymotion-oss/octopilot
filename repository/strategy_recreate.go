package repository

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/dailymotion/octopilot/update"
	"github.com/google/go-github/v28/github"
)

type RecreateStrategy struct {
	Repository Repository
	RepoPath   string
	Updaters   []update.Updater
	Options    UpdateOptions
}

func (s *RecreateStrategy) Run(ctx context.Context) (bool, *github.PullRequest, error) {
	gitRepo, err := cloneGitRepository(ctx, s.Repository.FullName(), s.RepoPath, s.Options.GitHub)
	if err != nil {
		return false, nil, fmt.Errorf("failed to clone repository %s: %w", s.Repository.FullName(), err)
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

	s.Options.Git.setDefaultValues(s.Updaters)
	s.Options.GitHub.setDefaultValues(s.Options.Git)

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
		GitHubToken: s.Options.GitHub.Token,
		BranchName:  branchName,
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
