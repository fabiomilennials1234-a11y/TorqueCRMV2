/**
 * F15 Settings — organization profile + webhooks + notification prefs.
 *
 * Admin-only mutations flow through useAppMutation so 403s show up as a
 * toast. GET /organization is member-accessible.
 */

import { useQuery } from '@tanstack/react-query'

import { del, get, patch, post, put } from '@/api/client'
import { useAppMutation } from '@/hooks/useAppMutation'

export interface Organization {
  id: string
  slug: string
  name: string
  legal_name?: string | null
  cnpj?: string | null
  timezone?: string | null
  logo_url?: string | null
  updated_at: string
}

export interface WebhookEndpoint {
  id: string
  url: string
  description?: string | null
  event_types: string[]
  is_active: boolean
  last_success_at?: string | null
  last_failure_at?: string | null
  last_error?: string | null
  created_at: string
}

export interface NotificationPref {
  channel: 'email' | 'in_app' | 'push'
  topic: string
  enabled: boolean
}

export function useOrganization() {
  return useQuery<Organization>({
    queryKey: ['settings', 'organization'],
    queryFn: () => get<Organization>('/api/v1/organization'),
    staleTime: 60 * 1000,
  })
}

export function useUpdateOrganization() {
  return useAppMutation<
    Organization,
    {
      name?: string
      legal_name?: string
      cnpj?: string
      timezone?: string
      logo_url?: string
    }
  >((body) => patch<Organization>('/api/v1/organization', body), {
    invalidate: [['settings', 'organization']],
    errorContext: 'organization.update',
  })
}

export function useWebhooks() {
  return useQuery<WebhookEndpoint[]>({
    queryKey: ['settings', 'webhooks'],
    queryFn: async () =>
      (await get<{ data: WebhookEndpoint[] }>('/api/v1/webhooks')).data,
    staleTime: 60 * 1000,
  })
}

export function useCreateWebhook() {
  return useAppMutation<
    WebhookEndpoint,
    { url: string; description?: string; event_types: string[]; secret: string }
  >((body) => post<WebhookEndpoint>('/api/v1/webhooks', body), {
    invalidate: [['settings', 'webhooks']],
    errorContext: 'webhook.create',
  })
}

export function useUpdateWebhook(id: string) {
  return useAppMutation<
    WebhookEndpoint,
    { description?: string; event_types?: string[]; is_active?: boolean }
  >((body) => patch<WebhookEndpoint>(`/api/v1/webhooks/${id}`, body), {
    invalidate: [['settings', 'webhooks']],
    errorContext: 'webhook.update',
  })
}

export function useDeleteWebhook(id: string) {
  return useAppMutation<void, void>(
    () => del<void>(`/api/v1/webhooks/${id}`),
    { invalidate: [['settings', 'webhooks']], errorContext: 'webhook.delete' }
  )
}

export function useNotificationPrefs() {
  return useQuery<NotificationPref[]>({
    queryKey: ['settings', 'notifications'],
    queryFn: async () =>
      (await get<{ data: NotificationPref[] }>('/api/v1/me/notifications')).data,
    staleTime: 60 * 1000,
  })
}

export function useSetNotificationPref() {
  return useAppMutation<void, NotificationPref>(
    (body) => put<void>('/api/v1/me/notifications', body),
    { invalidate: [['settings', 'notifications']], errorContext: 'notifications.set' }
  )
}
