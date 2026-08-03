---
name: cleanup-demo
description: Resets the Grafana LCF demo environment by closing all open PRs on lct45/grafana and moving every LCF-1 child Jira ticket back to To Do. Use only when the user explicitly invokes cleanup-demo or asks to reset/clean up the demo env.
disable-model-invocation: true
---

# Cleanup Demo

Reset this repo's demo environment so the next run starts clean.

**Invocation only** — do not run from ambient context. Proceed once the user attaches or names this skill (or explicitly asks to clean up / reset the demo).

## Targets

| System | Target |
| ------ | ------ |
| GitHub | `lct45/grafana` (always pass `--repo lct45/grafana`) |
| Jira site | `https://fe-anysphere-demo.atlassian.net` |
| Jira project | `LCF` |
| Jira scope | Issues with `parent = LCF-1` (not the epic itself) |
| Target status | `To Do` |

Resolve `cloudId` via `getAccessibleAtlassianResources` if needed.

## Checklist

```
Demo cleanup:
- [ ] Step 1: List open PRs
- [ ] Step 2: Close all open PRs
- [ ] Step 3: List LCF-1 children not in To Do
- [ ] Step 4: Transition each to To Do
- [ ] Step 5: Report summary
```

## Step 1 — List open PRs

```bash
gh pr list --repo lct45/grafana --state open --limit 100 \
  --json number,title,url,isDraft
```

If none, skip Step 2 and note that in the summary.

## Step 2 — Close all open PRs

Close every open PR (including drafts):

```bash
gh pr list --repo lct45/grafana --state open --limit 100 --json number \
  --jq '.[].number' | while read -r n; do
  gh pr close "$n" --repo lct45/grafana --comment "Demo cleanup: closing PR to reset env."
done
```

Do **not** close PRs on `grafana/grafana` or `fieldsphere/grafana` unless the user explicitly asks.

## Step 3 — List LCF-1 children not in To Do

```
searchJiraIssuesUsingJql(
  cloudId,
  jql="project = LCF AND parent = LCF-1 AND status != \"To Do\" ORDER BY key ASC",
  fields=["summary", "status", "issuetype"],
  maxResults=100
)
```

Paginate with `nextPageToken` until all issues are collected. Skip `LCF-1` itself (epic is not a child).

If none, skip Step 4.

## Step 4 — Transition each issue to To Do

For **each** issue key:

1. `getTransitionsForJiraIssue(cloudId, issueIdOrKey)`
2. Pick the transition whose target status name is `To Do` (case-insensitive match on status/name; prefer exact `"To Do"`).
3. `transitionJiraIssue(cloudId, issueIdOrKey, transition={id})`

If no transition to `To Do` is available, record the key as failed and continue with the rest. Do not invent status names or force unrelated transitions.

## Step 5 — Report summary

Return a short report:

```markdown
## Demo cleanup complete

**PRs closed** (`lct45/grafana`): N
- #… — title (or "none open")

**Jira reset to To Do** (parent `LCF-1`): N
- LCF-… — summary (or "already all To Do")

**Failed / skipped**: …
```

## Guardrails

- Do not modify branches, stashes, or local git state.
- Do not transition issues outside `parent = LCF-1`.
- Do not create, reopen, or merge PRs as part of this skill.
- Writes are intentional when this skill is invoked; still execute Steps 1 and 3 first so the summary is accurate.
