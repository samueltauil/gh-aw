## Cache-Memory Deduplication Protocol

Use this protocol to prevent duplicate issues or discussions from being created when a workflow is retried or triggered repeatedly within a short time window.

### Overview

Before creating a new issue or discussion, check `cache-memory` for a recently created matching output. If a recent duplicate is found, add a comment to the existing item instead of creating a new one.

### Step 1 – Define the Cache Path

Choose a stable, filesystem-safe path under `/tmp/gh-aw/cache-memory/` that is unique to the workflow and output type:

```
/tmp/gh-aw/cache-memory/<workflow-id>/last-output.json
```

Replace `<workflow-id>` with a short identifier for the workflow (e.g., `ci-doctor`, `weekly-report`).

### Step 2 – Deduplication Check (Before Creating)

Before creating a new issue or discussion:

1. Read the cache file at the path defined above (it may not exist on the first run — that is normal).
2. If the file exists, parse it as JSON and inspect the `created_at` field.
3. Calculate the age of the cached entry: `now - created_at`.
4. **If the entry is recent** (e.g., within the last hour, or within a configurable window):
   - Add a comment with the current findings to the existing issue or discussion referenced by the `url` field.
   - Call `noop` (do not create a new item).
   - Stop here — skip the creation phases.
5. **If the entry is absent or older than the threshold**: proceed to create a new item normally.

Example cache file format:

```json
{
  "url": "https://github.com/owner/repo/issues/42",
  "title": "[CI Failure Doctor] Build failure in main",
  "created_at": "2026-02-12-11-20-45-458",
  "run_id": "12345678901"
}
```

### Step 3 – Cache Write (After Creating)

After successfully creating the issue or discussion:

1. Construct a JSON object with the following fields:
   - `url` – the URL of the newly created item
   - `title` – the title of the newly created item
   - `created_at` – the current timestamp in filesystem-safe format (see below)
   - `run_id` – `${{ github.run_id }}`
2. Write the JSON object to the cache path defined in Step 1, overwriting any previous entry.

### Filesystem-Safe Timestamp Format

**Always use** the format `YYYY-MM-DD-HH-MM-SS-sss` for timestamps stored in filenames or cache fields:

```
2026-02-12-11-20-45-458
```

**Never use** ISO 8601 format with colons (`2026-02-12T11:20:45.458Z`). Colons are not valid in artifact filenames on NTFS filesystems and will cause `actions/upload-artifact` to fail.

### Configuring the Deduplication Window

The recency threshold is configurable per workflow. Common values:

| Window | Use case |
|--------|----------|
| 1 hour | High-frequency triggers (push, PR events) |
| 6 hours | Scheduled workflows running multiple times per day |
| 24 hours | Daily scheduled workflows |

Document the chosen threshold in your workflow so future contributors understand the deduplication behaviour.

### Example Prompt Snippet

Include the following in your workflow prompt to apply this protocol:

```
Before creating a new issue:
1. Read /tmp/gh-aw/cache-memory/<workflow-id>/last-output.json
2. If it exists and created_at is within the last hour, add a comment to the existing
   issue at `url` and call noop instead of creating a new issue.
3. After creating a new issue, write the url, title, created_at (YYYY-MM-DD-HH-MM-SS-sss),
   and run_id to /tmp/gh-aw/cache-memory/<workflow-id>/last-output.json.
```
