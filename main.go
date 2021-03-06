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

	"github.com/dailymotion-oss/octopilot/internal/git"
	"github.com/dailymotion-oss/octopilot/repository"
	"github.com/dailymotion-oss/octopilot/update"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

// the following build-related variables are set at release-time by goreleaser
// using ldflags
var (
	buildVersion = "dev"
	buildCommit  = "none"
	buildDate    = "unknown"
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
	pflag.StringVar(&options.GitHub.AuthMethod, "github-auth-method", "token", "Mandatory GitHub authentication method: can be `token` or `app`. Defaults to token.")
	assert(pflag.CommandLine.SetAnnotation("github-auth-method", "mandatory", []string{"true"}))

	// GitHub auth flags
	pflag.StringVar(&options.GitHub.Token, "github-token", os.Getenv("GITHUB_TOKEN"), "For the `token` GitHub auth method, contains the GitHub token. Default to the GITHUB_TOKEN env var.")
	pflag.Int64Var(&options.GitHub.AppID, "github-app-id", int64(getenvInt("GITHUB_APP_ID")), "For the `app` GitHub auth method, contains the GitHubApp AppID. Default to the GITHUB_APP_ID env var.")
	pflag.Int64Var(&options.GitHub.InstallationID, "github-installation-id", int64(getenvInt("GITHUB_INSTALLATION_ID")), "For the `app` GitHub auth method, contains the GitHubApp Installation ID. Default to the GITHUB_INSTALLATION_ID env var.")
	pflag.StringVar(&options.GitHub.PrivateKey, "github-privatekey", os.Getenv("GITHUB_PRIVATEKEY"), "For the `app` GitHub auth method, contains the GitHubApp Private key file in PEM format. Default to the GITHUB_PRIVATEKEY env var.")
	pflag.StringVar(&options.GitHub.PrivateKeyPath, "github-privatekey-path", os.Getenv("GITHUB_PRIVATEKEY_PATH"), "For the `app` GitHub auth method, contains the GitHubApp Private key file path `/some/key.pem` (used if the github-privatekey is empty). Default to the GITHUB_PRIVATEKEY_PATH env var.")

	// pull-request flags
	pflag.StringVar(&options.GitHub.PullRequest.Title, "pr-title", "", "")
	pflag.StringVar(&options.GitHub.PullRequest.TitleUpdateOperation, "pr-title-update-operation", "", "operation when updating the PR's title: ignore (keep old value), replace, prepend or append. Default is: ignore for append strategy, replace for reset strategy.")
	pflag.StringVar(&options.GitHub.PullRequest.Body, "pr-body", "", "")
	pflag.StringVar(&options.GitHub.PullRequest.BodyUpdateOperation, "pr-body-update-operation", "", "operation when updating the PR's body: ignore (keep old value), replace, prepend or append. Default is: ignore for append strategy, replace for reset strategy.")
	pflag.StringArrayVar(&options.GitHub.PullRequest.Comments, "pr-comment", []string{}, "")
	pflag.StringSliceVar(&options.GitHub.PullRequest.Labels, "pr-labels", []string{"octopilot-update"}, "List of labels set on the pull requests, and used to find existing pull requests to update.")
	pflag.StringVar(&options.GitHub.PullRequest.BaseBranch, "pr-base-branch", "master", "Name of the branch used as a base when creating pull requests.")
	pflag.BoolVar(&options.GitHub.PullRequest.Draft, "pr-draft", false, "")
	pflag.BoolVar(&options.GitHub.PullRequest.Merge.Enabled, "pr-merge", false, "")
	pflag.StringVar(&options.GitHub.PullRequest.Merge.Method, "pr-merge-method", "merge", "")
	pflag.StringVar(&options.GitHub.PullRequest.Merge.CommitTitle, "pr-merge-commit-title", "", "")
	pflag.StringVar(&options.GitHub.PullRequest.Merge.CommitMessage, "pr-merge-commit-message", "", "")
	pflag.StringVar(&options.GitHub.PullRequest.Merge.SHA, "pr-merge-sha", "", "")
	pflag.DurationVar(&options.GitHub.PullRequest.Merge.PollTimeout, "pr-merge-poll-timeout", 10*time.Minute, "")
	pflag.DurationVar(&options.GitHub.PullRequest.Merge.PollInterval, "pr-merge-poll-interval", 30*time.Second, "")
	pflag.IntVar(&options.GitHub.PullRequest.Merge.RetryCount, "pr-merge-retry-count", 3, "")

	// git-related flags
	pflag.StringVar(&options.UpdateOptions.Git.CloneDir, "git-clone-dir", temporaryDirectory(), "")
	pflag.StringArrayVar(&options.UpdateOptions.Git.StagePatterns, "git-stage-pattern", nil, "")
	pflag.BoolVar(&options.UpdateOptions.Git.StageAllChanged, "git-stage-all-changed", true, "")
	pflag.StringVar(&options.UpdateOptions.Git.AuthorName, "git-author-name", firstNonEmpyValue(os.Getenv("GIT_AUTHOR_NAME"), git.ConfigValue("user.name")), "")
	pflag.StringVar(&options.UpdateOptions.Git.AuthorEmail, "git-author-email", firstNonEmpyValue(os.Getenv("GIT_AUTHOR_EMAIL"), git.ConfigValue("user.email")), "")
	pflag.StringVar(&options.UpdateOptions.Git.CommitterName, "git-committer-name", firstNonEmpyValue(os.Getenv("GIT_COMMITTER_NAME"), git.ConfigValue("user.name")), "")
	pflag.StringVar(&options.UpdateOptions.Git.CommitterEmail, "git-committer-email", firstNonEmpyValue(os.Getenv("GIT_COMMITTER_EMAIL"), git.ConfigValue("user.email")), "")
	pflag.StringVar(&options.UpdateOptions.Git.CommitTitle, "git-commit-title", "", "")
	pflag.StringVar(&options.UpdateOptions.Git.CommitBody, "git-commit-body", "", "")
	pflag.StringVar(&options.UpdateOptions.Git.CommitFooter, "git-commit-footer", defaultCommitFooter(), "")
	pflag.StringVar(&options.UpdateOptions.Git.BranchPrefix, "git-branch-prefix", "octopilot-", "")

	pflag.StringVar(&options.Strategy, "strategy", "reset", "Update strategy: either 'reset' (reset any existing PR from the current base branch), 'append' (append new commit to any existing PR) or 'recreate' (always create a new PR).")
	pflag.BoolVar(&options.KeepFiles, "keep-files", false, "")
	pflag.BoolVarP(&options.DryRun, "dry-run", "n", false, "")
	pflag.StringVar(&options.logLevel, "log-level", "info", "Log level. Supported values: trace, debug, info, warning, error, fatal, panic.")

	pflag.BoolP("help", "h", false, "")
	pflag.Bool("version", false, "")
}

