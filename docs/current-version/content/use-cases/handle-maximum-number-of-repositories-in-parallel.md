---
title: "Handle maximum number of repositories in parallel"
anchor: "handle-maximum-number-of-repositories-in-parallel"
weight: 50
---

By default Octopilot handles all repositories in parallel - creating as many goroutines as there are repositories. 
Add the `--max-concurrent-repos` flag so that Octopilot handles them in a batch way to avoid issues such as github rate limiting, or high load on your CI platform.
