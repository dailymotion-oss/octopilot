package repository

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/utils/merkletrie"
	"github.com/google/go-github/v57/github"
	"github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
)

func cloneGitRepository(ctx context.Context, repo Repository, localPath string, options UpdateOptions) (*git.Repository, error) {
	gitURL, err := url.JoinPath(options.GitHub.URL, repo.GitFullName())
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

	_, token, err := githubClient(ctx, options.GitHub)
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

	recurseSubmodules := git.NoRecurseSubmodules
	if options.Git.RecurseSubmodules {
		recurseSubmodules = git.DefaultSubmoduleRecursionDepth
	}

	githubURL, err := url.Parse(options.GitHub.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Github URL: %w", err)
	}

	err = initSubmodules(ctx, gitRepo, token, recurseSubmodules, githubURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize submodules for git repository %s: %w", gitURL, err)
	}

	logrus.WithFields(logrus.Fields{
		"git-url":       gitURL,
		"git-reference": referenceName.String(),
		"local-path":    localPath,
	}).Debug("Git repository cloned")

	return gitRepo, nil
}

func initSubmodules(ctx context.Context, repo *git.Repository, token string, recurseSubmodules git.SubmoduleRescursivity, githubURL *url.URL) error {
	if recurseSubmodules == git.NoRecurseSubmodules {
		return nil
	}
	recurseSubmodules--

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	subModules, err := wt.Submodules()
	if err != nil {
		return fmt.Errorf("failed to get submodules: %w", err)
	}

	for _, s := range subModules {
		// Hack: rewrite Github hosted submodule SSH URLs to use HTTPS because token auth only works with that
		// go-git does not expose a way of doing insteadOf type rewrites for submodules, so this will have to do for now
		s.Config().URL = strings.Replace(s.Config().URL, fmt.Sprintf("git@%s:", githubURL.Hostname()), githubURL.String(), 1)

		// Only use basic auth for Github. This lets us use any public Git repo not hosted on Github.
		var auth transport.AuthMethod
		if strings.HasPrefix(s.Config().URL, githubURL.String()) {
			auth = &http.BasicAuth{
				Username: "x-access-token",
				Password: token,
			}
		} else {
			auth = nil
		}

		logrus.WithFields(logrus.Fields{
			"submodule-name":   s.Config().Name,
			"submodule-url":    s.Config().URL,
			"submodule-branch": s.Config().Branch,
			"submodule-path":   s.Config().Path,
		}).Trace("Initializing submodule")

		err = s.UpdateContext(ctx, &git.SubmoduleUpdateOptions{
			Init: true,
			Auth: auth,
		})
		if err != nil {
			return fmt.Errorf("failed to initialize submodule: %w", err)
		}

		sRepo, err := s.Repository()
		if err != nil {
			return fmt.Errorf("failed to get submodule repo: %w", err)
		}

		err = initSubmodules(ctx, sRepo, token, recurseSubmodules, githubURL)
		if err != nil {
			return err
		}
	}

	return nil
}

type createBranchOptions struct {
	GitHubOpts GitHubOptions
	Repository Repository
	BranchName string
	CommitSHA  string
}

func createBranchWithAPI(ctx context.Context, opts createBranchOptions) error {
	client, _, err := githubClient(ctx, opts.GitHubOpts)
	if err != nil {
		return fmt.Errorf("failed to create github client: %w", err)
	}

	repository, _, err := client.Repositories.Get(ctx, opts.Repository.Owner, opts.Repository.Name)
	if err != nil {
		return fmt.Errorf("failed to fetch repository: %w", err)
	}

	gqlClient, err := githubGraphqlClient(ctx, opts.GitHubOpts)
	if err != nil {
		return fmt.Errorf("failed to create github GraphQL client: %w", err)
	}

	inputs := githubv4.CreateRefInput{
		RepositoryID: githubv4.ID(repository.NodeID),
		Name:         githubv4.String(fmt.Sprintf("refs/heads/%s", opts.BranchName)),
		Oid:          githubv4.GitObjectID(opts.CommitSHA),
	}

	var mutation struct {
		CreateRefInput struct {
			ClientMutationID string
		} `graphql:"createRef(input: $input)"`
	}

	err = gqlClient.Mutate(ctx, &mutation, inputs, nil)
	if err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}
	return nil
}

type resetBranchOptions struct {
	GitHubOpts GitHubOptions
	Repository Repository
	BranchName string
	CommitSHA  string
}

func resetBranchWithAPI(ctx context.Context, opts resetBranchOptions) error {
	client, _, err := githubClient(ctx, opts.GitHubOpts)
	if err != nil {
		return fmt.Errorf("failed to create github client: %w", err)
	}

	branchRef := fmt.Sprintf("refs/heads/%s", opts.BranchName)

	_, _, err = client.Git.UpdateRef(
		ctx,
		opts.Repository.Owner,
		opts.Repository.Name,
		&github.Reference{
			Ref: &branchRef,
			Object: &github.GitObject{
				SHA: &opts.CommitSHA,
			},
		},
		true,
	)
	if err != nil {
		return fmt.Errorf("failed to update branch ref: %w", err)
	}
	return nil
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

func getLatestCommit(_ context.Context, gitRepo *git.Repository) (*object.Commit, error) {
	headCommitRef, err := gitRepo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch head: %w", err)
	}

	latestCommit, err := gitRepo.CommitObject(headCommitRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch commit: %w", err)
	}
	return latestCommit, nil
}

