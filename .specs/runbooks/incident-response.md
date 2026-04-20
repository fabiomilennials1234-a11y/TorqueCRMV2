# Incident Response Runbook

Oncall's first 15 minutes for any Sev1/Sev2 page.

## Severity classification

| Sev | Signal | Response time |
|-----|--------|---------------|
| Sev1 | Login down, DB unreachable, >5% 5xx sustained, active data corruption | Page. 5 min ack. |
| Sev2 | Single feature broken, elevated error rate on non-auth path, stuck worker queue | Page. 15 min ack. |
| Sev3 | Non-critical regression, cosmetic bug, slow but up | Next business day. |

## Immediate checklist (Sev1/Sev2)

1. **Acknowledge** the page.
2. **Status check** — open the dashboard URL in the env bookmark bar:
   - `GET ${BASE_URL}/healthz` — if 200, app is up.
   - `GET ${BASE_URL}/readyz` — if 200, app is serving traffic.
3. **Sentry** — open the project, filter `environment:<env>`, last 30 min.
   Look for a new top-10 exception introduced after the last deploy.
4. **Recent deploys** — GitHub Actions → Deploy workflow → last 10 runs. If
   a deploy in the last 30 min correlates with the incident, go to **Rollback** below.
5. **DB** — if Sentry shows pgx/postgres errors, check managed Postgres dashboard: connections, CPU, slow queries.
6. **Rate limiter state** — if 429 spike, verify no runaway loop in a client. Check `audit_log` for `permission.denied` burst.

## Rollback procedure

**When**: bad deploy identified within the last 30-60 minutes AND Sentry
error rate correlates with the deploy timestamp.

1. Identify the last known-good tag:
   `gh release list --limit 10` (pick the last one before the bad deploy).
2. Go to Actions → Deploy → Run workflow:
   - environment: `production` (or whichever env is burning)
   - image_tag: `v<previous-good>`
   - run_migrations: **false** (critical — old schema is rarely forward-compatible)
3. Monitor the `smoke` job; if it fails, escalate to Sev1 and page the engineering lead.

## Rollback migrations (only when schema change caused the outage)

**When**: deploy included a migration that broke reads for the previous app version AND rollback of the image alone doesn't help.

1. SSH to the bastion (or run via `make` target from a trusted host).
2. Check the pending migration:
   `migrate -path ./torque-api/migrations -database "${DATABASE_URL}" version`
3. Roll back ONE step:
   `migrate -path ./torque-api/migrations -database "${DATABASE_URL}" down 1`
4. Redeploy the old image with run_migrations=false.
5. Post-incident: open a ticket to fix the migration ordering.

Destructive migrations (DROP COLUMN, DROP TABLE) should be avoided. If a
down migration would lose production data, **do not run it** — escalate
and take a read-only window while the schema is reconciled by hand.

## DB emergency

- **Connection exhaustion**: scale down worker concurrency (`Concurrency` in
  `worker.New`) by redeploying with a lower value; investigate which
  kind is leaking. Check `pg_stat_activity` for idle_in_transaction.
- **Slow query storm**: `pg_stat_statements` top 10 by total_exec_time;
  check for a new query pattern from the last deploy.
- **Data corruption suspected**: take a snapshot immediately, then read-only mode. Do not attempt `DELETE`/`UPDATE` to "fix" — escalate.

## WebSocket hub saturation

- Check `SubscriberCount` via log grep (`hub: register` / `hub: unregister`).
- If goroutine count is climbing without bound, a client is reconnecting
  in a loop. Rate-limit websocket auth upstream, then investigate.
- In-process hub has a hard limit — at 10k connections per instance, add
  a second replica (stateless rollout; WS is bound to a connection, not a session).

## Billing webhook queue

- Failed webhook? `SELECT * FROM billing_events WHERE processed_at > now() - interval '1 hour' ORDER BY processed_at DESC LIMIT 50`.
- Duplicates are expected (provider replays); look for event_types that
  should drive state transitions but aren't reflected in `subscriptions`.

## Escalation contacts

(Replace with real rotation.)

- **Primary**: oncall@torque.com.br
- **Engineering lead**: eng-lead@torque.com.br
- **Database admin**: dba@torque.com.br
- **Security lead**: security@torque.com.br (any auth/access anomaly)

## Post-incident

Within 48h of resolution:
1. Write a post-mortem in `.specs/incidents/YYYY-MM-DD-<slug>.md` using the template.
2. File tickets for every `Action Item` in the post-mortem.
3. Schedule the blameless review.
