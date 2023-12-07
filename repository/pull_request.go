package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v36/github"
	"github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
)

func (r Repository) findMatchingPullRequest(ctx context.Context, options GitHubOptions) (*github.PullRequest, error) {
	logrus.WithFields(logrus.Fields{
		"repository": r.FullName(),
		"labels":     options.PullRequest.Labels,
	}).Trace("Looking for existing Pull Requests")
	client, _, err := githubClient(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create github client: %w", err)
	}
	prs, _, err := client.PullRequests.List(ctx, r.Owner, r.Name, &github.PullRequestListOptions{
		Base: options.PullRequest.BaseBranch,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list opened Pull Requests for repository %s: %w", r.FullName(), err)
	}

	for _, pr := range prs {
		if prHasLabels(pr, options.PullRequest.Labels) {
			logrus.WithFields(logrus.Fields{
				"repository":   r.FullName(),
				"labels":       options.PullRequest.Labels,
				"pull-request": pr.GetHTMLURL(),
			}).Info("Found existing Pull Request")
			return pr, nil
		}
	}

	logrus.WithFields(logrus.Fields{
		"repository": r.FullName(),
		"labels":     options.PullRequest.Labels,
	}).Debug("No existing Pull Request found")
	return nil, nil
}

func (r Repository) createPullRequest(ctx context.Context, options GitHubOptions, branchName string) (*github.PullRequest, error) {
	logrus.WithFields(logrus.Fields{
		"repository": r.FullName(),
	}).Trace("Creating new Pull Request")

	client, _, err := githubClient(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create github client: %w", err)
	}
	pr, _, err := client.PullRequests.Create(ctx, r.Owner, r.Name, &github.NewPullRequest{
		Title:               github.String(options.PullRequest.Title),
		Base:                github.String(options.PullRequest.BaseBranch),
		Head:                github.String(branchName),
		Body:                github.String(options.PullRequest.Body),
		MaintainerCanModify: github.Bool(true),
		Draft:               github.Bool(options.PullRequest.Draft),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create a new Pull Request for repository %s: %w", r.FullName(), err)
	}

	logrus.WithFields(logrus.Fields{
		"repository":   r.FullName(),
		"pull-request": pr.GetHTMLURL(),
	}).Info("New Pull Request created")

	err = r.ensurePullRequestLabels(ctx, options, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure that Pull Request %s has the right labels: %w", pr.GetHTMLURL(), err)
	}

	err = r.addPullRequestComments(ctx, options, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to add comments to Pull Request %s: %w", pr.GetHTMLURL(), err)
	}

	err = r.addPullRequestAssignees(ctx, options, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to add assignees to Pull Request %s: %w", pr.GetHTMLURL(), err)
	}

	return pr, nil
}

func (r Repository) updatePullRequest(ctx context.Context, options GitHubOptions, pr *github.PullRequest) (*github.PullRequest, error) {
	var needUpdate bool
	client, _, err := githubClient(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create github client: %w", err)
	}

	if len(options.PullRequest.Title) > 0 {
		switch options.PullRequest.TitleUpdateOperation {
		case IgnoreUpdateOperation:
			// nothing to do
		case ReplaceUpdateOperation:
			pr.Title = github.String(options.PullRequest.Title)
			needUpdate = true
		case PrependUpdateOperation:
			pr.Title = github.String(fmt.Sprintf("%s %s", options.PullRequest.Title, pr.GetTitle()))
			needUpdate = true
		case AppendUpdateOperation:
			pr.Title = github.String(fmt.Sprintf("%s %s", pr.GetTitle(), options.PullRequest.Title))
			needUpdate = true
		}
	}
	if len(options.PullRequest.Body) > 0 {
		switch options.PullRequest.BodyUpdateOperation {
		case IgnoreUpdateOperation:
			// nothing to do
		case ReplaceUpdateOperation:
			pr.Body = github.String(options.PullRequest.Body)
			needUpdate = true
		case PrependUpdateOperation:
			pr.Body = github.String(fmt.Sprintf("%s\n\n%s", options.PullRequest.Body, pr.GetBody()))
			needUpdate = true
		case AppendUpdateOperation:
			pr.Body = github.String(fmt.Sprintf("%s\n\n%s", pr.GetBody(), options.PullRequest.Body))
			needUpdate = true
		}
	}

	if needUpdate {
		logrus.WithFields(logrus.Fields{
			"repository":   r.FullName(),
			"pull-request": pr.GetHTMLURL(),
		}).Trace("Updating existing Pull Request")
		prURL := pr.GetHTMLURL()
		pr, _, err = client.PullRequests.Edit(ctx, r.Owner, r.Name, pr.GetNumber(), pr)
		if err != nil {
			return nil, fmt.Errorf("failed to update Pull Request %s: %w", prURL, err)
		}
		logrus.WithFields(logrus.Fields{
			"repository":   r.FullName(),
			"pull-request": pr.GetHTMLURL(),
		}).Info("Pull Request updated")
	} else {
		logrus.WithFields(logrus.Fields{
			"repository":   r.FullName(),
			"pull-request": pr.GetHTMLURL(),
		}).Debug("No need to update the Pull Request")
	}

	err = r.ensurePullRequestLabels(ctx, options, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure that Pull Request %s has the right labels: %w", pr.GetHTMLURL(), err)
	}

	err = r.addPullRequestComments(ctx, options, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to add comments to Pull Request %s: %w", pr.GetHTMLURL(), err)
	}

	err = r.addPullRequestAssignees(ctx, options, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to add assignees to Pull Request %s: %w", pr.GetHTMLURL(), err)
	}

	return pr, nil
}

func (r Repository) ensurePullRequestLabels(ctx context.Context, options GitHubOptions, pr *github.PullRequest) error {
	if prHasLabels(pr, options.PullRequest.Labels) {
		logrus.WithFields(logrus.Fields{
			"repository":   r.FullName(),
			"pull-request": pr.GetHTMLURL(),
		}).Debug("No labels to add to the Pull Request")
		return nil
	}

	client, _, err := githubClient(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to create github client: %w", err)
	}
	logrus.WithFields(logrus.Fields{
		"repository":   r.FullName(),
		"pull-request": pr.GetHTMLURL(),
	}).Trace("Adding labels to Pull Request")
	_, _, err = client.Issues.AddLabelsToIssue(ctx, r.Owner, r.Name, pr.GetNumber(), options.PullRequest.Labels)
	if err != nil {
		return fmt.Errorf("failed to add labels %v on PR %s: %w", options.PullRequest.Labels, pr.GetHTMLURL(), err)
	}

	logrus.WithFields(logrus.Fields{
		"repository":   r.FullName(),
		"pull-request": pr.GetHTMLURL(),
	}).Debug("Labels added to Pull Request")
	return nil
}

func (r Repository) addPullRequestComments(ctx context.Context, options GitHubOptions, pr *github.PullRequest) error {
	if len(options.PullRequest.Comments) == 0 {
		logrus.WithFields(logrus.Fields{
			"repository":   r.FullName(),
			"pull-request": pr.GetHTMLURL(),
		}).Debug("No comments to add to the Pull Request")
		return nil
	}

	client, _, err := githubClient(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to create github client: %w", err)
	}
	for i, comment := range options.PullRequest.Comments {
		logrus.WithFields(logrus.Fields{
			"repository":   r.FullName(),
			"pull-request": pr.GetHTMLURL(),
			"comment":      i,
		}).Trace("Adding a comment to the Pull Request")

		_, _, err := client.Issues.CreateComment(ctx, r.Owner, r.Name, pr.GetNumber(), &github.IssueComment{
			Body: github.String(comment),
		})
		if err != nil {
			return fmt.Errorf("failed to add comment %v on PR %s: %w", github.String(comment), pr.GetHTMLURL(), err)
		}

		logrus.WithFields(logrus.Fields{
			"repository":   r.FullName(),
			"pull-request": pr.GetHTMLURL(),
			"comment":      i,
		}).Debug("Comment added to the Pull Request")

		if len(options.PullRequest.Comments) > 1 && i < len(options.PullRequest.Comments)-1 {
			logrus.WithFields(logrus.Fields{
				"repository":   r.FullName(),
				"pull-request": pr.GetHTMLURL(),
				"comment":      i,
			}).Trace("Sleeping a little before adding next comment, for rate limiting...")
			time.Sleep(500 * time.Millisecond)
		}
	}

	return nil
}

func (r Repository) addPullRequestAssignees(ctx context.Context, options GitHubOptions, pr *github.PullRequest) error {
	if len(options.PullRequest.Assignees) == 0 {
		logrus.WithFields(logrus.Fields{
			"repository":   r.FullName(),
			"pull-request": pr.GetHTMLURL(),
		}).Debug("No assignees to add to the Pull Request")
		return nil
	}

	client, _, err := githubClient(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to create github client: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"repository":   r.FullName(),
		"pull-request": pr.GetHTMLURL(),
		"assignees":    options.PullRequest.Assignees,
	}).Trace("Adding assignees to the Pull Request")

	_, _, err = client.Issues.AddAssignees(ctx, r.Owner, r.Name, pr.GetNumber(), options.PullRequest.Assignees)

	if err != nil {
		return fmt.Errorf("failed to add assignees %v on PR %s: %w", options.PullRequest.Assignees, pr.GetHTMLURL(), err)
	}

	logrus.WithFields(logrus.Fields{
		"repository":   r.FullName(),
		"pull-request": pr.GetHTMLURL(),
		"assignees":    options.PullRequest.Assignees,
	}).Debug("Assignees added to the Pull Request")

	return nil
}

func (r Repository) mergePullRequest(ctx context.Context, options GitHubOptions, pr *github.PullRequest) error {
	if options.PullRequest.Merge.Auto {
		return r.mergePullRequestUsingAutoMerge(ctx, options, pr)
	}

	return r.mergePullRequestUsingClient(ctx, options, pr, 0)
}

func (r Repository) mergePullRequestUsingAutoMerge(ctx context.Context, options GitHubOptions, pr *github.PullRequest) error {
	prURL := pr.GetHTMLURL()
	mergeStrategy := "auto-merge"

	client, _, err := githubClient(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to create github client: %w", err)
	}

	gqlClient, _, err := githubGraphqlClient(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to create github GraphQL client: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"repository":     r.FullName(),
		"pull-request":   prURL,
		"timeout":        options.PullRequest.Merge.PollTimeout.String(),
		"merge-strategy": mergeStrategy,
	}).Trace("Starting Pull Request merge process")

	logrus.WithFields(logrus.Fields{
		"repository":     r.FullName(),
		"pull-request":   prURL,
		"merge-strategy": mergeStrategy,
	}).Trace("Getting Pull Request status")

	pr, _, err = client.PullRequests.Get(ctx, r.Owner, r.Name, pr.GetNumber())
	if err != nil {
		return fmt.Errorf("failed to retrieve status of Pull Request %s: %w", prURL, err)
	}

	if pr.GetMerged() {
		logrus.WithFields(logrus.Fields{
			"repository":     r.FullName(),
			"pull-request":   prURL,
			"merge-strategy": mergeStrategy,
		}).Info("Pull Request is already merged")
		return nil
	}

	logrus.WithFields(logrus.Fields{
		"repository":     r.FullName(),
		"pull-request":   prURL,
		"merge-strategy": mergeStrategy,
	}).Trace("Enabling auto-merge for Pull Request")

	var mutation struct {
		EnablePullPullRequestAutoMergeInput struct {
			ClientMutationID string
		} `graphql:"enablePullRequestAutoMerge(input: $input)"`
	}

	var mergeMethod githubv4.PullRequestMergeMethod
	switch strings.ToLower(options.PullRequest.Merge.Method) {
	case "merge":
		mergeMethod = githubv4.PullRequestMergeMethodMerge
	case "squash":
		mergeMethod = githubv4.PullRequestMergeMethodSquash
	case "rebase":
		mergeMethod = githubv4.PullRequestMergeMethodRebase
	default:
		mergeMethod = githubv4.PullRequestMergeMethodMerge
		logrus.WithFields(logrus.Fields{
			"repository":     r.FullName(),
			"pull-request":   prURL,
			"merge-strategy": mergeStrategy,
		}).Warnf(
			"Unknown Pull Request merge method %v. Falling back to 'merge'",
			options.PullRequest.Merge.Method,
		)
	}

	var expectedHeadOid *githubv4.GitObjectID
	if options.PullRequest.Merge.SHA != "" {
		expectedHeadOid = githubv4.NewGitObjectID(githubv4.GitObjectID(options.PullRequest.Merge.SHA))
	}

	var commitHeadLine *githubv4.String
	if options.PullRequest.Merge.CommitTitle != "" {
		commitHeadLine = githubv4.NewString(githubv4.String(options.PullRequest.Merge.CommitTitle))
	}

	var commitBody *githubv4.String
	if options.PullRequest.Merge.CommitMessage != "" {
		commitBody = githubv4.NewString(githubv4.String(options.PullRequest.Merge.CommitMessage))
	}

	inputs := githubv4.EnablePullRequestAutoMergeInput{
		PullRequestID:   pr.NodeID,
		MergeMethod:     &mergeMethod,
		ExpectedHeadOid: expectedHeadOid,
		CommitHeadline:  commitHeadLine,
		CommitBody:      commitBody,
	}

	attempted := 0

	// Should this be retried?
	for {
		err = gqlClient.Mutate(ctx, &mutation, inputs, nil)

		attempted++

		if err == nil {
			break
		}

		if attempted > options.PullRequest.Merge.RetryCount {
			return fmt.Errorf("failed to enable auto-merge for Pull Request: %w", err)
		}

		logrus.WithFields(logrus.Fields{
			"repository":     r.FullName(),
			"pull-request":   prURL,
			"attempted":      attempted,
			"merge-strategy": mergeStrategy,
		}).WithError(err).Warning("Failed to enable auto-merge for Pull Request - will retry")

		// To wait or not to wait?
		time.Sleep(options.PullRequest.Merge.PollInterval)
	}

	logrus.WithFields(logrus.Fields{
		"repository":     r.FullName(),
		"pull-request":   prURL,
		"merge-strategy": mergeStrategy,
	}).Debug("Enabled auto-merge for Pull Request")

	if !options.PullRequest.Merge.AutoWait {
		logrus.WithFields(logrus.Fields{
			"repository":     r.FullName(),
			"pull-request":   prURL,
			"merge-strategy": mergeStrategy,
		}).Debugf("Not waiting until Pull Request %s is merged.", prURL)
		return nil
	}

	logrus.WithFields(logrus.Fields{
		"repository":     r.FullName(),
		"pull-request":   prURL,
		"merge-strategy": mergeStrategy,
	}).Debug("Waiting for Pull Request to be merged")

	err = r.waitUntilPullRequestIsMerged(ctx, options, pr)
	if err != nil {
		return fmt.Errorf("failed to wait for Pull Request %s to be auto merged: %w", prURL, err)
	}

	return nil
}

