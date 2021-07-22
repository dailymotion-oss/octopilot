---
title: "Promoting a new application release"
anchor: "use-case-app-promotion"
weight: 10
---

If you're doing gitops, you'll most likely have:
- at least one "application" git repository, for your(s) application(s).
- at least one "environment" git repository, for your(s) environment(s).

When you release a new version of your application, you'll need to "promote" it to (at least) one of your environments. In the gitops world, we're doing that by creating a new Pull Request against the target environment git repository, to update the version of the application in a configuration file.

And this is where Octopilot shines. You can use it in your application's Continuous Delivery pipeline to create the Pull Request against your environment's git repository.

Let's see an example, where we want to:
- automatically deploy new releases in the staging environment
- manually deploy new releases in the production environment

To do that, we'll:
- create a Pull Request on the `staging-env` git repository, and ask Octopilot to automatically merge it - and thus deploying it
- create a Pull Request on the `prod-env` git repository, but not merge it. We'll even create the Pull Request as a "draft", making it explicit that the release is not ready yet.

Oh, and we'd like to get nice commit messages and Pull Request title/description, so that people can understand what really is changing with this Pull Request. If all you have is "Update my-app to version 1.2.3", and a diff that shows a version change in a config file, it won't help you understand if this release is just fixing a typo in a README, fixing a critical bug, or introducing a new feature. We're already generating release notes with a changelog - if not, you should, using [git-chglog](https://github.com/git-chglog/git-chglog) for example - so let's re-use it and include it in our commit messages and Pull Request.

Here is an example of an Octopilot invocation you can use to achieve our goal - we'll go over it in details later:

```bash
$ export GITHUB_TOKEN=<your_github_token>
$ export ORG_NAME=my-org
$ export APP_NAME=my-app
$ export VERSION=1.2.3
$ octopilot \
    --repo "${ORG_NAME}/staging-env(merge=true)" \
    --repo "${ORG_NAME}/prod-env(draft=true)" \
    --update "yq(file=helmfile.yaml,expression='.releases[] | select(.name == strenv(APP_NAME)) | .version',output=.git/previous-version.txt)" \
    --update "yq(file=helmfile.yaml,expression='(.releases[] | select(.name == strenv(APP_NAME)) | .version) = strenv(VERSION)')" \
    --git-commit-title 'chore(deps): update {{ env "APP_NAME" }} from {{ readFile ".git/previous-version.txt" | trim }} to {{ env "VERSION" }}' \
    --git-commit-body '{{ githubRelease (printf "%s/%s/v%s" (env "ORG_NAME") (env "APP_NAME") (env "VERSION")) | expandGithubLinks (printf "%s/%s" (env "ORG_NAME") (env "APP_NAME")) | extractMarkdownURLs | md2txt }}' \
    --git-branch-prefix "octopilot-update-${APP_NAME}-" \
    --pr-labels "update-${APP_NAME}" \
    --pr-title "Update ${APP_NAME} to ${VERSION}" \
    --pr-title-update-operation "replace" \
    --pr-body '{{ githubRelease (printf "%s/%s/v%s" (env "ORG_NAME") (env "APP_NAME") (env "VERSION")) | expandGithubLinks (printf "%s/%s" (env "ORG_NAME") (env "APP_NAME")) }}' \
    --pr-body-update-operation "prepend" \
    --strategy "append"
```

As you can see, we're making the same set of changes to 2 different repositories, with different configurations:
- the PR on the `staging-env` repository will be automatically merged - and thus our release deployed
- the PR on the `prod-env` repository will be created as "draft", and won't be automatically merged - thus requiring a human intervention to merge it

We're running the [YQ updater](#yq) twice, on the same `helmfile.yaml` file, which would look like the following:

```yaml
releases:
  - name: my-app
    version: 1.0.0
  - name: another-app
    version: 1.5.0
```

