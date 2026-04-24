#!/usr/bin/env bash
#
# verify-backup.sh — sanity-check a pg_dump custom-format archive without
# touching any database.
#
# Checks, in order:
#   1. File exists and is non-empty (> 1 KiB).
#   2. pg_restore --list succeeds (archive header is structurally valid).
#   3. The listing contains at least one TABLE DATA entry for a core table.
#
# Usage:
#   ./scripts/verify-backup.sh <dump-file>
#
# Exit codes:
#   0  dump appears valid
#   1  usage error or dump not found
#   2  dump failed an integrity check

set -euo pipefail

log() {
  local level="$1"
  shift
  printf '{"time":"%s","level":"%s","msg":"%s","script":"verify-backup"}\n' \
    "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$level" "$*"
}
log_info()  { log "info"  "$*"; }
log_error() { log "error" "$*" >&2; }

DUMP_FILE="${1:-}"

if [[ -z "$DUMP_FILE" ]]; then
  log_error "usage: verify-backup.sh <dump-file>"
  exit 1
fi

if [[ ! -f "$DUMP_FILE" ]]; then
  log_error "file not found: ${DUMP_FILE}"
  exit 1
fi

if ! command -v pg_restore >/dev/null 2>&1; then
  log_error "pg_restore not found on PATH — install postgresql-client-15"
  exit 1
fi

# --- Check 1: non-empty ------------------------------------------------------
DUMP_BYTES="$(wc -c < "$DUMP_FILE" | tr -d ' ')"
if [[ "$DUMP_BYTES" -lt 1024 ]]; then
  log_error "dump is suspiciously small: ${DUMP_BYTES} bytes (threshold 1024)"
  exit 2
fi
log_info "size ok bytes=${DUMP_BYTES}"

# --- Check 2: structural listing ---------------------------------------------
# Capture the listing; if pg_restore can't parse the archive, this fails.
LISTING="$(pg_restore --list "$DUMP_FILE" 2>/dev/null || true)"
if [[ -z "$LISTING" ]]; then
  log_error "pg_restore --list produced no output; archive header likely corrupt"
  exit 2
fi

ENTRY_COUNT="$(printf '%s\n' "$LISTING" | grep -cE '^[0-9]+; ' || true)"
if [[ "$ENTRY_COUNT" -lt 10 ]]; then
  log_error "dump listing has only ${ENTRY_COUNT} entries; too sparse to be a real archive"
  exit 2
fi
log_info "listing ok entries=${ENTRY_COUNT}"

# --- Check 3: contains at least one core table's data ------------------------
# A valid Torque dump must contain TABLE DATA for organizations OR leads.
# If neither is present the dump is either empty or schema-only.
if ! printf '%s\n' "$LISTING" \
  | grep -qE 'TABLE DATA[[:space:]]+public[[:space:]]+(organizations|leads|users|pipes|stages)'; then
  log_error "dump listing does not contain TABLE DATA for any core table"
  exit 2
fi
log_info "core-table data present"

# --- Preview (first 50 lines of the listing, for the operator's log) ---------
log_info "listing preview (first 50 entries):"
printf '%s\n' "$LISTING" | head -50

log_info "verify-backup.sh finished ok"
exit 0
