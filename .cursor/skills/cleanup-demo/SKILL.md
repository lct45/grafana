---
name: cleanup-demo
description: Resets the Grafana LCF demo environment by closing all open PRs on lct45/grafana, deleting non-main branches, clearing comments on grafana-labeled Jira tickets, and moving every LCF-1 child ticket back to To Do. Use only when the user explicitly invokes cleanup-demo or asks to reset/clean up the demo env.
disable-model-invocation: true
---

# Cleanup Demo

Reset this repo's demo environment so the next run starts clean.

**Invocation only** — do not run from ambient context. Proceed once the user attaches or names this skill (or explicitly asks to clean up / reset the demo).

## Targets

| System | Target |
| ------ | ------ |
| GitHub | `lct45/grafana` (always pass `--repo lct45/grafana`) |
| Git branches | Delete every local and remote branch except `main` |
| Jira site | `https://fe-anysphere-demo.atlassian.net` |
| Jira project | `LCF` |
| Jira status reset | Issues with `parent = LCF-1` (not the epic itself) → `To Do` |
| Jira comment wipe | Issues with `labels = grafana` (all comments removed) |
| Target status | `To Do` |

Resolve `cloudId` via `getAccessibleAtlassianResources` if needed.

## Checklist

```
Demo cleanup:
- [ ] Step 1: List open PRs
- [ ] Step 2: Close all open PRs
- [ ] Step 3: Delete all non-main branches (remote + local)
- [ ] Step 4: List grafana-labeled Jira issues and wipe their comments
- [ ] Step 5: List LCF-1 children not in To Do
- [ ] Step 6: Transition each to To Do
- [ ] Step 7: Report summary
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

## Step 3 — Delete all non-main branches

Work against `lct45/grafana` only. Never delete `main`.

### 3a — Remote branches on `lct45/grafana`

Use `git` for remote deletes (not `gh api -X DELETE`). Mutating `gh` commands are blocked by `.cursor/hooks/enforce-fieldsphere-gh.sh` unless they pass `--repo lct45/grafana`, and `gh api` does not support that flag.

Abort unless `origin` is `lct45/grafana`, then delete remote branches:

```bash
ORIGIN_URL="$(git remote get-url origin)"
case "$ORIGIN_URL" in
  *github.com[:/]lct45/grafana.git|*github.com[:/]lct45/grafana)
    ;;
  *)
    echo "ABORT: origin is '$ORIGIN_URL', expected lct45/grafana" >&2
    exit 1
    ;;
esac

git fetch origin --prune
git ls-remote --heads origin \
  | awk '{print $2}' \
  | sed 's#refs/heads/##' \
  | grep -vx 'main' \
  | while read -r b; do
      git push origin --delete "$b" \
        && echo "deleted remote: $b" \
        || echo "FAILED remote: $b"
    done
```

If a branch is protected or the default and cannot be deleted, record it as failed and continue.

### 3b — Local branches

From the repo root (or active worktree), ensure you are on `main` first:

```bash
git fetch origin --prune
git checkout main
git branch --format='%(refname:short)' \
  | grep -vx 'main' \
  | while read -r b; do
      git branch -D "$b" && echo "deleted local: $b" || echo "FAILED local: $b"
    done
```

Do **not** delete branches on other remotes (`grafana/grafana`, `fieldsphere/grafana`) unless the user explicitly asks.

## Step 4 — Remove all comments on grafana-labeled Jira tickets

### 4a — Find issues

```
searchJiraIssuesUsingJql(
  cloudId,
  jql="project = LCF AND labels = grafana ORDER BY key ASC",
  fields=["summary", "status", "labels"],
  maxResults=100
)
```

Paginate with `nextPageToken` until all issues are collected. If none, skip the rest of Step 4.

### 4b — List and delete comments

For **each** issue key:

1. `getJiraIssue(cloudId, issueIdOrKey, fields=["comment"])` — comments are in `fields.comment.comments` (each has an `id`).
2. For every comment `id`, delete it via Jira Cloud REST (Atlassian MCP has no delete-comment tool):

```bash
# Prefer site URL + authenticated curl/API token available in the environment.
# cloudId may also be used with api.atlassian.com/ex/jira/{cloudId}/...
curl -sS -X DELETE \
  -u "${JIRA_EMAIL}:${JIRA_API_TOKEN}" \
  -H "Accept: application/json" \
  "https://fe-anysphere-demo.atlassian.net/rest/api/3/issue/${ISSUE_KEY}/comment/${COMMENT_ID}"
```

If `JIRA_EMAIL` / `JIRA_API_TOKEN` (or equivalent Atlassian auth) are unavailable, try any other authenticated Jira CLI already configured for this site. If deletion still cannot be authenticated, **stop Step 4**, report the blocker, and continue with Steps 5–7 (do not invent a workaround that leaves partial silent failure).

Record counts: issues scanned, comments deleted, failures.

## Step 5 — List LCF-1 children not in To Do

```
searchJiraIssuesUsingJql(
  cloudId,
  jql="project = LCF AND parent = LCF-1 AND status != \"To Do\" ORDER BY key ASC",
  fields=["summary", "status", "issuetype"],
  maxResults=100
)
```

Paginate with `nextPageToken` until all issues are collected. Skip `LCF-1` itself (epic is not a child).

If none, skip Step 6.

## Step 6 — Transition each issue to To Do

For **each** issue key:

1. `getTransitionsForJiraIssue(cloudId, issueIdOrKey)`
2. Pick the transition whose target status name is `To Do` (case-insensitive match on status/name; prefer exact `"To Do"`).
3. `transitionJiraIssue(cloudId, issueIdOrKey, transition={id})`

If no transition to `To Do` is available, record the key as failed and continue with the rest. Do not invent status names or force unrelated transitions.

## Step 7 — Report summary

Return a short report:

```markdown
## Demo cleanup complete

**PRs closed** (`lct45/grafana`): N
- #… — title (or "none open")

**Branches deleted** (`lct45/grafana`, kept `main`): N remote, M local
- remote: … (or "none")
- local: … (or "none")

**Jira comments removed** (`labels = grafana`): N comments across M issues
- LCF-… — K comments (or "none found")

**Jira reset to To Do** (parent `LCF-1`): N
- LCF-… — summary (or "already all To Do")

**Failed / skipped**: …
```

## Guardrails

- Never delete the `main` branch (local or remote).
- Do not delete branches or close PRs on `grafana/grafana` or `fieldsphere/grafana` unless the user explicitly asks.
- Do not transition issues outside `parent = LCF-1`.
- Do not wipe comments on issues that lack the `grafana` label.
- Do not create, reopen, or merge PRs as part of this skill.
- Writes are intentional when this skill is invoked; still execute list steps first so the summary is accurate.
