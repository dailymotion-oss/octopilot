---
title: "Dynamic list"
anchor: "dynamic"
weight: 20
---

Octopilot can also be used with a "dynamic" list of repositories: repositories that are unknown when you write the command arguments.

It can retrieve the list of repositories to update from:
- one or more **environment variables**
- one or more [GitHub Search Query](https://docs.github.com/en/github/searching-for-information-on-github/searching-on-github/searching-for-repositories)

## Using environment variables

At runtime, Octopilot will read the list of repositories defined in one or more environment variables, such as:

```bash
$ export PROMOTE_TO_REPOSITORIES="my-github-org/my-first-repo my-github-org/my-second-repo(draft=true,merge=false)"
$ export ANOTHER_SET_OF_REPOSITORIES="some-org/some-repo;another-org/another-repo(draft=false)"
$ octopilot \
    --repo "discover-from(env=PROMOTE_TO_REPOSITORIES)" \
    --repo "discover-from(env=ANOTHER_SET_OF_REPOSITORIES,sep=;,draft=true)"
```

It supports the following parameters:

- `env` (string): the name of the environment variable to use, to retrieve the list of repositories.
- `sep` (string): the separator between each repository, default to `" "` (space).
- `merge` (boolean): if `true`, then the PRs created on the repositories from this env var will be automatically merged - see the [Pull Requests](#pull-request) section for more details. It overrides the value of the `--pr-merge` flag for the repositories defined in this env var.
- `draft` (boolean): if `true`, then the PRs will be created as [draft PRs](https://github.blog/2019-02-14-introducing-draft-pull-requests/) on GitHub. You will need to manually mark them as "ready for review" before being able to merge them. It overrides the value of the `--pr-draft` flag for the repositories defined in this env var.
- `branch` (string): the name of the base branch to use when cloning the repositories. Default to the `HEAD` branch - which means the default branch configured in GitHub: usually `main` or `master`.

Note that each repository listed in an environment variable supports all the parameters defined in the [static definition of a repository](#static).

## Using GitHub Search Query

A more powerful feature is the ability to load a list of repositories from a [GitHub Search Query](https://docs.github.com/en/github/searching-for-information-on-github/searching-on-github/searching-for-repositories), such as:

```bash
$ octopilot \
    --repo "discover-from(query=org:my-github-org topic:some-topic)" \
    --repo "discover-from(query=org:my-github-org in:readme some-specific-content-in-the-readme,draft=true)" \
    --repo "discover-from(query=org:my-github-org language:java is:private mirror:false archived:false,merge=true)"
```

At runtime, Octopilot will use the GitHub API to retrieve the list of repositories matching a given query. This is useful when you have a common library used/imported by many repositories, and you want to create a PR to update the version when there is a new release of your lib. Instead of hardcoding the list of "dependant" repositories in your library repository, you can use a GitHub Search Query to find all repositories with a specific topic, or specific content in the description of the repo, or specific content in the README.md file of the repo, and so on. So "dependant" repositories can easily opt-in to get automatic PRs just by adding a topic for example.

It supports the following parameters:

- `query` (string): the name of the environment variable to use, to retrieve the list of repositories.
- `merge` (boolean): if `true`, then the PRs created on the repositories from this query will be automatically merged - see the [Pull Requests](#pull-request) section for more details. It overrides the value of the `--pr-merge` flag for the repositories retrieved from this query.
- `draft` (boolean): if `true`, then the PRs will be created as [draft PRs](https://github.blog/2019-02-14-introducing-draft-pull-requests/) on GitHub. You will need to manually mark them as "ready for review" before being able to merge them. It overrides the value of the `--pr-draft` flag for the repositories retrieved from this query.
- `branch` (string): the name of the base branch to use when cloning the repositories. Default to the `HEAD` branch - which means the default branch configured in GitHub: usually `main` or `master`.

See the ["promoting a new library release" use-case](#use-case-lib-promotion) for a real-life example of what you can do with this feature.
