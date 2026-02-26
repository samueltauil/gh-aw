---
description: Flags newly-created GitHub Discussions that look like spam, exam-question dumps, AI/bot-generated, or non-English content, then creates a GitHub Issue with evidence
on:
  discussion:
    types: [created]
  workflow_dispatch:
    inputs:
      discussion_number:
        description: "Discussion number to (re-)analyze (for manual testing)"
        required: false
        type: string
permissions:
  contents: read
  discussions: read
  issues: read
tools:
  github:
    mode: local
    read-only: true
    toolsets: [issues, discussions]
if: needs.precompute.outputs.action == 'create'
jobs:
  precompute:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      discussions: read
      issues: read
    outputs:
      action: ${{ steps.score.outputs.action }}
      issue_title: ${{ steps.score.outputs.issue_title }}
      issue_body: ${{ steps.score.outputs.issue_body }}
    steps:
      - name: Score discussion
        id: score
        uses: actions/github-script@v7
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { owner, repo } = context.repo;

            // ── Default configuration ──────────────────────────────────────────────
            const DEFAULT_CONFIG = {
              min_score_to_flag: 30,
              threshold_low: 10,
              threshold_medium: 30,
              threshold_high: 60,
              weights: {
                spam_external_links: 15,
                url_shortener: 20,
                binary_download_link: 20,
                spam_keywords: 20,
                promotional_contact: 15,
                exam_dump: 35,
                ai_generated: 15,
                non_english: 25,
                new_account: 10,
                short_body: 10,
                all_caps: 8,
              },
            };

            // ── Load repo config (with fallback to defaults) ───────────────────────
            let cfg = {
              ...DEFAULT_CONFIG,
              weights: { ...DEFAULT_CONFIG.weights },
            };
            try {
              const { data: cfgFile } = await github.rest.repos.getContent({
                owner,
                repo,
                path: ".github/discussion-spam-detector-config.json",
              });
              const raw = Buffer.from(cfgFile.content, "base64").toString("utf8");
              const parsed = JSON.parse(raw);
              cfg = {
                ...cfg,
                ...parsed,
                weights: { ...cfg.weights, ...(parsed.weights || {}) },
              };
              core.info("Loaded repo config from .github/discussion-spam-detector-config.json");
            } catch {
              core.info("No repo config found — using built-in defaults");
            }

            // ── Resolve discussion ─────────────────────────────────────────────────
            let discussion = null;

            if (context.eventName === "workflow_dispatch") {
              const numStr = core.getInput("discussion_number") || "";
              const num = parseInt(numStr, 10);
              if (!num) {
                core.info("workflow_dispatch: no discussion_number input — nothing to do");
                core.setOutput("action", "none");
                return;
              }
              try {
                const gql = `
                  query($owner: String!, $repo: String!, $number: Int!) {
                    repository(owner: $owner, name: $repo) {
                      discussion(number: $number) {
                        number
                        title
                        body
                        url
                        author { login }
                        createdAt
                      }
                    }
                  }`;
                const result = await github.graphql(gql, { owner, repo, number: num });
                const d = result.repository.discussion;
                if (!d) {
                  core.info(`Discussion #${num} not found`);
                  core.setOutput("action", "none");
                  return;
                }
                discussion = {
                  number: d.number,
                  title: d.title || "",
                  body: d.body || "",
                  url: d.url,
                  authorLogin: d.author?.login || "",
                  createdAt: d.createdAt,
                };
              } catch (err) {
                core.warning(`Failed to fetch discussion via GraphQL: ${err.message}`);
                core.setOutput("action", "none");
                return;
              }
            } else if (context.payload.discussion) {
              const d = context.payload.discussion;
              discussion = {
                number: d.number,
                title: d.title || "",
                body: d.body || "",
                url: d.html_url,
                authorLogin: d.user?.login || "",
                createdAt: d.created_at,
              };
            }

            if (!discussion) {
              core.info("No discussion found in event payload");
              core.setOutput("action", "none");
              return;
            }

            const { number: discNum, title, body, url: discUrl, authorLogin } = discussion;
            const text = `${title}\n\n${body}`;

            core.info(`Scoring discussion #${discNum} by @${authorLogin}: "${title.slice(0, 80)}"`);

            // ── Fetch account age ──────────────────────────────────────────────────
            let accountAgeDays = null;
            if (authorLogin) {
              try {
                const { data: user } = await github.rest.users.getByUsername({ username: authorLogin });
                accountAgeDays = Math.floor((Date.now() - new Date(user.created_at).getTime()) / (86400 * 1000));
              } catch {
                // skip silently if user lookup fails
              }
            }

            // ── Scoring helpers ────────────────────────────────────────────────────
            const penalties = [];

            function addPenalty(signal, pts, reason, evidence) {
              if (pts > 0) {
                penalties.push({
                  signal,
                  pts,
                  reason,
                  evidence: evidence ? String(evidence).slice(0, 200) : "",
                });
              }
            }

            // ── 1. External links & URL signals ───────────────────────────────────
            const URL_RE = /https?:\/\/[^\s)\]>"]+/g;
            const rawUrls = text.match(URL_RE) || [];

            const GITHUB_HOSTS = new Set([
              "github.com",
              "raw.githubusercontent.com",
              "gist.github.com",
              "objects.githubusercontent.com",
            ]);
            const SHORTENERS = new Set([
              "bit.ly",
              "tinyurl.com",
              "t.co",
              "is.gd",
              "goo.gl",
              "ow.ly",
              "short.link",
              "rb.gy",
              "cutt.ly",
            ]);
            const BINARY_EXTS = [".exe", ".msi", ".pkg", ".dmg", ".apk", ".bat", ".ps1", ".sh"];
            const SUSPICIOUS_DOMAINS = new Set([
              "mediafire.com",
              "mega.nz",
              "sendspace.com",
              "rapidshare.com",
              "zippyshare.com",
              "4shared.com",
              "filehippo.com",
            ]);

            const externalUrls = [];
            const shortenerUrls = [];
            const binaryUrls = [];

            for (const raw of rawUrls) {
              let u;
              try {
                u = new URL(raw);
              } catch {
                continue;
              }
              const host = u.hostname.toLowerCase().replace(/^www\./, "");
              if (GITHUB_HOSTS.has(host)) continue;

              externalUrls.push(raw);
              if (SHORTENERS.has(host)) shortenerUrls.push(raw);
              if (SUSPICIOUS_DOMAINS.has(host) || BINARY_EXTS.some(ext => u.pathname.toLowerCase().endsWith(ext))) {
                binaryUrls.push(raw);
              }
            }

            if (externalUrls.length > 0) {
              // Scale: full weight at 3+ links; 1 link → weight/3, 2 links → 2×weight/3, 3+ → full weight
              const pts = Math.min(cfg.weights.spam_external_links, Math.ceil(cfg.weights.spam_external_links * externalUrls.length / 3));
              addPenalty("spam_external_links", pts, `${externalUrls.length} external link(s)`, externalUrls.slice(0, 3).join(" "));
            }
            if (shortenerUrls.length > 0) {
              addPenalty("url_shortener", cfg.weights.url_shortener, "URL shortener detected", shortenerUrls[0]);
            }
            if (binaryUrls.length > 0) {
              addPenalty("binary_download_link", cfg.weights.binary_download_link, "Binary/file download link", binaryUrls[0]);
            }

            // ── 2. Spam keywords ───────────────────────────────────────────────────
            const SPAM_PATTERNS = [
              /\bbuy\s+now\b/i,
              /\bfree\s+(download|trial|access|crack|license)\b/i,
              /\blimited[- ]time\s+offer\b/i,
              /\bclick\s+here\b/i,
              /\bvisit\s+(our\s+)?website\b/i,
              /\bget\s+(it\s+)?now\b/i,
              /\bcoupon\s*(code)?\b/i,
              /\bpromo(tion)?\s*(code)?\b/i,
              /\bspecial\s+(offer|deal|discount)\b/i,
              /\bmake\s+money\s+(online|fast)\b/i,
              /\bearn\s+\$\d+/i,
              /\b\d{1,3}\s*%\s*off\b/i,
              /\bno\s+(credit\s+card|cc)\s+required\b/i,
              /\bsatisfaction\s+guaranteed\b/i,
              /\bact\s+now\b/i,
              /\blimited\s+seats?\b/i,
            ];
            const spamHits = SPAM_PATTERNS.flatMap(re => { const m = text.match(re); return m ? [m[0]] : []; });
            if (spamHits.length > 0) {
              addPenalty("spam_keywords", cfg.weights.spam_keywords, `${spamHits.length} spam keyword(s)`, spamHits.slice(0, 3).join(", "));
            }

            // ── 3. Promotional contact info ────────────────────────────────────────
            const CONTACT_RE = [
              /whatsapp\s*[:\s]\s*[+\d()\s-]{7,}/i,
              /telegram\s*[:\s]\s*@\w+/i,
              /wechat\s*[:\s]\s*\w+/i,
              /contact\s+(us\s+)?at\s+[\w.+-]+@[\w-]+\.\w+/i,
              /email\s+(us\s+)?at\s+[\w.+-]+@[\w-]+\.\w+/i,
            ];
            const contactHits = CONTACT_RE.flatMap(re => { const m = text.match(re); return m ? [m[0]] : []; });
            if (contactHits.length > 0) {
              addPenalty("promotional_contact", cfg.weights.promotional_contact, "Promotional contact info", contactHits[0]);
            }

            // ── 4. Exam / question-dump patterns ──────────────────────────────────
            const EXAM_RE = [
              /\bwhich\s+of\s+the\s+following\b/i,
              /\bcorrect\s+answer\s*[:\-]/i,
              /\bexam\s+dumps?\b/i,
              /\bpractice\s+(exam|test|questions?)\b/i,
              /\b(aws|azure|gcp|comptia|cisco|ccna|ccnp|mcsa|mcse|cissp|ceh|pmp|prince2)\s+(cert\w*|exam|test|dump|question)/i,
              /\bquestion\s+bank\b/i,
              /\bpass\s+guarantee\b/i,
              /\bbraindumps?\b/i,
            ];
            const mcqLineCount = (text.match(/^\s*[A-D][.)]\s+\S/mg) || []).length;
            const examHits = EXAM_RE.flatMap(re => { const m = text.match(re); return m ? [m[0].trim()] : []; });

            if (examHits.length > 0 || mcqLineCount >= 3) {
              const pts = mcqLineCount >= 3 ? cfg.weights.exam_dump : Math.ceil(cfg.weights.exam_dump * 0.6);
              addPenalty(
                "exam_dump",
                pts,
                `Exam-dump patterns (${mcqLineCount} MCQ options, ${examHits.length} keyword(s))`,
                examHits.slice(0, 2).join("; ") || `${mcqLineCount} MCQ option lines`,
              );
            }

            // ── 5. AI / bot-generated language ────────────────────────────────────
            const AI_RE = [
              /\bdelve\s+into\b/i,
              /\bembark\s+on\b/i,
              /\bunlock\s+(your\s+)?(full\s+)?potential\b/i,
              /\bcomprehensive\s+(guide|overview|tutorial|approach|solution)\b/i,
              /\bin\s+today'?s?\s+(fast[-\s]paced|digital|modern)\s+world\b/i,
              /\bit'?s?\s+worth\s+noting\b/i,
              /\bkey\s+takeaways?\b/i,
              /\bas\s+an?\s+(ai|language model|llm|chatgpt|gpt)\b/i,
              /\u2014/,
              /\bseamless(?:ly)?\b/i,
              /\brobust\s+solution\b/i,
              /\beveryone'?s?\s+needs\b/i,
              /\bstate[-\s]of[-\s]the[-\s]art\b/i,
              /\bleverage\s+(the|our|this)\b/i,
              /\btailored\s+(to\s+)?(your|the)\s+(needs|requirement)\b/i,
            ];
            const aiHits = AI_RE.flatMap(re => { const m = text.match(re); return m ? [m[0].trim()] : []; });
            if (aiHits.length >= 2) {
              addPenalty("ai_generated", cfg.weights.ai_generated, `${aiHits.length} AI-language indicator(s)`, aiHits.slice(0, 3).join(", "));
            }

            // ── 6. Non-English / non-Latin script ─────────────────────────────────
            const NON_LATIN_RE = /[\u0600-\u06FF\u0750-\u077F\u4E00-\u9FFF\u3040-\u30FF\u1100-\u11FF\uAC00-\uD7AF\u0400-\u04FF\u0900-\u097F\u0980-\u09FF\u0A80-\u0AFF]/g;
            const nonLatinCount = (text.match(NON_LATIN_RE) || []).length;
            const totalChars = text.length;
            const nonLatinRatio = totalChars > 0 ? nonLatinCount / totalChars : 0;
            if (nonLatinRatio > 0.15 || (nonLatinCount > 30)) {
              addPenalty(
                "non_english",
                cfg.weights.non_english,
                `${Math.round(nonLatinRatio * 100)}% non-Latin characters (${nonLatinCount} chars)`,
                `Non-Latin script detected`,
              );
            }

            // ── 7. New account ─────────────────────────────────────────────────────
            if (accountAgeDays !== null && accountAgeDays < 30) {
              addPenalty(
                "new_account",
                cfg.weights.new_account,
                `Account is only ${accountAgeDays} day(s) old`,
                `@${authorLogin} created ${accountAgeDays}d ago`,
              );
            }

            // ── 8. Short body ──────────────────────────────────────────────────────
            const bodyTrimmed = body.trim();
            if (bodyTrimmed.length > 0 && bodyTrimmed.length < 50) {
              addPenalty("short_body", cfg.weights.short_body, `Body is very short (${bodyTrimmed.length} chars)`, bodyTrimmed.slice(0, 60));
            }

            // ── 9. All-caps title ──────────────────────────────────────────────────
            const letters = title.replace(/[^A-Za-z]/g, "");
            const upperRatio = letters.length > 5 ? title.replace(/[^A-Z]/g, "").length / letters.length : 0;
            if (upperRatio > 0.7) {
              addPenalty("all_caps", cfg.weights.all_caps, `Title is ${Math.round(upperRatio * 100)}% uppercase`, title.slice(0, 80));
            }

            // ── Aggregate score + severity ─────────────────────────────────────────
            const totalScore = Math.min(100, penalties.reduce((s, p) => s + p.pts, 0));
            let severity = "low";
            if (totalScore >= cfg.threshold_high) severity = "high";
            else if (totalScore >= cfg.threshold_medium) severity = "medium";

            core.info(`Score: ${totalScore}/100 (${severity}), penalties: ${penalties.map(p => `${p.signal}(+${p.pts})`).join(", ") || "none"}`);

            // ── Idempotency check ──────────────────────────────────────────────────
            // Title marker: includes the discussion number so duplicates are detected
            const TITLE_MARKER = `[Discussion #${discNum}]`;
            let alreadyFlagged = false;
            try {
              // Search open issues whose title contains the marker
              const q = `repo:${owner}/${repo} is:issue is:open "${TITLE_MARKER}" in:title`;
              const result = await github.rest.search.issuesAndPullRequests({ q, per_page: 5 });
              alreadyFlagged = (result.data.total_count || 0) > 0;
            } catch {
              // Fall back to listing issues if search fails
              try {
                const { data: openIssues } = await github.rest.issues.listForRepo({
                  owner,
                  repo,
                  state: "open",
                  per_page: 100,
                });
                alreadyFlagged = openIssues.some(i => (i.title || "").includes(TITLE_MARKER));
              } catch {
                core.warning("Could not check for existing issues — proceeding anyway");
              }
            }

            if (alreadyFlagged) {
              core.info(`Issue for discussion #${discNum} already exists — skipping (idempotent)`);
              core.setOutput("action", "none");
              return;
            }

            // ── Threshold gate ─────────────────────────────────────────────────────
            if (totalScore < cfg.min_score_to_flag) {
              core.info(`Score ${totalScore} is below threshold ${cfg.min_score_to_flag} — no action`);
              core.setOutput("action", "none");
              return;
            }

            // ── Build issue content ────────────────────────────────────────────────
            const BADGE = { high: "🔴 HIGH", medium: "🟡 MEDIUM", low: "🟢 LOW" }[severity];

            const evidenceRows = penalties
              .map(p => `| \`${p.signal}\` | +${p.pts} | ${p.reason} | ${p.evidence ? `\`${p.evidence.replace(/`/g, "'")}\`` : "—"} |`)
              .join("\n");

            const issueTitle = `[Discussion Spam Detector] ${TITLE_MARKER}: ${title.slice(0, 60)}`;

            const lines = [
              `## ⚠️ Discussion Quality Alert`,
              ``,
              `| Field | Value |`,
              `|-------|-------|`,
              `| Discussion | [#${discNum}: ${title.slice(0, 80)}](${discUrl}) |`,
              `| Author | @${authorLogin}${accountAgeDays !== null ? ` (account age: ${accountAgeDays}d)` : ""} |`,
              `| Score | **${totalScore}/100** |`,
              `| Severity | **${BADGE}** |`,
              `| Threshold | min_score_to_flag=${cfg.min_score_to_flag} |`,
              ``,
              `## 📊 Signal Analysis`,
              ``,
              `| Signal | Penalty | Reason | Evidence |`,
              `|--------|---------|--------|----------|`,
              evidenceRows || "| — | — | No signals triggered | — |",
              ``,
              `---`,
              `*Generated by [Discussion Spam Detector](${context.serverUrl}/${owner}/${repo}/actions/runs/${context.runId}). Review the discussion and take appropriate action if warranted.*`,
              `<!-- dsd:discussion=#${discNum} -->`,
            ];

            const issueBody = lines.join("\n");

            core.setOutput("action", "create");
            core.setOutput("issue_title", issueTitle);
            core.setOutput("issue_body", issueBody);
safe-outputs:
  create-issue:
    max: 1
    labels: [spam, moderation, discussion-quality]
  threat-detection: false
timeout-minutes: 10
strict: true
---

# Discussion Spam Detector

You are a lightweight moderation assistant. A precompute job has already scored a newly created GitHub Discussion and determined it warrants a moderation issue.

## Your Task

Create **exactly one** GitHub issue using the precomputed data below. Do not query GitHub for additional information; just emit the issue.

## Precomputed Data

- **Action**: `${{ needs.precompute.outputs.action }}`
- **Issue Title**: `${{ needs.precompute.outputs.issue_title }}`
- **Issue Body**: (verbatim, see below)

## Issue to Create

Use the `create-issue` safe output with:

- `title`: `${{ needs.precompute.outputs.issue_title }}`
- `body`: (verbatim content below — do not modify)

```
${{ needs.precompute.outputs.issue_body }}
```

## Important

- Use the body content **verbatim** — do not reformat, summarize, or add to it.
- Only call `create-issue` once.
- If the action is `none`, emit `noop` instead.

**Important**: You MUST call a safe-output tool. If action is `none`, call:

```json
{"noop": {"message": "No action needed: score below threshold or discussion already flagged"}}
```
