---
title: "Static list"
anchor: "static"
weight: 10
---

It is the easiest: just specify the repositories on the CLI using the `--repo` flag, such as:

```bash
$ octopilot \
    --repo "my-github-org/my-first-repo" \
    --repo "my-github-org/my-second-repo(merge=true)" \
    --repo "my-github-org/my-third-repo(draft=true,merge=false,branch=dev)"
```

You can add as much repositories as you want, each with different configuration.

It supports the following parameters:

- `merge` (boolean): if `true`, then the PR created on this repository will be automatically merged - see the [Pull Requests](#pull-request) section for more details. It overrides the value of the `--pr-merge` flag for this specific repository.
- `mergeauto` (boolean): if `true`, then the PR will merged by Github's auto-merge PR feature. See the [Pull Requests](#pull-request) section for more details. It overrides the value of the `--pr-merge-auto` flag for this specific repository.
- `mergeautowait` (boolean): if `true`, then wait until the PR is actually merged. See the [Pull Requests](#pull-request) section for more details. It overrides the value of the `--pr-merge-auto-wait` flag for this specific repository.
- `draft` (boolean): if `true`, then the PR will be created as a [draft PR](https://github.blog/2019-02-14-introducing-draft-pull-requests/) on GitHub. You will need to manually mark it as "ready for review" before being able to merge it. It overrides the value of the `--pr-draft` flag for this specific repository.
- `branch` (string): the name of the base branch to use when cloning the repository. Default to the `HEAD` branch - which means the default branch configured in GitHub: usually `main` or `master`.
