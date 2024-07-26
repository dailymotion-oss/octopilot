package repository

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dailymotion-oss/octopilot/update"
	"github.com/go-git/go-git/v5"
)

// definition of the different kind of Pull Request update operations
const (
	IgnoreUpdateOperation  = "ignore"
	ReplaceUpdateOperation = "replace"
	PrependUpdateOperation = "prepend"
	AppendUpdateOperation  = "append"

	PublicGithubURL = "https://github.com"
)

// UpdateOptions is the options entrypoint for a git repo update
type UpdateOptions struct {
	DryRun    bool
	KeepFiles bool
	Git       GitOptions
	GitHub    GitHubOptions
	Strategy  string
}

// GitOptions holds all the options required to perform git operations: clone, commit, ...
type GitOptions struct {
	CloneDir             string
	StagePatterns        []string
	StageAllChanged      bool
	AuthorName           string
	AuthorEmail          string
	CommitterName        string
	CommitterEmail       string
	CommitTitle          string
	CommitBody           string
	CommitFooter         string
	BranchPrefix         string
	SigningKeyPath       string
	SigningKeyPassphrase string
	RecurseSubmodules    bool
}

// GitHubOptions holds all the options required to perform github operations: auth, PRs, ...
type GitHubOptions struct {
	URL            string
	AuthMethod     string
	Token          string
	AppID          int64
	InstallationID int64
	PrivateKey     string
	PrivateKeyPath string
	PullRequest    PullRequestOptions
}

func (o *GitHubOptions) isEnterprise() bool {
	return o.URL != PublicGithubURL
}

// PullRequestOptions holds all the options required to perform github PR operations: title/body, merge, ...
type PullRequestOptions struct {
	Labels               []string
	BaseBranch           string
	Title                string
	TitleUpdateOperation string
	Body                 string
	BodyUpdateOperation  string
	Comments             []string
	Assignees            []string
	Draft                bool
	Merge                PullRequestMergeOptions
}

// BranchProtectionKind enumerates possible branch protections to wait for before attempting a PR merge.
type BranchProtectionKind string

const (
	// BranchProtectionKindStatusChecks waits for all required statues & checksuite runs to be passing ("success", "neutral", "skipped")
	BranchProtectionKindStatusChecks BranchProtectionKind = "statusChecks"

	// BranchProtectionKindAll wait for all protection rules
	BranchProtectionKindAll BranchProtectionKind = "all"

	// BranchProtectionKindBypass waits until the user can bypass branch protection rules
	BranchProtectionKindBypass BranchProtectionKind = "bypass"
)

func (b *BranchProtectionKind) String() string {
	return string(*b)
}

func (b *BranchProtectionKind) Set(s string) error {
	switch s {
	case string(BranchProtectionKindStatusChecks), string(BranchProtectionKindAll), string(BranchProtectionKindBypass):
		*b = BranchProtectionKind(s)
		return nil
	default:
		return errors.New("invalid value")
	}
}

func (b *BranchProtectionKind) Type() string {
	return strings.Join(
		[]string{
			string(BranchProtectionKindStatusChecks),
			string(BranchProtectionKindAll),
			string(BranchProtectionKindBypass),
		},
		"|",
	)
}

// PullRequestMergeOptions holds all the options required to merge github PRs
type PullRequestMergeOptions struct {
	Enabled          bool
	Auto             bool
	AutoWait         bool
	Method           string
	CommitTitle      string
	CommitMessage    string
	SHA              string
	PollInterval     time.Duration
	PollTimeout      time.Duration
	RetryCount       int
	BranchProtection BranchProtectionKind
}

func (o *GitOptions) setDefaultValues(updaters []update.Updater, tplExecutorFunc templateExecutor) error {
	if len(updaters) == 1 {
		title, body := updaters[0].Message()
		if len(o.CommitTitle) == 0 {
			o.CommitTitle = title
		}
		if len(o.CommitBody) == 0 {
			o.CommitBody = body
		}
	}
	if len(o.CommitTitle) == 0 {
		o.CommitTitle = "Octopilot update"
	}
	if len(o.CommitBody) == 0 {
		body := new(strings.Builder)
		body.WriteString("Updates:")
		for _, updater := range updaters {
			updaterTitle, updaterBody := updater.Message()
			body.WriteString("\n\n### ")
			body.WriteString(updater.String())
			body.WriteString("\n")
			body.WriteString(updaterTitle)
			body.WriteString("\n")
			body.WriteString(updaterBody)
		}
		o.CommitBody = body.String()
	}

	commitTitle, err := tplExecutorFunc(o.CommitTitle)
	if err != nil {
		return fmt.Errorf("failed to run template for git commit title %s: %w", o.CommitTitle, err)
	}
	o.CommitTitle = commitTitle

	commitBody, err := tplExecutorFunc(o.CommitBody)
	if err != nil {
		return fmt.Errorf("failed to run template for git commit body %s: %w", o.CommitBody, err)
	}
	o.CommitBody = commitBody

	return nil
}

func (o *GitHubOptions) setDefaultValues(git GitOptions, tplExecutorFunc templateExecutor) error {
	if len(o.PullRequest.Title) == 0 {
		o.PullRequest.Title = git.CommitTitle
	}
	if len(o.PullRequest.Body) == 0 {
		o.PullRequest.Body = git.CommitBody
	}
	if len(git.CommitFooter) > 0 {
		o.PullRequest.Body += fmt.Sprintf("\n\n-- \n%s", git.CommitFooter)
	}

	prTitle, err := tplExecutorFunc(o.PullRequest.Title)
	if err != nil {
		return fmt.Errorf("failed to run template for pull reequest title %s: %w", o.PullRequest.Title, err)
	}
	o.PullRequest.Title = prTitle

	prBody, err := tplExecutorFunc(o.PullRequest.Body)
	if err != nil {
		return fmt.Errorf("failed to run template for pull request body %s: %w", o.PullRequest.Body, err)
	}
	o.PullRequest.Body = prBody

	return nil
}

func (o *GitHubOptions) setDefaultUpdateOperation(defaultUpdateOperation string) {
	if len(o.PullRequest.TitleUpdateOperation) == 0 {
		o.PullRequest.TitleUpdateOperation = defaultUpdateOperation
	}
	if len(o.PullRequest.BodyUpdateOperation) == 0 {
		o.PullRequest.BodyUpdateOperation = defaultUpdateOperation
	}
}

func (o *GitHubOptions) adjustOptionsFromGitRepository(gitRepo *git.Repository) error {
	if len(o.PullRequest.BaseBranch) == 0 {
		if gitRepo == nil {
			return errors.New("failed to resolve repository branch referenced by HEAD: repository is null")
		}
		head, err := gitRepo.Head()
		if err != nil {
			return fmt.Errorf("failed to resolve repository branch referenced by HEAD: %w", err)
		}
		o.PullRequest.BaseBranch = head.Name().Short()
	}
	return nil
}
