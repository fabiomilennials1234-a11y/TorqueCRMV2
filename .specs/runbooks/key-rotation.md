# Key Rotation Runbook — Torque CRM v2

Status: draft, pre-GA. **Every key in the table below must have an escrow entry in the 1Password shared vault `Torque Ops` before the first paying tenant is admitted.**

Related runbooks:
- [backup-restore.md](./backup-restore.md) — a dump without `INTEGRATION_ENCRYPTION_KEY` is ciphertext; escrow is a hard prerequisite for useful backups.
- [incident-response.md](./incident-response.md) — suspected leak of any key on this table is at minimum Sev2 and upgrades to Sev1 if `INTEGRATION_ENCRYPTION_KEY` or `DATABASE_URL` are involved.

---

## Critical Keys

All keys live as GitHub Actions / EasyPanel environment variables and are escrowed in 1Password. None are ever committed, Slacked, emailed, or written to a plaintext file on disk.

| Key | Impact of Loss | Impact of Leak | Rotation Cadence | Escrow |
|-----|----------------|----------------|-------------------|--------|
| `INTEGRATION_ENCRYPTION_KEY` | Permanent loss of every OAuth refresh token in `integration_credentials` (Google Calendar, Meta Lead Ads, TinyERP, SZ.Chat) across every tenant. No recovery. Tenants must re-consent each integration. | Attacker can decrypt stolen DB dumps offline and harvest every tenant's third-party OAuth tokens. Treat as Sev1. | 6 months OR on suspected leak. | 1Password `Torque Ops` → `Integration Encryption Key`. Must also be present in the DR bucket's sealed envelope (see "Escrow" below). |
| `JWT_SECRET` | All active sessions invalidated; every user must log in again. No data loss. | Attacker can forge access tokens for any user; combined with a tenant ID they can read any org's data. Sev1 if exploitation is observed, Sev2 otherwise. | 12 months OR on suspected leak. | 1Password `Torque Ops` → `JWT Secret`. |
| `BILLING_WEBHOOK_SECRET` | Asaas webhooks fail HMAC check until both sides updated; checkout state diverges from payment state. | Attacker can forge payment events (`PAYMENT_CONFIRMED`, `PAYMENT_REFUNDED`) and manipulate subscription state. Sev1 — touches money. | 12 months OR on suspected leak. | 1Password + Asaas dashboard webhook configuration. |
| `ASAAS_API_KEY` | Checkout and subscription state updates fail until rotated. No data loss. | Attacker can issue charges, refunds, and read every Asaas record tied to our merchant account. Sev1 — touches money. | 6 months. | 1Password + Asaas dashboard. |
| `OPENROUTER_API_KEY` | Copilot chat/stream endpoints degrade (fall back to cached responses where possible, 503 otherwise). | Attacker burns quota and generates bills. Sev2. | 12 months. | 1Password. |
| `GEMINI_API_KEY` | RAG embeddings stop being generated; search degrades. | Same quota-burn risk. Sev2. | 12 months. | 1Password. |
| `ELEVENLABS_API_KEY` | TTS falls back to deterministic mock (by design). | Quota burn. Sev3. | 12 months. | 1Password. |
| `LEAD_WEBHOOK_SECRET` | Upstream lead-ingest senders (n8n, Meta ad forms proxy) start getting 403 until they update their signing secret. | Attacker can inject forged leads into any tenant's inbox. Sev2 (data pollution, possible phishing vector). | 12 months OR on suspected leak. | 1Password. |
| `META_APP_SECRET` | Meta Lead Ads webhooks fail signature check; ads-insights calls start failing appsecret_proof validation. | Attacker can forge Meta Lead Ads events. Sev2. | 12 months OR on suspected leak. | 1Password + Meta App Dashboard. |
| `SENTRY_DSN` | Backend telemetry stops shipping until rotated (old events already ingested are unaffected). | Attacker can submit fake events and exhaust quota. Sev3. | 24 months. | 1Password + Sentry UI. |
| `DATABASE_URL` (connection string containing password) | Application cannot reach Postgres; total outage. | Direct read/write access to every tenant's data. Sev1. | 12 months OR on suspected leak. Also rotate whenever anyone with current access leaves the team. | 1Password + Hostinger managed Postgres dashboard. |
| `GOOGLE_OAUTH_CLIENT_SECRET` | Google Calendar OAuth flow breaks until rotated. | Attacker can impersonate our app to Google, intercept consent redirects. Sev2. | 24 months (Google recommends biannual). | 1Password + Google Cloud Console. |

