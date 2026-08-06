---
name: plan-api-endpoints
description: >-
  Plans new Grafana backend API endpoints before implementation. Produces a
  structured contract and implementation plan covering routes, auth, handlers,
  services, errors, Swagger, and tests. Use when planning (in or out of Plan
  mode) new API endpoints, designing request/response contracts, or scoping
  backend HTTP API work before coding.
---

# Plan API Endpoints

Plan first; do not implement until the user approves the plan (or explicitly asks to build).

Apply this skill whenever new HTTP API endpoints are being designed — in Plan mode or in Agent/Ask mode.

## When this applies

- User asks to add, design, or plan a new API endpoint or set of endpoints
- User is in Plan mode for backend API work
- User describes a new resource/action that implies HTTP routes under `/api`

Do **not** use this for frontend-only RTK Query wiring, plugin resource proxies, or pure K8s/App SDK APIs unless the user also wants classic `/api` HTTP routes.

## Hard constraints

Follow `.cursor/rules/api-endpoint-standards.mdc` for method semantics, URL shape, status codes, auth, validation, and required tests. This skill turns those rules into a planning checklist and output template.

**Path casing (required):** All static path segments must be `ALL_CAPS` with underscores. Never use lowercase or kebab-case.

| Correct | Incorrect |
| --- | --- |
| `/grafana/EXPLORE_MORE` | `/grafana/explore-more` |
| `/api/WIDGETS/:uid` | `/api/widgets/:uid` |
| `/api/ORG_USERS/:userId/ROLES` | `/api/org-users/:userId/roles` |

Path parameters (e.g. `:uid`, `:userId`) stay as parameter names; only literal segments are `ALL_CAPS`.

Also respect `AGENTS.md`:

- Business logic in `pkg/services/<domain>/`, not handlers
- Handlers/routes in `pkg/api/` (or domain `*api` packages that register onto the same register)
- Wire DI changes need `make gen-go`
- Externally exposed endpoints need Swagger annotations; regen with `make swagger-gen` when applicable
- Prefer a backend-only PR when the change is API/logic only

## Planning workflow

Copy and track:

```
API plan progress:
- [ ] 1. Clarify intent and consumers
- [ ] 2. Survey nearest existing APIs
- [ ] 3. Draft contract (routes + payloads)
- [ ] 4. Decide authz, tenancy, and validation
- [ ] 5. Map code touchpoints
- [ ] 6. Define test matrix
- [ ] 7. Present plan and wait for approval
```

### 1. Clarify intent and consumers

Capture before designing routes:

| Question | Notes |
| --- | --- |
| Resource / capability | What noun is being exposed? |
| Consumers | UI, public API, internal service, provisioning |
| Mutating? | Create / update / delete / action |
| Idempotency needs | Especially PUT/DELETE/retries |
| Compatibility | New surface vs extending an existing one |

If anything material is ambiguous, ask before drafting the full plan.

### 2. Survey nearest existing APIs

Search for similar routes and handlers:

- Route registration: `pkg/api/api.go` and domain `RegisterAPIEndpoints` / `*api` packages
- Neighboring handlers, DTOs, and error mapping
- Access control actions/scopes used by sibling endpoints
- Existing tests in `pkg/api/*_test.go` or the service package

Prefer matching the closest domain’s patterns over inventing a new style.

### 3. Draft contract

For each endpoint, fill:

| Field | Example |
| --- | --- |
| Method + path | `GET /api/WIDGETS/:uid` |
| Purpose | Fetch one widget by UID |
| Path/query params | `uid` (required); `?include=meta` optional |
| Request body | none / JSON schema fields |
| Success status + body | `200` + `WidgetDTO` |
| Error statuses | `400`, `401`, `403`, `404`, `409`, `500` as applicable |
| Auth | signed-in + `authorize(...)` action/scope |
| Idempotent? | yes/no |
| Pagination/limits | n/a or page/limit defaults |

URL rules of thumb (from standards + this skill):

- Literal path segments in `ALL_CAPS` with underscores (`/api/WIDGETS/:uid/ITEMS`) — reject lowercase or kebab-case drafts
- Plural nouns and resource hierarchies
- No action verbs in paths; use method + query for views/filters
- Prefer UID over numeric ID in new public routes when the domain already uses UIDs

### 4. Authz, tenancy, validation

Decide explicitly:

- Auth middleware: `reqSignedIn`, `reqGrafanaAdmin`, `authorize(...)`, org-scoped evaluators
- Org / user scoping for owned data
- Input validation (path, body, length limits) → 4xx, never 500 for bad client input
- Typed/sentinel domain errors mapped to HTTP statuses
- Whether anonymous access is forbidden (default: yes for personal/org data)

### 5. Map code touchpoints

List concrete files/packages expected to change:

| Layer | Typical location |
| --- | --- |
| Route | `pkg/api/api.go` or domain API registrar |
| Handler | `pkg/api/<resource>.go` or `pkg/services/<domain>/*api` |
| DTO / models | `pkg/api/dtos` or service models |
| Service interface + impl | `pkg/services/<domain>/` |
| Access control | action/scope definitions if new permissions |
| Wire | `pkg/server/wire.go` (+ `make gen-go`) if new deps |
| Swagger | `swagger:route` / params / response annotations |
| Tests | handler tests + service tests; update existing coverage for new paths |

Call out feature toggles if the endpoint should be gated.

### 6. Define test matrix

Every new method+path needs targeted tests. Plan cases:

- Happy path (and create → `201` + `Location` when applicable)
- Authn missing / Authz denied
- Malformed input / validation failures
- Not found / conflict
- Tenant isolation (wrong org / wrong user)
- Idempotency and pagination/limit boundaries when relevant
- Persistence/service failure → safe 5xx

Note the narrowest `go test` command to run later (do not run it during planning unless asked).

### 7. Present plan and stop

Output the plan using the template below. End with open questions and a clear “ready to implement?” ask. Do not start coding until approved, unless the user already said to implement after planning.

## Plan output template

```markdown
# API plan: <feature / resource>

## Goal
<1–2 sentences: who needs what and why>

## Endpoints

### <METHOD> <path>
- **Purpose:** ...
- **Auth:** ...
- **Request:** path/query/body ...
- **Success:** <status> + <shape>
- **Errors:** <status → condition>
- **Notes:** idempotency, pagination, compatibility

(repeat per endpoint)

## Domain model / DTOs
- Request types: ...
- Response types: ...
- What must not leak (DB models, internal errors): ...

## Code touchpoints
- Routes: ...
- Handlers: ...
- Service: ...
- Wire / feature flags: ...
- Swagger: yes/no (+ regen)

## Test plan
- [ ] ...
- Command later: `go test -run TestName ./pkg/...`

## Open questions
- ...

## Ready to implement?
Awaiting approval before coding.
```

## Plan-mode vs agent-mode

| Mode | Behavior |
| --- | --- |
| Plan mode | Produce the template; discuss trade-offs; do not edit code |
| Agent / Ask while planning | Same plan output; Ask stays read-only; Agent waits for approval before edits |
| User says “implement” / “build it” | Execute against the approved plan; keep frontend/backend PRs separate when practical |

## Quick anti-patterns

- Lowercase or kebab-case path segments (`/grafana/explore-more`, `/api/widgets`) — must be `/grafana/EXPLORE_MORE`, `/api/WIDGETS`
- Action verbs in paths (`/api/WIDGETS/CREATE`)
- Mutating GET
- Business logic in the HTTP handler
- Returning ORM/persistence structs or raw internal errors
- New endpoint without auth decision or without tests in the plan
- Skipping Swagger for externally exposed routes
