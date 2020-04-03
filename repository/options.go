package repository

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/dailymotion/octopilot/update"
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
	CommitBodyFile string
	CommitFooter   string
	BranchPrefix   string
}

type GitHubOptions struct {
	Token       string
	PullRequest PullRequestOptions
}

type PullRequestOptions struct {
	Labels     []string
	BaseBranch string
	Title      string
	Body       string
	Comments   []string
	Draft      bool
	Merge      PullRequestMergeOptions
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

func (o *GitOptions) setDefaultValues(updaters []update.Updater) {
	if len(o.CommitBody) == 0 && len(o.CommitBodyFile) > 0 {
		data, _ := ioutil.ReadFile(o.CommitBodyFile)
		o.CommitBody = string(data)
	}
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
}

func (o *GitHubOptions) setDefaultValues(git GitOptions) {
	if len(o.PullRequest.Title) == 0 {
		o.PullRequest.Title = git.CommitTitle
	}
	if len(o.PullRequest.Body) == 0 {
		o.PullRequest.Body = git.CommitBody
	}
	if len(git.CommitFooter) > 0 {
		o.PullRequest.Body += fmt.Sprintf("\n\n-- \n%s", git.CommitFooter)
	}
}
