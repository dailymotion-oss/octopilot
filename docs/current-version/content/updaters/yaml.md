---
title: "YAML"
anchor: "yaml"
weight: 10
---

The YAML updater is great when you want to quickly set a value for a specific path in one or more files. Such as if you want to update a version used in a YAML file:

```
$ octopilot \
    --update "yaml(file=config.yaml,path='app.version')=file(path=VERSION)" \
    ...
```

Given the following `config.yaml` file:

```
app:
  name: foo
  version: 1.0.0
```

Octopilot will set the value of the `app.version` key to the content of the `VERSION` file.

The syntax is: `yaml(params)=value` - you can read more about the value in the ["value" section](#value).

It support the following parameters:

- `file` (string): mandatory path to the YAML file to update. Can be a file pattern - such as `config/*.yaml`. If it's a relative path, it will be relative to the root of the cloned git repository.
- `path` (string): mandatory path to the key to update in the YAML file(s). We support [yq v3 path expressions](https://mikefarah.gitbook.io/yq/v/v3.x/usage/path-expressions) or [yq v4 syntax](https://mikefarah.gitbook.io/yq/operators/traverse-read).
- `indent` (int): optional number of spaces used for indentation when writing the YAML file(s) after update. Default to `2`.
- `trim` (boolean): if `true`, the content will be "trimmed" before being written to disk - to avoid extra line break at the end of the file for example.
- `create` (boolean): if `true`, then the `path` will always be set to the given value, even if no such key existed before. The default behaviour (`false`) is to NOT create any new path/key.
- `style` (string): an optional style to apply to the new value: `double` (add double quotes), `single` (add single quotes), `literal`, `folded` or `flow` - see [yq style reference](https://mikefarah.gitbook.io/yq/operators/style).

Note that Octopilot will keep the comments in the YAML files - because we're using the great [go-yaml v3 lib](https://github.com/go-yaml/yaml/tree/v3). [Just that it might rewrite a bit your indentation](https://mikefarah.gitbook.io/yq/usage/output-format#indent).
