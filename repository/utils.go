package repository

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"slices"

	"github.com/google/go-github/v57/github"
)

// removeDuplicate removes duplicate repositories from the input list and returns a new slice of unique repositories.
// It checks for duplicates based on the combination of the "Owner" and "Name" attributes
func removeDuplicate(inputList []Repository) []Repository {
	if len(inputList) == 0 {
		return inputList
	}

	seen := make(map[string]bool)
	isDuplicate := func(repo Repository) bool {
		key := fmt.Sprintf("%s-%s", repo.Owner, repo.Name)
		if seen[key] {
			return true
		}
		seen[key] = true
		return false
	}
	repositories := slices.DeleteFunc(inputList, isDuplicate)

	return repositories
}

// parseSearchType converts a string representation to a SearchType.
// It maps specific string values to corresponding SearchType constants.
// If the input string is not recognized, it defaults to Repositories.
func parseSearchType(str string) SearchType {
	if searchType, ok := searchTypeMap[str]; ok {
		return searchType
	}

	return Repositories
}

// searchCodeRepositories searches GitHub repositories using the Github Code Search feature
func searchCodeRepositories(ctx context.Context, ghClient *github.Client, query string, opts *github.SearchOptions, params map[string]string) ([]Repository, *github.Response, error) {
	repos := []Repository{}
	codeResults, resp, err := ghClient.Search.Code(ctx, query, opts)
	if err != nil {
		return repos, nil, err
	}

	for _, result := range codeResults.CodeResults {
		repos = append(repos, Repository{
			Owner:  result.Repository.Owner.GetLogin(),
			Name:   result.Repository.GetName(),
			Params: params,
		})
	}

	return repos, resp, nil
}

// searchRepositories searches GitHub repositories using the Github Repositories Search feature
func searchRepositories(ctx context.Context, ghClient *github.Client, query string, opts *github.SearchOptions, params map[string]string) ([]Repository, *github.Response, error) {
	repos := []Repository{}
	repoResults, resp, err := ghClient.Search.Repositories(ctx, query, opts)
	if err != nil {
		return repos, nil, err
	}

	for _, result := range repoResults.Repositories {
		repos = append(repos, Repository{
			Owner:  result.Owner.GetLogin(),
			Name:   result.GetName(),
			Params: params,
		})
	}

	return repos, resp, nil
}

// base64EncodeFile returns the contents of a file in base64
func base64EncodeFile(path string) (string, error) {
	fileContent, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("unable to read file: %w", err)
	}
	return base64.StdEncoding.EncodeToString(fileContent), nil
}
