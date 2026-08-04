---
name: github-lct45-fork
description: Ensure GitHub context, issues, and pull request work targets the lct45/grafana fork. Use for any GitHub-related task, including issues, PRs, commits, searches, or repository context.
---

# GitHub fork targeting: lct45/grafana

## Instructions

- Always use `lct45/grafana` for GitHub queries and operations (issues, PRs, commits, code search, and repo context).
- For mutating `gh` CLI commands (for example `pr create`, `pr edit`, `pr merge`, issue/release writes), always pass `--repo lct45/grafana`.
- For read-only/sync operations, upstream `grafana/grafana` may be used when explicitly needed (for example comparing, listing, syncing, or fetching context), but never as the target for write actions.
- For GitHub MCP tools, set owner to `lct45` and repo to `grafana` (or include this in the tool query when required).
- For PR creation and PR updates, explicitly state the target repo in user-facing output, e.g. `Target repo: lct45/grafana`, and include the PR URL.
- Never create, edit, or merge PRs/issues/releases against `grafana/grafana` or `fieldsphere/grafana` unless the user explicitly requests that repo.
- If the target repo is ambiguous, check `git remote -v` and prefer `origin` when it is `lct45/grafana`; otherwise still use `lct45/grafana`.
