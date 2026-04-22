/**
 * S49 / F.1 + S50 / F.2 — integrations hooks.
 * Google Calendar + TinyERP (S49), Meta Ads + SZ.Chat + lead webhook (S50).
 *
 * The backend exposes:
 *   GET    /integrations                        — list credentials (member)
 *   GET    /integrations/meta/ads-insights      — cached Meta insights (member)
 *   POST   /integrations/google/...             — OAuth + disconnect (admin)
 *   POST   /integrations/tinyerp/...            — connect + disconnect + push-order + sync-products (admin)
 *   POST   /integrations/meta + /disconnect     — system-user token (admin)
 *   POST   /integrations/szchat + /disconnect   — API key (admin)
 */

import { useQuery } from '@tanstack/react-query'

import { get, post } from '@/api/client'
import { useAppMutation } from '@/hooks/useAppMutation'

export type IntegrationProvider = 'google' | 'tinyerp' | 'meta' | 'szchat'

export interface IntegrationCredential {
  provider: IntegrationProvider
  connected: boolean
  external_account_id?: string | null
  scopes?: string[] | null
  expires_at?: string | null
  last_success_at?: string | null
  last_error_text?: string | null
  last_error_at?: string | null
}

export interface MetaAdCampaign {
  campaign_id: string
  name: string
  spend_cents: number
  impressions: number
  clicks: number
  leads: number
}

export interface MetaAdAccountInsights {
  account_id: string
  date_range: string
  currency?: string
  spend_cents: number
  impressions: number
  clicks: number
  leads: number
  cpl_cents: number // -1 when leads == 0
  campaigns?: MetaAdCampaign[]
  fetched_at: string
}

export interface TinyERPSyncResult {
  fetched: number
  inserted: number
  updated: number
  skipped: number
}

function keys() {
  return {
    list: ['integrations'] as const,
    metaInsights: (accountId: string | null, dateRange: string) =>
      ['integrations', 'meta', 'ads-insights', accountId ?? '', dateRange] as const,
  }
}

export function useIntegrations() {
  return useQuery({
    queryKey: keys().list,
    queryFn: () => get<{ data: IntegrationCredential[] }>('/integrations'),
    staleTime: 60_000,
    select: (r) => r.data,
  })
}

// ---------------- Google ----------------------------------------------

export function googleConnectURL(): string {
  return '/api/v1/integrations/google/connect'
}

export function useDisconnectGoogle() {
  return useAppMutation<void, void>(
    () => post<void>('/integrations/google/disconnect', {}),
    { invalidate: [keys().list], errorContext: 'Desconectar Google' }
  )
}

// ---------------- TinyERP --------------------------------------------

export function useConnectTinyERP() {
  return useAppMutation<void, string>(
    (apiKey: string) => post<void>('/integrations/tinyerp', { api_key: apiKey }),
    { invalidate: [keys().list], errorContext: 'Conectar TinyERP' }
  )
}

export function useDisconnectTinyERP() {
  return useAppMutation<void, void>(
    () => post<void>('/integrations/tinyerp/disconnect', {}),
    { invalidate: [keys().list], errorContext: 'Desconectar TinyERP' }
  )
}

export function useSyncTinyERPProducts() {
  return useAppMutation<TinyERPSyncResult, void>(
    () => post<TinyERPSyncResult>('/integrations/tinyerp/sync-products', {}),
    { invalidate: [keys().list, ['products']], errorContext: 'Sincronizar produtos TinyERP' }
  )
}

// ---------------- Meta Ads -------------------------------------------

export interface MetaConnectInput {
  access_token: string
  account_id?: string
}

export function useConnectMeta() {
  return useAppMutation<void, MetaConnectInput>(
    (in_: MetaConnectInput) => post<void>('/integrations/meta', in_),
    { invalidate: [keys().list], errorContext: 'Conectar Meta Ads' }
  )
}

export function useDisconnectMeta() {
  return useAppMutation<void, void>(
    () => post<void>('/integrations/meta/disconnect', {}),
    { invalidate: [keys().list], errorContext: 'Desconectar Meta Ads' }
  )
}

/**
 * useMetaAdsInsights queries the cached insights endpoint. accountId is
 * optional — the backend falls back to the credential's
 * external_account_id when missing. Pass `enabled=false` to keep the
 * hook suspended until the caller picks a date range.
 */
export function useMetaAdsInsights(params: {
  accountId?: string | null
  dateRange: string
  enabled?: boolean
}) {
  const { accountId, dateRange, enabled } = params
  return useQuery({
    queryKey: keys().metaInsights(accountId ?? null, dateRange),
    queryFn: () => {
      const qs = new URLSearchParams()
      if (accountId) qs.set('account_id', accountId)
      qs.set('date_range', dateRange)
      return get<MetaAdAccountInsights>(`/integrations/meta/ads-insights?${qs.toString()}`)
    },
    staleTime: 5 * 60_000, // backend caches 15min; UI refresh hint at 5min
    enabled: enabled !== false,
  })
}

// ---------------- SZ.Chat --------------------------------------------

export interface SZChatConnectInput {
  api_key: string
  channel_id?: string
}

export function useConnectSZChat() {
  return useAppMutation<void, SZChatConnectInput>(
    (in_: SZChatConnectInput) => post<void>('/integrations/szchat', in_),
    { invalidate: [keys().list], errorContext: 'Conectar SZ.Chat' }
  )
}

export function useDisconnectSZChat() {
  return useAppMutation<void, void>(
    () => post<void>('/integrations/szchat/disconnect', {}),
    { invalidate: [keys().list], errorContext: 'Desconectar SZ.Chat' }
  )
}
