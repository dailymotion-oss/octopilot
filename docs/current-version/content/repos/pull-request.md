---
title: "Pull Requests"
anchor: "pull-request"
weight: 60
---

After running all the [updaters](#updaters) and creating a [git commit](#commit), octopilot will create a Pull Request for each [repository](#repos).

## Strategies

Octopilot has 3 strategies for creating a Pull Request:
- **reset** (the default): reset any existing Pull Request from the base branch
- **append**: append new commits to any existing Pull Request
- **recreate**: always create a new Pull Request

You can control which strategy to use using the `--strategy` CLI flag.

### Reset Strategy

This is the default strategy. With this strategy, Octopilot will reset any existing Pull Request from the base branch.

In detail, it will:
- clone the git repository
- find a "matching" Pull Request - based on the pre-configured labels. If there is a matching Pull Request, it will use the PR's branch. Otherwise, it will just create a new branch
- reset the branch to the base branch - usually `main` or `master`
- run the [updaters](#updaters)
- [commit](#commit) the changes and (force) push the commit
- update the existing Pull Request title/body/labels/comments/assignees or create a new one

Note that you can control how the existing Pull Request will be updated. For both the title and body, you can either:
- **ignore** the new changes. The existing PR won't be changed - except for the labels, comments, assignees, and of course the commit.
- **replace** the title and/or body with the new ones. This is the default for the **reset** strategy.
- **prepend** the title and/or body with the new ones. This is mostly useful for the body.
- **append** the title and/or body with the new ones. This is mostly useful for the body.

### Append Strategy

With this strategy, Octopilot will append new commits to any existing Pull Request.

In detail, it will:
- clone the git repository
- find a "matching" Pull Request - based on the pre-configured labels. If there is a matching Pull Request, it will switch to the PR's branch. Otherwise it will just create a new branch from the base branch, and switch to it.
- run the [updaters](#updaters)
- [commit](#commit) the changes and push the commit
- update the existing Pull Request title/body/labels/comments/asignees or create a new one

Note that you can control how the existing Pull Request will be updated. For both the title and body, you can either:
- **ignore** the new changes. The existing PR won't be changed - except for the labels, comments, assignees, and of course the commit. This is the default for the **append** strategy.
- **replace** the title and/or body with the new ones.
- **prepend** the title and/or body with the new ones. This is mostly useful for the body.
- **append** the title and/or body with the new ones. This is mostly useful for the body.

### Recreate Strategy

With this strategy, Octopilot will always create a new Pull Request.

In detail, it will:
- clone the git repository
- create a new branch from the base branch, and switch to it
- run the [updaters](#updaters)
- [commit](#commit) the changes and push the commit
- create a new Pull Request

## Creating / updating Pull Requests

You can control how the Pull Requests will be created or updated using the following CLI flags:

- `--strategy` (string): strategy to use when creating/updating the Pull Requests: either `reset` (reset any existing PR from the current base branch), `append` (append new commit to any existing PR) or `recreate` (always create a new PR). Default to `reset`.
- `--dry-run` (bool): if enabled, won't perform any operation on the remote git repository or on GitHub: all operations will be done in the local cloned repository. So no Pull Request will be created/updated. Default to `false`.
- `--pr-title` (string): the title of the Pull Request. Default to the commit title. Note that you can use the [templating](#templating) feature here.
- `--pr-title-update-operation` (string): the type of operation when updating a Pull Request's title: either `ignore` (keep old value), `replace`, `prepend` or `append`. Default is: `ignore` for "append" strategy, `replace` for "reset" strategy, and not applicable for "recreate" strategy.
- `--pr-body` (string): the body of the Pull Request. Default to the commit body and the commit footer. Note that you can use the [templating](#templating) feature here.
- `--pr-body-update-operation` (string): the type of operation when updating a Pull Request's body: either `ignore` (keep old value), `replace`, `prepend` or `append`. Default is: `ignore` for "append" strategy, `replace` for "reset" strategy, and not applicable for "recreate" strategy.
- `--pr-comment` (array of string): optional list of comments to add to the Pull Request.
- `--pr-assignees` (array of string): optional list of assignees (Github usernames) to add to the Pull Request.
- `--pr-labels` (array of string): optional list of labels to set on the pull requests, and used to find existing pull requests to update. Default to `["octopilot-update"]`.
- `--pr-base-branch` (string): name of the branch used as a base when creating pull requests. Default to `master`.
- `--pr-draft` (bool): if enabled, the Pull Request will be created as a draft - instead of regular ones. It means that the PRs can't be merged until marked as "ready for review". Default to `false`.

## Merging Pull Requests

Optionally, Octopilot can also automatically merge the Pull Requests it creates. Before merging a Pull Request, Octopilot will wait for the PR to be in a "mergable" state, and for all required status checks to pass.

- `--pr-merge` (bool): if enabled, the Pull Requests will be automatically merged. It will wait until the PRs are "mergeable" before merging them. Default to `false`.

All the following flags only apply if `--pr-merge` is enabled.

- `--pr-merge-auto` (bool):  merge PRs using Github's [auto-merge PR feature](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/incorporating-changes-from-a-pull-request/automatically-merging-a-pull-request).
  By default, this will not wait until the PR is merged.
  Note, this must also be enabled at the [repository level](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-auto-merge-for-pull-requests-in-your-repository) for it to work. This is a one-time task and `ocotopilot` will *not* do this for you. One way of doing this in bulk is using the `gh` CLI: `gh search repos --json url --jq '.[].url' ... | xargs -L1 gh repo edit --enable-auto-merge`.
- `--pr-merge-auto-wait` (bool):  when `--pr-merge-auto` is enabled, wait for the PR to be merged by Github.
- `--pr-merge-method` (string): the merge method to use. Either `merge`, `squash`, or `rebase`. Default to `merge`.
- `--pr-merge-commit-title` (string): optional title of the merge commit.
- `--pr-merge-commit-message` (string): optional body of the merge commit.
- `--pr-merge-sha` (string): optional SHA that pull request head must match to allow merge.
- `--pr-merge-poll-timeout` (string/duration): maximum duration to wait for a Pull Request to be mergeable/merged, using the [Golang syntax](https://golang.org/pkg/time/#ParseDuration). Default to `10m` (10 minutes).
- `--pr-merge-poll-interval` (string/duration): duration to wait for between each GitHub API call to check if a PR is mergeable/merged, using the [Golang syntax](https://golang.org/pkg/time/#ParseDuration). Default to `30s` (30 seconds).
- `--pr-merge-retry-count` (int): number of times to retry the merge operation in case of merge failure. Default to `3`.
