package repository

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v57/github"
	"github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
	"github.com/ybbus/httpretry"
	"golang.org/x/oauth2"
)

func githubAuthenticatedHTTPClient(ctx context.Context, ghOptions GitHubOptions) (*http.Client, string, error) {
	var (
		httpClient *http.Client
		token      string
		err        error
	)
	logrus.Tracef("Creating github client using auth method %q", ghOptions.AuthMethod)
	switch ghOptions.AuthMethod {
	case "token":
		httpClient, token, err = githubTokenClient(ctx, ghOptions.Token)
	case "app":
		httpClient, token, err = githubAppClient(ctx, ghOptions)
	default:
		return nil, "", fmt.Errorf("GitHub auth method unrecognized (allowed values: app, token): %s", ghOptions.AuthMethod)
	}
	if err != nil {
		return nil, "", err
	}
	httpClient = httpretry.NewCustomClient(httpClient)

	return httpClient, token, nil
}

func githubClient(ctx context.Context, ghOptions GitHubOptions) (*github.Client, string, error) {
	httpClient, token, err := githubAuthenticatedHTTPClient(ctx, ghOptions)

	if err != nil {
		return nil, "", err
	}

	var ghc *github.Client
	if ghOptions.isEnterprise() {
		var err error
		ghc, err = github.NewClient(httpClient).WithEnterpriseURLs(ghOptions.URL, ghOptions.URL)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create an enterprise client: %w", err)
		}
	} else {
		ghc = github.NewClient(httpClient)
	}
	return ghc, token, nil
}

type graphqlTransport struct {
	base http.RoundTripper
}

func (t *graphqlTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Enable PullRequest.mergeStateStatus
	req.Header.Add("accept", "application/vnd.github.merge-info-preview+json")

	return t.base.RoundTrip(req)
}

func githubGraphqlClient(ctx context.Context, ghOptions GitHubOptions) (*githubv4.Client, error) {
	httpClient, _, err := githubAuthenticatedHTTPClient(ctx, ghOptions)

	httpClient.Transport = &graphqlTransport{base: httpClient.Transport}

	if err != nil {
		return nil, err
	}

	if ghOptions.isEnterprise() {
		apiURL, err := url.JoinPath(ghOptions.URL, "/api/graphql")

		if err != nil {
			return nil, fmt.Errorf("failed to build GraphQL API URL: %w", err)
		}

		return githubv4.NewEnterpriseClient(apiURL, httpClient), nil
	}

	return githubv4.NewClient(httpClient), nil
}

func githubTokenClient(ctx context.Context, token string) (*http.Client, string, error) { //nolint: unparam // the returned error is not used, but we need it for the method signature
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	return oauth2.NewClient(ctx, tokenSource), token, nil
}

func githubAppClient(ctx context.Context, ghOptions GitHubOptions) (*http.Client, string, error) {
	var (
		tr  = http.DefaultTransport
		itr *ghinstallation.Transport
		err error
	)
	if len(ghOptions.PrivateKey) > 0 {
		itr, err = ghinstallation.New(tr, ghOptions.AppID, ghOptions.InstallationID, []byte(ghOptions.PrivateKey))
	} else {
		itr, err = ghinstallation.NewKeyFromFile(tr, ghOptions.AppID, ghOptions.InstallationID, ghOptions.PrivateKey)
	}
	if err != nil {
		return nil, "", err
	}
	if ghOptions.isEnterprise() {
		itr.BaseURL = ghOptions.URL + "/api/v3"
	}
	token, err := itr.Token(ctx)
	if err != nil {
		return nil, "", err
	}
	return &http.Client{Transport: itr}, token, err
}