func (r Repository) mergePullRequestUsingClient(ctx context.Context, options GitHubOptions, pr *github.PullRequest, retryCount int) error {
	prURL := pr.GetHTMLURL()
	mergeStrategy := "client"

	if retryCount >= options.PullRequest.Merge.RetryCount {
		return fmt.Errorf("failed to merge Pull Request %s after %d retries (max retry count is set to %d)", prURL, retryCount, options.PullRequest.Merge.RetryCount)
	}

	client, _, err := githubClient(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to create github client: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"repository":     r.FullName(),
		"pull-request":   prURL,
		"timeout":        options.PullRequest.Merge.PollTimeout.String(),
		"retry":          retryCount,
		"merge-strategy": mergeStrategy,
	}).Trace("Starting Pull Request merge process")

	err = r.waitUntilPullRequestIsMergeable(ctx, options, pr)
	if err != nil {
		return fmt.Errorf("failed to wait until Pull Request %s is mergeable: %w", prURL, err)
	}

	logrus.WithFields(logrus.Fields{
		"repository":     r.FullName(),
		"pull-request":   prURL,
		"retry":          retryCount,
		"merge-strategy": mergeStrategy,
	}).Trace("Getting Pull Request status")
	pr, _, err = client.PullRequests.Get(ctx, r.Owner, r.Name, pr.GetNumber())
	if err != nil {
		return fmt.Errorf("failed to retrieve status of Pull Request %s: %w", prURL, err)
	}
	if pr.GetMerged() {
		logrus.WithFields(logrus.Fields{
			"repository":     r.FullName(),
			"pull-request":   prURL,
			"retry":          retryCount,
			"merge-strategy": mergeStrategy,
		}).Info("Pull Request is already merged")
		return nil
	}

	logrus.WithFields(logrus.Fields{
		"repository":     r.FullName(),
		"pull-request":   prURL,
		"retry":          retryCount,
		"merge-strategy": mergeStrategy,
	}).Trace("Merging Pull Request")
	res, resp, err := client.PullRequests.Merge(ctx, r.Owner, r.Name, pr.GetNumber(), options.PullRequest.Merge.CommitMessage, &github.PullRequestOptions{
		MergeMethod: options.PullRequest.Merge.Method,
		CommitTitle: options.PullRequest.Merge.CommitTitle,
		SHA:         options.PullRequest.Merge.SHA,
	})
	if err != nil && shouldRetryMerge(resp, err) {
		logrus.WithFields(logrus.Fields{
			"repository":     r.FullName(),
			"pull-request":   prURL,
			"retry":          retryCount,
			"merge-strategy": mergeStrategy,
		}).WithError(err).Warning("Failed to merge Pull Request - will retry")
		retryCount++
		err = r.mergePullRequestUsingClient(ctx, options, pr, retryCount)
		if err == nil {
			return nil
		}
	}
	if err != nil {
		return fmt.Errorf("failed to merge Pull Request %s: %w", prURL, err)
	}
	if !res.GetMerged() {
		return fmt.Errorf("pull request %s was not merged: %s", prURL, res.GetMessage())
	}

	logrus.WithFields(logrus.Fields{
		"repository":     r.FullName(),
		"pull-request":   prURL,
		"retry":          retryCount,
		"merge-strategy": mergeStrategy,
	}).Info("Pull Request merged")
	return nil
}

