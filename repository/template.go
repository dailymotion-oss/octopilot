package repository

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	stripmd "github.com/writeas/go-strip-markdown"
)

type templateExecutor func(text string) (string, error)

func templateExecutorFor(options UpdateOptions, repo Repository, repoPath string) templateExecutor {
	return func(text string) (string, error) {
		return executeTemplate(options, repo, repoPath, text)
	}
}

func executeTemplate(options UpdateOptions, repo Repository, repoPath string, text string) (string, error) {
	t, err := template.
		New("").
		Funcs(sprig.TxtFuncMap()).
		Funcs(template.FuncMap{
			"readFile":            tplReadFileFunc(repoPath),
			"githubRelease":       tplGitHubReleaseFunc(options.GitHub),
			"expandGithubLinks":   tplExpandGitHubLinksToMarkdownFunc(),
			"extractMarkdownURLs": tplExtractMarkdownURLsFunc(),
			"md2txt":              stripmd.Strip,
		}).
		Parse(text)
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", text, err)
	}

	var buffer bytes.Buffer
	err = t.Execute(&buffer, map[string]interface{}{
		"repo": repo,
	})
	if err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", text, err)
	}

	return buffer.String(), nil
}

func tplReadFileFunc(repoPath string) func(string) string {
	return func(path string) string {
		if !filepath.IsAbs(path) {
			path = filepath.Join(repoPath, path)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			panic(fmt.Sprintf("failed to readFile %s: %v", path, err))
		}
		return string(content)
	}
}

func tplGitHubReleaseFunc(githubOpts GitHubOptions) func(string) string {
	return func(releaseID string) string {
		elems := strings.SplitN(releaseID, "/", 3)
		if len(elems) < 3 {
			panic("invalid syntax for the commitBodyFromRelease flag - expected 3 parts got " + fmt.Sprint(len(elems)))
		}
		owner, repo, tag := elems[0], elems[1], elems[2]

		ctx := context.Background()
		ghClient, _, err := githubClient(ctx, githubOpts)
		if err != nil {
			panic(fmt.Sprintf("failed to create github client: %s", err))
		}
		release, _, err := ghClient.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
		if err != nil {
			panic(fmt.Sprintf("failed to retrieve GitHub Release for %s/%s %s: %v", owner, repo, tag, err))
		}
		return fmt.Sprintf("# **%s** release [%s](%s)\n\nReleased %s\n\n%s",
			repo, tag, release.GetHTMLURL(), release.GetPublishedAt().Format("on Monday January 2, 2006 at 15:04 (UTC)"), release.GetBody(),
		)
	}
}

func tplExpandGitHubLinksToMarkdownFunc() func(string, string) string {
	linkReg := regexp.MustCompile(`([^[]|\s)(#([0-9]+))`)
	return func(fullRepoName, input string) string {
		return linkReg.ReplaceAllString(input, fmt.Sprintf("$1[$2](https://github.com/%s/issues/$3)", fullRepoName))
	}
}

func tplExtractMarkdownURLsFunc() func(string) string {
	linkReg := regexp.MustCompile(`\[(.*?)\][\[\(](.*?)[\]\)]`)
	return func(input string) string {
		return linkReg.ReplaceAllString(input, "$2")
	}
}
