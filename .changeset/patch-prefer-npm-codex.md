---
"gh-aw": patch
---
Prefer the npm-installed Codex CLI on self-hosted runners by extending the PATH setup in both AWF and non-AWF execution paths so the workflow-installed binary (with --dangerously-bypass-approvals-and-sandbox) runs instead of outdated vendored installs.
