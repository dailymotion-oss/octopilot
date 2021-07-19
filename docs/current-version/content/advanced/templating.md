---
title: "Templating"
anchor: "templating"
weight: 10
---

For some of the CLI flags - such as commit title/body or Pull Requests title/body - you can use our "templating" feature. This will allow you to write nice commits and Pull Requests.

Octopilot uses the [Go template](https://pkg.go.dev/text/template) syntax, and supports the following functions:
- all the [Go template functions](https://golang.org/pkg/text/template/#hdr-Functions)
- all the [sprig functions](http://masterminds.github.io/sprig/)
- Octopilot's own custom functions

## Octopilot's own custom functions

Octopilot comes with the following custom functions:
- `readFile` to read a file from the cloned git repository, and return its content
- `githubRelease` to retrieve the release notes for a specific GitHub release
- `expandGithubLinks` to transform GitHub short links - such as #123 - to absolute URLs
- `extractMarkdownURLs` to transform markdown links to plain URLs
- `md2txt` to strip all the markdown syntax from a string

### readFile

The `readFile` function will read a file from the cloned git repository, and return its content as a string. If the path of the file is relative, it will be relative to the root of the git repository.

Definition: `readFile(filePath string) string`.

Example: `{{readFile "README.md"}}` to print the content of the `README.md` file.

### githubRelease

The `githubRelease` function will retrieve the release notes for a specific GitHub release. The release is identified by the owner, repository and release version.

Definition: `githubRelease(releaseID string) string`.

Example: `{{ githubRelease "owner/repo/v1.2.3" }}` to print the release notes for the release `v1.2.3` of the `owner/repo` github repository.

### expandGithubLinks

The `expandGithubLinks` function will transform GitHub short links - such as #123 - to absolute URLs. It is mostly useful when combined with the `githubRelease` function, to ensure that the links in the release notes are always absolute.

Definition: `expandGithubLinks(fullRepositoryName string, markdownInput string) string`. The `fullRepositoryName` parameter is the full name of the repository, such as `owner/repo`. It is used to build the absolute URLs.

Example: `{{ githubRelease (print "owner/repo/" (env "VERSION")) | expandGithubLinks "owner/repo" }}`.

### extractMarkdownURLs

The `extractMarkdownURLs` function will transform markdown links to plain URLs. It is mostly useful when combined with the `githubRelease` function, to ensure that the links in the release notes are always plain URLs - when you want to print plain text, and not markdown, such as in a commit message.

Definition: `extractMarkdownURLs(markdownInput string) string`.

Example: `{{ githubRelease (print "owner/repo/" (env "VERSION")) | expandGithubLinks "owner/repo" | extractMarkdownURLs }}`.

### md2txt

The `md2txt` function will strip all the markdown syntax from a string. It is mostly useful when combined with the `githubRelease` function, to ensure that the release notes are always plain text, and not markdown. Use it for your commit messages.

Definition: `md2txt(markdownInput string) string`.

Example: `{{ githubRelease (print "owner/repo/" (env "VERSION")) | expandGithubLinks "owner/repo" | extractMarkdownURLs | md2txt }}`.
