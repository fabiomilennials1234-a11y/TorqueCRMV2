/**
 * F02 Confirmation hooks — 1:1 with pipe_entries in a confirmation-kind pipe.
 *
 * The server exposes:
 *   PUT    /api/v1/confirmations/:entryId            — upsert meeting
 *   GET    /api/v1/confirmations/:entryId            — fetch
 *   POST   /api/v1/confirmations/:entryId/confirm    — mark confirmed
 *   POST   /api/v1/confirmations/:entryId/no-show    — mark no-show
 *   GET    /api/v1/confirmations/overdue             — list unconfirmed past-due
 */

import { useQuery } from '@tanstack/react-query'

import { get, post, put } from '@/api/client'
import { useAppMutation } from '@/hooks/useAppMutation'

export interface Confirmation {
  pipe_entry_id: string
  lead_id: string
  meeting_at: string
  meeting_channel?: string | null
  meeting_notes?: string | null
  confirmed_at?: string | null
  no_show: boolean
  no_show_reason?: string | null
}

export interface ConfirmationUpsertPayload {
  lead_id: string
  meeting_at: string // ISO-8601 UTC
  meeting_channel?: string
  meeting_notes?: string
}

/** GET /api/v1/confirmations/:entryId */
export function useConfirmation(entryId: string | undefined) {
  return useQuery<Confirmation>({
    queryKey: entryId ? ['confirmations', 'detail', entryId] : ['confirmations', 'disabled'],
    enabled: Boolean(entryId),
    queryFn: () => get<Confirmation>(`/api/v1/confirmations/${entryId}`),
  })
}

/** PUT /api/v1/confirmations/:entryId */
export function useUpsertConfirmation(entryId: string) {
  return useAppMutation<Confirmation, ConfirmationUpsertPayload>(
    (body) => put<Confirmation>(`/api/v1/confirmations/${entryId}`, body),
    {
      invalidate: [['confirmations', 'detail', entryId]],
      errorContext: 'confirmation.upsert',
    }
  )
}

/** POST /api/v1/confirmations/:entryId/confirm */
export function useConfirm(entryId: string) {
  return useAppMutation<void, void>(
    () => post<void>(`/api/v1/confirmations/${entryId}/confirm`, {}),
    {
      invalidate: [['confirmations', 'detail', entryId]],
      errorContext: 'confirmation.confirm',
    }
  )
}

/** POST /api/v1/confirmations/:entryId/no-show */
export function useMarkNoShow(entryId: string) {
  return useAppMutation<void, { reason?: string }>(
    (body) => post<void>(`/api/v1/confirmations/${entryId}/no-show`, body),
    {
      invalidate: [['confirmations', 'detail', entryId]],
      errorContext: 'confirmation.no_show',
    }
  )
}

/** GET /api/v1/confirmations/overdue */
export function useOverdueConfirmations(limit = 50) {
  return useQuery<Confirmation[]>({
    queryKey: ['confirmations', 'overdue', limit],
    queryFn: async () => {
      const res = await get<{ data: Confirmation[] }>(`/api/v1/confirmations/overdue?limit=${limit}`)
      return res.data
    },
    staleTime: 30 * 1000,
  })
}