func (r Repository) waitUntilPullRequestIsMerged(ctx context.Context, options GitHubOptions, pr *github.PullRequest) error {
	var startTime = time.Now()

	client, _, err := githubClient(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to create github client: %w", err)
	}

	for {
		logrus.WithFields(logrus.Fields{
			"repository":   r.FullName(),
			"pull-request": pr.GetHTMLURL(),
		}).Trace("Getting Pull Request status")
		var (
			prURL = pr.GetHTMLURL()
			err   error
		)
		pr, _, err = client.PullRequests.Get(ctx, r.Owner, r.Name, pr.GetNumber())
		if err != nil {
			return fmt.Errorf("failed to retrieve status of Pull Request %s: %w", prURL, err)
		}

		if pr.GetMerged() {
			logrus.WithFields(logrus.Fields{
				"repository":   r.FullName(),
				"pull-request": pr.GetHTMLURL(),
			}).Debug("Pull Request merged")
			return nil
		}

		if time.Since(startTime) > options.PullRequest.Merge.PollTimeout {
			return fmt.Errorf("timeout after %s waiting for Pull Request %s to be merged", options.PullRequest.Merge.PollTimeout.String(), pr.GetHTMLURL())
		}

		logrus.WithFields(logrus.Fields{
			"repository":   r.FullName(),
			"pull-request": pr.GetHTMLURL(),
		}).Tracef("Waiting %s until next GitHub request...", options.PullRequest.Merge.PollInterval.String())

		time.Sleep(options.PullRequest.Merge.PollInterval)
	}
}

