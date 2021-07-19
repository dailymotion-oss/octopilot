---
title: "Regex"
anchor: "regex"
weight: 50
---

The **regex** updater can be used to update any kind of text file using a regular expression - with the [Golang syntax](https://golang.org/pkg/regexp/syntax/), such as:

```
$ octopilot \
    --update "regex(file=some-file.txt,pattern='version: \"(.*)\"')=${VERSION}" \
    ...
```

Given the following `some-file.txt` file:

```
version: "1.0.0"
...
```

Octopilot will replace the first line with `version: "1.2.3"` if the `$VERSION` env var is set to `1.2.3` for example.

The syntax is: `regex(params)=value` - you can read more about the value in the ["value" section](#value).

It supports the following parameters:

- `file` (string): mandatory path to the file to update. Can be a file pattern - such as `files/**/*.txt`. If it's a relative path, it will be relative to the root of the cloned git repository.
- `pattern` (string): mandatory regex pattern to find and replace something in the file(s). The pattern must be in the [Golang syntax](https://golang.org/pkg/regexp/syntax/). If this pattern includes a capturing group, then it will be replaced by the provided value.

A few things you can do with the regex updater:

- replace the whole content of a file
    ```
    $ octopilot \
        --update "regex(file=my-file,pattern='(?ms)(.*)')=new content" 
    ```
