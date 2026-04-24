/**
 * F05 Task hooks — aligned with ADR-007 unified Task entity.
 *
 * Server invariant: AT MOST ONE task in_progress per assignee. The backend
 * returns 409 ASSIGNEE_BUSY on a Start that would violate it. Consumers
 * surface that via `mutation.error` (the default silent toast is still
 * fine — toast text already says "Você não tem permissão" style, but the
 * server message reads "assignee already has an in_progress task").
 */

import { useQuery, useQueryClient } from '@tanstack/react-query'

import { get, post } from '@/api/client'
import { queryKeys } from '@/api/queryKeys'
import { useAppMutation } from '@/hooks/useAppMutation'
import { useWSSubscribe } from '@/hooks/useWSSubscribe'

export type TaskStatus = 'pending' | 'in_progress' | 'done' | 'cancelled' | 'missed'
export type TaskKind =
  | 'followup'
  | 'call'
  | 'qualification'
  | 'send_proposal'
  | 'confirm_meeting'
  | 'objection'
  | 'generic'
export type TaskPriority = 'low' | 'normal' | 'high' | 'urgent'

export interface Task {
  id: string
  lead_id?: string | null
  assigned_to: string
  kind: TaskKind
  title: string
  description?: string | null
  priority: TaskPriority
  status: TaskStatus
  due_at?: string | null
  started_at?: string | null
  completed_at?: string | null
  origin: string
  result_note?: string | null
  created_at: string
}

export interface TaskListFilters {
  assigned_to?: string
  status?: TaskStatus
  kind?: TaskKind
  lead_id?: string
}

export function useTasks(filters: TaskListFilters = {}) {
  const client = useQueryClient()

  useWSSubscribe<Task>(
    ['task.created', 'task.started', 'task.completed', 'task.cancelled', 'task.missed'],
    () => {
      void client.invalidateQueries({ queryKey: queryKeys.tasks.all() })
    }
  )

  return useQuery<Task[]>({
    queryKey: queryKeys.tasks.list(filters as Record<string, unknown>),
    queryFn: async () => {
      const qs = new URLSearchParams()
      for (const [k, v] of Object.entries(filters)) if (v) qs.set(k, String(v))
      const suffix = qs.toString() ? `?${qs.toString()}` : ''
      const res = await get<{ data: Task[] }>(`/api/v1/tasks${suffix}`)
      return res.data
    },
  })
}

export function useTask(id: string | undefined) {
  return useQuery<Task>({
    queryKey: id ? queryKeys.tasks.detail(id) : ['tasks', 'detail', 'disabled'],
    enabled: Boolean(id),
    queryFn: () => get<Task>(`/api/v1/tasks/${id}`),
  })
}

export function useCreateTask() {
  return useAppMutation<
    Task,
    {
      assigned_to: string
      title: string
      kind?: TaskKind
      description?: string
      priority?: TaskPriority
      due_at?: string
      lead_id?: string
    }
  >((body) => post<Task>('/api/v1/tasks', body), {
    invalidate: [queryKeys.tasks.all()],
    errorContext: 'task.create',
  })
}

export function useStartTask(id: string) {
  return useAppMutation<void, void>(() => post<void>(`/api/v1/tasks/${id}/start`, {}), {
    invalidate: [queryKeys.tasks.all()],
    errorContext: 'task.start',
  })
}

export function useCompleteTask(id: string) {
  return useAppMutation<void, { result_note?: string }>(
    (b) => post<void>(`/api/v1/tasks/${id}/complete`, b),
    { invalidate: [queryKeys.tasks.all()], errorContext: 'task.complete' }
  )
}

export function useCancelTask(id: string) {
  return useAppMutation<void, { reason?: string }>(
    (b) => post<void>(`/api/v1/tasks/${id}/cancel`, b),
    { invalidate: [queryKeys.tasks.all()], errorContext: 'task.cancel' }
  )
}
