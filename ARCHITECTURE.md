# Octopilot Architecture

This document describes the high-level architecture of Octopilot. If you want to familiarize yourself with the code base, you are just in the right place!

## Big Pictures

There are 2 main packages:
- `update`: the updaters that change content in the git repositories
- `repository`: everything related to working with git repositories hosted on GitHub: cloning, commits, creating branches, pushing, creating/updating/merging pull requests, and so on.

## Main entry point

The `main.go` file contains the main entry point, which:
- defines all the CLI flags
- parses the updaters and the repositories
- updates all the repositories in parallel

## Repositories

Everything related to working to git repositories hosted on GitHub is in the `repository` package:
- `repository.go`: the definition of a repository, and how to parse one from a command line argument
- `strategy_*.go`: the implementation of the different strategies for updating a repository
- `options.go`: definition of the options exposed by the CLI flags
- `git.go`: set of functions to work with git repositories: clone, commit, push, ...
- `pull_request.go`: find, create, update and merge a pull request
- `template.go`: definition and execution of the (golang) templates used to generate the commit and pull request title/body

## Updaters

The updaters, which change content in the git repositories, are defined in the `update` package: 1 package for each updater. Each updater is mainly defined by an `Update` function, which operates on a repository.

Note that an updater's `Update` function may be called by multiple goroutines at the same time - from multiple repositories - so it must be "thread-safe".

## Internal packages

There are a few small internal packages, in the `internal` directory - using the Go convention that makes these packages private by default:
- `git`: provides helper functions to work with Git repository - and mainly its configuration.
- `parameters`: provides functions to work with "parameters": key-value maps.

## Credits

- [matklad's blog post on ARCHITECTURE.md](https://matklad.github.io/2021/02/06/ARCHITECTURE.md.html)
