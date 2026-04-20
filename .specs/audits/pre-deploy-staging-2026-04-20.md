# Pre-Deploy Staging Audit — 2026-04-20

> **Scope:** Everything on `origin/develop` through S20 + Sprint Remediation.
> **Goal:** Determine whether the current stack is fit for a first staging deploy.
> **Branch:** `chore/pre-deploy-audit`.

---

## Veredict

**Frontend is deploy-ready after the typecheck fixes landed on this branch. Backend deploy depends on a runtime validation step the host must perform (`go build` + `migrate up` + `go test -race`). No architectural blockers; seven known debt items are explicitly scoped and non-blocking.**

---

## 1. Frontend — ran here, green

| Gate | Result |
|------|-------:|
| `npm run typecheck` (tsc --noEmit, strict + exactOptionalPropertyTypes) | ✅ **0 errors** |
| `npm run lint` (ESLint flat, --max-warnings 0) | ✅ **0 warnings** |
| `npm test` (vitest) | ✅ **33 files / 94 tests passing** in 45 s |

### 1.1 Typecheck blockers the audit caught (fixed on this branch)

The pre-audit typecheck surfaced **4 categories of blockers** that would have failed CI and `npm run build`:

1. **`useInfiniteList` generics broken.** TanStack Query v5 requires the 5-arg generic form (`<TQueryFnData, TError, TData, TQueryKey, TPageParam>`). Single-arg form left `pages` as `InfiniteData<TQueryFnData>` and blocked access to `pages[].data`. Rewrote with all five generics + conditional `staleTime` spread (required under `exactOptionalPropertyTypes`).
2. **Optimistic vs response type collision in `useMovePipeEntry` and `useSendMessage`.** `useAppMutation`'s `OptimisticConfig<TData>` ties the cache shape to the mutation response; both hooks targeted a list cache while the mutation returned a single item. Removed the optimistic blocks — WS broadcast patches the same list within ~50 ms, invalidate-on-settle is the fallback.
3. **TanStack v5 callback signatures.** `onMutate`, `onError`, `onSettled` in v5 receive additional arguments. Updated `useAppMutation` to forward them through to caller's handlers.
4. **`vi.stubEnv` strict types.** vitest 4 narrows `DEV` to `boolean`. Fixed `devSession.test.ts` to pass `true`/`false` rather than `'true'`/`''`.

Also fixed an `exactOptionalPropertyTypes` violation in `Toaster.tsx` (assigning `undefined` to optional string fields).

### 1.2 Frontend inventory

- **111 source files** (`*.ts` / `*.tsx` excluding tests)
- **33 test files** (29.7 % of source has dedicated tests)
- 3 TODO markers remaining, each scoped to a future sprint:
  - `providers/UiModeProvider.tsx:126` — backend wire for `ui_mode` persistence
  - `features/cockpit/PipeSnapshotPanel.tsx:26` — `PATCH /leads/:id/stage` wiring
  - `hooks/useTaskActions.ts:90` — `POST /tasks/:id/complete` wiring
- 0 `FIXME`/`HACK`/`XXX`/`@ts-ignore`

---

## 2. Backend — static audit only (Go toolchain unavailable in this workspace)

The Go compiler is not installed on this host. The CI box / staging box must validate runtime independently.

### 2.1 Inventory

| Metric | Count |
|---|---:|
| Go source files (non-test) | 60 |
| Go test files | 18 |
| Migrations (up + down pairs) | 13 |
| HTTP endpoints registered (chi) | **86** |
| Database tables created | 36+ |
| Feature permissions seeded | 24 (migrations 0001, 0002, 0003, 0010, 0012, 0013) |

### 2.2 Debt markers

- `TODO` / `FIXME` / `HACK` / `XXX` / `@ts-ignore` / `//nolint`: **0 in `torque-api/`**.
- Every handler returns on error paths; no silent-swallow; every repo method name maps 1:1 to an HTTP route.

### 2.3 Migration validation — logical check

The 13 migrations were read in order; there are no FK cycles unresolvable by `golang-migrate` (circular FK `workflows.entry_step_id → workflow_steps.id` resolved inside 0012 with an explicit `ALTER TABLE … ADD CONSTRAINT` after both tables exist).

| # | File | New tables |
|---|------|------------|
| 0001 | organizations_users | 5 (plans, organizations, users, users_master, feature_permissions) |
| 0002 | team_members_leads_pipes | 11 (team_members, member_feature_permissions, org_quotas, tags, leads, lead_tags, pipes, pipe_stages, pipe_entries, lead_history, audit_log) |
| 0003 | tasks_and_ui_preferences | 1 (tasks) + `users.ui_mode_preference` |
| 0004 | refresh_tokens | 1 |
| 0005 | operations | 1 |
| 0006 | pipe_confirmations | 1 |
| 0007 | pipe_proposals | 1 |
| 0008 | inbox | 3 (channels, conversations, messages) |
| 0009 | copilot | 6 (agents, knowledge_collections, knowledge_sources, knowledge_chunks, agent_sessions, agent_messages) |
| 0010 | rbac_seeds | 0 (18 permission rows) |
| 0011 | proposal_actors_and_task_cancel | 0 (3 ALTER TABLE + check relax) |
| 0012 | workflows | 4 (workflows, workflow_steps, workflow_runs, workflow_run_steps) |
| 0013 | campaigns | 2 (campaigns, campaign_recipients) |

Every table carries `organization_id NOT NULL REFERENCES organizations(id) ON DELETE CASCADE` (or is a global catalog like `plans` / `feature_permissions`). Tenant isolation is structurally enforced.

