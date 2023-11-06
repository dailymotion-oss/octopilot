---
title: "Promoting a new library release"
anchor: "use-case-lib-promotion"
weight: 15
---

Promoting a new release of a library is similar to [promoting a new release of an application](#use-case-app-promotion), except that instead of promoting to a small number of git repositories - representing the different environments - you will promote to a much larger number of git repositories: one for each application using the library.

The main change is that instead of using a [static list](#static) of repositories, you can use a [dynamic list](#dynamic) of repositories - based on a [GitHub Repositories Search Query](https://docs.github.com/en/github/searching-for-information-on-github/searching-on-github/searching-for-repositories) or a [GitHub Code Search Query](https://docs.github.com/en/search-github/searching-on-github/searching-code) for example.

## GitHub Repositories Search Query

Here is an example of an Octopilot invocation using a dynamic list of repos and GitHub Repositories Search Query - we'll go over it in details later:

```bash
$ export GITHUB_TOKEN=<your_github_token>
$ export VERSION=1.0.15
$ octopilot \
    --repo "discover-from(query=org:my-org topic:octopilot-my-base-chart)" \
    --update "helm(dependency=my-base-chart)=${VERSION}" \
    --git-commit-title 'chore(deps): update my-base-chart to {{ env "VERSION" }}' \
    --git-commit-body '{{ githubRelease (printf "my-org/my-base-chart/v%s" (env "VERSION")) | expandGithubLinks "my-org/my-base-chart" | extractMarkdownURLs | md2txt }}' \
    --git-branch-prefix "octopilot-update-my-base-chart-" \
    --pr-labels "update-my-base-chart" \
    --pr-title "Update Helm Base Chart to ${VERSION}" \
    --pr-title-update-operation "replace" \
    --pr-body '{{ githubRelease (printf "my-org/my-base-chart/v%s" (env "VERSION")) | expandGithubLinks "my-org/my-base-chart" }}' \
    --pr-body-update-operation "prepend" \
    --strategy "append"
```

We're using the following GitHub Repositories search query to discover the repositories that are using the `my-base-chart` Helm chart:

```
query=org:my-org topic:octopilot-my-base-chart
```

Which means that developers just need to add the `octopilot-my-base-chart` topic to their repositories - in the GitHub UI - to enable automatic update of the Helm base chart. It's easy to add, and easy to turn off if you don't want it. Note that you are not limited to searching by topic - you can also search for specific content in the repository's name, description, `README.md` file, and so on.

We're using the [Helm updater](#helm) to update all the [Helm](https://helm.sh/) charts, and set the version of their dependency on `my-base-chart` to the new release version.

Everything else is similar to the [promotion of a new application release](#use-case-app-promotion).

## GitHub Code Search Query

This search type is useful in case you experience some limitations with the default `Repositories` one
- 50 characters max
- 20 topics max per repository

Here is an example of an Octopilot invocation using a dynamic list of repos and GitHub Code Search Query - we'll go over it in details later:

```bash
$ export GITHUB_TOKEN=<your_github_token>
$ export VERSION=1.0.15
$ octopilot \
    --repo "discover-from(searchtype=code,query=org:my-org filename:Chart.yaml path:charts my-base-chart)" \
    --update "yaml(file=charts/*/Chart.yaml,path=dependencies.(name==my-base-chart).version,style=folded)=${VERSION}" \
    --git-commit-title 'chore(deps): update my-base-chart to {{ env "VERSION" }}' \
    --git-commit-body '{{ githubRelease (printf "my-org/my-base-chart/v%s" (env "VERSION")) | expandGithubLinks "my-org/my-base-chart" | extractMarkdownURLs | md2txt }}' \
    --git-branch-prefix "octopilot-update-my-base-chart-" \
    --pr-labels "update-my-base-chart" \
    --pr-title "Update Helm Base Chart to ${VERSION}" \
    --pr-title-update-operation "replace" \
    --pr-body '{{ githubRelease (printf "my-org/my-base-chart/v%s" (env "VERSION")) | expandGithubLinks "my-org/my-base-chart" }}' \
    --pr-body-update-operation "prepend" \
    --strategy "append"
```

We can use the following GitHub Code search type and query to discover the repositories that are using the `my-base-chart` Helm chart:

```
searchtype=code,query=org:my-org filename:Chart.yaml path:charts my-base-chart
```
By default `searchtype` is equal to `repositories`, because here we need to use the Github Code Search query, we have to put it to code. 
With Github Code Search query, developers don't need to do anything to enable automatic update of their Helm base chart.

We're using the [Helm updater](#helm) to update all the [Helm](https://helm.sh/) charts, and set the version of their dependency on `my-base-chart` to the new release version.

Everything else is similar to the [promotion of a new application release](#use-case-app-promotion).

## Result

This is the screenshot of the bottom of the library PR which is at the origin of the promotion - and which is referenced in the library's release notes.

![](screenshot-lib-promotion-pr-feedback.png)

You can see that multiple promotion pull requests have been created, one per repository matching the GitHub search query. This is an easy way to see which application repositories have upgraded to the new version of the library.
