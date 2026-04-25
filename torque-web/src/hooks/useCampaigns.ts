/**
 * F08 Campaign hooks — bulk outbound messaging.
 *
 * Admin-only surfaces backed by S19 endpoints. The WS stats are updated by
 * the campaign worker (kind `campaign.dispatch`) as each recipient is sent
 * or fails, so the progress bar updates live.
 */

import { useQuery, useQueryClient } from '@tanstack/react-query'

import { get, post } from '@/api/client'
import { useAppMutation } from '@/hooks/useAppMutation'
import { useWSSubscribe } from '@/hooks/useWSSubscribe'

export type CampaignStatus =
  | 'draft'
  | 'scheduled'
  | 'running'
  | 'paused'
  | 'completed'
  | 'cancelled'

export interface Campaign {
  id: string
  name: string
  description?: string | null
  channel_id?: string | null
  template_body: string
  status: CampaignStatus
  scheduled_at?: string | null
  started_at?: string | null
  ended_at?: string | null
  stats_queued: number
  stats_sent: number
  stats_failed: number
  stats_skipped: number
}

export interface Recipient {
  id: string
  lead_id: string
  status: 'queued' | 'sent' | 'failed' | 'opted_out' | 'skipped'
  message_id?: string | null
  sent_at?: string | null
  failed_at?: string | null
}

function keys() {
  return {
    list: (status?: CampaignStatus) =>
      status ? (['campaigns', { status }] as const) : (['campaigns'] as const),
    detail: (id: string) => ['campaigns', id] as const,
    recipients: (id: string) => ['campaigns', id, 'recipients'] as const,
  }
}

export function useCampaigns(status?: CampaignStatus) {
  const client = useQueryClient()

  useWSSubscribe(
    [
      'campaign.created',
      'campaign.launched',
      'campaign.paused',
      'campaign.resumed',
      'campaign.cancelled',
    ],
    () => void client.invalidateQueries({ queryKey: ['campaigns'] })
  )

  return useQuery<Campaign[]>({
    queryKey: keys().list(status),
    queryFn: async () => {
      const qs = status ? `?status=${status}` : ''
      return (await get<{ data: Campaign[] }>(`/api/v1/campaigns${qs}`)).data
    },
    staleTime: 30 * 1000,
  })
}

export function useCampaign(id: string | undefined) {
  return useQuery<Campaign>({
    queryKey: id ? keys().detail(id) : ['campaigns', 'disabled'],
    enabled: Boolean(id),
    queryFn: () => get<Campaign>(`/api/v1/campaigns/${id}`),
  })
}

export function useRecipients(campaignId: string | undefined) {
  return useQuery<Recipient[]>({
    queryKey: campaignId ? keys().recipients(campaignId) : ['campaigns', 'recipients', 'disabled'],
    enabled: Boolean(campaignId),
    queryFn: async () =>
      (await get<{ data: Recipient[] }>(`/api/v1/campaigns/${campaignId}/recipients`)).data,
  })
}

export function useCreateCampaign() {
  return useAppMutation<
    Campaign,
    {
      name: string
      description?: string
      channel_id?: string
      template_body: string
      audience_query?: unknown
      scheduled_at?: string
    }
  >((body) => post<Campaign>('/api/v1/campaigns', body), {
    invalidate: [['campaigns']],
    errorContext: 'campaign.create',
  })
}

export function useLaunchCampaign(id: string) {
  return useAppMutation<{ queued: number }, { lead_ids: string[] }>(
    (body) => post<{ queued: number }>(`/api/v1/campaigns/${id}/launch`, body),
    { invalidate: [['campaigns']], errorContext: 'campaign.launch' }
  )
}

export function usePauseCampaign(id: string) {
  return useAppMutation<void, void>(() => post<void>(`/api/v1/campaigns/${id}/pause`, {}), {
    invalidate: [['campaigns']],
    errorContext: 'campaign.pause',
  })
}

export function useResumeCampaign(id: string) {
  return useAppMutation<void, void>(() => post<void>(`/api/v1/campaigns/${id}/resume`, {}), {
    invalidate: [['campaigns']],
    errorContext: 'campaign.resume',
  })
}

export function useCancelCampaign(id: string) {
  return useAppMutation<void, void>(() => post<void>(`/api/v1/campaigns/${id}/cancel`, {}), {
    invalidate: [['campaigns']],
    errorContext: 'campaign.cancel',
  })
}
