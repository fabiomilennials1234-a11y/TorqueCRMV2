#!/usr/bin/env bash
#
# backup-postgres.sh — take a compressed pg_dump of Torque's Postgres DB and,
# optionally, upload it to S3-compatible object storage.
#
# Called by:
#   - .github/workflows/backup.yml (daily schedule)
#   - Operator's shell before a production migration (see .specs/runbooks/backup-restore.md)
#
# Required env:
#   DATABASE_URL                Postgres connection string.
#
# Optional env:
#   BACKUP_DIR                  Directory for the dump file. Default: /tmp
#   BACKUP_PREFIX               Filename prefix. Default: "torque"
#   S3_BUCKET                   If set, the dump is uploaded to s3://$S3_BUCKET/
#   S3_PREFIX                   Object key prefix inside the bucket. Default: "daily/"
#   S3_SSE                      Server-side encryption. Default: "AES256" (SSE-S3)
#   AWS_REGION                  AWS region. Default: left to awscli's own resolution.
#   KEEP_LOCAL                  "true" to keep the local dump after upload. Default: "false"
#
# Exit codes:
#   0  success
#   1  any failure (pg_dump, upload, integrity check)

set -euo pipefail

# --- Structured logging (zerolog-alike: ISO timestamp + level + msg) ---------
log() {
  local level="$1"
  shift
  printf '{"time":"%s","level":"%s","msg":"%s","script":"backup-postgres"}\n' \
    "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$level" "$*"
}

log_info()  { log "info"  "$*"; }
log_warn()  { log "warn"  "$*"; }
log_error() { log "error" "$*" >&2; }

on_error() {
  log_error "backup failed at line $1"
  exit 1
}
trap 'on_error $LINENO' ERR

# --- Preflight ---------------------------------------------------------------
if [[ -z "${DATABASE_URL:-}" ]]; then
  log_error "DATABASE_URL is required"
  exit 1
fi

if ! command -v pg_dump >/dev/null 2>&1; then
  log_error "pg_dump not found on PATH — install postgresql-client-15"
  exit 1
fi

BACKUP_DIR="${BACKUP_DIR:-/tmp}"
BACKUP_PREFIX="${BACKUP_PREFIX:-torque}"
S3_PREFIX="${S3_PREFIX:-daily/}"
S3_SSE="${S3_SSE:-AES256}"
KEEP_LOCAL="${KEEP_LOCAL:-false}"

# Normalize trailing slash on S3_PREFIX
[[ "$S3_PREFIX" != */ ]] && S3_PREFIX="${S3_PREFIX}/"

mkdir -p "$BACKUP_DIR"

TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
DUMP_FILE="${BACKUP_DIR}/${BACKUP_PREFIX}-${TIMESTAMP}.dump"

log_info "starting pg_dump to ${DUMP_FILE}"

# --- Dump --------------------------------------------------------------------
# --format=custom is required for pg_restore selectivity.
# --compress=9 maximizes compression; CPU cost is acceptable on a daily run.
# --no-owner / --no-privileges make the dump restorable into a DB owned by a
# different role (useful for staging drills).
pg_dump \
  "$DATABASE_URL" \
  --format=custom \
  --compress=9 \
  --no-owner \
  --no-privileges \
  --file="$DUMP_FILE"

# --- Integrity check ---------------------------------------------------------
if [[ ! -s "$DUMP_FILE" ]]; then
  log_error "dump file is empty after pg_dump"
  exit 1
fi

DUMP_BYTES="$(wc -c < "$DUMP_FILE" | tr -d ' ')"
if [[ "$DUMP_BYTES" -lt 1024 ]]; then
  log_error "dump file suspiciously small: ${DUMP_BYTES} bytes"
  exit 1
fi

# Quick structural check — pg_restore --list must succeed on a valid custom dump.
if ! pg_restore --list "$DUMP_FILE" >/dev/null 2>&1; then
  log_error "pg_restore --list failed; dump is not a valid custom archive"
  exit 1
fi

# SHA-256 for traceability (logged, not stored separately).
if command -v sha256sum >/dev/null 2>&1; then
  DUMP_SHA="$(sha256sum "$DUMP_FILE" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  DUMP_SHA="$(shasum -a 256 "$DUMP_FILE" | awk '{print $1}')"
else
  DUMP_SHA="unavailable"
fi

log_info "dump complete bytes=${DUMP_BYTES} sha256=${DUMP_SHA}"

# --- Upload (optional) -------------------------------------------------------
if [[ -n "${S3_BUCKET:-}" ]]; then
  if ! command -v aws >/dev/null 2>&1; then
    log_error "S3_BUCKET set but aws CLI not found on PATH"
    exit 1
  fi

  S3_KEY="${S3_PREFIX}$(basename "$DUMP_FILE")"
  S3_URI="s3://${S3_BUCKET}/${S3_KEY}"

  log_info "uploading to ${S3_URI} sse=${S3_SSE}"

  AWS_ARGS=(s3 cp "$DUMP_FILE" "$S3_URI" --sse "$S3_SSE")
  if [[ -n "${AWS_REGION:-}" ]]; then
    AWS_ARGS+=(--region "$AWS_REGION")
  fi

  aws "${AWS_ARGS[@]}"

  log_info "upload complete key=${S3_KEY}"

  if [[ "$KEEP_LOCAL" != "true" ]]; then
    rm -f "$DUMP_FILE"
    log_info "local dump removed (KEEP_LOCAL=false)"
  fi
else
  log_warn "S3_BUCKET not set — dump retained locally only at ${DUMP_FILE}"
fi

log_info "backup-postgres.sh finished ok"
exit 0