func compareCommits(base, head *object.Commit) (*CommitFileChanges, error) {
	baseTree, err := base.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get base commit tree: %w", err)
	}

	headTree, err := head.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get head commit tree: %w", err)
	}

	changes, err := baseTree.Diff(headTree)
	if err != nil {
		return nil, fmt.Errorf("failed to compare commit trees: %w", err)
	}

	commitFileChanges := CommitFileChanges{}
	for _, change := range changes {
		action, err := change.Action()
		if err != nil {
			return nil, fmt.Errorf("failed to get commit change action: %w", err)
		}

		if action == merkletrie.Delete {
			commitFileChanges.Deleted = append(commitFileChanges.Deleted, change.From.Name)
		} else {
			commitFileChanges.Upserted = append(commitFileChanges.Upserted, change.To.Name)
		}
	}
	return &commitFileChanges, nil
}

func pushChangesWithAPI(ctx context.Context, gitRepo *git.Repository, opts pushOptions) error {
	commit, err := getLatestCommit(ctx, gitRepo)
	if err != nil {
		return fmt.Errorf("failed to fetch latest commit: %w", err)
	}

	parentCommit, err := commit.Parent(0)
	if err != nil {
		return fmt.Errorf("failed to fetch parent of latest commit: %w", err)
	}

	parentCommitSHA := parentCommit.Hash.String()

	if opts.CreateBranch {
		err = createBranchWithAPI(ctx, createBranchOptions{
			GitHubOpts: opts.GitHubOpts,
			Repository: opts.Repository,
			BranchName: opts.BranchName,
			CommitSHA:  parentCommitSHA,
		})
		if err != nil {
			return fmt.Errorf("failed to create branch: %w", err)
		}
	} else if opts.ResetFromBase {
		err = resetBranchWithAPI(ctx, resetBranchOptions{
			GitHubOpts: opts.GitHubOpts,
			Repository: opts.Repository,
			BranchName: opts.BranchName,
			CommitSHA:  parentCommitSHA,
		})
		if err != nil {
			return fmt.Errorf("failed to reset branch: %w", err)
		}
	}

	changes, err := compareCommits(parentCommit, commit)
	if err != nil {
		return fmt.Errorf("failed to compare commits: %w", err)
	}

	deletions := make([]githubv4.FileDeletion, 0, len(changes.Deleted))
	for _, path := range changes.Deleted {
		deletions = append(deletions, githubv4.FileDeletion{
			Path: githubv4.String(path),
		})
	}

	additions := make([]githubv4.FileAddition, 0, len(changes.Upserted))
	repoDirPath := filepath.Join(opts.GitCloneDir, opts.Repository.Owner, opts.Repository.Name)
	for _, path := range changes.Upserted {
		base64FileContent, err := base64EncodeFile(filepath.Join(repoDirPath, path))
		if err != nil {
			return fmt.Errorf("failed to encode file to base64: %w", err)
		}
		additions = append(additions, githubv4.FileAddition{
			Path:     githubv4.String(path),
			Contents: githubv4.Base64String(base64FileContent),
		})
	}

	inputs := githubv4.CreateCommitOnBranchInput{
		Branch: githubv4.CommittableBranch{
			RepositoryNameWithOwner: githubv4.NewString(githubv4.String(opts.Repository.FullName())),
			BranchName:              githubv4.NewString(githubv4.String(opts.BranchName)),
		},
		Message: githubv4.CommitMessage{
			Headline: githubv4.String(opts.CommitMessage.Headline),
			Body:     githubv4.NewString(githubv4.String(opts.CommitMessage.Body)),
		},
		FileChanges: &githubv4.FileChanges{
			Additions: &additions,
			Deletions: &deletions,
		},
		ExpectedHeadOid: githubv4.GitObjectID(parentCommitSHA),
	}

	var mutation struct {
		CreateCommitOnBranchInput struct {
			ClientMutationID string
		} `graphql:"createCommitOnBranch(input: $input)"`
	}

	gqlClient, err := githubGraphqlClient(ctx, opts.GitHubOpts)
	if err != nil {
		return fmt.Errorf("failed to create github GraphQL client: %w", err)
	}

	err = gqlClient.Mutate(ctx, &mutation, inputs, nil)
	if err != nil {
		return fmt.Errorf("failed to push branch %s to %s: %w", opts.BranchName, opts.Repository.FullName(), err)
	}
	return nil
}

func pushChangesWithGit(ctx context.Context, gitRepo *git.Repository, opts pushOptions) error {
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

type pushOptions struct {
	GitHubOpts    GitHubOptions
	GitCloneDir   string
	Repository    Repository
	BranchName    string
	CreateBranch  bool
	ResetFromBase bool
	CommitMessage CommitMessage
}

func pushChanges(ctx context.Context, gitRepo *git.Repository, opts pushOptions) error {
	if opts.GitHubOpts.AlwaysPushChangesWithGit {
		return pushChangesWithGit(ctx, gitRepo, opts)
	}

	switch opts.GitHubOpts.AuthMethod {
	case "token":
		return pushChangesWithGit(ctx, gitRepo, opts)
	case "app":
		return pushChangesWithAPI(ctx, gitRepo, opts)
	default:
		return fmt.Errorf("GitHub auth method unrecognized (allowed values: app, token): %s", opts.GitHubOpts.AuthMethod)
	}
}
