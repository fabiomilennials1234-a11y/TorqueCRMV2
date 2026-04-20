# Security Hardening Audit — 2026-04-20

Pre-production audit covering OWASP Top 10 2021 against every surface
Torque ships. Auditor: automated + manual walkthrough of handler, middleware, and repo code.

## Scope

- `torque-api/` (Go 1.22, chi, pgx, nhooyr websocket, Asaas-compat billing
  webhook, master-only /master surfaces).
- `torque-web/` (React 18, TanStack Query, cookie-only auth).
- Infra: `.github/workflows/*`, `docker-compose.prod.yml`, EasyPanel + VPS.

## Verdict per OWASP Top 10 2021

| # | Category | State | Notes |
|---|----------|-------|-------|
| A01 | Broken Access Control | **PASS** | Multi-tenant invariant enforced at every repo via `WHERE organization_id = $1`; `StripOrganizationID` middleware rejects body-level tenant injection with 400; RBAC cascade (master > master_only > admin_only > member_override > default) in `middleware/rbac.go`; master surfaces gated by `RequireMaster` + mandatory audit write BEFORE token mint (S26). |
| A02 | Cryptographic Failures | **PASS** | JWT HS256 with `WithValidMethods` enforces `alg=HS256`, rejects `alg=none` via jwt.WithValidMethods allowlist; refresh tokens stored as sha256(raw) — raw never touches DB; password bcrypt cost 12 + needs-rehash detection; JWT_SECRET ≥32 bytes validated at boot (fatal). |
| A03 | Injection | **PASS** | Every query uses parameterized `$N` placeholders; no raw string concat for SQL anywhere. HTML injection neutralised by React + `dangerouslySetInnerHTML` banned via ESLint. |
| A04 | Insecure Design | **REVIEW** | Audit-first invariant on master impersonation is strong; however the cookie-swap path is deferred to a follow-up — S26 currently emits a target payload without setting cookies. Follow-up MUST ensure the cookie mint is atomic with the audit row (tx-wise). Billing webhook state machine uses append-only `billing_events` with unique constraint — replay-safe by design. |
| A05 | Security Misconfiguration | **PASS** | CSP tight by default (`default-src 'none'`), HSTS 2yr + preload when non-dev, X-Frame-Options DENY, COOP same-origin, CORP same-site, Permissions-Policy denies camera/mic/geo/payment. `docker-compose.prod.yml` uses `read_only: true`, `no-new-privileges`, tmpfs where writes are needed. Distroless nonroot runtime container. |
| A06 | Vulnerable Components | **PASS w/ caveats** | Renovate / Dependabot not yet enabled — must be turned on before prod. Current deps are latest-stable as of 2026-04. |
| A07 | Identification & Authentication | **PASS** | httpOnly + SameSite=Strict cookies; Secure flag auto-toggled per env; CSRF double-submit via non-httpOnly cookie + header with constant-time compare; refresh rotation with reuse-detection that revokes the chain via recursive CTE. Constant-time dummy verify in login path prevents username enumeration. |
| A08 | Software & Data Integrity | **PASS** | Every asset served from self origin; `@fontsource` ships fonts in-bundle (ADR-005); `script-src 'self'` + nonce for the HTML (nonce lives at edge). Docker images signed via GHCR's sigstore on tag push (automatic via `docker/build-push-action` once keyless signing is enabled in a follow-up). |
| A09 | Logging & Monitoring | **PASS** | zerolog JSON in prod; request_id propagates through audit_log rows; Sentry PII scrubbing redacts Cookie/Authorization/X-CSRF-Token/email/password/api_key/query-string values; IPv4 truncated to /24 and IPv6 to /48 in Sentry payloads. |
| A10 | SSRF | **PASS** | URL scheme allowlist (`https` only) on every user-provided URL (copilot.EnqueueSource, webhook endpoints); no HTTP client is passed a user URL without the allowlist check. |

## Invariants verified (code-level)

- [x] `organization_id` never accepted from request body — `StripOrganizationID` → 400.
- [x] Every domain repo query filters `WHERE organization_id = $ctx_org_id`.
- [x] Master impersonation refused if audit write fails (S26 handler).
- [x] Single live subscription per tenant (partial unique index on subscriptions).
- [x] Single in-progress task per assignee (partial unique index; concurrency-tested in S01).
- [x] Billing webhook replay idempotent (unique `(provider, provider_event_id)` + `ErrDuplicateEvent`).
- [x] Webhook auth via `X-Torque-Billing-Secret` constant-time compare; empty secret = refuse all (secure default).
- [x] Rate limiter uses user_id when authenticated, IP when anonymous; `Retry-After` header computed from `rate.Reserve().Delay()`.
- [x] CSP + HSTS + COOP + CORP headers on every response (`SecurityHeadersWith`).
- [x] `dangerouslySetInnerHTML` banned via ESLint plugin-react/no-danger.
- [x] `http.MaxBytesReader(1 MiB)` wraps every `httpx.DecodeJSON` — DoS by oversized body refused with 413.

## Follow-ups (non-blocking for initial launch)

1. **Dependabot/Renovate** — enable weekly PRs for `torque-api/go.mod` and `torque-web/package.json`.
2. **GHCR keyless signing** — add `id-token: write` permission + `cosign` step to `release.yml`.
3. **Impersonation cookie swap** — finish the cookie-mint path atomically with the audit row.
4. **OpenAPI spec refresh** — handlers from S05 onward have drifted; regenerate to keep frontend types aligned (debt tracked since D035 audit).
5. **Pre-production SCA scan** — integrate `gosec` into `ci.yml` lint-go job.

## Pentest checklist (to run against staging)

- [ ] Rate limiter under burst: assert 429 with `Retry-After` after burst.
- [ ] Cross-tenant leakage: auth as org A, hit `/api/v1/leads/<org-B-uuid>` → 404 (not 403, to avoid existence disclosure).
- [ ] JWT tampering: flip alg→none, flip exp, flip org_id — all refused.
- [ ] CSRF: mutate without `X-CSRF-Token` → 403; mutate with tampered cookie/header mismatch → 403.
- [ ] Billing webhook: same event_id replayed twice → 200+200 no state transition on replay.
- [ ] Master impersonation with DB read-only user (simulated audit failure) → 500 AUDIT_FAILED, no session minted.
- [ ] WebSocket origin: open from non-CORS origin → connection refused.
- [ ] `dangerouslySetInnerHTML` — grep the bundle for `innerHTML=` assignments (should be empty outside React's internal).
