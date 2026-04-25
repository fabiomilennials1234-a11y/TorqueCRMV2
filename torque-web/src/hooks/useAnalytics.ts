/**
 * F09 Analytics hooks — read-only summaries over the domain tables.
 *
 * Every query requires an explicit `since` (ISO-8601). The backend refuses
 * windowless queries to prevent full-table scans on large tenants.
 */

import { useQuery } from '@tanstack/react-query'

import { get } from '@/api/client'

export interface AnalyticsWindow {
  since: string // ISO-8601
  until?: string // ISO-8601; defaults to now() on the server
}

export interface LeadsSummary {
  total: number
  in_window: number
  assigned: number
  unassigned: number
}

export interface MessagesSummary {
  inbound: number
  outbound: number
}

export interface TasksSummary {
  pending: number
  in_progress: number
  done_in_window: number
  missed_in_window: number
  overdue: number
}

export interface ProposalsSummary {
  sent_in_window: number
  viewed_in_window: number
  accepted_in_window: number
  rejected_in_window: number
  won_amount_cents: number
}

export interface StageVolume {
  pipe_id: string
  stage_id: string
  count: number
}

export interface MemberStats {
  member_id: string
  leads_assigned: number
  tasks_completed_in_window: number
  proposals_accepted_in_window: number
}

function buildQS(w: AnalyticsWindow): string {
  const qs = new URLSearchParams()
  qs.set('since', w.since)
  if (w.until) qs.set('until', w.until)
  return qs.toString()
}

export function useLeadsSummary(w: AnalyticsWindow) {
  return useQuery<LeadsSummary>({
    queryKey: ['analytics', 'leads', w],
    enabled: Boolean(w.since),
    queryFn: () => get<LeadsSummary>(`/api/v1/analytics/leads?${buildQS(w)}`),
    staleTime: 60 * 1000,
  })
}

export function useMessagesSummary(w: AnalyticsWindow) {
  return useQuery<MessagesSummary>({
    queryKey: ['analytics', 'messages', w],
    enabled: Boolean(w.since),
    queryFn: () => get<MessagesSummary>(`/api/v1/analytics/messages?${buildQS(w)}`),
    staleTime: 60 * 1000,
  })
}

export function useTasksSummary(w: AnalyticsWindow) {
  return useQuery<TasksSummary>({
    queryKey: ['analytics', 'tasks', w],
    enabled: Boolean(w.since),
    queryFn: () => get<TasksSummary>(`/api/v1/analytics/tasks?${buildQS(w)}`),
    staleTime: 60 * 1000,
  })
}

export function useProposalsSummary(w: AnalyticsWindow) {
  return useQuery<ProposalsSummary>({
    queryKey: ['analytics', 'proposals', w],
    enabled: Boolean(w.since),
    queryFn: () => get<ProposalsSummary>(`/api/v1/analytics/proposals?${buildQS(w)}`),
    staleTime: 60 * 1000,
  })
}

export function useStageVolume(pipeId: string | undefined) {
  return useQuery<StageVolume[]>({
    queryKey: pipeId
      ? ['analytics', 'stage-volume', pipeId]
      : ['analytics', 'stage-volume', 'disabled'],
    enabled: Boolean(pipeId),
    queryFn: async () =>
      (await get<{ data: StageVolume[] }>(`/api/v1/analytics/pipes/${pipeId}/stages`)).data,
    staleTime: 30 * 1000,
  })
}

export function useLeaderboard(w: AnalyticsWindow) {
  return useQuery<MemberStats[]>({
    queryKey: ['analytics', 'leaderboard', w],
    enabled: Boolean(w.since),
    queryFn: async () =>
      (await get<{ data: MemberStats[] }>(`/api/v1/analytics/leaderboard?${buildQS(w)}`)).data,
    staleTime: 60 * 1000,
  })
}
