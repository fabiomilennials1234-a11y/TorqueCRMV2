/**
 * F16 Master Admin — cross-org surfaces. Every hook here hits master-only
 * endpoints; non-master callers get 403 which the client toasts globally.
 */

import { useQuery } from '@tanstack/react-query'

import { get, post } from '@/api/client'
import { useAppMutation } from '@/hooks/useAppMutation'

export interface MasterOrg {
  id: string
  slug: string
  name: string
  plan_id?: string | null
  payment_status: string
  member_count: number
  lead_count: number
  created_at: string
}

export interface SystemHealth {
  org_count: number
  active_org_count: number
  user_count: number
  lead_count: number
  active_subscriptions: number
  pending_subscriptions: number
  operations_running: number
  operations_failed_24h: number
}

export interface ImpersonationTarget {
  organization_id: string
  team_member_id: string
  user_id: string
  expires_at: string
}

export function useSystemHealth() {
  return useQuery<SystemHealth>({
    queryKey: ['master', 'health'],
    queryFn: () => get<SystemHealth>('/api/v1/master/health'),
    staleTime: 30 * 1000,
  })
}

export function useMasterOrganizations() {
  return useQuery<MasterOrg[]>({
    queryKey: ['master', 'organizations'],
    queryFn: async () =>
      (await get<{ data: MasterOrg[] }>('/api/v1/master/organizations')).data,
    staleTime: 60 * 1000,
  })
}

export function useMasterOrganization(id: string | undefined) {
  return useQuery<MasterOrg>({
    queryKey: id ? ['master', 'organizations', id] : ['master', 'organizations', 'disabled'],
    enabled: Boolean(id),
    queryFn: () => get<MasterOrg>(`/api/v1/master/organizations/${id}`),
  })
}

export function useImpersonate() {
  return useAppMutation<ImpersonationTarget, { organization_id: string }>(
    ({ organization_id }) =>
      post<ImpersonationTarget>(`/api/v1/master/organizations/${organization_id}/impersonate`, {}),
    { errorContext: 'master.impersonate' }
  )
}
