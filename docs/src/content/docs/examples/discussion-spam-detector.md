---
title: Discussion Spam Detector
description: Flags newly-created GitHub Discussions for spam, exam dumps, AI-generated content, and non-English posts
---

## Overview

The **Discussion Spam Detector** is an agentic workflow that automatically monitors every new GitHub Discussion in your repository. It applies a weighted, evidence-based scoring system to detect low-quality or abusive content without relying on an AI/LLM by default. When a discussion exceeds the configured threshold, it creates a GitHub Issue with a full evidence table so maintainers can make an informed decision.

**It never auto-deletes, auto-bans, or posts comments** — it only creates a triage issue.

## How It Works

```
New Discussion Created
        │
        ▼
  Precompute Job (JavaScript)
  ┌──────────────────────────────────────────────┐
  │  1. Load config from repo (with fallback)    │
  │  2. Score 9 weighted signals (see below)     │
  │  3. Check for existing issue (idempotency)   │
  │  4. If score ≥ threshold → build issue body  │
  └──────────────────────────────────────────────┘
        │
        ▼  (only when score ≥ threshold AND no dup)
  Agent Job
  ┌──────────────────────────────────────────────┐
  │  Creates GitHub Issue with evidence table    │
  └──────────────────────────────────────────────┘
```

All scoring is deterministic and runs without calling an LLM. The agent job is minimal — it only formats and submits the precomputed evidence as a GitHub Issue.

## Signals & Weights

| Signal | Default Weight | Description |
|--------|---------------|-------------|
| `spam_external_links` | 15 | External (non-GitHub) links present |
| `url_shortener` | 20 | URL-shortener domain (bit.ly, tinyurl, etc.) |
| `binary_download_link` | 20 | Link to binary download or suspicious file host |
| `spam_keywords` | 20 | Promotional phrases ("buy now", "limited offer", etc.) |
| `promotional_contact` | 15 | Contact info spam (WhatsApp, Telegram, etc.) |
| `exam_dump` | 35 | MCQ format, "which of the following", certification dump keywords |
| `ai_generated` | 15 | AI-language indicators ("delve into", em-dashes, etc.) |
| `non_english` | 25 | More than 15% non-Latin Unicode characters |
| `new_account` | 10 | Author account is less than 30 days old |
| `short_body` | 10 | Discussion body is fewer than 50 characters |
| `all_caps` | 8 | Title is more than 70% uppercase |

Total score is capped at 100.

## Severity Thresholds (default)

| Severity | Score Range |
|----------|------------|
| 🟢 LOW | 10 – 29 |
| 🟡 MEDIUM | 30 – 59 |
| 🔴 HIGH | 60 – 100 |

A moderation issue is **only created when score ≥ `min_score_to_flag`** (default: 30).

## Configuration

Edit `.github/discussion-spam-detector-config.json` to tune the detector for your community:

```json
{
  "min_score_to_flag": 30,
  "threshold_low": 10,
  "threshold_medium": 30,
  "threshold_high": 60,
  "weights": {
    "spam_external_links": 15,
    "url_shortener": 20,
    "binary_download_link": 20,
    "spam_keywords": 20,
    "promotional_contact": 15,
    "exam_dump": 35,
    "ai_generated": 15,
    "non_english": 25,
    "new_account": 10,
    "short_body": 10,
    "all_caps": 8
  }
}
```

**Tips:**
- Increase `min_score_to_flag` to reduce false positives in active communities.
- Disable a signal by setting its weight to `0`.
- Set `threshold_high` lower to expand the "high" severity band.

## Idempotency

The detector checks for an existing open issue whose title contains `[Discussion #<number>]` before creating a new one. Re-running the workflow (e.g., via `workflow_dispatch`) for the same discussion will be a no-op if an issue already exists.

## Files

| File | Purpose |
|------|---------|
| `.github/workflows/discussion-spam-detector.md` | Workflow definition + agent prompt |
| `.github/workflows/discussion-spam-detector.lock.yml` | Compiled GitHub Actions YAML (auto-generated) |
| `.github/discussion-spam-detector-config.json` | Configurable weights and thresholds |

## Manual Testing

Use `workflow_dispatch` to test the detector against an existing discussion:

1. Go to **Actions** → **Discussion Spam Detector**
2. Click **Run workflow**
3. Enter a discussion number in the `discussion_number` input
4. The workflow will score the discussion and create an issue if the score is high enough

## Output Example

When a discussion is flagged, an issue is created with this format:

```markdown
## ⚠️ Discussion Quality Alert

| Field      | Value                                                        |
|------------|--------------------------------------------------------------|
| Discussion | [#42: FREE DOWNLOAD click here](https://github.com/...)     |
| Author     | @some-user (account age: 2d)                                 |
| Score      | **75/100**                                                   |
| Severity   | **🔴 HIGH**                                                  |

## 📊 Signal Analysis

| Signal               | Penalty | Reason                      | Evidence                 |
|----------------------|---------|-----------------------------|--------------------------|
| `spam_external_links`| +15     | 2 external link(s)          | `http://example.com ...` |
| `url_shortener`      | +20     | URL shortener detected      | `http://bit.ly/...`      |
| `spam_keywords`      | +20     | 2 spam keyword(s)           | `FREE DOWNLOAD, click here` |
| `new_account`        | +10     | Account is only 2 day(s) old| `@some-user created 2d ago` |
| `all_caps`           | +8      | Title is 82% uppercase      | `FREE DOWNLOAD CLICK ...` |
```
