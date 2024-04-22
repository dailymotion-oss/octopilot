package repository

import (
	"context"
	"fmt"

	"github.com/dailymotion-oss/octopilot/update"
	"github.com/google/go-github/v57/github"
	"github.com/sirupsen/logrus"
)

// Strategy defines how the pull request will be created or updated if one already exists.
type Strategy struct {
	Repository              Repository
	RepoPath                string
	Updaters                []update.Updater
	Options                 UpdateOptions
	FindMatchingPullRequest bool
	DefaultUpdateOperation  string
	ForcePush               bool
	ForceBranchCreation     bool
}

// Run executes the strategy. It returns:
// - a boolean indicating whether changes have been made to the repository
// - a pull request if one has been created (or updated)
func (s *Strategy) Run(ctx context.Context) (bool, *github.PullRequest, error) {
	gitRepo, err := cloneGitRepository(ctx, s.Repository, s.RepoPath, s.Options.GitHub)
	if err != nil {
		return false, nil, fmt.Errorf("failed to clone repository %s: %w", s.Repository.FullName(), err)
	}

	err = s.Options.GitHub.adjustOptionsFromGitRepository(gitRepo)
	if err != nil {
		return false, nil, fmt.Errorf("failed to adjust options for repository %s: %w", s.Repository.FullName(), err)
	}

	var existingPR *github.PullRequest
	if s.FindMatchingPullRequest {
		existingPR, err = s.Repository.findMatchingPullRequest(ctx, s.Options.GitHub)
		if err != nil {
			return false, nil, fmt.Errorf("failed to find matching pull request for repository %s: %w", s.Repository.FullName(), err)
		}
	}

	var branchName string
	if existingPR != nil {
		branchName = existingPR.Head.GetRef()
	} else {
		branchName = s.Repository.newBranchName(s.Options.Git.BranchPrefix)
	}
	err = switchBranch(ctx, gitRepo, switchBranchOptions{
		BranchName:   branchName,
		CreateBranch: s.ForceBranchCreation || existingPR == nil,
	})
	if err != nil {
		return false, existingPR, fmt.Errorf("failed to switch to branch %s: %w", branchName, err)
	}

	repoUpdated, err := s.Repository.runUpdaters(ctx, s.Updaters, s.RepoPath)
	if err != nil {
		return false, existingPR, fmt.Errorf("failed to update repository %s: %w", s.Repository.FullName(), err)
	}
	if !repoUpdated {
		return false, existingPR, nil
	}

	if err = s.Options.Git.setDefaultValues(s.Updaters, templateExecutorFor(s.Options, s.Repository, s.RepoPath)); err != nil {
		return false, existingPR, fmt.Errorf("failed to set default git values: %w", err)
	}
	if err = s.Options.GitHub.setDefaultValues(s.Options.Git, templateExecutorFor(s.Options, s.Repository, s.RepoPath)); err != nil {
		return false, existingPR, fmt.Errorf("failed to set default github values: %w", err)
	}
	if len(s.DefaultUpdateOperation) > 0 {
		s.Options.GitHub.setDefaultUpdateOperation(IgnoreUpdateOperation)
	}

	changesCommitted, err := commitChanges(ctx, gitRepo, s.Options)
	if err != nil {
		return false, existingPR, fmt.Errorf("failed to commit changes to git repository %s: %w", s.Repository.FullName(), err)
	}
	if !changesCommitted {
		logrus.WithField("repository", s.Repository.FullName()).Debug("No changes recorded, nothing to push")
		return false, existingPR, nil
	}
	if s.Options.DryRun {
		logrus.WithField("repository", s.Repository.FullName()).Warning("Running in dry-run mode, not pushing changes")
		return false, existingPR, nil
	}

	err = pushChanges(ctx, gitRepo, pushOptions{
		GitHubOpts: s.Options.GitHub,
		BranchName: branchName,
		ForcePush:  s.ForcePush,
	})
	if err != nil {
		return false, existingPR, fmt.Errorf("failed to push changes to git repository %s: %w", s.Repository.FullName(), err)
	}

	var pr *github.PullRequest
	if existingPR != nil {
		pr, err = s.Repository.updatePullRequest(ctx, s.Options.GitHub, existingPR)
	} else {
		pr, err = s.Repository.createPullRequest(ctx, s.Options.GitHub, branchName)
	}
	if err != nil {
		return false, existingPR, fmt.Errorf("failed to create or update Pull Request: %w", err)
	}

	return true, pr, nil
}
