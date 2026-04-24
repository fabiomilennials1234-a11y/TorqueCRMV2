/**
 * F13 Agenda hooks (S48) — meetings CRUD.
 */

import { useQuery, useQueryClient } from '@tanstack/react-query'

import { del, get, post } from '@/api/client'
import { useAppMutation } from '@/hooks/useAppMutation'
import { useWSSubscribe } from '@/hooks/useWSSubscribe'

export type MeetingStatus = 'scheduled' | 'completed' | 'cancelled' | 'no_show'

export interface Meeting {
  id: string
  title: string
  description?: string | null
  starts_at: string
  ends_at: string
  location?: string | null
  lead_id?: string | null
  owner_member_id?: string | null
  status: MeetingStatus
  external_provider?: string | null
  external_id?: string | null
  created_at: string
  updated_at: string
}

function keys() {
  return {
    range: (from: string, to: string) => ['meetings', from, to] as const,
    detail: (id: string) => ['meetings', id] as const,
  }
}

export function useMeetings(window: { from: string; to: string } | undefined) {
  const client = useQueryClient()
  useWSSubscribe(
    ['meeting.created', 'meeting.updated', 'meeting.deleted'],
    () => void client.invalidateQueries({ queryKey: ['meetings'] })
  )
  return useQuery<{ data: Meeting[]; from: string; to: string }>({
    queryKey: window ? keys().range(window.from, window.to) : ['meetings', 'disabled'],
    enabled: Boolean(window),
    queryFn: () => {
      const qs = new URLSearchParams({ from: window!.from, to: window!.to }).toString()
      return get<{ data: Meeting[]; from: string; to: string }>(`/api/v1/meetings?${qs}`)
    },
    staleTime: 30 * 1000,
  })
}

export function useCreateMeeting() {
  return useAppMutation<
    Meeting,
    {
      title: string
      description?: string
      starts_at: string
      ends_at: string
      location?: string
      lead_id?: string
      owner_member_id?: string
    }
  >((body) => post<Meeting>('/api/v1/meetings', body), {
    invalidate: [['meetings']],
    errorContext: 'meeting.create',
  })
}

export function useSetMeetingStatus(id: string) {
  return useAppMutation<void, { status: MeetingStatus }>(
    (body) => post<void>(`/api/v1/meetings/${id}/status`, body),
    {
      invalidate: [['meetings']],
      errorContext: 'meeting.status',
    }
  )
}

export function useDeleteMeeting(id: string) {
  return useAppMutation<void, void>(() => del<void>(`/api/v1/meetings/${id}`), {
    invalidate: [['meetings']],
    errorContext: 'meeting.delete',
  })
}
