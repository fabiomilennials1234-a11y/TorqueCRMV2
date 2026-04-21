/**
 * F09 Performance hooks — ranking, goals, commissions, awards (S47).
 *
 * All reads are member-accessible; writes are admin-only on the backend.
 * WS invalidation fires on goal.created / commission.updated / award.created
 * so multiple tabs stay in sync without polling.
 */

import { useQuery, useQueryClient } from '@tanstack/react-query'

import { get, post } from '@/api/client'
import { useAppMutation } from '@/hooks/useAppMutation'
import { useWSSubscribe } from '@/hooks/useWSSubscribe'

export type GoalMetric =
  | 'deals_won'
  | 'revenue_cents'
  | 'leads_contacted'
  | 'response_time_ms'
  | 'first_response_minutes'

export type CommissionStatus = 'pending' | 'approved' | 'paid' | 'cancelled'

export interface RankingEntry {
  member_id: string
  member_name: string
  deals_won: number
  revenue_cents: number
}

export interface Goal {
  id: string
  member_id?: string | null
  metric: GoalMetric
  target: number
  period_start: string
  period_end: string
  created_at: string
}

export interface Commission {
  id: string
  proposal_id?: string | null
  member_id: string
  percentage: number
  amount_cents: number
  currency: string
  status: CommissionStatus
  earned_at: string
  approved_at?: string | null
  paid_at?: string | null
  notes?: string | null
}

export interface Award {
  id: string
  title: string
  description?: string | null
  criteria: unknown
  winners: unknown
  awarded_at?: string | null
  created_at: string
}

function keys() {
  return {
    ranking: (since: string, until: string) =>
      ['performance', 'ranking', since, until] as const,
    goals: () => ['performance', 'goals'] as const,
    commissions: (memberId?: string) =>
      memberId
        ? (['performance', 'commissions', memberId] as const)
        : (['performance', 'commissions'] as const),
    awards: () => ['performance', 'awards'] as const,
  }
}

// -------- ranking ---------------------------------------------------

export function useRanking(window: { since: string; until: string } | undefined) {
  return useQuery<{ data: RankingEntry[]; since: string; until: string }>({
    queryKey: window
      ? keys().ranking(window.since, window.until)
      : ['performance', 'ranking', 'disabled'],
    enabled: Boolean(window),
    queryFn: () => {
      const qs = new URLSearchParams({
        since: window!.since,
        until: window!.until,
      }).toString()
      return get<{ data: RankingEntry[]; since: string; until: string }>(
        `/api/v1/performance/ranking?${qs}`
      )
    },
    staleTime: 60 * 1000,
  })
}

// -------- goals -----------------------------------------------------

export function useGoals() {
  const client = useQueryClient()
  useWSSubscribe(['goal.created'], () =>
    void client.invalidateQueries({ queryKey: keys().goals() })
  )
  return useQuery<Goal[]>({
    queryKey: keys().goals(),
    queryFn: async () => (await get<{ data: Goal[] }>('/api/v1/performance/goals')).data,
    staleTime: 30 * 1000,
  })
}

export function useCreateGoal() {
  return useAppMutation<
    Goal,
    {
      member_id?: string
      metric: GoalMetric
      target: number
      period_start: string
      period_end: string
    }
  >((body) => post<Goal>('/api/v1/performance/goals', body), {
    invalidate: [['performance', 'goals']],
    errorContext: 'performance.goal.create',
  })
}

// -------- commissions -----------------------------------------------

export function useCommissions(memberId?: string) {
  const client = useQueryClient()
  useWSSubscribe(['commission.updated'], () =>
    void client.invalidateQueries({ queryKey: ['performance', 'commissions'] })
  )
  return useQuery<Commission[]>({
    queryKey: keys().commissions(memberId),
    queryFn: async () => {
      const qs = memberId ? `?member_id=${memberId}` : ''
      return (await get<{ data: Commission[] }>(`/api/v1/performance/commissions${qs}`)).data
    },
    staleTime: 30 * 1000,
  })
}

export function useSetCommissionStatus(commissionId: string) {
  return useAppMutation<void, { status: CommissionStatus }>(
    (body) =>
      post<void>(`/api/v1/performance/commissions/${commissionId}/status`, body),
    {
      invalidate: [['performance', 'commissions']],
      errorContext: 'performance.commission.status',
    }
  )
}

// -------- awards ----------------------------------------------------

export function useAwards() {
  const client = useQueryClient()
  useWSSubscribe(['award.created'], () =>
    void client.invalidateQueries({ queryKey: keys().awards() })
  )
  return useQuery<Award[]>({
    queryKey: keys().awards(),
    queryFn: async () => (await get<{ data: Award[] }>('/api/v1/performance/awards')).data,
    staleTime: 30 * 1000,
  })
}

export function useCreateAward() {
  return useAppMutation<
    Award,
    { title: string; description?: string; criteria?: unknown; winners?: unknown; awarded_at?: string }
  >((body) => post<Award>('/api/v1/performance/awards', body), {
    invalidate: [['performance', 'awards']],
    errorContext: 'performance.award.create',
  })
}
