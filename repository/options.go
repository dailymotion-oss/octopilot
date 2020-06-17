package repository

import (
	"fmt"
	"strings"
	"time"

	"github.com/dailymotion/octopilot/update"
)

const (
	IgnoreUpdateOperation  = "ignore"
	ReplaceUpdateOperation = "replace"
	PrependUpdateOperation = "prepend"
	AppendUpdateOperation  = "append"
)

type UpdateOptions struct {
	DryRun    bool
	KeepFiles bool
	Git       GitOptions
	GitHub    GitHubOptions
	Strategy  string
}

type GitOptions struct {
	CloneDir       string
	AuthorName     string
	AuthorEmail    string
	CommitterName  string
	CommitterEmail string
	CommitTitle    string
	CommitBody     string
	CommitFooter   string
	BranchPrefix   string
}

type GitHubOptions struct {
	Token       string
	PullRequest PullRequestOptions
}

type PullRequestOptions struct {
	Labels               []string
	BaseBranch           string
	Title                string
	TitleUpdateOperation string
	Body                 string
	BodyUpdateOperation  string
	Comments             []string
	Draft                bool
	Merge                PullRequestMergeOptions
}

type PullRequestMergeOptions struct {
	Enabled       bool
	Method        string
	CommitTitle   string
	CommitMessage string
	SHA           string
	PollInterval  time.Duration
	PollTimeout   time.Duration
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
		o.CommitTitle = "OctoPilot update"
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
	} else {
		o.CommitTitle = commitTitle
	}
	commitBody, err := tplExecutorFunc(o.CommitBody)
	if err != nil {
		return fmt.Errorf("failed to run template for git commit body %s: %w", o.CommitBody, err)
	} else {
		o.CommitBody = commitBody
	}

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
	} else {
		o.PullRequest.Title = prTitle
	}
	prBody, err := tplExecutorFunc(o.PullRequest.Body)
	if err != nil {
		return fmt.Errorf("failed to run template for pull request body %s: %w", o.PullRequest.Body, err)
	} else {
		o.PullRequest.Body = prBody
	}

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
