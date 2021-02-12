package repository

import (
	"context"
	"fmt"
	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"
	"github.com/ybbus/httpretry"
	"golang.org/x/oauth2"
	"net/http"
)

func githubClient(ctx context.Context, ghOptions GitHubOptions) (*github.Client, string, error) {
	var httpClient *http.Client
	var token string
	var err error
	logrus.Debugf("Creating gh client: %v", ghOptions.AuthMethod)
	if ghOptions.AuthMethod == "token" {
		httpClient, token, err = githubTokenClient(ctx, ghOptions.Token)
	} else if ghOptions.AuthMethod == "app" {
		httpClient, token, err = githubAppClient(ctx, ghOptions.AppID, ghOptions.InstallationID, ghOptions.PrivateKey, ghOptions.PrivateKeyPath)
	} else {
		return nil, "", fmt.Errorf("GitHub auth method unrecognized (allowed values: app, token): %s", ghOptions.AuthMethod)
	}
	if err != nil {
		return nil, "", err
	}
	httpClient = httpretry.NewCustomClient(httpClient)
	ghc := github.NewClient(httpClient)
	return ghc, token, nil
}

func githubTokenClient(ctx context.Context, token string) (*http.Client, string, error) {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	return oauth2.NewClient(ctx, tokenSource), token, nil
}

func githubAppClient(ctx context.Context, appId int64, installationId int64, privateKey string, privateKeyPath string) (*http.Client, string, error) {
	tr := http.DefaultTransport
	var itr *ghinstallation.Transport
	var err error
	if len(privateKey) > 0 {
		itr, err = ghinstallation.New(tr, appId, installationId, []byte(privateKey))
	} else {
		itr, err = ghinstallation.NewKeyFromFile(tr, appId, installationId, privateKeyPath)
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
