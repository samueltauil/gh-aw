---
description: Example workflow demonstrating the dangerously-github-MCP-write feature flag for non-read-only GitHub MCP toolsets
timeout-minutes: 5
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
tools:
  github:
    toolsets: [repos, issues, pull_requests]
    read-only: false
strict: false
features:
  dangerously-github-MCP-write: true
---

# Example: Non-Read-Only GitHub MCP Without Write Permissions

This workflow demonstrates using the `dangerously-github-MCP-write` feature flag to allow
non-read-only GitHub MCP toolsets without declaring write permissions in the frontmatter.

The workflow uses three GitHub toolsets with `read-only: false`, but only declares read permissions:
- The `repos` toolset would normally require `contents: write`
- The `issues` toolset would normally require `issues: write`
- The `pull_requests` toolset would normally require `pull-requests: write`

By enabling `dangerously-github-MCP-write: true`, the compiler skips the check that
non-read-only GitHub toolsets require write permissions to be declared. This allows the
workflow to compile without warnings even though the declared permissions don't include write access.

⚠️  Use this feature flag only when you understand the security implications of allowing
the GitHub MCP server to perform write operations without explicit permission declarations.
