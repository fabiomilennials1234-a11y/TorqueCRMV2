// Load test baseline for Torque API (k6 >= 0.50).
//
// Usage:
//   BASE_URL=https://staging.torque.com.br \
//   TORQUE_SESSION_COOKIE='your-session-cookie' \
//   TORQUE_CSRF_TOKEN='your-csrf' \
//   k6 run api-baseline.k6.js
//
// Target: p95 < 500ms under 100 concurrent users for 5 minutes on the
// hottest read paths. Write paths are exercised at 5% of the mix; the
// load test is read-dominant by design because write paths are
// rate-limited and would skew the p95 upward for reasons unrelated to
// latency regressions we're actually watching.

import http from 'k6/http'
import { sleep, check } from 'k6'
import { Trend, Rate } from 'k6/metrics'

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080'
const SESSION = __ENV.TORQUE_SESSION_COOKIE || ''
const CSRF = __ENV.TORQUE_CSRF_TOKEN || ''

export const options = {
  // Ramp to 100 VUs, hold 5 min, ramp down. Standard p95 regression gate.
  stages: [
    { duration: '1m', target: 50 },
    { duration: '5m', target: 100 },
    { duration: '1m', target: 0 },
  ],
  thresholds: {
    // Hard gates — the run fails if any of these trip.
    http_req_duration: ['p(95)<500', 'p(99)<1500'],
    http_req_failed: ['rate<0.01'], // < 1% error rate
    checks: ['rate>0.99'],
  },
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)'],
}

const dashboardLatency = new Trend('dashboard_latency_ms', true)
const leadListLatency = new Trend('lead_list_latency_ms', true)
const quotaLookupLatency = new Trend('quota_lookup_latency_ms', true)
const metaInsightsLatency = new Trend('meta_insights_latency_ms', true)
const authRate = new Rate('auth_ok')
const quotaOKRate = new Rate('quota_lookup_ok')

function authHeaders() {
  const h = { 'Content-Type': 'application/json' }
  if (CSRF) h['X-CSRF-Token'] = CSRF
  return h
}

function cookieJar() {
  const jar = http.cookieJar()
  if (SESSION) {
    jar.set(BASE_URL, '__torque_session', SESSION, { path: '/', secure: BASE_URL.startsWith('https'), httpOnly: true })
  }
  return jar
}

export default function () {
  const url = (p) => `${BASE_URL}${p}`
  const headers = authHeaders()

  // 1. Bootstrap (unauthenticated, cacheable)
  const boot = http.get(url('/api/bootstrap'), { tags: { name: 'bootstrap' } })
  check(boot, { 'bootstrap 200': (r) => r.status === 200 })

  // 2. /me to keep session warm
  const me = http.get(url('/api/v1/auth/me'), { headers, tags: { name: 'me' } })
  authRate.add(me.status === 200)
  if (me.status !== 200) {
    sleep(1)
    return
  }

  // 3. Dashboard-equivalent mix: health + leads list + pipes + quotas + integrations
  //    S52 — added /quotas (S51) and /integrations (S49) to the hot-path batch
  //    so the load profile reflects what the frontend dashboard actually
  //    fetches on Plano e Faturamento + Integrações tabs.
  const dashboardStart = Date.now()
  const batch = http.batch([
    { method: 'GET', url: url('/api/v1/leads?page_size=25'), headers, tags: { name: 'leads_list' } },
    { method: 'GET', url: url('/api/v1/pipes'), headers, tags: { name: 'pipes_list' } },
    { method: 'GET', url: url('/api/v1/analytics/leads?since=2026-03-01T00:00:00Z'), headers, tags: { name: 'analytics_leads' } },
    { method: 'GET', url: url('/api/v1/quotas'), headers, tags: { name: 'quotas_list' } },
    { method: 'GET', url: url('/api/v1/integrations'), headers, tags: { name: 'integrations_list' } },
  ])
  dashboardLatency.add(Date.now() - dashboardStart)
  leadListLatency.add(batch[0].timings.duration)
  quotaLookupLatency.add(batch[3].timings.duration)

  check(batch[0], { 'leads 200': (r) => r.status === 200 })
  check(batch[1], { 'pipes 200': (r) => r.status === 200 })
  check(batch[2], { 'analytics 200': (r) => r.status === 200 })
  check(batch[3], { 'quotas 200': (r) => r.status === 200 })
  check(batch[4], { 'integrations 200': (r) => r.status === 200 })
  quotaOKRate.add(batch[3].status === 200)

  // 4. 10% mix: Meta Ads insights (cache-hit path — S50 backend serves
  //    from meta_insights_cache; we want the p95 for the cached read).
  if (Math.random() < 0.1) {
    const insightsStart = Date.now()
    const insights = http.get(
      url('/api/v1/integrations/meta/ads-insights?date_range=last_7d'),
      { headers, tags: { name: 'meta_insights' } }
    )
    metaInsightsLatency.add(Date.now() - insightsStart)
    // 200 (cached) or 412 (not connected) are both expected success
    // shapes; other statuses are regressions.
    check(insights, {
      'meta insights ok or not connected': (r) => r.status === 200 || r.status === 412,
    })
  }

  // 5. 5% mix: quota-protected write path (POST /leads). The middleware
  //    runs a DB roundtrip before the handler — we're measuring the
  //    cost of admission, not the lead repo insert itself.
  //    Skips when CSRF absent (unauthenticated runs).
  if (CSRF && Math.random() < 0.05) {
    http.post(
      url('/api/v1/leads'),
      JSON.stringify({
        name: 'Lead k6 ' + Math.floor(Math.random() * 1e9),
        phone: '+5511999990000',
      }),
      { headers, tags: { name: 'lead_create_quota' } }
    )
  }

  // 6. 5% mix: create a task on self (pre-S50 baseline retained so
  //    we can diff p95 against pre-hardening runs).
  if (CSRF && Math.random() < 0.05) {
    http.post(
      url('/api/v1/tasks'),
      JSON.stringify({ title: 'loadtest', assigned_to_self: true }),
      { headers, tags: { name: 'task_create' } }
    )
  }

  sleep(1)
  // Silence unused-var linter for cookieJar while the function stays ready
  // for auth flows that need it.
  void cookieJar
}
