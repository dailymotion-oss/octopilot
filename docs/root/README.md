# Octopilot root documentation

This is the "root" documentation project for octopilot, it contains:
- a high-level overview of what is octopilot
- links to the detailed per-version documentation

It's a static website build with [Hugo](https://gohugo.io/).
- the content is written in markdown format, in the [content](./content/) directory
- to render the website:
  - install [Hugo](https://gohugo.io/) - see the [.github/workflows/release.yml](../../.github/workflows/release.yml) file for the version of Hugo to use
  - run `hugo server` in this directory, and open <http://localhost:1313/>

