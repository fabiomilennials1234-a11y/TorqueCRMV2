/**
 * Pipe hooks (F01) — list, stages, entries, and atomic move.
 *
 * Every query is tenant-scoped by the auth cookie on the wire; no pipe id
 * from another tenant can reach here.
 */

import { useQuery, useQueryClient } from '@tanstack/react-query'

import { get, post } from '@/api/client'
import { queryKeys } from '@/api/queryKeys'
import { useAppMutation } from '@/hooks/useAppMutation'
import { useWSSubscribe } from '@/hooks/useWSSubscribe'

export interface Pipe {
  id: string
  kind: string
  name: string
  is_default: boolean
  is_archived: boolean
  position: number
}

export interface PipeStage {
  id: string
  name: string
  color_token?: string | null
  position: number
  is_final_positive: boolean
  is_final_negative: boolean
}

export interface PipeEntry {
  id: string
  stage_id: string
  lead_id: string
  entered_stage_at: string
}

interface DataEnvelope<T> {
  data: T[]
}

/** GET /api/v1/pipes */
export function usePipes() {
  return useQuery<Pipe[]>({
    queryKey: queryKeys.pipes.list(),
    queryFn: async () => (await get<DataEnvelope<Pipe>>('/api/v1/pipes')).data,
    staleTime: 5 * 60 * 1000,
  })
}

/** GET /api/v1/pipes/:id/stages */
export function usePipeStages(pipeId: string | undefined) {
  return useQuery<PipeStage[]>({
    queryKey: pipeId ? queryKeys.pipes.stages(pipeId) : ['pipes', 'stages', 'disabled'],
    enabled: Boolean(pipeId),
    queryFn: async () => (await get<DataEnvelope<PipeStage>>(`/api/v1/pipes/${pipeId}/stages`)).data,
    staleTime: 5 * 60 * 1000,
  })
}

/**
 * GET /api/v1/pipes/:id/entries + WS auto-patch.
 *
 * Entries are the Kanban card positions. When `pipe_entry.moved` fires we
 * move the card in-place rather than refetching — the patch already carries
 * the destination stage and the entry id.
 */
export function usePipeEntries(pipeId: string | undefined) {
  const client = useQueryClient()

  useWSSubscribe<{ entry_id: string; pipe_id: string; stage_id: string; lead_id: string }>(
    'pipe_entry.moved',
    (evt) => {
      if (!pipeId || evt.patch?.pipe_id !== pipeId) return
      client.setQueryData<PipeEntry[]>(queryKeys.pipes.entries(pipeId), (prev) => {
        if (!prev) return prev
        const others = prev.filter((e) => e.lead_id !== evt.patch!.lead_id)
        return [
          ...others,
          {
            id: evt.patch!.entry_id,
            stage_id: evt.patch!.stage_id,
            lead_id: evt.patch!.lead_id,
            entered_stage_at: evt.occurred_at,
          },
        ]
      })
    },
    [pipeId]
  )

  return useQuery<PipeEntry[]>({
    queryKey: pipeId ? queryKeys.pipes.entries(pipeId) : ['pipes', 'entries', 'disabled'],
    enabled: Boolean(pipeId),
    queryFn: async () => (await get<DataEnvelope<PipeEntry>>(`/api/v1/pipes/${pipeId}/entries`)).data,
  })
}

export interface MovePayload {
  lead_id: string
  new_stage_id: string
}

/**
 * POST /api/v1/pipes/:id/entries/move — optimistic placement so the card
 * snaps into the target column instantly; the WS event confirms and/or
 * rolls back.
 */
export function useMovePipeEntry(pipeId: string) {
  return useAppMutation<PipeEntry, MovePayload>(
    (body) => post<PipeEntry>(`/api/v1/pipes/${pipeId}/entries/move`, body),
    {
      optimistic: {
        queryKey: queryKeys.pipes.entries(pipeId),
        updater: (prev, body) => {
          if (!prev) return prev
          return prev.map((e) =>
            e.lead_id === body.lead_id ? { ...e, stage_id: body.new_stage_id } : e
          )
        },
      },
      invalidate: [queryKeys.pipes.entries(pipeId)],
      errorContext: 'pipe.move',
    }
  )
}
