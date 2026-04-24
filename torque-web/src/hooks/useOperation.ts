/**
 * Operation hooks — the 202-Accepted + poll + WS-push pattern (ADR-006).
 *
 * Three hooks:
 *
 *   - `useOperationSubmit` → fires the POST; returns the new operation id.
 *   - `useOperationStatus` → polls GET /operations/:id until terminal, with
 *                            the interval set short while running and long
 *                            when idle. WS `operation.updated/.succeeded/
 *                            .failed` short-circuits the poll with a cache
 *                            patch.
 *   - `useOperationCancel` → DELETE; invalidates the status cache.
 *
 * The frontend rarely owns the polling budget — when a matching WS event
 * fires, we prefer the server-pushed patch and skip the next poll.
 */

import { useQuery, useQueryClient, type UseQueryResult } from '@tanstack/react-query'
import { useCallback } from 'react'

import { del, get, post } from '@/api/client'
import { queryKeys } from '@/api/queryKeys'
import { useAppMutation } from '@/hooks/useAppMutation'
import { useWSSubscribe, type TorqueEvent } from '@/hooks/useWSSubscribe'

export type OperationStatus = 'pending' | 'running' | 'succeeded' | 'failed' | 'cancelled'

export interface Operation {
  id: string
  kind: string
  status: OperationStatus
  progress?: number | null
  result?: unknown
  error?: unknown
  retry_remaining?: number
  created_at: string
  started_at?: string | null
  ended_at?: string | null
}

export interface OperationPatch {
  id: string
  status: OperationStatus
  progress?: number | null
  result?: unknown
  error?: unknown
}

const TERMINAL = new Set<OperationStatus>(['succeeded', 'failed', 'cancelled'])

/** POST /api/v1/operations — submit a new async job. */
export function useOperationSubmit() {
  return useAppMutation<Operation, { kind: string; input?: unknown; retry_remaining?: number }>(
    (vars) => post<Operation>('/api/v1/operations', vars),
    {
      invalidate: [queryKeys.operations.all()],
      errorContext: 'operation.submit',
    }
  )
}

/**
 * GET /api/v1/operations/:id with adaptive polling.
 *
 * - running/pending → polls every 2 s.
 * - terminal         → stops polling (refetchInterval returns false).
 *
 * WS events patch the cache directly, so the poll is a fallback, not the
 * primary path.
 */
export function useOperationStatus(id: string | undefined): UseQueryResult<Operation, unknown> {
  const client = useQueryClient()

  useWSSubscribe<OperationPatch>(
    ['operation.updated', 'operation.succeeded', 'operation.failed'],
    (evt: TorqueEvent<OperationPatch>) => {
      const patch = evt.patch
      if (!patch || !id || patch.id !== id) return
      client.setQueryData<Operation>(queryKeys.operations.detail(id), (prev) => {
        if (!prev) return prev
        return { ...prev, ...patch }
      })
    },
    [id]
  )

  return useQuery<Operation, unknown>({
    queryKey: id ? queryKeys.operations.detail(id) : ['operations', 'detail', 'disabled'],
    enabled: Boolean(id),
    queryFn: () => get<Operation>(`/api/v1/operations/${id}`),
    refetchInterval: (query) => {
      const data = query.state.data
      if (!data) return 2000
      return TERMINAL.has(data.status) ? false : 2000
    },
  })
}

/** DELETE /api/v1/operations/:id — cancel an in-flight op. */
export function useOperationCancel(id: string | undefined) {
  const client = useQueryClient()
  const invalidate = useCallback(
    () =>
      id ? client.invalidateQueries({ queryKey: queryKeys.operations.detail(id) }) : undefined,
    [client, id]
  )
  return useAppMutation<void, void>(
    () => {
      if (!id) return Promise.reject(new Error('operation id required'))
      return del<void>(`/api/v1/operations/${id}`)
    },
    {
      errorContext: 'operation.cancel',
      onSuccess: () => void invalidate(),
    }
  )
}
