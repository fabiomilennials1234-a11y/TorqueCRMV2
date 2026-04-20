# Oncall Basics

A concise reference for anyone picking up oncall rotation for the first time.

## Access prerequisites

- GitHub access to the Torque-v2 repo with `write` on `develop`.
- Environment access: GitHub Environment `staging` and `production` (required
  reviewers list includes you).
- Managed Postgres dashboard bookmark.
- Sentry project access.
- EasyPanel admin access.
- Slack `#torque-oncall` channel.

## Daily health check (5 minutes, start of shift)

1. Sentry — any new issue introduced overnight?
2. `/healthz` and `/readyz` on every environment — green?
3. `pg_stat_activity` — any long-running query > 5 min?
4. `operations` table — backlog of `pending` older than 1 hour?
5. `billing_events` — failed event types in the last 24h?

## Metrics to watch (dashboards)

- p95 request latency per route (target < 500ms).
- Error rate (target < 0.1% for GET paths, < 0.5% for writes).
- 429 rate limit hits per minute (sustained spike = runaway client).
- WS connection count per tenant (spike = client reconnect loop).
- DB connection pool utilization (alert at >80%).

## Common symptoms → first checks

| Symptom | First check |
|---------|-------------|
| Users can't log in | `/auth/login` in Sentry; JWT_SECRET env var present. |
| Dashboard blank | `/api/v1/auth/me` returning 401? Session cookie expired. |
| Kanban stuck | WS connection alive? Check hub logs. |
| Messages not sending | Billing webhook alive? Worker queue `campaign.dispatch` running? |
| Copilot silent | `agents.kill_switch` enabled by an admin? |
| Master impersonation fails | `audit_log` writable? Check DB write perms. |

## Quick SQL queries

```sql
-- Last 20 audit rows (any org)
SELECT created_at, organization_id, actor_type, action, entity_type
  FROM audit_log
 ORDER BY created_at DESC
 LIMIT 20;

-- Tenants with stuck pending subs older than 1 hour
SELECT s.id, s.organization_id, o.name, s.created_at
  FROM subscriptions s JOIN organizations o ON o.id = s.organization_id
 WHERE s.status = 'pending'
   AND s.created_at < now() - interval '1 hour'
 ORDER BY s.created_at;

-- Failed operations in last 24h grouped by kind
SELECT kind, COUNT(*) AS cnt
  FROM operations
 WHERE status = 'failed' AND ended_at > now() - interval '24 hours'
 GROUP BY kind
 ORDER BY cnt DESC;

-- Orgs with no active admins (impersonation would refuse)
SELECT o.id, o.slug
  FROM organizations o
  LEFT JOIN team_members tm ON tm.organization_id = o.id
       AND tm.role = 'admin' AND tm.is_active = true
 WHERE tm.id IS NULL AND o.deleted_at IS NULL;
```

## Do not / Always

- **Never** run destructive SQL (`DELETE`/`UPDATE` without a `WHERE organization_id`) on a live tenant — wrong tenant, wrong rows, wrong day.
- **Never** bypass the CI pipeline to deploy a "hotfix" image that wasn't built from a tag.
- **Always** prefer rolling back the deploy (re-dispatch with previous tag) over patching a prod config directly.
- **Always** leave a one-line summary in `#torque-oncall` when you take an action, even if it resolved itself.
