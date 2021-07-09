---
title: "Usage"
anchor: "usage"
weight: 20
---

You can download the binary or see the `docker pull` commands from the [release page on GitHub](https://github.com/dailymotion-oss/octopilot/releases/latest).

There are no dependencies - not even on `git`.

There are no configuration files - we use CLI flags for everything. You can run `octopilot -h` to see all the flags. Some of them can be use multiple times, such as the `--repo` or `--update` flags.

Here is an example of using Octopilot to update multiple repositories, using multiple updaters:

```
$ octopilot \
    --repo "my-org/some-repo" \
    --repo "my-org/another-repo(merge=true)" \
    --repo "discover-from(env=PROMOTE_TO_REPOSITORIES)" \
    --repo "discover-from(query=org:my-org topic:my-topic)" \
    --update "yaml(file=config.yaml,path='version')=file(path=VERSION)" \
    --update `yq(file=helmfile.yaml,expression='(.releases[] | select(.chart == "repo/my-chart") | .version ) = strenv(VERSION)')` \
    --update `sops(file=secrets.yaml,key=path.to.base64encodedCertificateKey)=$(kubectl -n cert-manager get secrets tls-myapp -o template='{{index .data "tls.key"}}')` \
    --pr-title "Updating some files" \
    ...
```
