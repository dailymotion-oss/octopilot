---
title: "Updating Go dependencies"
anchor: "use-case-go-deps"
weight: 30
---

One of the downside of using a micro-services architecture is that it requires a lot of maintenance on the different git repositories. Even more when you have more repositories than teams or developers. For example, you'll need to ensure that:
- the dependencies are up to date
- the "config files" for your Continuous Delivery pipeline(s) are up to date
- and so on...

Automating the creation of Pull Requests to keep your git repositories "in sync" is a good way to reduce the maintenance effort. [Dependabot](https://dependabot.com/) is one way to do it, Octopilot is another. The benefits of using Octopilot are:
- you control where it is executed - in your own infrastructure
- you control exactly what it does - including running your own custom scripts/binaries

One use-case we have is to update the [Go](https://golang.org/) dependencies, by running the following commands:
- `go get -d -t -u`
- `go mod tidy`
- `go mod vendor`
- `go mod verify`

For example you can do it on octopilot's own repository, by running:

```bash
$ export GITHUB_TOKEN=<your_github_token>
$ octopilot \
    --repo "dailymotion-oss/octopilot" \
    --update "exec(cmd=go,args=get -d -t -u)" \
    --update "exec(cmd=go,args=mod tidy)" \
    --update "exec(cmd=go,args=mod vendor)" \
    --update "exec(cmd=go,args=mod verify)" \
    --git-stage-pattern "vendor" \
    --git-commit-title "chore(deps): update Go dependencies"
```

You can then run it on a regular basis, such as every week, to update all your dependencies at once. And you can use the [dynamic repositories](#dynamic) feature to do it on all your repositories with a specific GitHub topic.

## Result

This is a screenshot of a Pull Request on a git repository, which updates all the Go modules.

![](screenshot-go-deps-pr.png)

Using the [exec updater](#exec) we can run any command we want, capture its stdout and/or stderr, and use it in the commit message and/or Pull Request description.
