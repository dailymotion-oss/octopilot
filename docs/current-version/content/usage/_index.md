---
title: "Usage"
anchor: "usage"
weight: 20
---

You can download the binary or see the `docker pull` commands from the [release page on GitHub](https://github.com/dailymotion-oss/octopilot/releases/latest).

There are no dependencies - not even on `git`.

There are no configuration files - we use CLI flags for everything. You can run `octopilot -h` to see all the flags. Some of them can be used multiple times, such as the `--repo` or `--update` flags.

You will need a GitHub token or app to authenticate with GitHub. See the [GitHub Auth](#github-auth) section for more details.

Here is an example of using Octopilot to update multiple repositories, using multiple updaters:

```bash
$ octopilot \
    --github-token "my-github-token" \
    --repo "my-org/some-repo" \
    --repo "my-org/another-repo(merge=true)" \
    --repo "discover-from(env=PROMOTE_TO_REPOSITORIES)" \
    --repo "discover-from(query=org:my-org topic:my-topic)" \
    --repo "discover-from(searchtype=code,query=org:my-org filename:my-file path:dir-path in-file-text)" \
    --repo "discover-from(searchtype=code,query=org:my-org filename:my-file path:dir-path fork:true)" \
    --update "yaml(file=config.yaml,path='version')=file(path=VERSION)" \
    --update "yq(file=helmfile.yaml,expression='(.releases[] | select(.chart == \"repo/my-chart\") | .version ) = strenv(VERSION)')" \
    --update "sops(file=secrets.yaml,key=path.to.base64encodedCertificateKey)=$(kubectl -n cert-manager get secrets tls-myapp -o template='{{index .data \"tls.key\"}}')" \
    --pr-title "Updating some files" \
    ...
```

## Continuous Delivery Pipelines

Octopilot has been designed to be used in a Continuous Delivery pipeline: no dependencies, no configuration file, only 1 command to update multiple repositories...

You can use it with [Jenkins](https://www.jenkins.io/), [Jenkins X](https://jenkins-x.io/), [Tekton](https://tekton.dev/), [GitHub Actions](https://github.com/features/actions), ...

At Dailymotion, we're using it through [Jenkins X](https://jenkins-x.io/)/[Tekton](https://tekton.dev/) pipelines, and also a few [Jenkins](https://www.jenkins.io/) pipelines.
