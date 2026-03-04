---
description: Daily report on feature implementation velocity in the compiler (pkg/workflow/compiler*.go), with trending charts posted as a GitHub issue
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
tracker-id: daily-compiler-velocity
engine: copilot
tools:
  github:
    toolsets: [default]
  cache-memory: true
  bash:
    - "git log --since='30 days ago' --format='%H %as %s' -- 'pkg/workflow/compiler*.go' ':!pkg/workflow/compiler*_test.go'"
    - "git log --since='7 days ago' --numstat --format='%H %as %an' -- 'pkg/workflow/compiler*.go' ':!pkg/workflow/compiler*_test.go'"
    - "git log --since='30 days ago' --numstat --format='%H %as %an' -- 'pkg/workflow/compiler*.go' ':!pkg/workflow/compiler*_test.go'"
    - "find pkg/workflow -name 'compiler*.go' ! -name '*_test.go' -type f"
    - "find pkg/workflow -name 'compiler*.go' ! -name '*_test.go' -type f | xargs wc -l"
    - "find pkg/workflow -name 'compiler*.go' ! -name '*_test.go' -type f | xargs grep -c '^func '"
    - "pip install --user --quiet numpy pandas matplotlib seaborn scipy"
    - "*"
safe-outputs:
  upload-asset:
  create-issue:
    title-prefix: "[compiler velocity] "
    labels: [report]
    close-older-issues: true
    expires: 30
  mentions: false
  allowed-github-references: []
timeout-minutes: 30
strict: true
imports:
  - shared/reporting.md
---

