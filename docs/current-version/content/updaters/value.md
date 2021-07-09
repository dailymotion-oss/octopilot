---
title: "Value"
anchor: "value"
weight: 100
---

Some updaters accept a **value** in their syntax, such as `updater(params)=value`.

This value can be either:
- a raw value
- the content of a file

## Raw value

This is the easiest way to set a value: just use a raw value, such as:

```
$ octopilot \
    --update "yaml(file=config.yaml,path='version')=v1.2.3" \
    ...
```

Note that you can also use an environment variable:

```
$ export VERSION=v1.2.3
$ octopilot \
    --update "yaml(file=config.yaml,path='version')=${VERSION}" \
    ...
```

or any command you want:

```
$ echo v1.2.3 > /tmp/VERSION
$ octopilot \
    --update "yaml(file=config.yaml,path='version')=$(cat /tmp/VERSION)" \
    ...
```

## File content

If you want to use the content of a file, you can use the **file** valuer:

```
$ octopilot \
    --update "yaml(file=config.yaml,path='version')=file(path=VERSION)" \
    ...
```

It will read the `VERSION` file located at the root of the cloned git repository, and use its content as the value.

The syntax is: `file(params)`.

It support the following parameters:

- `path` (string): mandatory path to the file to read. If it's a relative path, it will be relative to the root of the cloned git repository.
