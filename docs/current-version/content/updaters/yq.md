---
title: "YQ"
anchor: "yq"
weight: 20
---

The YQ updater is based on the excellent [yq](https://github.com/mikefarah/yq) application - and Go lib. It is much more powerful than the basic [YAML updater](#yaml) - because it supports all the [yq operators](https://mikefarah.gitbook.io/yq/operators), and because you are not limited to setting a value for a specific key. You can do very powerful things, such as manipulating YAML comments, use variables, output to json (note that it can also read JSON input), ...

The syntax is: `yq(params)`, such as:

```
$ octopilot \
    --update "yq(file=config.yaml,expression='.path.to.version = strenv(VERSION)')" \
    ...
```

It supports the following parameters:

- `file` (string): mandatory path to the YAML (or JSON) file to update. Can be a file pattern - such as `config/*.yaml`. If it's a relative path, it will be relative to the root of the cloned git repository.
- `expression` (string): mandatory [yq v4 expression](https://mikefarah.gitbook.io/yq/commands/evaluate) that will be evaluated against each file.
- `output` (string): optional output of the result. By default the result is written to the source file - in-place editing. But you can send the result to `stdout`, `stderr` or a specific file.
- `json` (boolean): if `true`, then the output will be written in JSON format instead of YAML format.
- `indent` (int): optional number of spaces used for indentation when writing the YAML file(s) after update. See [yq doc on indent](https://mikefarah.gitbook.io/yq/usage/output-format#indent). Default to `2`.
- `trim` (boolean): if `true`, the content will be "trimmed" before being written to disk - to avoid extra line break at the end of the file for example.
- `unwrapscalar` (boolean): if `true` (the default), only the value will be printed - not the comments. See [yq doc on unwrap scalars](https://mikefarah.gitbook.io/yq/usage/output-format#unwrap-scalars).

Note that Octopilot will keep the comments in the YAML files - because we're using the great [go-yaml v3 lib](https://github.com/go-yaml/yaml/tree/v3). [Just that it might rewrite a bit your indentation](https://mikefarah.gitbook.io/yq/usage/output-format#indent).

See the ["promoting a new release" use-case](#use-case-app-promotion) for a real-life example of what you can do with this updater.
