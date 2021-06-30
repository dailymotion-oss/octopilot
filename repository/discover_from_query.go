package repository

import (
	"context"
	"fmt"

	"github.com/google/go-github/v36/github"
)

func discoverRepositoriesFromQuery(ctx context.Context, query string, params map[string]string, githubOpts GitHubOptions) ([]Repository, error) {
	repos := []Repository{}
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
		result, resp, err := ghClient.Search.Repositories(ctx, query, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list all repositories matching query %s on GitHub (page %d): %w", query, page, err)
		}

		for _, ghRepo := range result.Repositories {
			repos = append(repos, Repository{
				Owner:  ghRepo.Owner.GetLogin(),
				Name:   ghRepo.GetName(),
				Params: params,
			})
		}

		page = resp.NextPage
		if resp.NextPage == 0 {
			break
		}
	}

	return repos, nil
}
