package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dailymotion/scribe/internal/git"
	"github.com/dailymotion/scribe/repository"
	"github.com/dailymotion/scribe/update"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

var options struct {
	updates []string
	repos   []string
	repository.UpdateOptions
	logLevel string
}

func init() {
	// required flags
	pflag.StringArrayVarP(&options.updates, "update", "u", nil, "")
	assert(pflag.CommandLine.SetAnnotation("update", "mandatory", []string{"true"}))
	pflag.StringArrayVarP(&options.repos, "repo", "r", nil, "")
	assert(pflag.CommandLine.SetAnnotation("repo", "mandatory", []string{"true"}))
	pflag.StringVar(&options.GitHub.Token, "github-token", os.Getenv("GITHUB_TOKEN"), "Mandatory GitHub token. Default to the GITHUB_TOKEN env var.")
	assert(pflag.CommandLine.SetAnnotation("github-token", "mandatory", []string{"true"}))

	// pull-request flags
	pflag.StringVar(&options.GitHub.PullRequest.Title, "pr-title", "", "")
	pflag.StringVar(&options.GitHub.PullRequest.Body, "pr-body", "", "")
	pflag.StringArrayVar(&options.GitHub.PullRequest.Comments, "pr-comment", []string{}, "")
	pflag.StringSliceVar(&options.GitHub.PullRequest.Labels, "pr-labels", []string{"scribe-update"}, "List of labels set on the pull requests, and used to find existing pull requests to update.")
	pflag.StringVar(&options.GitHub.PullRequest.BaseBranch, "pr-base-branch", "master", "Name of the branch used as a base when creating pull requests.")
	pflag.BoolVar(&options.GitHub.PullRequest.Draft, "pr-draft", false, "")
	pflag.BoolVar(&options.GitHub.PullRequest.Merge.Enabled, "pr-merge", false, "")
	pflag.StringVar(&options.GitHub.PullRequest.Merge.Method, "pr-merge-method", "merge", "")
	pflag.StringVar(&options.GitHub.PullRequest.Merge.CommitTitle, "pr-merge-commit-title", "", "")
	pflag.StringVar(&options.GitHub.PullRequest.Merge.CommitMessage, "pr-merge-commit-message", "", "")
	pflag.StringVar(&options.GitHub.PullRequest.Merge.SHA, "pr-merge-sha", "", "")
	pflag.DurationVar(&options.GitHub.PullRequest.Merge.PollTimeout, "pr-merge-poll-timeout", 10*time.Minute, "")
	pflag.DurationVar(&options.GitHub.PullRequest.Merge.PollInterval, "pr-merge-poll-interval", 30*time.Second, "")

	// git-related flags
	pflag.StringVar(&options.UpdateOptions.Git.CloneDir, "git-clone-dir", temporaryDirectory(), "")
	pflag.StringVar(&options.UpdateOptions.Git.AuthorName, "git-author-name", firstNonEmpyValue(os.Getenv("GIT_AUTHOR_NAME"), git.ConfigValue("user.name")), "")
	pflag.StringVar(&options.UpdateOptions.Git.AuthorEmail, "git-author-email", firstNonEmpyValue(os.Getenv("GIT_AUTHOR_EMAIL"), git.ConfigValue("user.email")), "")
	pflag.StringVar(&options.UpdateOptions.Git.CommitterName, "git-committer-name", firstNonEmpyValue(os.Getenv("GIT_COMMITTER_NAME"), git.ConfigValue("user.name")), "")
	pflag.StringVar(&options.UpdateOptions.Git.CommitterEmail, "git-committer-email", firstNonEmpyValue(os.Getenv("GIT_COMMITTER_EMAIL"), git.ConfigValue("user.email")), "")
	pflag.StringVar(&options.UpdateOptions.Git.CommitTitle, "git-commit-title", "", "")
	pflag.StringVar(&options.UpdateOptions.Git.CommitBody, "git-commit-body", "", "")
	pflag.StringVar(&options.UpdateOptions.Git.CommitFooter, "git-commit-footer", defaultCommitFooter(), "")
	pflag.StringVar(&options.UpdateOptions.Git.BranchPrefix, "git-branch-prefix", "scribe-", "")

	pflag.StringVar(&options.Strategy, "strategy", "reset", "Update strategy: either 'reset' (reset any existing PR from the current base branch), 'append' (append new commit to any existing PR) or 'recreate' (always create a new PR).")
	pflag.BoolVar(&options.KeepFiles, "keep-files", false, "")
	pflag.BoolVarP(&options.DryRun, "dry-run", "n", false, "")
	pflag.StringVar(&options.logLevel, "log-level", "info", "Log level. Supported values: trace, debug, info, warning, error, fatal, panic.")

	pflag.BoolP("help", "h", false, "")
	pflag.Bool("version", false, "")
}

