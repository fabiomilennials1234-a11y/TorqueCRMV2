/**
 * F07 Workflow Builder hooks — CRUD on workflows, steps, and runs.
 *
 * The backend (S17) gates everything under RequireRole(admin). Non-admin
 * callers get 403; the toast pipeline surfaces it and the UI should hide
 * the admin surfaces when the session is not admin.
 */

import { useQuery, useQueryClient } from '@tanstack/react-query'

import { del, get, post, put } from '@/api/client'
import { useAppMutation } from '@/hooks/useAppMutation'
import { useWSSubscribe } from '@/hooks/useWSSubscribe'

export type WorkflowTrigger =
  | 'manual'
  | 'lead_created'
  | 'lead_stage_changed'
  | 'message_inbound'
  | 'schedule'

export type WorkflowStatus = 'draft' | 'active' | 'paused' | 'archived'

export type WorkflowStepKind =
  | 'send_message'
  | 'wait'
  | 'create_task'
  | 'branch'
  | 'update_lead'
  | 'call_agent'
  | 'http'

export type WorkflowRunStatus = 'pending' | 'running' | 'succeeded' | 'failed' | 'cancelled'

export interface Workflow {
  id: string
  name: string
  description?: string | null
  trigger: WorkflowTrigger
  status: WorkflowStatus
  entry_step_id?: string | null
}

export interface WorkflowStep {
  id: string
  kind: WorkflowStepKind
  name: string
  config: unknown
  next_step_ids: string[]
  position_x?: number | null
  position_y?: number | null
}

export interface WorkflowRun {
  id: string
  workflow_id: string
  lead_id?: string | null
  trigger_source: string
  status: WorkflowRunStatus
  current_step_id?: string | null
  created_at: string
}

function keys() {
  return {
    list: () => ['workflows'] as const,
    detail: (id: string) => ['workflows', id] as const,
    steps: (id: string) => ['workflows', id, 'steps'] as const,
    runs: (id: string) => ['workflows', id, 'runs'] as const,
  }
}

export function useWorkflows() {
  const client = useQueryClient()

  useWSSubscribe(
    [
      'workflow.created',
      'workflow.published',
      'workflow.paused',
      'workflow.archived',
      'workflow.entry_set',
    ],
    () => void client.invalidateQueries({ queryKey: keys().list() })
  )

  return useQuery<Workflow[]>({
    queryKey: keys().list(),
    queryFn: async () => (await get<{ data: Workflow[] }>('/api/v1/workflows')).data,
    staleTime: 30 * 1000,
  })
}

export function useWorkflow(id: string | undefined) {
  return useQuery<Workflow>({
    queryKey: id ? keys().detail(id) : ['workflows', 'disabled'],
    enabled: Boolean(id),
    queryFn: () => get<Workflow>(`/api/v1/workflows/${id}`),
  })
}

export function useCreateWorkflow() {
  return useAppMutation<
    Workflow,
    { name: string; description?: string; trigger: WorkflowTrigger; trigger_config?: unknown }
  >((body) => post<Workflow>('/api/v1/workflows', body), {
    invalidate: [keys().list()],
    errorContext: 'workflow.create',
  })
}

export function usePublishWorkflow(id: string) {
  return useAppMutation<void, void>(
    () => post<void>(`/api/v1/workflows/${id}/publish`, {}),
    { invalidate: [keys().list(), keys().detail(id)], errorContext: 'workflow.publish' }
  )
}

export function usePauseWorkflow(id: string) {
  return useAppMutation<void, void>(
    () => post<void>(`/api/v1/workflows/${id}/pause`, {}),
    { invalidate: [keys().list(), keys().detail(id)], errorContext: 'workflow.pause' }
  )
}

export function useArchiveWorkflow(id: string) {
  return useAppMutation<void, void>(
    () => post<void>(`/api/v1/workflows/${id}/archive`, {}),
    { invalidate: [keys().list(), keys().detail(id)], errorContext: 'workflow.archive' }
  )
}

// -------- steps -----------------------------------------------------

export function useWorkflowSteps(workflowId: string | undefined) {
  const client = useQueryClient()

  useWSSubscribe(
    ['workflow_step.upserted', 'workflow_step.deleted'],
    () => {
      if (workflowId)
        void client.invalidateQueries({ queryKey: keys().steps(workflowId) })
    },
    [workflowId]
  )

  return useQuery<WorkflowStep[]>({
    queryKey: workflowId ? keys().steps(workflowId) : ['workflows', 'steps', 'disabled'],
    enabled: Boolean(workflowId),
    queryFn: async () =>
      (await get<{ data: WorkflowStep[] }>(`/api/v1/workflows/${workflowId}/steps`)).data,
  })
}

