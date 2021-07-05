---
title: "Introduction"
anchor: "intro"
weight: 10
---

**Octopilot** is a CLI tool designed to help you automate your Gitops workflow, by automatically creating and merging GitHub Pull Requests to update specific content in Git repositories.

It works by:
- cloning one or more [repositories](#repos), defined either:
  - [statically](#static)
  - [dynamically](#dynamic), using environment variables or GitHub search queries
- running one or more [updaters](#updaters) on each cloned repository, using either:
  - the [YAML updater](#yaml), to quickly update YAML files
  - the [YQ updater](#yq), based on [mikefarah's yq](https://github.com/mikefarah/yq), to manipulate YAML or JSON files as you want
  - the [Helm updater](#helm), to easily update the dependencies of an [Helm](https://helm.sh/) chart
  - The [sops updater](#sops), to manipulate files encrypted with [mozilla's sops](https://github.com/mozilla/sops)
  - The [regex updater](#regex), to update any kind of text file using a regular expression
  - The [exec updater](#exec), to execute any command you want
- [commit/push](#commit) the changes
- create [Pull Requests](#pull-request) and optionally merge them