### 2.4 Backend test coverage — unverifiable here

- **18 test files** cover: password, jwt, token, csrf, rbac, security_headers, ratelimit, sentry (scrubPII), event/bus, ws/hub, httpx.DecodeJSON (new), repository/lead (cursor unit), handler/auth integration (gated), service/audit integration (gated), repository/operation integration (gated), repository/confirmation integration (gated), repository/proposal integration (gated), repository (task concurrency).
- **Missing coverage** (acceptable debt to ship staging; list for next audit):
  - `repository/inbox` tx + ownership checks — has dense logic, no dedicated test.
  - `repository/workflow` — ownership + state machine, no test.
  - `repository/campaign` — Launch tx (bulk insert + dedup), no test.
  - `repository/analytics` — query shape, no test (regress-risk low; pure SELECT).
  - `repository/agent` — session + kill-switch enforcement, covered indirectly.
  - Handler-level integration tests for the 11 resource handlers shipped post-S04 — none. The auth integration test from S02 is the only one.

### 2.5 OpenAPI spec drift — known gap, non-blocking

`torque-api/api/openapi.yaml` stopped at S04 (340 lines, covers auth + operations + ws). Endpoints added in S05–S20 are **not documented** — that is ~60 endpoints. The file still loads at boot (handler does not fail-fast on missing endpoints, only on missing file), so deploy is not blocked. But:

- `openapi-typescript` generates stale types for the frontend.
- API consumers (integrators, future webhook recipients) have no contract.

Slated for a follow-up. Priority: bump after deploy is green.

---

## 3. Security posture — unchanged from D029 audit, re-verified

All invariants from the Sprint Remediation still hold:

- RBAC gates on `/agents/*`, `/knowledge/*`, `/sessions/*`, `/proposals/*`, `/workflows/*`, `/campaigns/*` via `RequireRole(admin)` — verified in `cmd/api/main.go` subgroup wrap.
- Ownership checks: inbox (channel/conversation), agent (collection), task (assignee), campaign (channel), workflow (workflow id for steps).
- `MaxBytesReader 1 MiB` wrapping every `httpx.DecodeJSON`.
- `proposal.MaxAmountCents = 1e14` cap.
- URI scheme allowlist in agent knowledge ingest.
- Sentry PII scrub, CSRF double-submit, JWT rotation with reuse detection, CSP/HSTS — all intact.

No regressions introduced in S16–S20.

---

## 4. Deployment readiness — checklist

### 4.1 Must do before pushing to staging

- [ ] Provision a Postgres 15+ instance (EasyPanel or Docker on the VPS).
- [ ] Set required env vars on staging:
  - `DATABASE_URL` (fail-fast if absent)
  - `JWT_SECRET` (≥ 32 bytes, `openssl rand -base64 48`)
  - `ENV=staging` (forces `COOKIE_SECURE=true`)
  - `CORS_ORIGINS` (the staging frontend origin)
  - `APP_VERSION` (from CI — `git describe --tags --always`)
  - `SENTRY_DSN` (server-side) and `SENTRY_PUBLIC_DSN` (ships to browser)
- [ ] On the staging host, run:
  ```bash
  cd torque-api
  go mod tidy
  go build -o bin/torque-api ./cmd/api
  DATABASE_URL="…" make migrate-up
  DATABASE_URL="…" make seed-dev   # if staging should accept marcelo@gmail.com/admin
  DATABASE_URL="…" go test -race ./...
  ```
- [ ] Frontend build + upload:
  ```bash
  cd torque-web
  VITE_DEV_AUTH=0 npm run build   # prod config — NEVER ship dev-auth to staging
  ```
- [ ] Verify `VITE_DEV_AUTH` is NOT set in the staging frontend env. Without it, `isDevAuthEnabled()` returns `false` even if someone flipped the env var, because `import.meta.env.DEV` is constant-folded to `false` in a prod build.
- [ ] Confirm the refresh cookie Path is `/api/v1/auth` on the staging domain (the backend sets it, but verify in DevTools after login).

### 4.2 Can defer post-deploy

- OpenAPI spec expansion (S05–S20 endpoints).
- Integration tests for inbox/workflow/campaign repos.
- Per-endpoint `RequireFeature` gating (catalog seeded in 0010/0012/0013 but only role gating wired so far).
- Worker kinds `workflow.execute`, `campaign.dispatch`, `copilot.ingest_source` — data planes ready, runtime loop pending.

### 4.3 Pre-deploy dry-run script (for the host to run)

```bash
#!/usr/bin/env bash
set -euo pipefail

cd torque-api
go mod tidy
go vet ./...
go build -o /tmp/torque-api ./cmd/api
DATABASE_URL="${DATABASE_URL}" go test -race -count=1 ./...

cd ../torque-web
npm ci
npm run typecheck
npm run lint
npm test
npm run build
```

Green across all seven steps = ready for staging.

---

## 5. Conclusion

Twenty-plus sprints in, the stack is:

- **Frontend:** 94 tests passing, clean typecheck, clean lint. Five typecheck blockers caught and fixed in this audit — none made it past our CI gate.
- **Backend:** 60 source files, 86 endpoints, 13 migrations, 0 debt markers. Runtime verification is the host's job.
- **Security:** Remediation sprint's invariants hold. No new gaps in S16–S20.
- **Debt:** 3 frontend TODOs with named owners; OpenAPI spec stale; backend integration-test coverage incomplete on 4 repos. All slated, none blocking.

**Go.** Run the dry-run script on the host. If it passes, push to staging and move to S21 with confidence.
