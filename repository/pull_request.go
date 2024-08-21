package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
	"github.com/zoumo/goset"
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

	page := 1
	for {
		prs, resp, err := client.PullRequests.List(ctx, r.Owner, r.Name, &github.PullRequestListOptions{
			Base: options.PullRequest.BaseBranch,
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: 100,
			},
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

		page = resp.NextPage
		if resp.NextPage == 0 {
			break
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

	err = r.enrichPullRequestWithContextualData(ctx, options, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to enrich the Pull Request %s: %w", pr.GetHTMLURL(), err)
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

	err = r.enrichPullRequestWithContextualData(ctx, options, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to enrich the Pull Request %s: %w", pr.GetHTMLURL(), err)
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

func (r Repository) addPullRequestReviewers(ctx context.Context, options GitHubOptions, pr *github.PullRequest) error {
	if len(options.PullRequest.Reviewers) == 0 && len(options.PullRequest.TeamReviewers) == 0 {
		logrus.WithFields(logrus.Fields{
			"repository":   r.FullName(),
			"pull-request": pr.GetHTMLURL(),
		}).Debug("No reviewers to add to the Pull Request")
		return nil
	}

	client, _, err := githubClient(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to create github client: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"repository":     r.FullName(),
		"pull-request":   pr.GetHTMLURL(),
		"reviewers":      options.PullRequest.Reviewers,
		"team-reviewers": options.PullRequest.TeamReviewers,
	}).Trace("Adding reviewers to the Pull Request")

	reviewers := github.ReviewersRequest{
		Reviewers:     options.PullRequest.Reviewers,
		TeamReviewers: options.PullRequest.TeamReviewers,
	}
	_, _, err = client.PullRequests.RequestReviewers(ctx, r.Owner, r.Name, pr.GetNumber(), reviewers)

	if err != nil {
		return fmt.Errorf("failed to add reviewers to PR %s: %w", pr.GetHTMLURL(), err)
	}

	logrus.WithFields(logrus.Fields{
		"repository":     r.FullName(),
		"pull-request":   pr.GetHTMLURL(),
		"reviewers":      options.PullRequest.Reviewers,
		"team-reviewers": options.PullRequest.TeamReviewers,
	}).Debug("Reviewers added to the Pull Request")

	return nil
}

func (r Repository) enrichPullRequestWithContextualData(ctx context.Context, options GitHubOptions, pr *github.PullRequest) error {
	var err error

	err = r.ensurePullRequestLabels(ctx, options, pr)
	if err != nil {
		return fmt.Errorf("failed to ensure that Pull Request %s has the right labels: %w", pr.GetHTMLURL(), err)
	}

	err = r.addPullRequestComments(ctx, options, pr)
	if err != nil {
		return fmt.Errorf("failed to add comments to Pull Request %s: %w", pr.GetHTMLURL(), err)
	}

	err = r.addPullRequestAssignees(ctx, options, pr)
	if err != nil {
		return fmt.Errorf("failed to add assignees to Pull Request %s: %w", pr.GetHTMLURL(), err)
	}

	err = r.addPullRequestReviewers(ctx, options, pr)
	if err != nil {
		return fmt.Errorf("failed to add reviewers for Pull Request %s: %w", pr.GetHTMLURL(), err)
	}

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

	gqlClient, err := githubGraphqlClient(ctx, options)
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

	if err == nil {
		return nil
	}

	logrus.WithFields(logrus.Fields{
		"repository":     r.FullName(),
		"pull-request":   prURL,
		"merge-strategy": mergeStrategy,
	}).WithError(err).Warning("Timed out waiting for PR to be auto merged. Disabling auto-merge.")

	err = r.disableAutoMerge(ctx, options, pr)

	if err != nil {
		return fmt.Errorf("failed to disable auto-merge for PR %s: %w", prURL, err)
	}

	return nil
}

func (r Repository) disableAutoMerge(ctx context.Context, options GitHubOptions, pr *github.PullRequest) error {
	gqlClient, err := githubGraphqlClient(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to create github GraphQL client: %w", err)
	}

	var mutation struct {
		DisablePullRequestAutoMergeInput struct {
			ClientMutationID string
		} `graphql:"disablePullRequestAutoMerge(input: $input)"`
	}

	inputs := githubv4.DisablePullRequestAutoMergeInput{
		PullRequestID: pr.NodeID,
	}

	err = gqlClient.Mutate(ctx, &mutation, inputs, nil)

	if err != nil {
		return fmt.Errorf("GraphQL mutation failed: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"repository":   r.FullName(),
		"pull-request": pr.GetHTMLURL(),
	}).Debug("PR auto-merge disabled")

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

func (r Repository) pollPullRequestIsMergeable(ctx context.Context, client *github.Client, gqlClient *githubv4.Client, options GitHubOptions, pr *github.PullRequest) (bool, error) {
	logrus.WithFields(logrus.Fields{
		"repository":   r.FullName(),
		"pull-request": pr.GetHTMLURL(),
	}).Trace("Getting Pull Request status")

	var (
		prURL = pr.GetHTMLURL()
		err   error
	)

	var statusQuery struct {
		Repository struct {
			PullRequest struct {
				Mergeable             githubv4.MergeableState
				Merged                githubv4.Boolean
				MergeStateStatus      githubv4.String
				ViewerCanMergeAsAdmin githubv4.Boolean
				BaseRef               struct {
					RefUpdateRule struct {
						RequiredStatusCheckContexts []string
					}
				}
				HeadRef struct {
					Target struct {
						Commit struct {
							Status struct {
								Contexts []struct {
									Context string
									State   githubv4.StatusState
								}
							}
							CheckSuites struct {
								Nodes []struct {
									CheckRuns struct {
										Nodes []struct {
											Name       string
											Status     githubv4.CheckStatusState
											Conclusion *githubv4.CheckConclusionState
										}
									} `graphql:"checkRuns(last: 100)"`
								}
							} `graphql:"checkSuites(last:100)"`
						} `graphql:"... on Commit"`
					}
				}
			} `graphql:"pullRequest(number: $prNumber)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	statusQueryVars := map[string]interface{}{
		"owner":    githubv4.String(r.Owner),
		"name":     githubv4.String(r.Name),
		"prNumber": githubv4.Int(pr.GetNumber()),
	}

	err = gqlClient.Query(ctx, &statusQuery, statusQueryVars)
	if err != nil {
		return false, fmt.Errorf("failed to retrieve status of Pull Request %s: %w", prURL, err)
	}

	if statusQuery.Repository.PullRequest.Merged {
		logrus.WithFields(logrus.Fields{
			"repository":   r.FullName(),
			"pull-request": pr.GetHTMLURL(),
		}).Trace("Pull Request is already merged")

		return true, nil
	}

	if s := statusQuery.Repository.PullRequest.Mergeable; s == githubv4.MergeableStateConflicting {
		return false, fmt.Errorf("pull request %s is not mergeable: %s", pr.GetHTMLURL(), s)
	}

	if s := statusQuery.Repository.PullRequest.Mergeable; s == githubv4.MergeableStateUnknown {
		logrus.WithFields(logrus.Fields{
			"repository":   r.FullName(),
			"pull-request": pr.GetHTMLURL(),
		}).Trace("Pull Request mergeability is still being calculated")
		return false, nil
	}

	switch options.PullRequest.Merge.BranchProtection {
	case BranchProtectionKindBypass:
		{
			if statusQuery.Repository.PullRequest.ViewerCanMergeAsAdmin {
				logrus.WithFields(logrus.Fields{
					"repository":   r.FullName(),
					"pull-request": pr.GetHTMLURL(),
				}).Trace("Bypassing branch protection rules")
				return true, nil
			}

			logrus.WithFields(logrus.Fields{
				"repository":   r.FullName(),
				"pull-request": pr.GetHTMLURL(),
			}).Trace("Not able to bypass branch protection rules yet")

			// it's possible that the viewer could bypass protection rules at some later point, so don't return an error
			return false, nil
		}
	case BranchProtectionKindAll:
		{
			if statusQuery.Repository.PullRequest.MergeStateStatus == "CLEAN" {
				logrus.WithFields(logrus.Fields{
					"repository":   r.FullName(),
					"pull-request": pr.GetHTMLURL(),
				}).Trace("All branch protection rules satisfied")
				return true, nil
			}

			logrus.WithFields(logrus.Fields{
				"repository":   r.FullName(),
				"pull-request": pr.GetHTMLURL(),
			}).Trace("All branch protection rules not satisfied")
			return false, nil
		}
	case BranchProtectionKindStatusChecks:
		fallthrough
	default:
	}

	if pr.GetBase().Ref == nil {
		return false, errors.New("failed to get PR base ref")
	}

	requiredContexts := goset.NewSetFromStrings(statusQuery.Repository.PullRequest.BaseRef.RefUpdateRule.RequiredStatusCheckContexts)

	rules, _, err := client.Repositories.GetRulesForBranch(ctx, r.Owner, r.Name, pr.GetBase().GetRef())
	if err != nil {
		return false, fmt.Errorf("failed to fetch Rules for base ref: %w", err)
	}

	for _, rule := range rules {
		if rule.Type != "required_status_checks" {
			continue
		}

		params := github.RequiredStatusChecksRuleParameters{}
		if err := json.Unmarshal(*rule.Parameters, &params); err != nil {
			return false, fmt.Errorf("failed to parse rule: %w", err)
		}

		for _, c := range params.RequiredStatusChecks {
			err := requiredContexts.Add(c.Context)
			if err != nil {
				return false, fmt.Errorf("failed to add rule context to required set: %w", err)
			}
		}
	}

	commit := statusQuery.Repository.PullRequest.HeadRef.Target.Commit

	passingContexts := goset.NewSet()

	for _, c := range commit.Status.Contexts {
		if !requiredContexts.Contains(c.Context) {
			continue
		}

		if c.State == githubv4.StatusStateSuccess {
			err := passingContexts.Add(c.Context)
			if err != nil {
				return false, fmt.Errorf("failed to add status context to passing set: %w", err)
			}
		} else {
			logrus.WithFields(logrus.Fields{
				"repository":     r.FullName(),
				"pull-request":   pr.GetHTMLURL(),
				"status-context": c.Context,
				"status-state":   c.State,
			}).Trace("Waiting for status")
		}
	}

	for _, cs := range commit.CheckSuites.Nodes {
		for _, c := range cs.CheckRuns.Nodes {
			if !requiredContexts.Contains(c.Name) {
				continue
			}

			if c.Status != githubv4.CheckStatusStateCompleted || !isCheckConclusionPassing(c.Conclusion) {
				logrus.WithFields(logrus.Fields{
					"repository":       r.FullName(),
					"pull-request":     pr.GetHTMLURL(),
					"check-name":       c.Name,
					"check-status":     c.Status,
					"check-conclusion": c.Conclusion,
				}).Trace("Waiting for check")
			} else {
				err := passingContexts.Add(c.Name)
				if err != nil {
					return false, fmt.Errorf("failed to add check context to passing set: %w", err)
				}
			}
		}
	}

	if !passingContexts.IsSupersetOf(requiredContexts) {
		logrus.WithFields(logrus.Fields{
			"repository":        r.FullName(),
			"pull-request":      pr.GetHTMLURL(),
			"required-contexts": requiredContexts,
			"passing-contexts":  passingContexts,
		}).Trace("Required status checks not passing")
		return false, nil
	}

	logrus.WithFields(logrus.Fields{
		"repository":        r.FullName(),
		"pull-request":      pr.GetHTMLURL(),
		"required-contexts": requiredContexts,
		"passing-contexts":  passingContexts,
	}).Trace("All status checks passing")

	// Github's API is not race-free apparently.
	// Sometime the merge fails if done too quickly, even if the status/checks report success.
	// So wait a little.
	time.Sleep(5 * time.Second)

	return true, nil
}

func (r Repository) waitUntilPullRequestIsMergeable(ctx context.Context, options GitHubOptions, pr *github.PullRequest) error {
	var startTime = time.Now()

	client, _, err := githubClient(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to create github client: %w", err)
	}

	gqlClient, err := githubGraphqlClient(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to create github GraphQL client: %w", err)
	}

	for {
		mergeable, err := r.pollPullRequestIsMergeable(ctx, client, gqlClient, options, pr)

		if err != nil {
			return err
		}

		if mergeable {
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
			"repository":   r.FullName(),
			"pull-request": pr.GetHTMLURL(),
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

func isCheckConclusionPassing(c *githubv4.CheckConclusionState) bool {
	if c == nil {
		return false
	}
	switch *c { //nolint: exhaustive // default should catch the rest
	case githubv4.CheckConclusionStateSuccess, githubv4.CheckConclusionStateNeutral, githubv4.CheckConclusionStateSkipped:
		return true
	default:
		return false
	}
}
