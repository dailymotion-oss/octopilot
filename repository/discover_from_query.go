package repository

import (
	"context"
	"fmt"

	"github.com/google/go-github/v36/github"
)

func discoverRepositoriesFromQuery(ctx context.Context, searchType SearchType, query string, params map[string]string, githubOpts GitHubOptions) ([]Repository, error) {
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
			repos, resp, err = searchCodeRepositories(ctx, ghClient, query, opts, params)
		} else {
			repos, resp, err = searchRepositories(ctx, ghClient, query, opts, params)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to list all repositories matching query %s on GitHub (page %d): %w", query, page, err)
		}

		page = resp.NextPage
		if resp.NextPage == 0 {
			break
		}
	}

	return repos, nil
}
