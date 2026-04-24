# Backup & Restore — Torque CRM v2

Status: draft, pre-GA. **Must be provisioned and drilled before the first paying tenant is admitted.**

Related runbooks:
- [incident-response.md](./incident-response.md) — invoked when a backup restore is part of recovery.
- [key-rotation.md](./key-rotation.md) — `INTEGRATION_ENCRYPTION_KEY` escrow is a hard prerequisite for a useful backup (ciphertext without the key is garbage).

---

## RPO / RTO Targets

| Metric | Target (pre-GA) | Target (post 10 paying tenants) | Rationale |
|--------|-----------------|----------------------------------|-----------|
| **RPO** — Recovery Point Objective | 24h (daily `pg_dump`) | 1h (continuous WAL archiving via Hostinger PITR or wal-g to S3) | Pre-GA a lost day is painful but recoverable; once tenants pay, losing an hour of CRM activity is already a contract breach. |
| **RTO** — Recovery Time Objective | 1h from decision-to-restore until `/healthz` green on staging | 30 min | Measured on the last completed drill. If the drill missed the target, downgrade the RTO on this table until a new drill proves the smaller number. |
| **Retention** | 30 days rolling daily + 1 monthly snapshot in cold storage for 12 months | Same + 1 yearly snapshot kept 7 years (LGPD/fiscal) | Daily window covers "oh no we just corrupted data" horizon; monthly covers legal/audit requests; yearly covers tax/fiscal disputes. |
| **Encryption** | SSE-S3 at rest + TLS in transit | SSE-KMS with customer-managed key | Pre-GA SSE-S3 is fine because the bucket policy itself is the authorization boundary. Post-GA we want an auditable key per-tenant-class. |

**Invariant:** no backup counts as real until it has been restored, end-to-end, at least once. A dump that has never been restored is a hope, not a backup.

---

## Cadência

- **Daily full dump** — 03:00 UTC (00:00 BRT, lowest activity window) via GitHub Actions scheduled workflow `backup.yml`.
- **Manual snapshot before any prod migration** — part of the deploy runbook: operator runs `scripts/backup-postgres.sh` against `DATABASE_URL_PROD` and pins the dump in S3 under prefix `pre-migration/<tag>/` before flipping `run_migrations=true`.
- **Weekly rclone of object storage** — Sunday 04:00 UTC. Captures uploaded integration artifacts, audio TTS caches, and any tenant-uploaded attachments. Stored in a separate bucket/region from the Postgres dumps.
- **Monthly cold snapshot** — 1st of each month, 05:00 UTC. Copies the latest daily dump to a cold-tier bucket (B2 "Archive", R2 with `X-Amz-Storage-Class: GLACIER`, or Spaces cold tier) with a 12-month lifecycle.
- **Drill** — first Monday of every month. Runbook execution timed and logged at `.specs/runbooks/backup-drill-YYYY-MM-DD.md`.

---

## O que backupar

### Included
- **Postgres** — full `pg_dump --format=custom --compress=9` against `DATABASE_URL_PROD`. Includes every table, including high-volume `lead_history`, `agent_messages`, `audit_log`, `workflow_runs`, `plan_quotas`, `org_quotas`, `agent_triggers`. Size estimates tracked in `audits/` after first production run.
- **Object storage** (when S3/B2/R2/Spaces is provisioned) — weekly rclone sync of the live bucket to a cross-region backup bucket. Captures: integration OAuth artifacts the app caches, audio TTS responses, tenant-uploaded attachments (once F08/F12 land).
- **Secrets inventory** — annually on 15 January export `gh secret list` output (names only, never values) into `.specs/audits/secrets-inventory-YYYY.md` for governance review. Actual values live in 1Password (see [key-rotation.md](./key-rotation.md)).

### Excluded
- **zerolog stdout logs** — Sentry retains events 90 days; raw logs are ephemeral by design.
- **Redis cache** — if/when introduced, it is recomputable from Postgres.
- **Docker images** — `ghcr.io` retains image history; release tags are immutable.
- **Node / Go build artifacts** — rebuildable from the commit SHA, which is embedded in every image via `APP_VERSION`.

### Out of scope until decided
- **Append-only table rotation** — `lead_history`, `agent_messages`, `audit_log` will eventually need partition-based rotation (drop partitions > 90d after archiving to S3). Tracked as a follow-up ticket; for now they are dumped in full.

---

## Procedure

### Backup (automated, daily)

Implemented by `.github/workflows/backup.yml` calling `scripts/backup-postgres.sh`.

1. Scheduled run fires at `0 3 * * *` (03:00 UTC). Manual runs via `workflow_dispatch`.
2. Runner installs `postgresql-client-15` (must match the server major version to avoid `pg_dump` version mismatches) and the AWS CLI v2.
3. `scripts/backup-postgres.sh` runs:
   - `pg_dump "$DATABASE_URL" --format=custom --compress=9 --file=/tmp/torque-$(date -u +%Y%m%dT%H%M%SZ).dump`
   - Size and SHA-256 logged to stdout.
   - If `S3_BUCKET` is set, uploads via `aws s3 cp --sse AES256`.
