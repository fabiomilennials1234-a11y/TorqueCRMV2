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
const authRate = new Rate('auth_ok')

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

  // 3. Dashboard-equivalent mix: health + leads list + pipes
  const dashboardStart = Date.now()
  const batch = http.batch([
    { method: 'GET', url: url('/api/v1/leads?page_size=25'), headers, tags: { name: 'leads_list' } },
    { method: 'GET', url: url('/api/v1/pipes'), headers, tags: { name: 'pipes_list' } },
    { method: 'GET', url: url('/api/v1/analytics/leads?since=2026-03-01T00:00:00Z'), headers, tags: { name: 'analytics_leads' } },
  ])
  dashboardLatency.add(Date.now() - dashboardStart)
  leadListLatency.add(batch[0].timings.duration)

  check(batch[0], { 'leads 200': (r) => r.status === 200 })
  check(batch[1], { 'pipes 200': (r) => r.status === 200 })

  // 4. 5% mix: write path (create a task on self) — skips if CSRF absent
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
