# Octopilot Documentation

The documentation is written in markdown format, and we use [Hugo](https://gohugo.io/) to render a nice website from the markdown files. You can see the result at <https://dailymotion-oss.github.io/octopilot/>.

The documentation is in 2 parts:
- the [root](./root/) directory, which contains only a high-level overview
- the [current-version](./current-version/) directory, which contains the full documentation

Each of these 2 directories is a full Hugo project/website. We're doing this because we want to keep version-specific documentation, so for each new release/tag, we're generating a new documentation website, alongside the older ones. This allow users to read the documentation for older releases - if they don't care about upgrading for example.
The root website contains links to the most recent releases / documentation.
