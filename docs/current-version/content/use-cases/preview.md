---
title: "Previsualizing changes"
anchor: "use-case-preview"
weight: 40
---

If you're working on your workflow with Octopilot, at some point you might want to "preview" the changes Octopilot will do, without actually creating the Pull Request(s).

There is an easy way to do that. You'll need to:
- use the `--dry-run` CLI flag, to ensure that no operation will be performed on the remote git repository: not pushing branches/commits, not creating/updating Pull Requests, ...
- use the `--keep-files` CLI flag, to ensure that the changes to the local cloned git repositories won't be lost at the end of the process - because by default Octopilot removes all temporary files
- use a verbose log level - such as `debug` or `trace` - with the `--log-level` CLI flag, to retrieve the path of the temporary files created by Octopilot

For example, if you run:

```
$ export GITHUB_TOKEN=<your_github_token>
$ octopilot \
    --repo "dailymotion-oss/octopilot" \
    --update "yq(file=.goreleaser.yml,expression='(.dockers[] | select(.dockerfile == \"Dockerfile.goreleaser\") | .dockerfile) = \"a.new.Dockerfile\"')" \
    --update "yaml(file=.golangci.yml,path=run.timeout)=42m" \
    --update "regex(file=README.md,pattern='(?ms)(.*)')=replacing the content of the README.md file with this new content" \
    --dry-run --keep-files --log-level=debug
```

Then you should see something like:

```
DEBU[0000] Updaters ready                                updaters="[YQ[file=.goreleaser.yml,expression=(.dockers[] | select(.dockerfile == \"Dockerfile.goreleaser\") | .dockerfile) = \"a.new.Dockerfile\",output=,indent=2] YAML[path=run.timeout,file=.golangci.yml,style=,create=false,trim=false,indent=2] Regex[pattern=(?ms)(.*),file=README.md]]"
DEBU[0000] Repositories ready                            repositories="[{dailymotion-oss octopilot map[]}]"
DEBU[0000] Using 'reset' strategy                        repository=dailymotion-oss/octopilot
DEBU[0003] Git repository cloned                         git-reference=HEAD git-url="https://github.com/dailymotion-oss/octopilot.git" local-path=/var/folders/v0/fx5l3skn17785d8f4l883m6w0000gp/T/octopilot092369223/dailymotion-oss/octopilot
DEBU[0003] No existing Pull Request found                labels="[octopilot-update]" repository=dailymotion-oss/octopilot
DEBU[0004] Switched Git branch                           branch=octopilot-c3qif432dnc2961tuuh0 repository-name=octopilot
DEBU[0004] Updater finished                              changes=true repository=dailymotion-oss/octopilot updater="YQ[file=.goreleaser.yml,expression=(.dockers[] | select(.dockerfile == \"Dockerfile.goreleaser\") | .dockerfile) = \"a.new.Dockerfile\",output=,indent=2]"
DEBU[0004] Updater finished                              changes=true repository=dailymotion-oss/octopilot updater="YAML[path=run.timeout,file=.golangci.yml,style=,create=false,trim=false,indent=2]"
DEBU[0004] Updater finished                              changes=true repository=dailymotion-oss/octopilot updater="Regex[pattern=(?ms)(.*),file=README.md]"
DEBU[0004] All updaters finished                         repository=dailymotion-oss/octopilot
DEBU[0005] Git status                                    repository-name=octopilot status=" M README.md\n M .golangci.yml\n M .goreleaser.yml\n"
DEBU[0006] Git commit                                    commit=d263de874faf26a6ccc8bb2325bc4eb47e0a7029 repository-name=octopilot
WARN[0006] Running in dry-run mode, not pushing changes  repository=dailymotion-oss/octopilot
WARN[0006] Repository update has no changes              repository=dailymotion-oss/octopilot
INFO[0006] Updates finished                              repositories-count=1
```

The 2 interesting lines are:
- `Git repository cloned  local-path=/var/folders/v0/fx5l3skn17785d8f4l883m6w0000gp/T/octopilot092369223/dailymotion-oss/octopilot`
- `Git commit  commit=d263de874faf26a6ccc8bb2325bc4eb47e0a7029`

If you go to the directory identified by the `local-path` value, you can inspect the git repository - for example:

```
$ cd /var/folders/v0/fx5l3skn17785d8f4l883m6w0000gp/T/octopilot092369223/dailymotion-oss/octopilot
$ git show d263de874faf26a6ccc8bb2325bc4eb47e0a7029
```

and you should see:

```
commit d263de874faf26a6ccc8bb2325bc4eb47e0a7029 (HEAD -> octopilot-c3qif432dnc2961tuuh0)
Author: author <author@example.com>
Date:   Mon Jul 19 09:19:46 2021 +0200

    Octopilot update

    Updates:

    ### YQ[file=.goreleaser.yml,expression=(.dockers[] | select(.dockerfile == "Dockerfile.goreleaser") | .dockerfile) = "a.new.Dockerfile",output=,indent=2]
    Update .goreleaser.yml
    Updating file(s) `.goreleaser.yml` using yq expression `(.dockers[] | select(.dockerfile == "Dockerfile.goreleaser") | .dockerfile) = "a.new.Dockerfile"`

    ### YAML[path=run.timeout,file=.golangci.yml,style=,create=false,trim=false,indent=2]
    Update .golangci.yml
    Updating path `run.timeout` in file(s) `.golangci.yml`

    ### Regex[pattern=(?ms)(.*),file=README.md]
    Update README.md
    Updating file(s) `README.md` using pattern `(?ms)(.*)`

    --
    Generated by [Octopilot](https://github.com/dailymotion-oss/octopilot) [v0.2.16](https://github.com/dailymotion-oss/octopilot/releases/tag/v0.2.16) from https://github.com/dailymotion-oss/octopilot

diff --git a/.golangci.yml b/.golangci.yml
index 9a84d2c..b8a04d9 100644
--- a/.golangci.yml
+++ b/.golangci.yml
@@ -1,4 +1,4 @@
 # See https://golangci-lint.run/usage/configuration/#config-file

 run:
-  timeout: 3m
+  timeout: 42m
diff --git a/.goreleaser.yml b/.goreleaser.yml
index 3d8734f..1b96958 100644
--- a/.goreleaser.yml
+++ b/.goreleaser.yml
@@ -12,12 +12,10 @@ builds:
       - -X main.buildDate={{.Date}}
     env:
       - CGO_ENABLED=0
-
 archives:
   - format: binary
-
 dockers:
-  - dockerfile: Dockerfile.goreleaser
+  - dockerfile: a.new.Dockerfile
     image_templates:
       - "ghcr.io/dailymotion-oss/{{.ProjectName}}:{{ .Version }}"
       - "ghcr.io/dailymotion-oss/{{.ProjectName}}:{{ .Tag }}"
@@ -31,6 +29,5 @@ dockers:
       - "--label=org.opencontainers.image.revision={{.FullCommit}}"
       - "--label=org.opencontainers.image.version={{.Version}}"
       - "--label=org.opencontainers.image.source={{.GitURL}}"
-
 changelog:
   sort: asc
diff --git a/README.md b/README.md
index a1dbefa..b247a46 100644
--- a/README.md
+++ b/README.md
@@ -1,115 +1 @@
-# Octo Pilot
-...
+replacing the content of the README.md file with this new content
```

This is a good way to ensure that you have the right syntax for your [updater(s)](#updaters), and/or that your git commit is what you want.
