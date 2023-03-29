// Package repository contains everything related to working with git repositories hosted on GitHub: cloning, commits, creating branches, pushing, creating/updating/merging pull requests, and so on.
package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/dailymotion-oss/octopilot/internal/parameters"
	"github.com/dailymotion-oss/octopilot/update"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
)

var (
	// type(params)
	repoRegexp = regexp.MustCompile(`^(?P<type>[A-Za-z0-9._\-/]+)(?:\((?P<params>.+)\))?$`)

	// owner/name(params)
	repoWithNameRegexp = regexp.MustCompile(`^(?P<owner>[A-Za-z0-9_\-]+)/(?P<name>[A-Za-z0-9._\-]+)(?:\((?P<params>.+)\))?$`)
)

// Repository is a representation of a GitHub repository.
type Repository struct {
	Owner  string
	Name   string
	Params map[string]string
}

// Parse parses a set of repositories defined as string - from the CLI for example - and returns properly formatted Repositories
// expected syntax is documented in the user documentation: docs/current-version/content/repos/{static,dynamic}.md
func Parse(ctx context.Context, repos []string, githubOpts GitHubOptions) ([]Repository, error) {
	var repositories []Repository
	for _, repo := range repos {
		matches := repoRegexp.FindStringSubmatch(repo)
		if len(matches) < 2 {
			return nil, fmt.Errorf("invalid syntax for %s: missing repo type or name", repo)
		}

		switch matches[1] {
		case "discover-from":
			discoveredRepos, err := discoverRepositoriesFrom(ctx, parameters.Parse(matches[2]), githubOpts)
			if err != nil {
				return nil, fmt.Errorf("failed to discover repositories: %w", err)
			}
			repositories = append(repositories, discoveredRepos...)
		default:
			matches := repoWithNameRegexp.FindStringSubmatch(repo)
			if len(matches) < 4 {
				return nil, fmt.Errorf("invalid syntax for %s: found %d matches instead of 4: %v", repo, len(matches), matches)
			}

			repositories = append(repositories, Repository{
				Owner:  matches[1],
				Name:   matches[2],
				Params: parameters.Parse(matches[3]),
			})
		}
	}
	return repositories, nil
}

func discoverRepositoriesFrom(ctx context.Context, params map[string]string, githubOpts GitHubOptions) ([]Repository, error) {
	if query, ok := params["query"]; ok {
		return discoverRepositoriesFromQuery(ctx, query, params, githubOpts)
	}

	if envVar, ok := params["env"]; ok {
		delete(params, "env")
		return discoverRepositoriesFromEnvironment(ctx, envVar, params, githubOpts)
	}

	return nil, fmt.Errorf("can't discover repositories from params %v: missing either query or env param", params)
}

// Update is the entrypoint to update a repository with a set of updaters.
// It returns a boolean indicating whether the repository was updated or not.
func (r Repository) Update(ctx context.Context, updaters []update.Updater, options UpdateOptions) (bool, error) {
	r.adjustOptionsFromParams(&options)

	repoPath := filepath.Join(options.Git.CloneDir, r.Owner, r.Name)
	if !options.KeepFiles {
		defer func() {
			logrus.WithFields(logrus.Fields{
				"repository": r.FullName(),
				"path":       repoPath,
			}).Trace("Deleting temporary files")
			if err := os.RemoveAll(repoPath); err != nil {
				logrus.WithFields(logrus.Fields{
					"repository": r.FullName(),
					"path":       repoPath,
				}).WithError(err).Warning("Failed to delete temporary files")
			}
		}()
	}

	var strategy Strategy
	switch options.Strategy {
	case "recreate":
		logrus.WithFields(logrus.Fields{
			"repository": r.FullName(),
		}).Debug("Using 'recreate' strategy")
		strategy = &RecreateStrategy{
			Repository: r,
			RepoPath:   repoPath,
			Updaters:   updaters,
			Options:    options,
		}
	case "append":
		logrus.WithFields(logrus.Fields{
			"repository": r.FullName(),
		}).Debug("Using 'append' strategy")
		strategy = &AppendStrategy{
			Repository: r,
			RepoPath:   repoPath,
			Updaters:   updaters,
			Options:    options,
		}
	default:
		logrus.WithFields(logrus.Fields{
			"repository": r.FullName(),
		}).Debug("Using 'reset' strategy")
		strategy = &ResetStrategy{
			Repository: r,
			RepoPath:   repoPath,
			Updaters:   updaters,
			Options:    options,
		}
	}

	repoUpdated, pr, err := strategy.Run(ctx)
	if err != nil {
		return false, fmt.Errorf("%w", err)
	}
	if !repoUpdated {
		return false, nil
	}

	if !options.GitHub.PullRequest.Merge.Enabled {
		logrus.WithFields(logrus.Fields{
			"repository": r.FullName(),
		}).Debug("Pull Request merging is disabled")
		return true, nil
	}
	if pr == nil {
		logrus.WithFields(logrus.Fields{
			"repository": r.FullName(),
		}).Warning("No Pull Request was created - can't merge it!")
		return true, nil
	}

	err = r.mergePullRequest(ctx, options.GitHub, pr)
	if err != nil {
		return true, fmt.Errorf("failed to merge Pull Request %s: %w", pr.GetHTMLURL(), err)
	}

	return true, nil
}

func (r Repository) runUpdaters(ctx context.Context, updaters []update.Updater, repoPath string) (bool, error) {
	var repoUpdated bool
	for _, updater := range updaters {
		logrus.WithFields(logrus.Fields{
			"repository": r.FullName(),
			"updater":    updater.String(),
		}).Trace("Running updater")
		updated, err := updater.Update(ctx, repoPath)
		if err != nil {
			return false, fmt.Errorf("failed to update repository %s: %w", r.FullName(), err)
		}
		if updated {
			repoUpdated = true
		}
		logrus.WithFields(logrus.Fields{
			"repository": r.FullName(),
			"updater":    updater.String(),
			"changes":    repoUpdated,
		}).Debug("Updater finished")
	}
	logrus.WithField("repository", r.FullName()).Debug("All updaters finished")
	return repoUpdated, nil
}

func (r Repository) newBranchName(prefix string) string {
	branchName := fmt.Sprintf("%s%s", prefix, xid.New().String())
	logrus.WithFields(logrus.Fields{
		"repository": r.FullName(),
		"branch":     branchName,
	}).Trace("Using new branch")
	return branchName
}

func (r Repository) adjustOptionsFromParams(options *UpdateOptions) {
	if draftStr, found := r.Params["draft"]; found {
		if draft, err := strconv.ParseBool(draftStr); err == nil {
			options.GitHub.PullRequest.Draft = draft
		}
	}
	if mergeStr, found := r.Params["merge"]; found {
		if merge, err := strconv.ParseBool(mergeStr); err == nil {
			options.GitHub.PullRequest.Merge.Enabled = merge
		}
	}
}

// FullName returns the repository full name.
func (r Repository) FullName() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Name)
}

// FullName returns the repository full name with the git extension
func (r Repository) GitFullName() string {
	return fmt.Sprintf("%s/%s.git", r.Owner, r.Name)
}
