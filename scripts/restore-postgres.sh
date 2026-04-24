#!/usr/bin/env bash
#
# restore-postgres.sh — restore a pg_dump (custom format) into a Torque DB.
#
# DESTRUCTIVE. Drops and recreates every object in the target schema.
#
# Usage:
#   DATABASE_URL=... ./scripts/restore-postgres.sh <dump-file>           # non-prod only
#   ENV=production DATABASE_URL=... ./scripts/restore-postgres.sh <dump-file> --confirm-prod
#
# Required env:
#   DATABASE_URL  Postgres connection string for the TARGET database.
#
# Optional env:
#   ENV           If set to "production" (or "prod"), the script refuses to
#                 run unless the `--confirm-prod` flag is passed as the second
#                 positional argument. Defaults to unset (non-production).
#
# Exit codes:
#   0  success
#   1  any failure (missing tools, refused prod without confirm, pg_restore error)
#   2  validation failure after restore (row counts zero, etc.)

set -euo pipefail

# --- Structured logging ------------------------------------------------------
log() {
  local level="$1"
  shift
  printf '{"time":"%s","level":"%s","msg":"%s","script":"restore-postgres"}\n' \
    "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$level" "$*"
}
log_info()  { log "info"  "$*"; }
log_warn()  { log "warn"  "$*"; }
log_error() { log "error" "$*" >&2; }

on_error() {
  log_error "restore failed at line $1"
  exit 1
}
trap 'on_error $LINENO' ERR

# --- Args --------------------------------------------------------------------
DUMP_FILE="${1:-}"
CONFIRM_PROD_FLAG="${2:-}"

if [[ -z "$DUMP_FILE" ]]; then
  log_error "usage: restore-postgres.sh <dump-file> [--confirm-prod]"
  exit 1
fi

if [[ ! -f "$DUMP_FILE" ]]; then
  log_error "dump file not found: ${DUMP_FILE}"
  exit 1
fi

# --- Preflight ---------------------------------------------------------------
if [[ -z "${DATABASE_URL:-}" ]]; then
  log_error "DATABASE_URL is required"
  exit 1
fi

if ! command -v pg_restore >/dev/null 2>&1; then
  log_error "pg_restore not found on PATH — install postgresql-client-15"
  exit 1
fi

if ! command -v psql >/dev/null 2>&1; then
  log_error "psql not found on PATH — install postgresql-client-15"
  exit 1
fi

# --- Production guardrail ----------------------------------------------------
ENV_NORMALIZED="$(echo "${ENV:-}" | tr '[:upper:]' '[:lower:]')"
if [[ "$ENV_NORMALIZED" == "production" || "$ENV_NORMALIZED" == "prod" ]]; then
  if [[ "$CONFIRM_PROD_FLAG" != "--confirm-prod" ]]; then
    log_error "ENV=${ENV} detected — refusing to restore without --confirm-prod"
    log_error "prod restore requires written CTO approval (see .specs/runbooks/backup-restore.md)"
    exit 1
  fi
  log_warn "running destructive restore against PRODUCTION (confirmed)"
fi

# --- Verify dump integrity before touching the DB ----------------------------
if ! pg_restore --list "$DUMP_FILE" >/dev/null 2>&1; then
  log_error "dump is not a valid custom-format archive: ${DUMP_FILE}"
  exit 1
fi

DUMP_BYTES="$(wc -c < "$DUMP_FILE" | tr -d ' ')"
log_info "restoring ${DUMP_FILE} (${DUMP_BYTES} bytes) into DATABASE_URL"

# --- Restore -----------------------------------------------------------------
# --clean + --if-exists: drop objects first so a restore over an existing
#   database is idempotent.
# --no-owner / --no-privileges: ignore ownership in the dump; the target role
#   becomes the new owner. Required when restoring between environments.
# --exit-on-error: stop at the first error (defensive; default is to continue).
# --single-transaction: atomic — either the whole thing lands or nothing does.
#   Incompatible with parallel restore, so we trade speed for atomicity here.
pg_restore \
  --dbname="$DATABASE_URL" \
  --clean --if-exists \
  --no-owner --no-privileges \
  --exit-on-error \
  --single-transaction \
  --verbose \
  "$DUMP_FILE"

log_info "pg_restore completed; running post-restore validation"

# --- Post-restore validation -------------------------------------------------
# Expected schema sanity: every Torque deployment has these tables since S01.
# If any are missing, the restore landed a partial/wrong dump and we fail loudly.
REQUIRED_TABLES=(
  "organizations"
  "users"
  "leads"
  "pipes"
  "stages"
  "audit_log"
)

for table in "${REQUIRED_TABLES[@]}"; do
  if ! psql "$DATABASE_URL" -tAc \
      "SELECT to_regclass('public.${table}') IS NOT NULL;" | grep -q '^t$'; then
    log_error "post-restore validation failed: missing table ${table}"
    exit 2
  fi
done

# Row counts for the two highest-signal tables.
ORG_COUNT="$(psql "$DATABASE_URL" -tAc 'SELECT count(*) FROM organizations;' | tr -d ' ')"
LEAD_COUNT="$(psql "$DATABASE_URL" -tAc 'SELECT count(*) FROM leads;' | tr -d ' ')"

log_info "validation ok tables=${#REQUIRED_TABLES[@]} organizations=${ORG_COUNT} leads=${LEAD_COUNT}"
log_info "restore-postgres.sh finished ok — run application smoke tests next"

exit 0
