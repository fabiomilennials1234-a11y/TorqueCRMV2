/**
 * Dev-only session bypass.
 *
 * When the app is built in Vite dev mode (`import.meta.env.DEV === true`) AND
 * the explicit opt-in `VITE_DEV_AUTH=1` is set, AuthProvider skips the
 * `/api/v1/auth/me` fetch and injects this deterministic session instead.
 *
 * This exists so the UI can be navigated without a running backend. Every
 * read/write that hits the network will STILL fail (there is no server to
 * answer), but the shell, routing, and every purely-visual page become
 * walkable.
 *
 * Guards:
 *   1. `import.meta.env.DEV` is `false` in any production build — Vite strips
 *      the branch at tree-shake time, so the mock cannot ship.
 *   2. `VITE_DEV_AUTH` must be exactly `"1"` or `"true"`. Default is off.
 *
 * The session mirrors the shape that `/auth/me` → `toSessionBundle` produces,
 * with every permission granted so no PermissionGate blocks the walk.
 */

import type { SessionBundle } from '@/contracts/manual'

const FEATURE_KEYS = [
  'pipeline.view',
  'pipeline.edit',
  'inbox.view',
  'inbox.reply',
  'workflows.view',
  'workflows.edit',
  'campaigns.view',
  'campaigns.edit',
  'copilot.view',
  'copilot.configure',
  'analytics.view',
  'settings.view',
  'settings.edit',
  'ui.view_manager_mode',
  'ui.view_salesperson_mode',
  'leads.view',
  'leads.create',
  'leads.update',
  'leads.delete',
  'leads.assign',
  'tags.manage',
] as const

function allPermissions(): Record<string, boolean> {
  const out: Record<string, boolean> = {}
  for (const k of FEATURE_KEYS) out[k] = true
  return out
}

/** Deterministic dev session. Matches the seeded backend admin for parity. */
export const DEV_SESSION: SessionBundle = {
  user: {
    id: '00000000-0000-0000-0000-0000000000b1',
    email: 'marcelo@gmail.com',
    displayName: 'Marcelo Montemezzo (dev)',
  },
  org: {
    id: '00000000-0000-0000-0000-0000000000a1',
    name: 'Torque Dev',
    slug: 'torque-dev',
    planId: 'enterprise',
    paymentStatus: 'active',
    logoUrl: null,
  },
  role: 'admin',
  isMaster: false,
  featurePermissions: allPermissions(),
  quotas: {},
}

/**
 * Read the runtime toggle. Kept as a function so consumers can re-check in
 * tests after a `vi.stubEnv` call; the branch is constant-folded in prod.
 */
export function isDevAuthEnabled(): boolean {
  if (!import.meta.env.DEV) return false
  const raw = (import.meta.env.VITE_DEV_AUTH as string | undefined)?.toLowerCase?.()
  return raw === '1' || raw === 'true' || raw === 'yes' || raw === 'on'
}
