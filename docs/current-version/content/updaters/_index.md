---
title: "Updaters"
anchor: "updaters"
weight: 40
---

The core feature of Octopilot is to update git repositories, and to do it you can use one or more of the available "updaters":
- the [YAML updater](#yaml), to quickly update YAML files
- the [YQ updater](#yq), based on [mikefarah's yq](https://github.com/mikefarah/yq), to manipulate YAML or JSON files as you want
- the [Helm updater](#helm), to easily update the dependencies of an [Helm](https://helm.sh/) chart
- The [sops updater](#sops), to manipulate files encrypted with [mozilla's sops](https://github.com/mozilla/sops)
- The [regex updater](#regex), to update any kind of text file using a regular expression
- The [exec updater](#exec), to execute any command you want

Each updater can be used once or more, such as:

```
$ octopilot \
    --update "yaml(file=config.yaml,path='version')=file(path=VERSION)" \
    --update "regex(file=some-file.txt,pattern='version: \"(.*)\"')=${VERSION}" \
    --update "yaml(file=another-config.yaml,path='path.to.version')=$(cat VERSION)" \
    ...
```
