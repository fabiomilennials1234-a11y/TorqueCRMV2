# scripts/

Operational scripts for Torque CRM v2. All scripts are bash, `set -euo pipefail`,
no hardcoded paths. Output is structured JSON on stdout (zerolog-alike) so they
can be piped into the same log stream as the Go API during an incident.

## Inventory

| Script | Purpose | Runs from |
|--------|---------|-----------|
| `backup-postgres.sh` | `pg_dump` → local file → optional S3 upload | GitHub Actions (`backup.yml`) + operator bastion |
| `restore-postgres.sh` | `pg_restore` into a target DB, with a prod guardrail | Operator laptop / bastion only — never CI |
| `verify-backup.sh` | Structural sanity-check of a dump, no DB required | Anywhere; good as a pre-restore gate |

## Dependencies

- `bash` 4+ (associative arrays, `set -euo pipefail`).
- `postgresql-client-15` providing `pg_dump`, `pg_restore`, `psql`. Major version **must** match the server to avoid dump-format mismatches.
- `aws` CLI v2 if uploading to S3 (`backup-postgres.sh` only when `S3_BUCKET` is set).
- `sha256sum` or `shasum` for dump hashing. Optional — logs `sha256=unavailable` if neither is present.

## Usage

### Daily backup (automated)

Runs via `.github/workflows/backup.yml` at 03:00 UTC. Secrets required:

| Secret | Purpose |
|--------|---------|
| `DATABASE_URL_PROD` | Postgres connection string passed in as `DATABASE_URL`. |
| `BACKUP_S3_BUCKET` | Target bucket for dumps. |
| `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` | IAM user scoped to the bucket. |
| `AWS_REGION` | Optional; defaults to the CLI's own resolution. |
| `SENTRY_WEBHOOK` | Optional; used for failure notifications. |

### Pre-migration snapshot (manual)

```bash
export DATABASE_URL="$(op read 'op://Torque Ops/Prod Postgres/connection')"
export S3_BUCKET=torque-backups
export S3_PREFIX="pre-migration/$(git describe --tags --abbrev=0)"
./scripts/backup-postgres.sh
```

### Verify a local dump before restoring

```bash
./scripts/verify-backup.sh /tmp/torque-20260424T030001Z.dump
```

### Restore into staging

```bash
export DATABASE_URL="$DATABASE_URL_STAGING"
./scripts/restore-postgres.sh /tmp/torque-20260424T030001Z.dump
```

### Restore into production (CTO approval required)

```bash
export ENV=production
export DATABASE_URL="$DATABASE_URL_PROD"
./scripts/restore-postgres.sh /tmp/torque-20260424T030001Z.dump --confirm-prod
```

The script refuses to run without the `--confirm-prod` flag when `ENV=production`.

## Executable bit

These scripts should be executable. On Unix:

```bash
chmod +x scripts/*.sh
```

On Windows (where chmod is effectively a no-op in NTFS), invoke via `bash`
directly or ensure the files get mode 755 when checked out in WSL / on the CI
runner (Linux runners apply the mode from the index correctly).

If you see "permission denied" on Linux after cloning, run:

```bash
git update-index --chmod=+x scripts/backup-postgres.sh
git update-index --chmod=+x scripts/restore-postgres.sh
git update-index --chmod=+x scripts/verify-backup.sh
git commit -m "chore(scripts): mark backup scripts executable"
```

## Related runbooks

- [`/.specs/runbooks/backup-restore.md`](../.specs/runbooks/backup-restore.md) — RPO/RTO targets, drill protocol, ownership.
- [`/.specs/runbooks/key-rotation.md`](../.specs/runbooks/key-rotation.md) — key escrow and rotation (a backup without `INTEGRATION_ENCRYPTION_KEY` is ciphertext).
- [`/.specs/runbooks/incident-response.md`](../.specs/runbooks/incident-response.md) — pager protocol that calls into these scripts.
