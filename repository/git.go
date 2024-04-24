package repository

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/sirupsen/logrus"
)

func cloneGitRepository(ctx context.Context, repo Repository, localPath string, options GitHubOptions) (*git.Repository, error) {
	gitURL, err := url.JoinPath(options.URL, repo.GitFullName())
	if err != nil {
		// likely the Url passed is malformed
		return nil, fmt.Errorf("invalid github url format: %w", err)
	}

	branch := "HEAD"
	if b, ok := repo.Params["branch"]; ok && strings.TrimSpace(b) != "" {
		branch = fmt.Sprintf("refs/heads/%s", b)
	}
	referenceName := plumbing.ReferenceName(branch)
	logrus.WithFields(logrus.Fields{
		"git-url":       gitURL,
		"git-reference": referenceName.String(),
		"local-path":    localPath,
	}).Trace("Cloning git repository")

	_, token, err := githubClient(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create github client: %w", err)
	}

	gitRepo, err := git.PlainCloneContext(ctx, localPath, false, &git.CloneOptions{
		ReferenceName: referenceName,
		URL:           gitURL,
		Auth: &http.BasicAuth{
			Username: "x-access-token", // For GitHub Apps, the username must be `x-access-token`. For Personal Tokens, it doesn't matter.
			Password: token,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to clone git repository from %s to %s: %w", gitURL, localPath, err)
	}

	logrus.WithFields(logrus.Fields{
		"git-url":       gitURL,
		"git-reference": referenceName.String(),
		"local-path":    localPath,
	}).Debug("Git repository cloned")

	return gitRepo, nil
}

type switchBranchOptions struct {
	Repository   Repository
	BranchName   string
	CreateBranch bool
}

func switchBranch(_ context.Context, gitRepo *git.Repository, opts switchBranchOptions) error {
	branchRefName := plumbing.NewBranchReferenceName(opts.BranchName)

	if !opts.CreateBranch {
		// for an existing branch, we need to create a local reference to the remote branch
		remoteBranchRefName := plumbing.NewRemoteReferenceName("origin", opts.BranchName)
		remoteBranchRef, err := gitRepo.Reference(remoteBranchRefName, true)
		if err != nil {
			return fmt.Errorf("failed to get the reference for %s: %w", remoteBranchRef, err)
		}

		branchRef := plumbing.NewHashReference(branchRefName, remoteBranchRef.Hash())
		err = gitRepo.Storer.SetReference(branchRef)
		if err != nil {
			return fmt.Errorf("failed to store the reference for branch %s: %w", opts.BranchName, err)
		}
	}

	workTree, err := gitRepo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to open worktree: %w", err)
	}

	if err := workTree.Checkout(&git.CheckoutOptions{
		Branch: branchRefName,
		Create: opts.CreateBranch,
	}); err != nil {
		return fmt.Errorf("failed to checkout the branch %s: %w", opts.BranchName, err)
	}

	logrus.WithFields(logrus.Fields{
		"repository": opts.Repository.FullName(),
		"branch":     opts.BranchName,
	}).Debug("Switched Git branch")
	return nil
}

type commitOptions struct {
	Repository    Repository
	CommitMessage CommitMessage
	GitOpts       GitOptions
}

func commitChanges(_ context.Context, gitRepo *git.Repository, opts commitOptions) (bool, error) {
	workTree, err := gitRepo.Worktree()
	if err != nil {
		return false, fmt.Errorf("failed to open worktree: %w", err)
	}

	status, err := workTree.Status()
	if err != nil {
		return false, fmt.Errorf("failed to get the worktree status: %w", err)
	}
	if status.IsClean() {
		return false, nil
	}
	logrus.WithFields(logrus.Fields{
		"repository": opts.Repository.FullName(),
		"status":     status.String(),
	}).Debug("Git status")

	for _, pattern := range opts.GitOpts.StagePatterns {
		err = workTree.AddGlob(pattern)
		if err != nil {
			return false, fmt.Errorf("failed to stage files using pattern %s: %w", pattern, err)
		}
	}

	signingKey, err := parseSigningKey(opts.GitOpts.SigningKeyPath, opts.GitOpts.SigningKeyPassphrase)
	if err != nil {
		return false, err
	}

	now := time.Now()

	commit, err := workTree.Commit(opts.CommitMessage.String(),
		&git.CommitOptions{
			All: opts.GitOpts.StageAllChanged,
			Author: &object.Signature{
				Name:  opts.GitOpts.AuthorName,
				Email: opts.GitOpts.AuthorEmail,
				When:  now,
			},
			Committer: &object.Signature{
				Name:  opts.GitOpts.CommitterName,
				Email: opts.GitOpts.CommitterEmail,
				When:  now,
			},
			SignKey: signingKey,
		},
	)
	if err != nil {
		return false, fmt.Errorf("failed to commit: %w", err)
	}
	logrus.WithFields(logrus.Fields{
		"repository": opts.Repository.FullName(),
		"commit":     commit.String(),
	}).Debug("Git commit")

	return true, nil
}

func parseSigningKey(signingKeyPath, signingKeyPassphrase string) (*openpgp.Entity, error) {
	if signingKeyPath == "" {
		return nil, nil
	}

	b, err := os.ReadFile(signingKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read signing key file %q: %w", signingKeyPath, err)
	}

	el, err := openpgp.ReadArmoredKeyRing(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("failed to read signing key file content: %w", err)
	}
	signingKey := el[0]

	if signingKeyPassphrase != "" {
		if err := signingKey.PrivateKey.Decrypt([]byte(signingKeyPassphrase)); err != nil {
			return nil, fmt.Errorf("failed to decrypt signing key: %w", err)
		}
	} else if signingKey.PrivateKey.Encrypted {
		return nil, errors.New("signing key is encrypted, please provide a passphrase")
	}

	return signingKey, nil
}

type pushOptions struct {
	GitHubOpts    GitHubOptions
	Repository    Repository
	BranchName    string
	ResetFromBase bool
}

func pushChanges(ctx context.Context, gitRepo *git.Repository, opts pushOptions) error {
	refSpec := fmt.Sprintf("refs/heads/%[1]s:refs/heads/%[1]s", opts.BranchName)
	if opts.ResetFromBase {
		// https://git-scm.com/book/en/v2/Git-Internals-The-Refspec
		// The + tells Git to update the reference even if it isn’t a fast-forward.
		refSpec = fmt.Sprintf("+%s", refSpec)
	}

	_, token, err := githubClient(ctx, opts.GitHubOpts)
	if err != nil {
		return fmt.Errorf("failed to create github client: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"repository": opts.Repository.FullName(),
		"branch":     opts.BranchName,
		"force":      opts.ResetFromBase,
	}).Trace("Pushing git changes")
	err = gitRepo.PushContext(ctx, &git.PushOptions{
		RefSpecs: []config.RefSpec{
			config.RefSpec(refSpec),
		},
		Auth: &http.BasicAuth{
			Username: "x-access-token", // For GitHub Apps, the username must be `x-access-token`. For Personal Tokens, it doesn't matter.
			Password: token,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to push branch %s to %s: %w", opts.BranchName, opts.Repository.FullName(), err)
	}

	logrus.WithFields(logrus.Fields{
		"repository": opts.Repository.FullName(),
		"branch":     opts.BranchName,
	}).Debug("Git changes pushed")
	return nil
}
