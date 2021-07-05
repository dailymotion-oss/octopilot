# Octo Pilot

[![Go Report Card](https://goreportcard.com/badge/github.com/dailymotion-oss/octopilot)](https://goreportcard.com/report/github.com/dailymotion-oss/octopilot)
[![Release Status](https://github.com/dailymotion-oss/octopilot/workflows/release/badge.svg)](https://github.com/dailymotion-oss/octopilot/actions?query=workflow%3Arelease)
[![Latest Release](https://img.shields.io/github/v/release/dailymotion-oss/octopilot)](https://github.com/dailymotion-oss/octopilot/releases)

**OctoPilot** is a tool designed to help you automate your Gitops workflow, by automatically creating and merging GitHub Pull Requests to update specific content in Git repositories.

It supports updating:
- [sops](https://github.com/mozilla/sops) files
- [Helm](https://helm.sh/) dependencies versions
- YAML files
- Generic updates based on regular expressions
- Running any tool you like

It can update a single repository or multiple at the same time, with different strategies:
- always create a new pull request each time it will run
- re-use existing pull requests based on labels, by adding commits to the existing branch
- re-use existing pull requests based on labels, but re-creating the branch from the master branch

It can update repositories based on specific branch instead of the HEAD one's if mentionned
- The option `--repo "xxx/yyy(branch=xyz)"` will update the repo `xxx/yyy` based on the `xyz` branch if it exists
- The option `--repo "xxx/yyy"` will update the repo xxx/yyy based on the `HEAD` branch as usual

It can exec command and expand files on the arguments, only if using `sh -c` with `sh` as cmd and args starting by `-c` and using after backquotes
- It is possible because of the lib https://github.com/cosiner/argv
- It doesn't manage pipe on the args

It is somewhat based on [updatebot](https://github.com/jenkins-x/updatebot), but written in [Go](https://golang.org/), and with more features:
- supports running multiple "updaters" in the same execution - for example create a PR with both a sops change and a regex change
- run any tool to update a repo
- update YAML files without loosing comments
- more configurable github strategies
- no external dependencies - for example on the `git` command. Everything is bundled in the binary.
- and maintained ;-)

## Use cases

### Update certificates

If you store your certificates in git, with the certificate itself in clear text in a YAML file (base64-encoded), and the secret key in a sops-encrypted file, you can update both with the following command:

```
$ octopilot \
    --update "sops(file=certificates/secrets.yaml,key=certificates.myapp.b64encKey)=$(kubectl -n cert-manager get secrets tls-myapp -o jsonpath=\"{.data.tls\\\.key}\")" \
    --update "regex(file=certificates/values.yaml,pattern='myapp:\s+b64encCertificate: (.*)')=$(kubectl -n cert-manager get secrets tls-myapp -o jsonpath=\"{.data.tls\\\.crt}\"))" \
    --repo "myorg/my-gitops-env"
```

### Update Helm dependencies

If you release a new version of your app, you can update all the apps that depends on you:

```
$ octopilot \
    --update "helm(dependency=my-app)=file(path=VERSION)" \
    --repo "myorg/some-app" \
    --repo "myorg/another-app"
```

### Update a specific value in a YAML file

For example to update the version of an app in a YAML file with a format that is not natively supported by OctoPilot, you can use the YAML updater:

```
$ octopilot \
    --update "yaml(file=helmfile.yaml,path='releases.(chart==example/my-chart).version')=file(path=VERSION)"
```

An alternative is to use the regex updater:

```
$ octopilot \
    --update "regex(file=helmfile.yaml,pattern='chart: example/my-chart\s+version: \"(.*)\"')=file(path=VERSION)"
```

### YQ

Uses an [yq](https://mikefarah.gitbook.io/yq) expression, and write the result either in-place (the default) or to a specific file. All the [yq operators](https://mikefarah.gitbook.io/yq/operators) are supported, so you can do very powerful things, such as manipulating YAML comments, use variables, output to json (note that it can also read json input), ...

Equivalent of the previous example with the `yaml` updater:

```
$ export VERSION=$(cat VERSION) && octopilot \
    --update `yq(file=helmfile.yaml,expression='(.releases[] | select(.chart == "example/my-chart") | .version ) = strenv(VERSION)')`
```

Or read the previous version, store it in a temporary file, and use it to write the commit message:

```
$ octopilot \
    --update `yq(file=helmfile.yaml,expression='.releases[] | select(.name == strenv(RELEASE)) | .version',output=.git/previous-version.txt)` \
    --update `yq(file=helmfile.yaml,expression='(.releases[] | select(.name == strenv(RELEASE)) | .version) = strenv(VERSION)')` \
    --git-commit-title 'chore(deps): update {{ env "RELEASE" }} from {{ readFile ".git/previous-version.txt" | trim }} to {{ env "VERSION" }}'
```

### Update a whole file

To replace the whole content of a file:

```
$ octopilot \
    --update "regex(file=README.md,pattern='(?ms)(.*)')=new content" 
```

### Generic update by running a command

You can also run any command(s), and OctoPilot will just add/commit everything, and create/update the pull request. For example to automatically update all your Go dependencies to the latest patch version:

```
$ octopilot \
    --update "exec(cmd='go get -u=patch')" \
    --update "exec(cmd='go mod tidy')" \
    --update "exec(cmd='go mod vendor')"
```
