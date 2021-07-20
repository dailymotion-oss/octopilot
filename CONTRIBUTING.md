# Contributing to Octopilot

So if you're here, it's most likely because you want to contribute to Octopilot. Thanks!

First, if you're not sure of how to proceed, or need some help, you should start with [opening an issue](https://github.com/dailymotion-oss/octopilot/issues/new) to start a discussion.

## Setup

If you're new to [GitHub](https://github.com/) and/or open-source, you should read GitHub's guide on [forking a repository](https://docs.github.com/en/get-started/quickstart/fork-a-repo).

### Tooling

The best place to find the exact versions of the tools - [Go](https://golang.org/), [Hugo](https://gohugo.io/), ... - used is the `.github/workflows` directory.

## Making a change

- If you want to **fix a bug** or **add a new feature**, the best place to start is the [ARCHITECTURE.md](ARCHITECTURE.md) file.
  - If you're adding a new feature, please don't forget the documentation ;-)
  - If you're changing an existing feature, please do so in a backward-compatible way if possible.
- If you want to **improve the documentation**, have a look at the [docs/README.md](docs/README.md) file.

## Submitting a change

- Please follow the [Conventional Commits](https://conventionalcommits.org/) spec when writing your git commits.
  - We're using this spec to generate the next release version, using [jx-release-version](https://github.com/jenkins-x-plugins/jx-release-version).
- Add as much details as possible to your commit message and/or pull request.
  - In particular, you should explain what you want to achieve, it helps reviewers to understand what you're trying to do and if you are using the right approach.
- Once your pull request will be merged, a new release will be created automatically.

## Reporting an issue

Contributing is not limited to fixing bugs and adding features. If you have a question, a bug report or a feature request, please [create an issue](https://github.com/dailymotion-oss/octopilot/issues/new).