func (r Repository) waitUntilPullRequestIsMergeable(ctx context.Context, options GitHubOptions, pr *github.PullRequest) error {
	var startTime = time.Now()

	client, _, err := githubClient(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to create github client: %w", err)
	}

	// wait until PR is mergeable
	// i.e. no conflicts with target branch & all checks from statuses API, checks API & other branch protection rules are satisfied
	for {
		logrus.WithFields(logrus.Fields{
			"repository":   r.FullName(),
			"pull-request": pr.GetHTMLURL(),
		}).Trace("Getting Pull Request status")
		var (
			prURL = pr.GetHTMLURL()
			err   error
		)
		pr, _, err = client.PullRequests.Get(ctx, r.Owner, r.Name, pr.GetNumber())
		if err != nil {
			return fmt.Errorf("failed to retrieve status of Pull Request %s: %w", prURL, err)
		}

		if pr.GetMerged() {
			logrus.WithFields(logrus.Fields{
				"repository":   r.FullName(),
				"pull-request": pr.GetHTMLURL(),
			}).Debug("Pull Request is already merged")
			return nil
		}

		if pr.Mergeable != nil && !pr.GetMergeable() {
			return fmt.Errorf("pull request %s is not mergeable: %s", pr.GetHTMLURL(), pr.GetMergeableState())
		}

		if pr.GetMergeableState() == "clean" {
			logrus.WithFields(logrus.Fields{
				"repository":   r.FullName(),
				"pull-request": pr.GetHTMLURL(),
			}).Debug("Pull Request is mergeable")

			return nil
		}

		if time.Since(startTime) > options.PullRequest.Merge.PollTimeout {
			return fmt.Errorf("timeout after %s waiting for Pull Request %s mergeable status", options.PullRequest.Merge.PollTimeout.String(), pr.GetHTMLURL())
		}

		logrus.WithFields(logrus.Fields{
			"repository":      r.FullName(),
			"pull-request":    pr.GetHTMLURL(),
			"mergeable-state": pr.GetMergeableState(),
		}).Debug("Pull Request is not mergeable yet")

		logrus.WithFields(logrus.Fields{
			"repository":   r.FullName(),
			"pull-request": pr.GetHTMLURL(),
		}).Tracef("Waiting %s until next GitHub request...", options.PullRequest.Merge.PollInterval.String())
		time.Sleep(options.PullRequest.Merge.PollInterval)
	}
}

func prHasLabels(pr *github.PullRequest, labels []string) bool {
	matchingLabels := 0
	for _, requiredLabel := range labels {
		for _, label := range pr.Labels {
			if label.GetName() == requiredLabel {
				matchingLabels++
				break
			}
		}
	}
	return matchingLabels == len(labels)
}

// shouldRetryMerge returns true if we should retry the merge operation at a later time
// see https://github.com/jenkins-x/lighthouse/blob/v0.0.922/pkg/keeper/keeper.go#L1110 for more context
func shouldRetryMerge(resp *github.Response, err error) bool {
	var githubErr *github.ErrorResponse
	if !errors.As(err, &githubErr) {
		return false
	}

	return resp.StatusCode == 405 && githubErr.Message == "Base branch was modified. Review and try the merge again."
}
