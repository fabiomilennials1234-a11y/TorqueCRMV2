/**
 * F10 Equipe — membership + per-member permission overrides.
 *
 * Read surfaces are member-accessible (the team tab is visible to anyone in
 * the org). Mutations are admin-only — a non-admin calling POST/PATCH/DELETE
 * hits 403 which flows through `notifyAppError` as a toast.
 */

import { useQuery, useQueryClient } from '@tanstack/react-query'

import { del, get, patch, post, put } from '@/api/client'
import { useAppMutation } from '@/hooks/useAppMutation'
import { useWSSubscribe } from '@/hooks/useWSSubscribe'

export type MemberRole = 'admin' | 'membro'

export interface TeamMember {
  id: string
  user_id: string
  email: string
  display_name: string
  role: MemberRole
  avatar_url?: string | null
  is_active: boolean
  invited_by?: string | null
  invited_at?: string | null
  joined_at: string
  deactivated_at?: string | null
}

export interface MemberOverride {
  feature_key: string
  value: boolean
}

function memberKeys() {
  return {
    list: (includeInactive: boolean) => ['members', { includeInactive }] as const,
    detail: (id: string) => ['members', 'detail', id] as const,
    overrides: (id: string) => ['members', id, 'overrides'] as const,
  }
}

export function useMembers(includeInactive = false) {
  const client = useQueryClient()

  useWSSubscribe(
    ['member.created', 'member.updated', 'member.deactivated', 'member.permission_changed'],
    () => void client.invalidateQueries({ queryKey: ['members'] })
  )

  return useQuery<TeamMember[]>({
    queryKey: memberKeys().list(includeInactive),
    queryFn: async () => {
      const qs = includeInactive ? '?include_inactive=1' : ''
      return (await get<{ data: TeamMember[] }>(`/api/v1/members${qs}`)).data
    },
    staleTime: 60 * 1000,
  })
}

export function useMember(id: string | undefined) {
  return useQuery<TeamMember>({
    queryKey: id ? memberKeys().detail(id) : ['members', 'detail', 'disabled'],
    enabled: Boolean(id),
    queryFn: () => get<TeamMember>(`/api/v1/members/${id}`),
  })
}

export function useMemberOverrides(id: string | undefined) {
  return useQuery<MemberOverride[]>({
    queryKey: id ? memberKeys().overrides(id) : ['members', 'overrides', 'disabled'],
    enabled: Boolean(id),
    queryFn: async () =>
      (await get<{ data: MemberOverride[] }>(`/api/v1/members/${id}/permissions`)).data,
  })
}

// -------- mutations --------------------------------------------------

export function useAddMember() {
  return useAppMutation<TeamMember, { email: string; display_name: string; role: MemberRole }>(
    (body) => post<TeamMember>('/api/v1/members', body),
    {
      invalidate: [['members']],
      errorContext: 'member.add',
    }
  )
}

export function useUpdateMember(id: string) {
  return useAppMutation<
    TeamMember,
    {
      display_name?: string
      role?: MemberRole
      avatar_url?: string | null
      is_active?: boolean
    }
  >((body) => patch<TeamMember>(`/api/v1/members/${id}`, body), {
    invalidate: [['members']],
    errorContext: 'member.update',
  })
}

export function useDeactivateMember(id: string) {
  return useAppMutation<void, void>(() => del<void>(`/api/v1/members/${id}`), {
    invalidate: [['members']],
    errorContext: 'member.deactivate',
  })
}

export function useSetMemberPermission(id: string) {
  return useAppMutation<void, { feature_key: string; value: boolean }>(
    ({ feature_key, value }) =>
      put<void>(`/api/v1/members/${id}/permissions/${feature_key}`, { value }),
    {
      invalidate: [['members', id, 'overrides'], ['members']],
      errorContext: 'member.permission_set',
    }
  )
}

export function useClearMemberPermission(id: string) {
  return useAppMutation<void, { feature_key: string }>(
    ({ feature_key }) => del<void>(`/api/v1/members/${id}/permissions/${feature_key}`),
    {
      invalidate: [['members', id, 'overrides'], ['members']],
      errorContext: 'member.permission_clear',
    }
  )
}

export { memberKeys }