func main() {
	ctx := context.Background()
	pflag.Parse()
	printHelpOrVersion()
	setLogLevel()
	checkMandatoryFlags()

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
	repositories, err := repository.Parse(ctx, options.repos, options.GitHub)
	if err != nil {
		logrus.
			WithError(err).
			WithField("repos", options.repos).
			Fatal("Failed to parse repos")
	}
	logrus.WithField("repositories", repositories).Debug("Repositories ready")

	logrus.WithField("repositories-count", len(repositories)).Trace("Starting updates")
	var wg sync.WaitGroup
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
		fmt.Printf("version %v, commit %v, built at %v", buildVersion, buildCommit, buildDate)
		os.Exit(0)
	}
}

func temporaryDirectory() string {
	dir, err := ioutil.TempDir("", "octopilot")
	if err != nil {
		dir = filepath.Join(os.TempDir(), "octopilot")
	}
	return dir
}

func defaultCommitFooter() string {
	footer := new(strings.Builder)
	footer.WriteString("Generated by [OctoPilot](https://github.com/dailymotion-oss/octopilot)")
	if buildVersion == "dev" {
		footer.WriteString(" (dev version)")
	} else {
		footer.WriteString(fmt.Sprintf(" [v%[1]s](https://github.com/dailymotion-oss/octopilot/releases/tag/v%[1]s)", buildVersion))
	}
	if repoURL := git.CurrentRepositoryURL(); len(repoURL) > 0 {
		footer.WriteString(fmt.Sprintf(" from %s", repoURL))
	} else if currentDir, err := os.Getwd(); err == nil {
		dirName := filepath.Base(currentDir)
		footer.WriteString(fmt.Sprintf(" from %s", dirName))
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

func getenvInt(key string) int {
	s := os.Getenv(key)
	if s != "" {
		v, err := strconv.Atoi(s)
		assert(err)
		return v
	}
	return 0
}