func main() {
	pflag.Parse()
	setLogLevel()
	checkMandatoryFlags()
	printHelpOrVersion()

	logrus.WithField("updates", options.updates).Trace("Parsing updates")
	updaters, err := update.Parse(options.updates)
	if err != nil {
		logrus.
			WithError(err).
			WithField("updates", options.updates).
			Fatal("Failed to parse updates")
	}
	logrus.WithField("updaters", updaters).Debug("Updaters ready")

	logrus.WithField("repos", options.repos).Trace("Parsing repositories")
	repositories, err := repository.Parse(options.repos)
	if err != nil {
		logrus.
			WithError(err).
			WithField("repos", options.repos).
			Fatal("Failed to parse repos")
	}
	logrus.WithField("repositories", repositories).Debug("Repositories ready")

	logrus.WithField("repositories-count", len(repositories)).Trace("Starting updates")
	var (
		ctx = context.Background()
		wg  sync.WaitGroup
	)
	for _, repo := range repositories {
		wg.Add(1)
		go func(repo repository.Repository) {
			defer wg.Done()
			logrus.WithField("repository", repo.FullName()).Trace("Starting repository update")
			updated, err := repo.Update(ctx, updaters, options.UpdateOptions)
			if err != nil {
				logrus.
					WithError(err).
					WithField("repository", repo.FullName()).
					Error("Repository update failed")
				return
			}
			if !updated {
				logrus.WithField("repository", repo.FullName()).Warn("Repository update has no changes")
				return
			}
			logrus.WithField("repository", repo.FullName()).Info("Repository update finished")
		}(repo)
	}
	wg.Wait()
	logrus.WithField("repositories-count", len(repositories)).Info("Updates finished")
}

func checkMandatoryFlags() {
	var missingFlags []string
	pflag.CommandLine.VisitAll(func(flag *pflag.Flag) {
		if mandatory, found := flag.Annotations["mandatory"]; found {
			for _, v := range mandatory {
				if isMandatory, _ := strconv.ParseBool(v); isMandatory {
					switch flag.Value.Type() {
					case "string":
						if len(flag.Value.String()) == 0 {
							missingFlags = append(missingFlags, flag.Name)
						}
					case "stringSlice":
						if flag.Value.String() == "[]" {
							missingFlags = append(missingFlags, flag.Name)
						}
					}
				}
			}
		}
	})

	if len(missingFlags) == 0 {
		return
	}

	logrus.WithField("missing-flags", missingFlags).Fatal("Mandatory fields not defined")
}

func setLogLevel() {
	level, err := logrus.ParseLevel(options.logLevel)
	if err != nil {
		logrus.
			WithError(err).
			WithField("log-level", options.logLevel).
			Fatal("Invalid log level")
	}
	logrus.SetLevel(level)
}

func printHelpOrVersion() {
	if printHelp, _ := pflag.CommandLine.GetBool("help"); printHelp {
		pflag.Usage()
		os.Exit(0)
	}

	if printVersion, _ := pflag.CommandLine.GetBool("version"); printVersion {
		fmt.Printf("version ...")
		os.Exit(0)
	}
}

func temporaryDirectory() string {
	dir, err := ioutil.TempDir("", "scribe")
	if err != nil {
		dir = filepath.Join(os.TempDir(), "scribe")
	}
	return dir
}

func defaultCommitFooter() string {
	footer := new(strings.Builder)
	footer.WriteString("This is an automatic commit generated by Scribe vX.Y.Z")
	if repoURL := git.CurrentRepositoryURL(); len(repoURL) > 0 {
		footer.WriteString(fmt.Sprintf("\nRunnning from repository %s", repoURL))
	} else if currentDir, err := os.Getwd(); err == nil {
		dirName := filepath.Base(currentDir)
		footer.WriteString(fmt.Sprintf("\nRunning from %s", dirName))
	}
	return footer.String()
}

func firstNonEmpyValue(values ...string) string {
	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}
	return ""
}

func assert(err error) {
	if err != nil {
		panic(err)
	}
}
