/**
 * Pipe hooks (F01) — list, stages, entries, and atomic move.
 *
 * Every query is tenant-scoped by the auth cookie on the wire; no pipe id
 * from another tenant can reach here.
 */

import { useQuery, useQueryClient } from '@tanstack/react-query'

import { del, get, patch, post } from '@/api/client'
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
/**
 * F12 — admin-only pipe/stage lifecycle. Non-admin callers hit 403 which
 * flows through notifyAppError as a toast.
 */
export function useCreatePipe() {
  return useAppMutation<
    Pipe,
    { kind: 'whatsapp' | 'confirmation' | 'proposal' | 'custom'; name: string; is_default?: boolean; position?: number }
  >((body) => post<Pipe>('/api/v1/pipes', body), {
    invalidate: [queryKeys.pipes.list()],
    errorContext: 'pipe.create',
  })
}

export function useUpdatePipe(id: string) {
  return useAppMutation<Pipe, { name?: string; is_default?: boolean; position?: number }>(
    (body) => patch<Pipe>(`/api/v1/pipes/${id}`, body),
    { invalidate: [queryKeys.pipes.list()], errorContext: 'pipe.update' }
  )
}

export function useArchivePipe(id: string) {
  return useAppMutation<void, void>(
    () => del<void>(`/api/v1/pipes/${id}`),
    { invalidate: [queryKeys.pipes.list()], errorContext: 'pipe.archive' }
  )
}

export function useCreateStage(pipeId: string) {
  return useAppMutation<
    PipeStage,
    {
      name: string
      color_token?: string
      position: number
      is_final_positive?: boolean
      is_final_negative?: boolean
    }
  >((body) => post<PipeStage>(`/api/v1/pipes/${pipeId}/stages`, body), {
    invalidate: [queryKeys.pipes.stages(pipeId)],
    errorContext: 'pipe.stage.create',
  })
}

export function useUpdateStage(pipeId: string, stageId: string) {
  return useAppMutation<
    PipeStage,
    {
      name?: string
      color_token?: string
      position?: number
      is_final_positive?: boolean
      is_final_negative?: boolean
    }
  >((body) => patch<PipeStage>(`/api/v1/pipes/${pipeId}/stages/${stageId}`, body), {
    invalidate: [queryKeys.pipes.stages(pipeId)],
    errorContext: 'pipe.stage.update',
  })
}

export function useDeleteStage(pipeId: string, stageId: string) {
  return useAppMutation<void, void>(
    () => del<void>(`/api/v1/pipes/${pipeId}/stages/${stageId}`),
    { invalidate: [queryKeys.pipes.stages(pipeId)], errorContext: 'pipe.stage.delete' }
  )
}

export function useMovePipeEntry(pipeId: string) {
  // The optimistic-snap on this hook targets a list cache (PipeEntry[]) while
  // the mutation returns a single PipeEntry — useAppMutation ties
  // TOptimisticData to TData, so the two would clash. The WS broadcast of
  // `pipe_entry.moved` already patches the list in usePipeEntries within
  // ~50 ms, and invalidate on settle is a belt-and-suspenders refresh.
  return useAppMutation<PipeEntry, MovePayload>(
    (body) => post<PipeEntry>(`/api/v1/pipes/${pipeId}/entries/move`, body),
    {
      invalidate: [queryKeys.pipes.entries(pipeId)],
      errorContext: 'pipe.move',
    }
  )
}