This is a very simplified configuration file for [helmfile](https://github.com/roboll/helmfile), an application used to describe [Helm](http://helm.sh/) releases. You can use whatever you want, it's just to base our example on a real-life use-case.

So we're running the [YQ updater](#yq) twice:
- the first time, with the `.releases[] | select(.name == strenv(APP_NAME)) | .version` expression, to extract the current version for our application named `my-app`. And we're sending the output to the `.git/previous-version.txt` file - which will contain just `1.0.0`. We're doing that to store the "previous" version before changing it without new version. You'll notice that the output file is located in the `.git` directory, this is to avoid committing it by default.
- the second time, we're replacing the version with the new one - stored in the `VERSION` environment variable - with the following expression: `(.releases[] | select(.name == strenv(APP_NAME)) | .version) = strenv(VERSION)`.

Now, we have:
- a locally modified `helmfile.yaml`
- a new `.git/previous-version.txt` file

So it's time to commit!
- in the commit title, we want to include both the previous version and the new version. We'll use the [templating feature](#templating) to include both the content of a file and an environment variable: `chore(deps): update {{ env "APP_NAME" }} from {{ readFile ".git/previous-version.txt" | trim }} to {{ env "VERSION" }}`
- in the commit body, we want to include the release notes for the new version. We'll use the [templating feature](#templating) to retrieve the release notes, convert all the GitHub short links to absolute URLs, and transform the markdown into raw text: `{{ githubRelease (printf "%s/%s/v%s" (env "ORG_NAME") (env "APP_NAME") (env "VERSION")) | expandGithubLinks (printf "%s/%s" (env "ORG_NAME") (env "APP_NAME")) | extractMarkdownURLs | md2txt }}`

We'll also use a custom prefix for the branch name, to make it easier to find which branch belongs to which app, if you have multiple promotion PRs for different applications opened at the same time.

Next, the Pull Request. We'll use the `append` strategy, which means that if we need to promote a release in prod, and the previous one hasn't been merged/deployed yet, we'll just append a new commit on the existing branch/PR. So that your production PR for your application will stay the same, accumulating releases until it is merged. We're using a specific label for our Pull Request: `update-${APP_NAME}` - to make sure we'll find any existing PR for our application, and that each application will get its own PR.

Same as for the commit, we'll use the [templating feature](#templating) to write a nice title and description for our Pull Request - just that this time we won't need to convert from markdown to raw text. And we'll use specific "update operations":
- we'll always replace the PR title, because we want a short title, with only the app name and the latest release's version - using the `--pr-title-update-operation "replace"` flag
- we'll "prepend" the new PR body, before the existing one, using the `--pr-body-update-operation "prepend"` flag. So that the release notes for the latest release will be first - just as when you read a changelog. And of course we don't want to remove the previous release notes.

## Result

### Staging environment

This is a screenshot of a Pull Request on the staging environment git repository, which has been automatically merged. You can see the release notes, and you'll notice the "signature" at the bottom: we're making it easy for people to know:
- that this PR has been generated by an application - and not created manually by a human
- which version of Octopilot has been used
- from where it has been executed. This will most likely be the application's GitHub repository, because it is where you (should) define your application's Continuous Delivery pipeline, which contains a step to execute Octopilot.

![](screenshot-app-promotion-pr-single-commit.png)

### Production environment

This is a screenshot of a Pull Request on the production environment git repository:
- the PR has been created to promote `v3.14.1` of the application
- later, `v3.15.0` has been released - and promoted. Thus adding a new commit to the PR
- and then, `some-user` merged the Pull Request, to deploy in prod

You'll notice that we have 2 release notes in the PR: 1 for each release. So you can see the full changelog for every release that will be deployed when you'll merge this PR.

![](screenshot-app-promotion-pr-multi-commits.png)

In this screenshot you can see the 2 commits:

![](screenshot-app-promotion-pr-multi-commits-commits.png)

### Feedback

The benefit of adding the application's release notes in the promotion pull request body, is that not only will you know exactly what you'll deploy, but you'll also get links between the application pull request and the promotion pull requests. So that if you go back to the application's PR, you'll see something like:

![](screenshot-app-promotion-pr-feedback.png)

You can see at the bottom the links to our 2 promotion pull requests, with their statuses - both have been merged already in this case.

## Going further

As you can see, it's easy to adapt this example for your own use-case. For example, you might want:
- to create PRs on a different set of [repositories](#repos): QA, staging and production
- to update different kinds of files, using different [updaters](#updaters)
- to customize the [commit](#commit) or the [Pull Request](#pull-request)
