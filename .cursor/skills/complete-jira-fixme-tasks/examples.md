# FIXME Task Examples

Walkthroughs for two real LCF backlog tickets. Use these as templates for branch naming, tests, commits, and PRs.

---

## Example 1: LCF-5 — Frontend bug

**Ticket**: [LCF-5](https://fe-anysphere-demo.atlassian.net/browse/LCF-5) — Fix geomap panel suggestion preview legend disable

**Source**: `public/app/plugins/panel/geomap/suggestions.ts` (~line 35)

```ts
// FIXME: this doesn't work. I want to disable legends in the preview.
s.options?.layers?.forEach((layer) => {
  layer.config = layer.config || {};
  layer.config.showLegend = false;
});
```

**Problem**: Setting `showLegend = false` on suggestion preview layers does not hide legends in the geomap panel preview.

**Suggested fix**: Find the correct option path for legend visibility and apply it in the suggestion transformer. Verify in the panel suggestion UI.

### Workflow

| Step | Action |
|------|--------|
| Branch | `lcf-5-geomap-preview-legend` |
| Investigate | Read `suggestions.ts`, geomap layer config types, and how legends render in previews |
| Implement | Use the correct legend-disable property; remove `FIXME` |
| Test | `yarn jest --no-watch public/app/plugins/panel/geomap/suggestions.test.ts` |
| Lint | `yarn lint public/app/plugins/panel/geomap/suggestions.ts` |
| Commit | `LCF-5: disable legends in geomap suggestion preview` |
| PR title | `LCF-5: Fix geomap panel suggestion preview legend disable` |
| PR repo | `lct45/grafana` |

**Commit message example:**
```
LCF-5: disable legends in geomap suggestion preview

Use the correct layer option to hide legends in panel suggestions.
Removes FIXME in suggestions.ts.
```

**PR body snippet:**
```markdown
## Summary
- Fix legend visibility in geomap panel suggestion previews

## Jira
https://fe-anysphere-demo.atlassian.net/browse/LCF-5

## Test plan
- [x] yarn jest --no-watch public/app/plugins/panel/geomap/suggestions.test.ts
```

---

## Example 2: LCF-2 — Backend task

**Ticket**: [LCF-2](https://fe-anysphere-demo.atlassian.net/browse/LCF-2) — Handle not-found properly in support bundles store

**Source**: `pkg/services/supportbundles/supportbundlesimpl/store.go` (~line 117)

```go
if !ok {
    // FIXME: handle not found
    return nil, errors.New("not found")
}
```

**Problem**: Missing support bundles return a generic `errors.New("not found")` instead of a typed not-found error callers can distinguish.

**Suggested fix**: Return a proper not-found error (e.g. `ErrNotFound` or `apierrors.NewNotFound`) and update callers/tests.

### Workflow

| Step | Action |
|------|--------|
| Branch | `lcf-2-support-bundle-not-found` |
| Investigate | Read `store.go`, callers, and existing not-found patterns in `pkg/` |
| Implement | Add typed error; update callers; remove `FIXME` |
| Test | `go test -run TestSupportBundle ./pkg/services/supportbundles/...` (adjust test name to match) |
| Lint | `make lint-go` if needed |
| Commit | `LCF-2: return typed not-found from support bundles store` |
| PR title | `LCF-2: Handle not-found properly in support bundles store` |
| PR repo | `lct45/grafana` |

**Commit message example:**
```
LCF-2: return typed not-found from support bundles store

Replace generic errors.New with a typed not-found error so callers
can distinguish missing bundles from other failures.
```

**PR body snippet:**
```markdown
## Summary
- Return typed not-found error from support bundles store
- Update callers and tests

## Jira
https://fe-anysphere-demo.atlassian.net/browse/LCF-2

## Test plan
- [x] go test ./pkg/services/supportbundles/...
```

---

## Quick reference

| Ticket type | Typical test command | PR scope |
|-------------|---------------------|----------|
| Frontend (`public/app/`) | `yarn jest --no-watch <test-file>` | Frontend only |
| Backend (`pkg/`) | `go test -run <Test> ./pkg/...` | Backend only |
| Bug | Reproduce broken behavior in test if possible | Same as touched area |
| Task | Match ticket's suggested test coverage | Same as touched area |
