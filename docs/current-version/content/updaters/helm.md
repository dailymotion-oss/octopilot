---
title: "Helm"
anchor: "helm"
weight: 30
---

The Helm updater is made to easily update the dependencies of one or more [Helm](https://helm.sh/) charts. It supports both:
- Helm v3, with the dependencies declared in the `Chart.yaml` file
- Helm v2, with the dependencies declared in the `requirements.yaml` file

If you run the following command:

```bash
$ octopilot \
    --update "helm(dependency=chart-name)=1.2.3" \
    ...
```

Octopilot will discover all charts stored in the cloned repository, and for each, try to change the version of the `chart-name` dependency to `1.2.3`.

The syntax is: `helm(params)=value` - you can read more about the value in the ["value" section](#value).

It supports the following parameters:

- `dependency` (string): mandatory name of the dependency to update. Must exist in the dependencies list - it won't be added.
- `indent` (int): optional number of spaces used for indentation when writing the YAML file(s) after update. Default to `2`.

Note that Octopilot will keep the comments in the YAML files - because we're using the great [go-yaml v3 lib](https://github.com/go-yaml/yaml/tree/v3). [Just that it might rewrite a bit your indentation](https://mikefarah.gitbook.io/yq/usage/output-format#indent).
