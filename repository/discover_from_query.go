package repository

import (
	"context"
	"fmt"

	"github.com/google/go-github/v57/github"
)

func discoverRepositoriesFromQuery(ctx context.Context, searchType SearchType, query string, params map[string]string, githubOpts GitHubOptions) ([]Repository, error) {
	var pageRepos []Repository
	var repos []Repository
	var resp *github.Response

	ghClient, _, err := githubClient(ctx, githubOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create github client: %w", err)
	}

	page := 1
	for {
		opts := &github.SearchOptions{
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: 100,
			},
		}
		if searchType == Code {
			pageRepos, resp, err = searchCodeRepositories(ctx, ghClient, query, opts, params)
		} else {
			pageRepos, resp, err = searchRepositories(ctx, ghClient, query, opts, params)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list all repositories matching query %s on GitHub (page %d): %w", query, page, err)
		}

		repos = append(repos, pageRepos...)

		page = resp.NextPage
		if resp.NextPage == 0 {
			break
		}
	}

	return repos, nil
}
