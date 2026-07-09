---
name: complete-jira-fixme-tasks
description: Implements FIXME fixes from LCF Jira tickets in the Grafana repo, validates with targeted tests, commits, and opens PRs to lct45/grafana. Use when the user asks to work on a Jira ticket (e.g. LCF-5), complete a FIXME task, or pick up backlog work from the LCF project.
---

# Complete Jira FIXME Tasks

End-to-end workflow for LCF backlog tickets derived from `FIXME` comments in the Grafana codebase.

## Jira targeting

- **Site**: `https://fe-anysphere-demo.atlassian.net`
- **Cloud ID**: resolve via `getAccessibleAtlassianResources` if needed
- **Project**: `LCF` (parent epic `LCF-1`, "grafana backlog")
- **Default JQL** (open FIXME work): `project = LCF AND parent = LCF-1 AND status != Done ORDER BY created ASC`
- **Ticket URL pattern**: `https://fe-anysphere-demo.atlassian.net/browse/LCF-XX`

## Ticket description format

LCF FIXME tickets use this structure — parse each section before coding:


| Section            | Use                                               |
| ------------------ | ------------------------------------------------- |
| `## Source`        | File path(s), line hints, and the `FIXME` snippet |
| `## Problem`       | Current broken behavior                           |
| `## Suggested fix` | Primary acceptance criteria                       |
| `## Why workable`  | Scope and risk notes                              |




## Git and PR targeting

When this skill is active, prefer `lct45/grafana` over other fork skills.

- Remote: `git@github.com:lct45/grafana.git`
- All `gh` write commands: `--repo lct45/grafana`
- State target repo in user-facing output: `Target repo: lct45/grafana`



## Workflow

Copy this checklist and track progress:

```
Task progress:
- [ ] Step 1: Select ticket
- [ ] Step 2: Parse ticket
- [ ] Step 3: Verify FIXME in codebase
- [ ] Step 4: Create branch
- [ ] Step 5: Implement fix
- [ ] Step 6: Validate with targeted tests
- [ ] Step 7: Commit (after user approval)
- [ ] Step 8: Push and open PR (after user approval)
- [ ] Step 9: Comment on Jira with PR link
```



### Step 1 — Select ticket

**User named a key** (e.g. `LCF-5`):

```
getJiraIssue(cloudId, issueIdOrKey="LCF-5", fields=["summary","description","status","issuetype"])
```

**No key given** — list open work and let the user pick (or take the oldest):

```
searchJiraIssuesUsingJql(
  cloudId,
  jql="project = LCF AND parent = LCF-1 AND status != Done ORDER BY created ASC",
  fields=["summary","description","status","issuetype","priority"],
  maxResults=50
)
```



### Step 2 — Parse ticket

Extract from the description:

1. Source file path(s) and approximate line
2. Exact `FIXME` comment text
3. Problem statement
4. Suggested fix (treat as acceptance criteria)



### Step 3 — Verify before coding

1. Open cited file(s) and confirm the `FIXME` exists and matches the ticket
2. Read surrounding code and callers to understand impact
3. **Stop and report** if:
  - `FIXME` is already gone → suggest closing or updating the ticket
  - Code diverged materially from the ticket → comment on Jira with findings first



### Step 4 — Create branch

Pattern: `lcf-XX-short-description`

Examples: `lcf-5-geomap-preview-legend`, `lcf-2-support-bundle-not-found`

```bash
git checkout -b lcf-XX-short-description
```



### Step 5 — Implement

- Minimal scoped change; follow patterns in the cited package
- Add or update tests when the ticket or `AGENTS.md` calls for it
- Remove the `FIXME` comment when resolved
- Do not add unrelated refactors

**Path guidance** (see `AGENTS.md`):


| Area            | Location      |
| --------------- | ------------- |
| Backend         | `pkg/`        |
| Frontend        | `public/app/` |
| Shared packages | `packages/`   |


Prefer separate PRs for backend vs frontend (different deploy cadences).

### Step 6 — Validate

Run targeted checks only — not full suite unless the change is broad.

**Go** (backend):

```bash
go test -run <RelevantTest> ./pkg/services/<package>/...
make lint-go   # when touching pkg/
```

**TypeScript** (frontend):

```bash
yarn jest --no-watch public/app/path/to/file.test.ts
yarn lint public/app/path/to/file.ts
yarn typecheck   # when types may be affected
```

Summarize what ran and the result before proposing commit.

### Step 7 — Commit

Only commit when the user explicitly asks.

**Message format:**

```
LCF-XX: <imperative summary>

<1-2 sentences on what changed and why>
```

Show a summary of staged changes and the proposed message; wait for approval.

### Step 8 — Push and open PR

**Never push without explicit human approval** (see `AGENTS.md` review gate).

```bash
git push -u origin HEAD
```

```bash
gh pr create --repo lct45/grafana \
  --title "LCF-XX: <summary>" \
  --body "$(cat <<'EOF'
## Summary
- <what changed>

## Jira
https://fe-anysphere-demo.atlassian.net/browse/LCF-XX

## Test plan
- [ ] <targeted test command and result>

EOF
)"
```

Return the PR URL and state `Target repo: lct45/grafana`.

## Guardrails

Stop and escalate instead of silently expanding scope when:


| Condition                                   | Action                                        |
| ------------------------------------------- | --------------------------------------------- |
| `FIXME` already resolved                    | Update or close ticket; do not code           |
| Fix needs feature-flag rollout or migration | Report blocker; do not implement full rollout |
| Ticket spans backend + frontend             | Split into separate PRs or ask user           |
| Unclear acceptance criteria                 | Ask user or comment on Jira before coding     |
| Push or PR requested                        | Wait for explicit human approval              |




## MCP tools reference


| Tool                              | When                        |
| --------------------------------- | --------------------------- |
| `getAccessibleAtlassianResources` | Resolve `cloudId`           |
| `getJiraIssue`                    | Fetch a single ticket       |
| `searchJiraIssuesUsingJql`        | List open FIXME backlog     |
| `addCommentToJiraIssue`           | Link PR and test results    |
| `getTransitionsForJiraIssue`      | Before transitioning status |
| `transitionJiraIssue`             | Only when user requests     |




## Examples

For full walkthroughs of LCF-5 (frontend bug) and LCF-2 (backend task), see [examples.md](examples.md).