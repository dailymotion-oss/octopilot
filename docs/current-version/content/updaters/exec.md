---
title: "Exec"
anchor: "exec"
weight: 60
---

The **exec** updater can execute any command you want, so you can change files in the cloned git repository with any tool you have available.

For example to update all your Go dependencies to the latest version:

```bash
$ octopilot \
    --update "exec(cmd=go,args=get -d -t -u)" \
    --update "exec(cmd=go,args=mod tidy)" \
    --update "exec(cmd=go,args=mod vendor)" \
    --git-stage-pattern "vendor" \
    ...
```

This will execute 3 commands, that will update the `go.mod` & `go.sum` files, and the `vendor` dir. Octopilot will then add/commit all the changes, including the new files in the `vendor` directory.

The syntax is: `exec(params)`.

It supports the following parameters:

- `cmd` (string): mandatory command to execute.
- `path` (string): optional path to execute the command in.
- `args` (string): optional arguments for the command. The arguments are space-separated. If you have a space in an argument, you can quote it, such as: `-c 'some arg' -x another`.
- `stdout` (string): optional path to a file where the std output of the command will be written. If it's a relative path, it will be relative to the root of the cloned git repository.
- `stderr` (string): optional path to a file where the std error output of the command will be written. If it's a relative path, it will be relative to the root of the cloned git repository.
- `timeout` (string/duration): optional maximum duration to wait for the command to finish, using the [Golang syntax](https://golang.org/pkg/time/#ParseDuration).

A few things you can do with the regex updater:

- use a bash command to enable shell expansion (disabled by default when invoking commands through Octopilot)
    ```bash
    $ octopilot \
        --update "exec(cmd=sh,args=-c 'cat files/*.txt',stdout=output.txt)"
    ```

- use the an external tool (here kustomize) and change the current working directory before executing the command
    ```bash
    $ octopilot \
        --update "exec(cmd=kustomize, path=k8s/overlays/dev, args=edit set image containername=registry.tld/repo/image:newtag)"
    ```

See the ["updating go dependencies" use-case](#use-case-go-deps) for a real-life example of what you can do with this updater.