export interface StepPayload {
  kind: WorkflowStepKind
  name: string
  config?: unknown
  next_step_ids?: string[]
  position_x?: number
  position_y?: number
}

export function useCreateStep(workflowId: string) {
  return useAppMutation<WorkflowStep, StepPayload>(
    (body) => post<WorkflowStep>(`/api/v1/workflows/${workflowId}/steps`, body),
    { invalidate: [keys().steps(workflowId)], errorContext: 'workflow.step.create' }
  )
}

export function useUpdateStep(workflowId: string, stepId: string) {
  return useAppMutation<WorkflowStep, StepPayload>(
    (body) => put<WorkflowStep>(`/api/v1/workflows/${workflowId}/steps/${stepId}`, body),
    { invalidate: [keys().steps(workflowId)], errorContext: 'workflow.step.update' }
  )
}

export function useDeleteStep(workflowId: string, stepId: string) {
  return useAppMutation<void, void>(
    () => del<void>(`/api/v1/workflows/${workflowId}/steps/${stepId}`),
    { invalidate: [keys().steps(workflowId)], errorContext: 'workflow.step.delete' }
  )
}

export function useSetEntryStep(workflowId: string) {
  return useAppMutation<void, { step_id: string }>(
    (body) => post<void>(`/api/v1/workflows/${workflowId}/entry/${body.step_id}`, {}),
    { invalidate: [keys().detail(workflowId), keys().list()], errorContext: 'workflow.entry' }
  )
}

// -------- runs ------------------------------------------------------

export function useWorkflowRuns(workflowId: string | undefined) {
  const client = useQueryClient()

  useWSSubscribe(
    ['workflow_run.enqueued', 'workflow_run.cancelled'],
    () => {
      if (workflowId)
        void client.invalidateQueries({ queryKey: keys().runs(workflowId) })
    },
    [workflowId]
  )

  return useQuery<WorkflowRun[]>({
    queryKey: workflowId ? keys().runs(workflowId) : ['workflows', 'runs', 'disabled'],
    enabled: Boolean(workflowId),
    queryFn: async () =>
      (await get<{ data: WorkflowRun[] }>(`/api/v1/workflows/${workflowId}/runs`)).data,
  })
}

export function useEnqueueRun(workflowId: string) {
  return useAppMutation<
    WorkflowRun,
    { lead_id?: string; trigger_source?: string; input?: unknown }
  >(
    (body) => post<WorkflowRun>(`/api/v1/workflows/${workflowId}/runs`, body ?? {}),
    { invalidate: [keys().runs(workflowId)], errorContext: 'workflow.run.enqueue' }
  )
}

export function useCancelRun(runId: string) {
  return useAppMutation<void, void>(
    () => post<void>(`/api/v1/runs/${runId}/cancel`, {}),
    { errorContext: 'workflow.run.cancel' }
  )
}

// -------- S45 run-step trace ----------------------------------------

/**
 * S45 — one row in workflow_run_steps, surfaced by GET /runs/:id/steps.
 * The executions UI renders these as a timeline (status badges + JSON
 * output preview) after the user clicks a run.
 */
export interface WorkflowRunStep {
  id: string
  run_id: string
  step_id: string
  status: WorkflowRunStatus
  input?: unknown
  output?: unknown
  error_payload?: unknown
  started_at?: string | null
  ended_at?: string | null
  created_at: string
}

export function useWorkflowRunSteps(runId: string | undefined) {
  return useQuery<WorkflowRunStep[]>({
    queryKey: runId ? ['runs', runId, 'steps'] : ['runs', 'steps', 'disabled'],
    enabled: Boolean(runId),
    queryFn: async () =>
      (await get<{ data: WorkflowRunStep[] }>(`/api/v1/runs/${runId}/steps`)).data,
    // Poll while any step is non-terminal so the timeline animates
    // without depending on WS for the run-step lifecycle (those
    // events are not on the bus yet — follow-up).
    refetchInterval: (query) => {
      const data = query.state.data
      if (!data || data.some((s) => s.status === 'pending' || s.status === 'running')) {
        return 2_000
      }
      return false
    },
  })
}