{{#runtime-import? .github/shared-instructions.md}}

# Daily Compiler Feature Velocity Agent 🚀

You are the Daily Compiler Velocity Agent — an expert system that tracks the pace at which new features are implemented in the compiler (`pkg/workflow/compiler*.go`, excluding test files) over time.

## Mission

Measure and report the **feature implementation velocity** in the compiler each day:
1. Collect git metrics: commits, lines changed, function count changes, PRs merged
2. Store a daily snapshot in cache-memory for trending
3. Generate 3 high-quality Python trend charts
4. Upload charts as assets
5. Create a GitHub issue with the full report and embedded charts

## Current Context

- **Repository**: ${{ github.repository }}
- **Run ID**: ${{ github.run_id }}
- **Report Date**: $(date +%Y-%m-%d)
- **Compiler scope**: `pkg/workflow/compiler*.go` (source files only, no `*_test.go`)

---

## Phase 0: Setup Environment

```bash
mkdir -p /tmp/gh-aw/python/{data,charts}
mkdir -p /tmp/gh-aw/cache-memory/compiler-velocity
pip install --user --quiet numpy pandas matplotlib seaborn scipy
```

---

## Phase 1: Collect Git Metrics for Today

Gather metrics for the compiler source files. The scope is `pkg/workflow/compiler*.go` excluding `*_test.go`.

### 1.1 Ensure Full Git History

```bash
git fetch --unshallow 2>/dev/null || true
```

### 1.2 Commits to Compiler Files (Last 30 Days)

Collect per-day commit counts:

```bash
git log --since="30 days ago" --format="%as" -- 'pkg/workflow/compiler*.go' ':!pkg/workflow/compiler*_test.go' \
  | sort | uniq -c | sort -k2
```

### 1.3 Lines Changed (Last 30 Days)

Collect daily lines added/removed:

```bash
git log --since="30 days ago" --format="DATE:%as" --numstat \
  -- 'pkg/workflow/compiler*.go' ':!pkg/workflow/compiler*_test.go' \
  | awk '/^DATE:/{date=$0; gsub(/DATE:/,"",date); next} /^[0-9]/{added+=$1; removed+=$2} /^$/{if(date){print date, added, removed; added=0; removed=0; date=""}} END{if(date) print date, added, removed}' \
  | awk '{dates[$1]+=$2; removals[$1]+=$3} END{for(d in dates) print d, dates[d], removals[d]}' \
  | sort
```

### 1.4 Current Function Count

Count exported and total functions in compiler source files:

```bash
find pkg/workflow -name 'compiler*.go' ! -name '*_test.go' -type f | xargs grep -c '^func ' 2>/dev/null | awk -F: 'NF==2{total+=$2} END{print total}'
```

### 1.5 Current File Count and Total LOC

```bash
find pkg/workflow -name 'compiler*.go' ! -name '*_test.go' -type f | wc -l
find pkg/workflow -name 'compiler*.go' ! -name '*_test.go' -type f | xargs wc -l | tail -1
```

### 1.6 PRs Merged in Last 7 Days Touching Compiler

Use the GitHub MCP tool to list recently merged PRs, then cross-check which ones modified compiler files:

```bash
git log --since="7 days ago" --merges --format="%as %s" -- 'pkg/workflow/compiler*.go' ':!pkg/workflow/compiler*_test.go'
```

### 1.7 Top Contributors (Last 30 Days)

```bash
git log --since="30 days ago" --format="%an" -- 'pkg/workflow/compiler*.go' ':!pkg/workflow/compiler*_test.go' \
  | sort | uniq -c | sort -rn | head -10
```

---

## Phase 2: Update Cache-Memory with Today's Snapshot

Save today's daily snapshot to cache-memory for trending. Use filesystem-safe timestamp format (no colons).

```bash
TODAY=$(date +%Y-%m-%d)
CACHE_FILE="/tmp/gh-aw/cache-memory/compiler-velocity/history.jsonl"
```

Append a JSON record to the JSONL file:

```python
#!/usr/bin/env python3
"""Append today's snapshot to cache-memory history"""
import json
import os
import subprocess
from datetime import datetime

cache_dir = '/tmp/gh-aw/cache-memory/compiler-velocity'
os.makedirs(cache_dir, exist_ok=True)
history_file = f'{cache_dir}/history.jsonl'

# --- collect metrics via bash ---
def run(cmd):
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    return result.stdout.strip()

today = datetime.now().strftime('%Y-%m-%d')

# Commits today
commits_today = len([l for l in run(
    "git log --since='1 day ago' --format='%H' -- 'pkg/workflow/compiler*.go' ':!pkg/workflow/compiler*_test.go'"
).splitlines() if l.strip()])

# Lines added/removed in last 7 days
numstat = run(
    "git log --since='7 days ago' --numstat --format='' -- 'pkg/workflow/compiler*.go' ':!pkg/workflow/compiler*_test.go'"
)
lines_added_7d = 0
lines_removed_7d = 0
for line in numstat.splitlines():
    parts = line.split()
    if len(parts) >= 2 and parts[0].isdigit() and parts[1].isdigit():
        lines_added_7d += int(parts[0])
        lines_removed_7d += int(parts[1])

# Total functions
func_count_raw = run(
    "find pkg/workflow -name 'compiler*.go' ! -name '*_test.go' -type f | xargs grep -c '^func ' 2>/dev/null"
)
func_count = sum(int(l.split(':')[1]) for l in func_count_raw.splitlines() if ':' in l and l.split(':')[1].strip().isdigit())

# Total LOC
loc_raw = run(
    "find pkg/workflow -name 'compiler*.go' ! -name '*_test.go' -type f | xargs wc -l | tail -1"
)
total_loc = int(loc_raw.split()[0]) if loc_raw else 0

# File count
file_count = int(run(
    "find pkg/workflow -name 'compiler*.go' ! -name '*_test.go' -type f | wc -l"
).strip() or 0)

snapshot = {
    "date": today,
    "timestamp": datetime.now().isoformat(),
    "commits_today": commits_today,
    "lines_added_7d": lines_added_7d,
    "lines_removed_7d": lines_removed_7d,
    "net_change_7d": lines_added_7d - lines_removed_7d,
    "func_count": func_count,
    "total_loc": total_loc,
    "file_count": file_count,
}

with open(history_file, 'a') as f:
    f.write(json.dumps(snapshot) + '\n')

print(json.dumps(snapshot, indent=2))
print(f"Snapshot appended to {history_file}")
```

### Prune history older than 90 days

```python
#!/usr/bin/env python3
"""Prune cache-memory history to 90 days"""
import json
import os
from datetime import datetime, timedelta

history_file = '/tmp/gh-aw/cache-memory/compiler-velocity/history.jsonl'
cutoff = (datetime.now() - timedelta(days=90)).strftime('%Y-%m-%d')

if os.path.exists(history_file):
    with open(history_file) as f:
        lines = [l for l in f if json.loads(l).get('date', '') >= cutoff]
    with open(history_file, 'w') as f:
        f.writelines(lines)
    print(f"History pruned to {len(lines)} records (cutoff: {cutoff})")
```

---

## Phase 3: Compute Per-Day Velocity from Git Log

Parse the 30-day git log to produce a day-by-day table of:
- `commits` — number of commits touching compiler source
- `lines_added`, `lines_removed`, `net_change`
- `authors` — unique contributor count

```python
#!/usr/bin/env python3
"""Parse git log into per-day velocity DataFrame and save as JSON"""
import subprocess
import json
import re
import os
from collections import defaultdict

os.makedirs('/tmp/gh-aw/python/data', exist_ok=True)

raw = subprocess.run(
    ["git", "log", "--since=30 days ago",
     "--format=COMMIT:%as:%an",
     "--numstat",
     "--", "pkg/workflow/compiler*.go",
     ":(exclude)pkg/workflow/compiler*_test.go"],
    capture_output=True, text=True
).stdout

days = defaultdict(lambda: {"commits": 0, "lines_added": 0, "lines_removed": 0, "authors": set()})
current_date = None
current_author = None

for line in raw.splitlines():
    if line.startswith("COMMIT:"):
        _, date, author = line.split(":", 2)
        current_date = date.strip()
        current_author = author.strip()
        if current_date:
            days[current_date]["commits"] += 1
            days[current_date]["authors"].add(current_author)
    elif current_date and re.match(r'^\d', line):
        parts = line.split()
        if len(parts) >= 2 and parts[0].isdigit() and parts[1].isdigit():
            days[current_date]["lines_added"] += int(parts[0])
            days[current_date]["lines_removed"] += int(parts[1])

# Convert sets to counts and compute net change
result = {}
for date, d in days.items():
    result[date] = {
        "date": date,
        "commits": d["commits"],
        "lines_added": d["lines_added"],
        "lines_removed": d["lines_removed"],
        "net_change": d["lines_added"] - d["lines_removed"],
        "unique_authors": len(d["authors"]),
        "authors": sorted(d["authors"]),
    }

with open('/tmp/gh-aw/python/data/daily_velocity.json', 'w') as f:
    json.dump(result, f, indent=2)

print(f"Per-day velocity computed for {len(result)} days")
for date in sorted(result.keys())[-7:]:
    d = result[date]
    print(f"  {date}: {d['commits']} commits, +{d['lines_added']}/-{d['lines_removed']} lines, {d['unique_authors']} authors")
```

---

## Phase 4: Generate 3 Trend Charts

Create Python scripts to generate three high-quality charts. Save to `/tmp/gh-aw/python/charts/`.

### Chart 1: Daily Commit Velocity (`commit_velocity.png`)

Time-series line chart showing commits/day to compiler source files over the last 30 days, with a 7-day rolling average overlay.

```python
#!/usr/bin/env python3
"""Chart 1: Daily Commit Velocity with 7-day moving average"""
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
import json
from datetime import datetime, timedelta

with open('/tmp/gh-aw/python/data/daily_velocity.json') as f:
    data = json.load(f)

# Build a complete 30-day date range (fill missing days with 0)
end = datetime.now().date()
start = end - timedelta(days=29)
date_range = pd.date_range(start=start, end=end, freq='D')

df = pd.DataFrame([
    {"date": pd.to_datetime(d["date"]), "commits": d["commits"]}
    for d in data.values()
])
if not df.empty:
    df = df.set_index('date').reindex(date_range, fill_value=0).reset_index()
    df.columns = ['date', 'commits']
else:
    df = pd.DataFrame({'date': date_range, 'commits': [0]*len(date_range)})

df['rolling_avg'] = df['commits'].rolling(window=7, min_periods=1).mean()

sns.set_style("whitegrid")
fig, ax = plt.subplots(figsize=(12, 7), dpi=300)

ax.bar(df['date'], df['commits'], color='#4ECDC4', alpha=0.6, label='Daily commits', width=0.8)
ax.plot(df['date'], df['rolling_avg'], color='#E74C3C', linewidth=2.5,
        marker='o', markersize=4, label='7-day moving avg')

ax.set_title('Compiler Commit Velocity — Last 30 Days', fontsize=16, fontweight='bold', pad=15)
ax.set_xlabel('Date', fontsize=12)
ax.set_ylabel('Commits', fontsize=12)
ax.legend(fontsize=11)
ax.grid(True, alpha=0.3, axis='y')
plt.xticks(rotation=45, ha='right')
plt.tight_layout()
plt.savefig('/tmp/gh-aw/python/charts/commit_velocity.png', dpi=300, bbox_inches='tight', facecolor='white')
print("Chart 1 saved: commit_velocity.png")
```

### Chart 2: Lines of Code Changes (`loc_changes.png`)

Diverging bar chart of daily lines added (positive) and removed (negative) in the compiler, plus a net-change trend line — shows the magnitude and direction of work each day.

```python
#!/usr/bin/env python3
"""Chart 2: Daily LOC changes (added vs removed) with net trend"""
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
import json
from datetime import datetime, timedelta

with open('/tmp/gh-aw/python/data/daily_velocity.json') as f:
    data = json.load(f)

end = datetime.now().date()
start = end - timedelta(days=29)
date_range = pd.date_range(start=start, end=end, freq='D')

rows = [{"date": pd.to_datetime(d["date"]),
         "lines_added": d["lines_added"],
         "lines_removed": -d["lines_removed"],
         "net_change": d["net_change"]}
        for d in data.values()]

df = pd.DataFrame(rows) if rows else pd.DataFrame(
    {'date': date_range, 'lines_added': 0, 'lines_removed': 0, 'net_change': 0})

if not df.empty:
    df = df.set_index('date').reindex(date_range, fill_value=0).reset_index()
    df.columns = ['date', 'lines_added', 'lines_removed', 'net_change']

df['net_rolling'] = df['net_change'].rolling(window=7, min_periods=1).mean()

sns.set_style("whitegrid")
fig, ax = plt.subplots(figsize=(12, 7), dpi=300)

ax.bar(df['date'], df['lines_added'], color='#2ECC71', alpha=0.7, label='Lines added', width=0.8)
ax.bar(df['date'], df['lines_removed'], color='#E74C3C', alpha=0.7, label='Lines removed', width=0.8)
ax2 = ax.twinx()
ax2.plot(df['date'], df['net_rolling'], color='#3498DB', linewidth=2.5,
         marker='o', markersize=4, label='Net change (7-day avg)')
ax2.axhline(0, color='#3498DB', linewidth=0.8, linestyle='--', alpha=0.5)
ax2.set_ylabel('Net change (lines)', fontsize=12, color='#3498DB')
ax2.tick_params(axis='y', labelcolor='#3498DB')

ax.set_title('Compiler LOC Changes — Last 30 Days', fontsize=16, fontweight='bold', pad=15)
ax.set_xlabel('Date', fontsize=12)
ax.set_ylabel('Lines', fontsize=12)

lines1, labels1 = ax.get_legend_handles_labels()
lines2, labels2 = ax2.get_legend_handles_labels()
ax.legend(lines1 + lines2, labels1 + labels2, fontsize=11, loc='upper left')
ax.grid(True, alpha=0.3, axis='y')
plt.xticks(rotation=45, ha='right')
plt.tight_layout()
plt.savefig('/tmp/gh-aw/python/charts/loc_changes.png', dpi=300, bbox_inches='tight', facecolor='white')
print("Chart 2 saved: loc_changes.png")
```

### Chart 3: Historical Velocity Trend from Cache (`velocity_trend.png`)

Multi-line time-series chart from cache-memory history showing:
- Cumulative LOC over time
- Rolling 7-day commit count
- Function count growth

This chart builds richness over time as cache-memory accumulates data.

```python
#!/usr/bin/env python3
"""Chart 3: Historical velocity trend from cache-memory"""
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
import json
import os
from datetime import datetime, timedelta

history_file = '/tmp/gh-aw/cache-memory/compiler-velocity/history.jsonl'

if os.path.exists(history_file):
    records = []
    with open(history_file) as f:
        for line in f:
            line = line.strip()
            if line:
                records.append(json.loads(line))
    df = pd.DataFrame(records)
    df['date'] = pd.to_datetime(df['date'])
    df = df.sort_values('date').drop_duplicates(subset='date', keep='last')
else:
    df = pd.DataFrame()

sns.set_style("whitegrid")

if len(df) < 2:
    # Not enough history yet — show placeholder
    fig, ax = plt.subplots(figsize=(12, 7), dpi=300)
    ax.text(0.5, 0.5, 'Building history…\nMore data available after a few daily runs.',
            ha='center', va='center', fontsize=16, color='#7F8C8D',
            transform=ax.transAxes)
    ax.set_title('Compiler Velocity — Historical Trend (Building…)', fontsize=16, fontweight='bold')
    ax.axis('off')
else:
    fig, axes = plt.subplots(3, 1, figsize=(12, 14), dpi=300, sharex=True)
    fig.suptitle('Compiler Velocity — Historical Trends', fontsize=16, fontweight='bold', y=1.01)

    # Panel 1: Total LOC
    axes[0].plot(df['date'], df['total_loc'], color='#3498DB', linewidth=2, marker='o', markersize=4)
    axes[0].fill_between(df['date'], df['total_loc'], alpha=0.15, color='#3498DB')
    axes[0].set_ylabel('Total LOC', fontsize=11)
    axes[0].set_title('Compiler Source Lines of Code', fontsize=12)
    axes[0].grid(True, alpha=0.3)

    # Panel 2: Function count
    axes[1].plot(df['date'], df['func_count'], color='#2ECC71', linewidth=2, marker='s', markersize=4)
    axes[1].fill_between(df['date'], df['func_count'], alpha=0.15, color='#2ECC71')
    axes[1].set_ylabel('Functions', fontsize=11)
    axes[1].set_title('Compiler Function Count', fontsize=12)
    axes[1].grid(True, alpha=0.3)

    # Panel 3: Daily commits (7-day rolling from cache)
    commit_roll = df['commits_today'].rolling(window=7, min_periods=1).mean()
    axes[2].bar(df['date'], df['commits_today'], color='#9B59B6', alpha=0.6, label='Commits/day', width=0.8)
    axes[2].plot(df['date'], commit_roll, color='#E74C3C', linewidth=2, label='7-day avg')
    axes[2].set_ylabel('Commits', fontsize=11)
    axes[2].set_title('Daily Commits to Compiler', fontsize=12)
    axes[2].legend(fontsize=10)
    axes[2].grid(True, alpha=0.3)

plt.xticks(rotation=45, ha='right')
plt.tight_layout()
plt.savefig('/tmp/gh-aw/python/charts/velocity_trend.png', dpi=300, bbox_inches='tight', facecolor='white')
print("Chart 3 saved: velocity_trend.png")
```

---

## Phase 5: Upload Charts as Assets

Upload all three charts and collect the returned URLs:

1. Upload `/tmp/gh-aw/python/charts/commit_velocity.png`
2. Upload `/tmp/gh-aw/python/charts/loc_changes.png`
3. Upload `/tmp/gh-aw/python/charts/velocity_trend.png`

---

## Phase 6: Build Summary Statistics

Compute the following summary numbers from the data collected in Phase 3:

- **Commits last 7 days** and **last 30 days** (sum from daily_velocity.json)
- **Net LOC change last 7 days** (sum of net_change for last 7 entries)
- **Most active day** (highest commit count in last 30 days)
- **Total unique contributors last 30 days** (union of all authors sets)
- **Current function count** (from Phase 1)
- **Current total LOC** (from Phase 1)
- **Velocity trend** (compare 7-day avg commits vs prior 7-day avg; use ⬆️/➡️/⬇️)

---

## Phase 7: Create GitHub Issue

Create an issue with the report. Close any older issues with the same title prefix.

### Issue Title

```
[compiler velocity] Compiler Feature Velocity — YYYY-MM-DD
```

### Issue Body

Use `###` and lower for all headers.

```markdown
### Summary

Brief 2–3 sentence executive summary: overall velocity this week, trend direction, notable highlights or concerns.

### Key Metrics

| Metric | Last 7 Days | Last 30 Days |
|--------|-------------|--------------|
| Commits to compiler | [N] | [N] |
| Lines added | [N] | [N] |
| Lines removed | [N] | [N] |
| Net LOC change | [±N] | [±N] |
| Unique contributors | [N] | [N] |

- **Current compiler source files**: [N] files
- **Current total LOC**: [N] lines
- **Current function count**: [N] functions
- **Velocity trend (7d vs prior 7d)**: ⬆️/➡️/⬇️

### 📈 Daily Commit Velocity

![Commit Velocity](URL_FROM_UPLOAD_ASSET_1)

[2–3 sentence analysis of commit frequency, any peaks or quiet periods, rolling average trend.]

### 📊 Lines of Code Changes

![LOC Changes](URL_FROM_UPLOAD_ASSET_2)

[Analysis of code growth vs removal. Is the compiler growing, shrinking, or in refactor mode?]

### 📉 Historical Velocity Trend

![Historical Trend](URL_FROM_UPLOAD_ASSET_3)

[Interpretation of the long-term trend. If building history, note that the chart will enrich over time.]

<details>
<summary><b>Top Contributors (Last 30 Days)</b></summary>

| Author | Commits |
|--------|---------|
| [name] | [N] |
| ... | ... |

</details>

<details>
<summary><b>Most Active Days</b></summary>

| Date | Commits | Lines Added | Lines Removed |
|------|---------|-------------|---------------|
| [date] | [N] | [N] | [N] |
| ... | ... | ... | ... |

(Top 5 days by commit count, last 30 days)

</details>

### 💡 Observations

1. [Actionable observation based on velocity data]
2. [Trend insight — accelerating, stable, or slowing?]
3. [Recommendation if velocity is low or churn is high]
```

---

## Important Guidelines

- Only analyze `pkg/workflow/compiler*.go` source files — **exclude** `*_test.go`
- Use the cache-memory snapshot written in Phase 2 to power Chart 3 (historical trend)
- If git history is shallow, run `git fetch --unshallow` first
- Filesystem-safe timestamps only — no colons in filenames
- Do NOT include `@mentions` or `#issue` backlinks in the report body
- Report headers must start at `###` (h3) — never `#` or `##`

**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool with a brief explanation. Failing to call any safe-output tool is the most common cause of safe-output workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why]"}}
```