---

## Procedure — `INTEGRATION_ENCRYPTION_KEY` (most sensitive)

This key is different from every other key on the table: rotation requires
**re-encrypting existing ciphertext**, not just swapping values. The backend
must accept both the old and the new key for the duration of the rotation.

### One-time setup (before first rotation)

Backend work required before the first rotation is possible:
1. Config loader accepts both `INTEGRATION_ENCRYPTION_KEY` and
   `INTEGRATION_ENCRYPTION_KEY_NEW` (the latter optional).
2. Decrypt path: try the primary key first, fall back to the secondary. Log a
   structured `key.rotation.secondary_used` event when the fallback fires so
   we can see the rotation window closing in telemetry.
3. Re-encrypt worker: a one-shot Go binary or migration-like command that
   iterates `integration_credentials` under `FOR UPDATE SKIP LOCKED`, decrypts
   with the primary, re-encrypts with the secondary, writes the new ciphertext
   in a transaction. Idempotent (a marker column `encrypted_with_key_version`
   tells it which rows are already migrated).

Track implementation as a dedicated ticket; do not attempt rotation in a
tenant-facing window until the re-encrypt worker has been drilled on staging.

### Rotation execution

1. **Snapshot.** Run the pre-migration backup from
   [backup-restore.md](./backup-restore.md) before touching anything.
2. **Generate the new key.**
   ```bash
   openssl rand -base64 32
   ```
   Paste into 1Password `Torque Ops → Integration Encryption Key (pending)`.
3. **Ship dual-key config.** Update EasyPanel env vars:
   - `INTEGRATION_ENCRYPTION_KEY` — unchanged (still the primary).
   - `INTEGRATION_ENCRYPTION_KEY_NEW` — the newly generated value.
   Redeploy. Confirm `/healthz` green.
4. **Run the re-encrypt worker** against prod DB from the bastion. Monitor:
   - Row throughput (target: `integration_credentials` table fully migrated in < 15 min at current tenant count).
   - Sentry for any `decrypt failed` events.
5. **Validate.** For a known tenant, trigger a background refresh of a Google
   OAuth token (e.g. hit a Calendar endpoint that forces refresh). Confirm the
   event `key.rotation.secondary_used` appears — this proves the worker
   re-encrypted successfully and the secondary key decrypts correctly.
6. **Promote.** Swap the env vars:
   - `INTEGRATION_ENCRYPTION_KEY` ← old value of `_NEW`.
   - Remove `INTEGRATION_ENCRYPTION_KEY_NEW`.
   Redeploy.
7. **Retire.** In 1Password, move the old key from the main entry into a
   dated archive entry titled `Integration Encryption Key (retired YYYY-MM-DD)`.
   **Do not delete for at least 90 days** in case a backup from before the
   rotation needs to be restored and re-migrated.
8. **Audit log.** Insert a row into `audit_log` with
   `actor_type='system', action='key.rotated', target='integration_encryption_key'`
   and the ticket reference as payload. Sentry breadcrumb optional but nice.
9. **Cross-check DR.** Update the sealed envelope in the DR cold-storage
   bucket (see Escrow below) with the new key material.

### Rotation rollback (if the worker fails mid-run)

The dual-key design makes this safe: revert EasyPanel to the single-key
config (drop `_NEW`). The rows that were already re-encrypted cannot be
decrypted by the old key alone and will fail next integration refresh —
so instead, keep both keys deployed, investigate the failure, and resume
the worker. Never drop the secondary key while any row still references it.

---

## Procedure — Generic Rotation (every other key)

Simpler because no re-encryption is needed. The app just starts using the new
value after a redeploy.

1. **Snapshot.** Daily backup already exists; for money-touching keys
   (`BILLING_WEBHOOK_SECRET`, `ASAAS_API_KEY`) take a manual pre-rotation
   snapshot first.
