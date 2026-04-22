/**
 * S49 / Fase F.1 — integrations hooks. Google Calendar + TinyERP.
 * Full UI (IntegrationsSection rich surface, disconnect confirm, tinyerp
 * modal) lands in S50 — this module exposes just the read hook + the
 * mutations the settings tab needs today.
 */

import { useQuery } from '@tanstack/react-query'

import { get, post } from '@/api/client'
import { useAppMutation } from '@/hooks/useAppMutation'

export type IntegrationProvider = 'google' | 'tinyerp' | 'meta'

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

function keys() {
  return {
    list: ['integrations'] as const,
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

/**
 * googleConnectURL returns the same-origin path that kicks off the OAuth
 * redirect. The caller sets `window.location.href` — a full navigation is
 * required because the backend responds with a 302 to Google, and the
 * browser must follow it in the top window.
 */
export function googleConnectURL(): string {
  return '/api/v1/integrations/google/connect'
}

export function useDisconnectGoogle() {
  return useAppMutation<void, void>(
    () => post<void>('/integrations/google/disconnect', {}),
    { invalidate: [keys().list], errorContext: 'Desconectar Google' }
  )
}

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
