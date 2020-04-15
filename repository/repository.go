package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/dailymotion/octopilot/internal/parameters"
	"github.com/dailymotion/octopilot/update"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
)

var (
	// owner/name(params)
	repoRegexp = regexp.MustCompile(`^(?P<owner>[A-Za-z0-9_\-]+)/(?P<name>[A-Za-z0-9_\-]+)(?:\((?P<params>.+)\))?$`)
)

type Repository struct {
	Owner  string
	Name   string
	Params map[string]string
}

func Parse(repos []string) ([]Repository, error) {
	var repositories []Repository
	for _, repo := range repos {
		matches := repoRegexp.FindStringSubmatch(repo)
		if len(matches) < 4 {
			return nil, fmt.Errorf("invalid syntax for %s: found %d matches instead of 4: %v", repo, len(matches), matches)
		}

		r := Repository{
			Owner:  matches[1],
			Name:   matches[2],
			Params: parameters.Parse(matches[3]),
		}

		repositories = append(repositories, r)
	}
	return repositories, nil
}

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

func (r Repository) FullName() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Name)
}