2. **Generate** the new value at the source of truth:
   - `JWT_SECRET`: `openssl rand -base64 48`.
   - `BILLING_WEBHOOK_SECRET`: `openssl rand -base64 32` + update in Asaas webhook config.
   - `ASAAS_API_KEY`: rotate in Asaas dashboard → copy new key.
   - `LEAD_WEBHOOK_SECRET`: `openssl rand -base64 32` + notify every upstream sender.
   - `META_APP_SECRET`: rotate in Meta App Dashboard.
   - `SENTRY_DSN`: create a new client key in Sentry UI, mark old one as disabled after cutover.
   - `GOOGLE_OAUTH_CLIENT_SECRET`: reset in Google Cloud Console.
   - `DATABASE_URL`: rotate the Postgres role password in Hostinger UI, update connection string.
3. **Store** the new value in 1Password `Torque Ops` (create a dated entry for the retired version).
4. **Deploy** the change to EasyPanel env vars. For prod this is a single env var update + service restart.
5. **Validate** immediately:
   - `JWT_SECRET` — all existing sessions should now 401; confirm login still succeeds with fresh creds.
   - `BILLING_WEBHOOK_SECRET` — trigger a test webhook from Asaas dashboard, confirm `billing_events` row appears.
   - `ASAAS_API_KEY` — run `scripts/ping-asaas.sh` (if implemented) or trigger a sandbox charge.
   - `LEAD_WEBHOOK_SECRET` — coordinate with upstream sender to replay a test event.
   - `META_APP_SECRET` — trigger a Lead Ads test event from Meta's tool.
   - Every other key — the feature that uses it should work end-to-end within 5 minutes of the deploy.
6. **Audit log.** Same pattern as above: `action='key.rotated', target='<key name>'`.

---

## Escrow

### 1Password shared vault `Torque Ops`

- Members: 2 infra admins + CTO. No one else, ever.
- Every entry has: current value, previous value (dated), generation command used, deploy-ETA reminder.
- Access review: quarterly. Departing members lose access on their last day; any key they knew gets rotated within 72h.

### DR cold-storage envelope

- Sealed envelope in a cold-storage bucket separate from the backup bucket: `gpg --encrypt --recipient <cto-key> keys.json`.
- Contents: the current production value of every key in the table above plus the current `DATABASE_URL`.
- Updated within 48h of any rotation. Verified quarterly (download, decrypt, diff against 1Password, re-seal).
- This is the "bus factor" insurance. If 1Password access is lost entirely, the CTO's PGP key opens this envelope and restores operations.

### What NOT to do

- Never store keys in `.env` files checked into any repository, including private ones.
- Never paste keys into Slack, email, GitHub issues, or Linear comments.
- Never store keys on a developer laptop outside of an encrypted 1Password session.
- Never reuse a rotated key value for a different purpose. Retired means retired.

---

## Incident Response — Suspected Leak

Follow this sequence for ANY credible leak signal (accidental commit, log exfiltration, departing employee with unreturned access, suspicious API usage pattern).

1. **Revoke at the source** before touching our own config:
   - `ASAAS_API_KEY` — revoke in Asaas dashboard.
   - `GOOGLE_OAUTH_CLIENT_SECRET` — revoke in Google Cloud Console.
   - `META_APP_SECRET` — revoke in Meta App Dashboard.
   - `SENTRY_DSN` — disable the client key in Sentry.
   - DB credentials — revoke the role in Hostinger Postgres dashboard.
   - Provider-issued keys without a dashboard (OpenRouter, ElevenLabs, Gemini) — delete via API or provider UI.
2. **Rotate** following the procedures above. For `INTEGRATION_ENCRYPTION_KEY`, the dual-key procedure is mandatory — do not attempt a single-step swap.
3. **Audit.**
   - `grep 'key.rotated'` in the audit log to confirm the rotation event landed.
   - Sentry search for any usage of the leaked credential in the incident window.
   - Provider-side audit logs (Asaas, Meta, Google) if available.
4. **Notify.** CTO + security@ (when the role exists) + any affected tenant if customer data was touched. LGPD obligations may apply; default to notifying the DPO.
5. **Incident ticket.** Open within 1h of detection. Post-mortem within 48h in `.specs/incidents/YYYY-MM-DD-<slug>.md`. Root cause + blast radius + action items are non-negotiable sections.