4. Retention sweep deletes objects older than 30 days in the daily prefix.
5. On any failure, the workflow pings `SENTRY_WEBHOOK` with a structured event so the on-call rotation pages the infra owner.

### Backup (manual, pre-migration)

Runs from the operator's laptop or the bastion, never from CI.

```bash
# 1. Load prod creds (1Password CLI or equivalent — never plaintext env file)
export DATABASE_URL="$(op read 'op://Torque Ops/Prod Postgres/connection')"
export S3_BUCKET=torque-backups
export S3_PREFIX="pre-migration/$(git describe --tags --abbrev=0)"

# 2. Take the snapshot
./scripts/backup-postgres.sh

# 3. Verify the archive is non-zero and intact
./scripts/verify-backup.sh /tmp/torque-*.dump

# 4. Proceed with deploy — `run_migrations=true` is now safe to flip
```

### Restore (staging rehearsal or incident recovery)

**Never run this against production without CTO written approval.** The script enforces a `--confirm-prod` flag; the approval is the human gate.

```bash
# 1. Decide which dump. Default is "latest daily".
aws s3 ls "s3://$S3_BUCKET/daily/" --recursive | tail -5

# 2. Download. Use a tmpfs-backed path if the host disk is unencrypted.
aws s3 cp "s3://$S3_BUCKET/daily/torque-YYYYMMDDTHHMMSSZ.dump" /tmp/

# 3. Verify integrity before touching any database.
./scripts/verify-backup.sh /tmp/torque-YYYYMMDDTHHMMSSZ.dump

# 4. Restore into staging (destructive — wipes staging schema).
export DATABASE_URL="$DATABASE_URL_STAGING"
./scripts/restore-postgres.sh /tmp/torque-YYYYMMDDTHHMMSSZ.dump

# 5. Validate row counts against the production snapshot manifest.
psql "$DATABASE_URL" -c "SELECT count(*) AS leads FROM leads;"
psql "$DATABASE_URL" -c "SELECT count(*) AS orgs FROM organizations;"
psql "$DATABASE_URL" -c "SELECT count(*) AS audit FROM audit_log;"

# 6. Smoke test: healthz + login + pull one pipe.
curl -fsS https://staging.torquecrm.com.br/healthz
curl -fsS https://staging.torquecrm.com.br/readyz

# 7. If anything is off, escalate via incident-response.md and do NOT
#    promote the restore to prod.
```

### Restore (production — incident recovery)

Identical to staging except:
- `--confirm-prod` flag on `restore-postgres.sh` must be present AND the CTO approval reference (ticket ID or Slack permalink) must be in the operator's scratchpad before the command is typed.
- A read-only window is declared first: flip the EasyPanel web service to a maintenance page, or null out `CORS_ORIGINS` so the frontend bounces. No writes during the restore.
- Post-restore: rotate `JWT_SECRET` and `INTEGRATION_ENCRYPTION_KEY` if the incident was a suspected compromise. See [key-rotation.md](./key-rotation.md).

---

## Drill

Run on the first Monday of every month. Skip only with a written reason (code freeze, holiday). Three consecutive skips = lift the incident flag and block deploys until drilled.

Drill transcript template lives at `.specs/runbooks/backup-drill-YYYY-MM-DD.md`:

```markdown
# Backup Drill — YYYY-MM-DD

- Operator:
- Approver:
- Dump used: s3://torque-backups/daily/torque-YYYYMMDDTHHMMSSZ.dump
- Staging target: <name>
- Start: HH:MM UTC
- Restore finished: HH:MM UTC
- Smoke test passed: HH:MM UTC
- Total RTO: HH:MM
- Issues found:
- Action items (owner + ticket):
```

RTO regression blocks the next deploy.

---

## Ownership

- **Executor** — on-call infra for the week (see PagerDuty/Linear rotation).
- **Approver for prod restore** — CTO only. No exceptions, no delegation.
- **Escalation** — Sentry alert → Slack `#infra` → on-call via PagerDuty → CTO.
- **Audit** — every prod restore writes `audit_log` rows with `actor_type='system', action='backup.restored', target='database'` before the app accepts traffic again.

---

## Provisioning checklist (pre-first-run)

Tick these off before the workflow can succeed:

- [ ] Object storage bucket created: `torque-backups` (pick one provider, document which).
- [ ] Bucket policy denies public access, enforces TLS, blocks deletes without MFA for objects older than 7 days.
- [ ] IAM user/key with scoped `s3:PutObject`, `s3:GetObject`, `s3:ListBucket`, `s3:DeleteObject` on this bucket only.
- [ ] GitHub Actions secrets populated:
  - `DATABASE_URL_PROD`
  - `BACKUP_S3_BUCKET`
  - `AWS_ACCESS_KEY_ID`
  - `AWS_SECRET_ACCESS_KEY`
  - `AWS_REGION` (defaults to `us-east-1` if unset)
  - `SENTRY_WEBHOOK` (optional but recommended)
- [ ] Staging DB provisioned with identical extensions (`vector`, `pgcrypto`, etc.) so restore does not die on `CREATE EXTENSION`.
- [ ] First manual `workflow_dispatch` run succeeds end-to-end.
- [ ] First full restore drill completed and timed.
