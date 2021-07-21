package repository

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v36/github"
	"github.com/sirupsen/logrus"
	"github.com/ybbus/httpretry"
	"golang.org/x/oauth2"
)

func githubClient(ctx context.Context, ghOptions GitHubOptions) (*github.Client, string, error) {
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
		httpClient, token, err = githubAppClient(ctx, ghOptions.AppID, ghOptions.InstallationID, ghOptions.PrivateKey, ghOptions.PrivateKeyPath)
	default:
		return nil, "", fmt.Errorf("GitHub auth method unrecognized (allowed values: app, token): %s", ghOptions.AuthMethod)
	}
	if err != nil {
		return nil, "", err
	}
	httpClient = httpretry.NewCustomClient(httpClient)
	ghc := github.NewClient(httpClient)
	return ghc, token, nil
}

func githubTokenClient(ctx context.Context, token string) (*http.Client, string, error) { // nolint: unparam // the returned error is not used, but we need it for the method signature
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	return oauth2.NewClient(ctx, tokenSource), token, nil
}

func githubAppClient(ctx context.Context, appID int64, installationID int64, privateKey string, privateKeyPath string) (*http.Client, string, error) {
	var (
		tr  = http.DefaultTransport
		itr *ghinstallation.Transport
		err error
	)
	if len(privateKey) > 0 {
		itr, err = ghinstallation.New(tr, appID, installationID, []byte(privateKey))
	} else {
		itr, err = ghinstallation.NewKeyFromFile(tr, appID, installationID, privateKeyPath)
	}
	if err != nil {
		return nil, "", err
	}
	token, err := itr.Token(ctx)
	if err != nil {
		return nil, "", err
	}
	return &http.Client{Transport: itr}, token, err
}
