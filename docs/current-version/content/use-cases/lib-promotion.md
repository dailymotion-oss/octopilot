---
title: "Promoting a new library release"
anchor: "use-case-lib-promotion"
weight: 15
---

Promoting a new release of a library is similar to [promoting a new release of an application](#use-case-app-promotion), except that instead of promoting to a small number of git repositories - representing the different environments - you will promote to a much larger number of git repositories: one for each application using the library.

The main change is that instead of using a [static list](#static) of repositories, you can use a [dynamic list](#dynamic) of repositories - based on a [GitHub Search Query](https://docs.github.com/en/github/searching-for-information-on-github/searching-on-github/searching-for-repositories) for example.

Here is an example of an Octopilot invocation using a dynamic list of repos - we'll go over it in details later:

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

We're using the following GitHub query to discover the repositories that are using the `my-base-chart` Helm chart:

```
query=org:my-org topic:octopilot-my-base-chart
```

Which means that developers just need to add the `octopilot-my-base-chart` topic to their repositories - in the GitHub UI - to enable automatic update of the Helm base chart. It's easy to add, and easy to turn off if you don't want it. Note that you are not limited to searching by topic - you can also search for specific content in the repository's name, description, `README.md` file, and so on.

We're using the [Helm updater](#helm) to update all the [Helm](https://helm.sh/) charts, and set the version of their dependency on `my-base-chart` to the new release version.

Everything else is similar to the [promotion of a new application release](#use-case-app-promotion).

## Result

This is the screenshot of the bottom of the library PR which is at the origin of the promotion - and which is referenced in the library's release notes.

![](screenshot-lib-promotion-pr-feedback.png)

You can see that multiple promotion pull requests have been created, one per repository matching the GitHub search query. This is an easy way to see which application repositories have upgraded to the new version of the library.
