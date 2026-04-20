# CI/CD Workflows

## Pipeline overview

```
push → develop ──────────► ci.yml (frontend + backend + golangci-lint)
push → main ─────────────► ci.yml
tag  → v*  ───────────────► release.yml (Docker build + push to GHCR + draft release)
manual dispatch ─────────► deploy.yml (migrate → deploy → smoke)
```

## ci.yml

Runs on every push and PR to `main` / `develop`.

- **frontend**: `npm ci` → `tsc --noEmit` → `eslint` → `prettier --check` → `vitest` → `vite build`. Uploads `dist/` as an artifact on main/develop.
- **backend**: boots ephemeral Postgres service, applies migrations via `golang-migrate`, runs `go vet` + `go build` + `go test -race -timeout 5m` with `DATABASE_URL` set so integration tests run.
- **lint-go**: `golangci-lint` v1.60 with the config under `torque-api/.golangci.yml`.

## release.yml

Triggered by a `v*` tag on `main`.

Builds distroless Docker images for both surfaces, pushes to `ghcr.io/<repo>/torque-api:<tag>` and `.../torque-web:<tag>` (plus `:latest`), emits a draft GitHub release with auto-generated notes.

Required secrets: none beyond `GITHUB_TOKEN` (scoped via `permissions: packages: write`).

## deploy.yml

Manual dispatch — runs per environment with required reviewers.

Inputs:
- `environment`: `staging` | `production`.
- `image_tag`: the `v*` tag or commit sha from `release.yml`.
- `run_migrations`: apply pending migrations before cutover. Default true.

Jobs:
1. `migrate`: downloads `golang-migrate`, runs `up` against the env's `DATABASE_URL`.
2. `deploy`: POSTs the tag to the EasyPanel deploy webhook.
3. `smoke`: polls `${BASE_URL}/healthz` 5× with 10s spacing. Fails the run if non-200 after retries.

Required GitHub Environment secrets per env:
- `DATABASE_URL` — managed Postgres DSN.
- `EASYPANEL_DEPLOY_WEBHOOK` — EasyPanel-issued URL.

Required GitHub Environment vars:
- `BASE_URL` — e.g. `https://app.torque.com.br` (prod) or `https://staging.torque.com.br`.

## Rollback

`deploy.yml` is re-dispatchable with any previous tag:

```
Actions → Deploy → Run workflow → environment=production, image_tag=v1.2.3, run_migrations=false
```

The `run_migrations=false` flag is critical on rollback — older schema is almost never forward-compatible with a newer image. If a migration was shipped with the bad tag, follow `DEPLOYMENT.md → Rollback migrations` (see the runbook added in S29).

## Zero-downtime deploy

EasyPanel runs the image behind its internal reverse proxy. A deploy rotates containers in `rollout` mode: spin up N new, health-check, drain + kill old. The API's graceful shutdown (already wired at `cfg.ShutdownTimeout`) honors `SIGTERM` with up to 15s to drain in-flight requests.
