/**
 * F07 Workflow Builder — WorkflowExecutionsPage (S45).
 *
 * Route: /workflows/:id/executions. Admin-only per backend guard.
 *
 * Lists the most recent runs for a workflow (via useWorkflowRuns) and
 * lets the user drill into a run to see its per-step timeline (via
 * useWorkflowRunSteps). A "Debug run" button enqueues a manual run
 * against a fake lead id so the admin can validate the DAG without
 * waiting for a real trigger. The runner picks it up on its next
 * poll interval (≤2s).
 */

import { ArrowLeft, Play } from 'lucide-react'
import { useMemo, useState } from 'react'
import { Link, Navigate, useParams } from 'react-router-dom'

import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { PageHeader } from '@/ui/page-header'
import { Skeleton } from '@/ui/skeleton'
import { EmptyState } from '@/ui/empty-state'
import {
  useEnqueueRun,
  useWorkflow,
  useWorkflowRunSteps,
  useWorkflowRuns,
  type WorkflowRun,
  type WorkflowRunStatus,
  type WorkflowRunStep,
} from '@/hooks/useWorkflows'

const statusTone: Record<WorkflowRunStatus, 'success' | 'danger' | 'neutral'> = {
  succeeded: 'success',
  failed: 'danger',
  cancelled: 'neutral',
  pending: 'neutral',
  running: 'neutral',
}

export function WorkflowExecutionsPage() {
  const { id } = useParams<{ id: string }>()
  if (!id) return <Navigate to="/workflows" replace />
  return <ExecutionsInner workflowId={id} />
}

function ExecutionsInner({ workflowId }: { workflowId: string }) {
  const workflow = useWorkflow(workflowId)
  const runs = useWorkflowRuns(workflowId)
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null)

  if (workflow.isLoading) return <ExecutionsSkeleton />
  if (workflow.isError || !workflow.data) {
    return (
      <div className="mx-auto max-w-md p-12 text-center">
        <p className="text-sm text-danger">Workflow indisponível.</p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-5xl px-8 py-8">
      <Link
        to={`/workflows/${workflowId}`}
        className="mb-3 inline-flex items-center gap-1.5 text-xs text-ink-dim hover:text-ink-muted"
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        Voltar ao canvas
      </Link>
      <PageHeader
        eyebrow="Workflow · Execuções"
        title={workflow.data.name}
        description="Runs recentes (50 mais novos). Clique em uma run para ver o trace."
        actions={<DebugRunButton workflowId={workflowId} />}
      />

      {runs.isLoading ? (
        <Skeleton className="mt-6 h-40 w-full" />
      ) : (runs.data ?? []).length === 0 ? (
        <EmptyState
          title="Nenhuma execução ainda"
          description='Use "Rodar em debug" para enfileirar uma execução de teste.'
        />
      ) : (
        <div className="mt-6 grid grid-cols-[minmax(280px,380px)_1fr] gap-6">
          <RunsList runs={runs.data!} selectedRunId={selectedRunId} onSelect={setSelectedRunId} />
          <RunTimeline runId={selectedRunId} />
        </div>
      )}
    </div>
  )
}

function DebugRunButton({ workflowId }: { workflowId: string }) {
  const enqueue = useEnqueueRun(workflowId)
  async function handleClick() {
    await enqueue.mutateAsync({ trigger_source: 'debug' })
  }
  return (
    <Button
      type="button"
      variant="primary"
      size="sm"
      onClick={() => void handleClick()}
      disabled={enqueue.isPending}
    >
      <Play className="mr-1 h-3.5 w-3.5" />
      {enqueue.isPending ? 'Enfileirando…' : 'Rodar em debug'}
    </Button>
  )
}

function RunsList({
  runs,
  selectedRunId,
  onSelect,
}: {
  runs: WorkflowRun[]
  selectedRunId: string | null
  onSelect: (id: string) => void
}) {
  return (
    <ul className="space-y-1.5">
      {runs.map((run) => {
        const isActive = run.id === selectedRunId
        return (
          <li key={run.id}>
            <button
              type="button"
              onClick={() => onSelect(run.id)}
              className={
                'block w-full rounded-md px-3 py-2 text-left text-xs shadow-hairline transition ' +
                (isActive ? 'bg-surface shadow-elev-1' : 'bg-elevated/30 hover:bg-elevated/60')
              }
            >
              <div className="flex items-center gap-2">
                <Badge tone={statusTone[run.status]}>{run.status}</Badge>
                <span className="truncate text-ink-dim">{run.trigger_source}</span>
              </div>
              <div className="mt-0.5 text-2xs text-ink-dim">
                {new Date(run.created_at).toLocaleString('pt-BR')}
              </div>
            </button>
          </li>
        )
      })}
    </ul>
  )
}

function RunTimeline({ runId }: { runId: string | null }) {
  const steps = useWorkflowRunSteps(runId ?? undefined)

  if (!runId) {
    return (
      <div className="rounded-md bg-elevated/20 p-4 text-xs text-ink-dim">
        Selecione uma execução à esquerda para ver o trace.
      </div>
    )
  }
  if (steps.isLoading) return <Skeleton className="h-40 w-full" />
  const data = steps.data ?? []
  if (data.length === 0) {
    return <p className="text-xs text-ink-dim">Sem steps registrados.</p>
  }
  return (
    <ol className="space-y-2">
      {data.map((step, idx) => (
        <TimelineStep key={step.id} step={step} ord={idx + 1} />
      ))}
    </ol>
  )
}

function TimelineStep({ step, ord }: { step: WorkflowRunStep; ord: number }) {
  const duration = useMemo(() => {
    if (!step.started_at || !step.ended_at) return null
    const ms = new Date(step.ended_at).getTime() - new Date(step.started_at).getTime()
    return Math.max(0, ms)
  }, [step.started_at, step.ended_at])

  const outputPreview = useMemo(() => {
    if (step.status === 'failed' && step.error_payload) {
      return JSON.stringify(step.error_payload, null, 2)
    }
    if (step.output) return JSON.stringify(step.output, null, 2)
    return null
  }, [step.output, step.error_payload, step.status])

  return (
    <li className="rounded-md bg-surface p-3 shadow-elev-1">
      <div className="flex items-center gap-2 text-xs">
        <span className="font-mono text-ink-dim">#{ord}</span>
        <Badge tone={statusTone[step.status]}>{step.status}</Badge>
        <span className="font-mono text-2xs text-ink-dim">{step.step_id.slice(0, 8)}</span>
        {duration != null && <span className="ml-auto text-2xs text-ink-dim">{duration} ms</span>}
      </div>
      {outputPreview && (
        <pre className="mt-2 overflow-x-auto rounded bg-elevated/40 p-2 font-mono text-2xs text-ink">
          {outputPreview}
        </pre>
      )}
    </li>
  )
}

function ExecutionsSkeleton() {
  return (
    <div className="mx-auto max-w-5xl space-y-4 px-8 py-8">
      <Skeleton className="h-8 w-64" />
      <Skeleton className="h-40 w-full" />
    </div>
  )
}
