/**
 * S51 / Fase G.1 — Quota observability hooks.
 *
 * Reads GET /quotas (list) + GET /quotas/:resource (detail). Writes
 * (PATCH /quotas/:resource) are master-only and kept out of this hook
 * file on purpose — master surfaces live in features/master/.
 *
 * The quota cap is the backstop, not the UI — pages read these hooks
 * to render gentle upsell nudges ("80 de 100 leads usados") before the
 * 402 lands.
 */

import { useQuery } from '@tanstack/react-query'

import { get } from '@/api/client'

export type QuotaResource = string

/** Well-known keys — mirrors the Go quotarepo.Resource* constants. */
export const KnownQuotaResource = {
  Leads: 'leads',
  TeamMembers: 'team_members',
  Workflows: 'workflows',
  Agents: 'agents',
} as const

export interface Quota {
  resource: QuotaResource
  plan_base: number
  purchased_addons: number
  admin_adjustment: number
  current_usage: number
  effective_limit: number
  remaining: number
}

function keys() {
  return {
    list: ['quotas'] as const,
    detail: (resource: QuotaResource) => ['quotas', resource] as const,
  }
}

/**
 * useQuotas lists every (resource → quota) tuple for the tenant. Used
 * by the Plan & Billing settings tab.
 */
export function useQuotas() {
  return useQuery({
    queryKey: keys().list,
    queryFn: () => get<{ data: Quota[] }>('/quotas'),
    staleTime: 60_000,
    select: (r) => r.data,
  })
}

/**
 * useQuota(resource) — single-resource accessor used by resource
 * pages (Leads page, Workflows page, etc) to render an upsell hint.
 */
export function useQuota(resource: QuotaResource) {
  return useQuery({
    queryKey: keys().detail(resource),
    queryFn: () => get<Quota>(`/quotas/${encodeURIComponent(resource)}`),
    staleTime: 60_000,
  })
}
