---
title: "GitHub Auth"
anchor: "github-auth"
weight: 50
---

Octopilot needs a way to authenticate against the GitHub API, to:
- clone the repositories
- push new changes
- and create/update/merge Pull Requests

It supports 2 ways to authenticate:
- using a personal access **token**, which is the default
- using a GitHub **app**

You can define which method to use, using the `--github-auth-method` CLI flag.

## Personal Access Token

By default, the `--github-auth-token` flag is set to `token`, so Octopilot will use a **Personal Access Token** - or `PAT`. This token can be defined either by the `GITHUB_TOKEN` environment variable, or by setting the `--github-token` CLI flag.

You can read GitHub's documentation on [creating a personal access token](https://docs.github.com/en/github/authenticating-to-github/keeping-your-account-and-data-secure/creating-a-personal-access-token). You'll need at least the `repo` permissions.

## GitHub App

An alternative to the "simple" token is to use a [GitHub App](https://docs.github.com/en/developers/apps).

First, you'll need to set the `--github-auth-token` flag value to `app`, and then configure the following settings:
- `--github-app-id` (int): the GitHub App ID. Default to the value of the `GITHUB_APP_ID` environment variable.
- `--github-installation-id` (int): the GitHub App installation ID. Default to the value of the `GITHUB_INSTALLATION_ID` environment variable.
- `--github-privatekey` (string): the app's private key - used to sign access token requests - in PEM format. Default to the value of the `GITHUB_PRIVATEKEY` environment variable. You can either set this, or the `--github-privatekey-path`.
- `--github-privatekey-path` (string): the path to the app's private key - used to sign access token requests - in PEM format. Default to the value of the `GITHUB_PRIVATEKEY_PATH` environment variable. Will be used if the `--github-privatekey` flag is not set.

See GitHub's documentation for more details on:
- [Creating a GitHub App](https://docs.github.com/en/developers/apps/building-github-apps/creating-a-github-app)
- [Authenticating with GitHub Apps](https://docs.github.com/en/developers/apps/building-github-apps/authenticating-with-github-apps)
