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
	ResetFromBase           bool
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
		Repository:   s.Repository,
		BranchName:   branchName,
		CreateBranch: s.ResetFromBase || existingPR == nil,
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

	commitMessage := NewCommitMessage(s.Options.Git.CommitTitle, s.Options.Git.CommitBody, s.Options.Git.CommitFooter)

	changesCommitted, err := commitChanges(ctx, gitRepo, commitOptions{
		Repository:    s.Repository,
		CommitMessage: commitMessage,
		GitOpts:       s.Options.Git,
	})
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
		GitHubOpts:    s.Options.GitHub,
		Repository:    s.Repository,
		BranchName:    branchName,
		ResetFromBase: s.ResetFromBase,
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
