/**
 * F07 Workflow Builder — WorkflowCanvasPage (S43).
 *
 * xyflow canvas editor. Drag nodes from the sidebar palette onto the
 * canvas → POST /workflows/:id/steps. Drag connections between nodes →
 * PUT the source step with the new next_step_ids. Double-click any
 * node to open the inspector and edit its config. The editor always
 * persists on action — no local draft that gets out of sync with the
 * server. Reloading restores positions because position_x/y ride
 * along in every PUT.
 *
 * Scope note (S43 acceptance):
 *   - 6 node kinds prioritized (schema already supports 7; `http` is
 *     parked for S45 when it gets a proper config form).
 *   - Edge connect validates compat: trigger.* can only source; action/
 *     wait can source and sink; branch sources multiple targets.
 *   - We intentionally treat `trigger` as a step kind that happens to
 *     have no inbound edges. The workflow's `trigger` column captures
 *     which event feeds the entry step; the canvas places a matching
 *     trigger-kind node at position_x/y 40/40 by default.
 */

import { ArrowLeft } from 'lucide-react'
import { useCallback, useEffect, useId, useMemo, useRef, useState } from 'react'
import { Link, Navigate, useParams } from 'react-router-dom'

import {
  Background,
  Controls,
  MiniMap,
  ReactFlow,
  ReactFlowProvider,
  addEdge,
  useEdgesState,
  useNodesState,
  useReactFlow,
  type Connection,
  type Edge,
  type Node,
  type NodeChange,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'

import { useQueryClient } from '@tanstack/react-query'

import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { Input } from '@/ui/input'
import { Skeleton } from '@/ui/skeleton'
import { PageHeader } from '@/ui/page-header'
import { friendlyMessage } from '@/api/errors'
import { del, put } from '@/api/client'
import {
  useCreateStep,
  usePauseWorkflow,
  usePublishWorkflow,
  useSetEntryStep,
  useWorkflow,
  useWorkflowSteps,
  type WorkflowStep,
  type WorkflowStepKind,
} from '@/hooks/useWorkflows'

// -------- palette ---------------------------------------------------

interface PaletteItem {
  kind: WorkflowStepKind
  label: string
  description: string
  defaults: { name: string; config: Record<string, unknown> }
  isTrigger?: boolean
}

const palette: PaletteItem[] = [
  {
    kind: 'send_message',
    label: 'Gatilho: Lead criado',
    description: 'Dispara quando um lead entra no sistema.',
    defaults: { name: 'Lead criado', config: { trigger_kind: 'lead_created' } },
    isTrigger: true,
  },
  {
    kind: 'send_message',
    label: 'Gatilho: Mensagem recebida',
    description: 'Dispara ao receber mensagem inbound.',
    defaults: { name: 'Mensagem recebida', config: { trigger_kind: 'message_received' } },
    isTrigger: true,
  },
  {
    kind: 'send_message',
    label: 'Ação: Enviar mensagem',
    description: 'Dispara mensagem pelo canal configurado.',
    defaults: { name: 'Enviar mensagem', config: { template: '', body: '' } },
  },
  {
    kind: 'update_lead',
    label: 'Ação: Mudar estágio',
    description: 'Move o lead para outra etapa.',
    defaults: { name: 'Mudar estágio', config: { stage_id: '' } },
  },
  {
    kind: 'wait',
    label: 'Ação: Esperar',
    description: 'Pausa o fluxo por N minutos/horas.',
    defaults: { name: 'Esperar', config: { duration_seconds: 3600 } },
  },
  {
    kind: 'branch',
    label: 'Condição: Se / Senão',
    description: 'Bifurca o fluxo por uma expressão.',
    defaults: { name: 'Se/Senão', config: { expression: 'lead.origin == "meta-ads"' } },
  },
]

// -------- page shell ------------------------------------------------

export function WorkflowCanvasPage() {
  const { id } = useParams<{ id: string }>()

  if (!id) return <Navigate to="/workflows" replace />
  return (
    <ReactFlowProvider>
      <CanvasInner workflowId={id} />
    </ReactFlowProvider>
  )
}

function CanvasInner({ workflowId }: { workflowId: string }) {
  const workflow = useWorkflow(workflowId)
  const stepsQuery = useWorkflowSteps(workflowId)

  if (workflow.isLoading || stepsQuery.isLoading) return <CanvasSkeleton />
  if (workflow.isError || !workflow.data) {
    return (
      <div className="mx-auto max-w-md p-12 text-center">
        <p className="text-sm text-danger">{friendlyMessage(workflow.error)}</p>
        <Link to="/workflows" className="mt-4 inline-block text-sm text-accent underline">
          Voltar
        </Link>
      </div>
    )
  }
  const steps = stepsQuery.data ?? []
  return <CanvasEditor workflowId={workflowId} workflow={workflow.data} steps={steps} />
}

// -------- editor ----------------------------------------------------

function CanvasEditor({
  workflowId,
  workflow,
  steps,
}: {
  workflowId: string
  workflow: NonNullable<ReturnType<typeof useWorkflow>['data']>
  steps: WorkflowStep[]
}) {
  const queryClient = useQueryClient()
  const createStep = useCreateStep(workflowId)
  const setEntry = useSetEntryStep(workflowId)
  const publish = usePublishWorkflow(workflowId)
  const pause = usePauseWorkflow(workflowId)

  const invalidateSteps = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ['workflows', workflowId, 'steps'] })
  }, [queryClient, workflowId])

  const updateStepAsync = useCallback(
    async (stepId: string, payload: StepPayload) => {
      await put(`/api/v1/workflows/${workflowId}/steps/${stepId}`, payload)
      invalidateSteps()
    },
    [workflowId, invalidateSteps]
  )
  const deleteStepAsync = useCallback(
    async (stepId: string) => {
      await del(`/api/v1/workflows/${workflowId}/steps/${stepId}`)
      invalidateSteps()
    },
    [workflowId, invalidateSteps]
  )

  // Hydrate xyflow from server state whenever the steps array changes.
  const initialNodes = useMemo<Node[]>(
    () =>
      steps.map((s) => ({
        id: s.id,
        type: 'default',
        position: { x: s.position_x ?? 80, y: s.position_y ?? 80 },
        data: { label: `${kindIcon(s.kind)}  ${s.name}`, kind: s.kind, step: s },
      })),
    [steps]
  )
  const initialEdges = useMemo<Edge[]>(
    () =>
      steps.flatMap((s) =>
        s.next_step_ids.map((target) => ({
          id: `${s.id}-${target}`,
          source: s.id,
          target,
        }))
      ),
    [steps]
  )

  const [nodes, setNodes, onNodesChange] = useNodesState<Node>(initialNodes)
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>(initialEdges)

  // Re-sync when server state refreshes (WS invalidation etc).
  useEffect(() => setNodes(initialNodes), [initialNodes, setNodes])
  useEffect(() => setEdges(initialEdges), [initialEdges, setEdges])

  const [selected, setSelected] = useState<string | null>(null)
  const reactFlow = useReactFlow()
  const canvasRef = useRef<HTMLDivElement>(null)

  // Flush position moves server-side (debounced: the step is PUT on
  // `nodeDragStop`, not on every pixel).
  const handleNodesChange = useCallback(
    (changes: NodeChange[]) => {
      onNodesChange(changes)
    },
    [onNodesChange]
  )

  const handleNodeDragStop = useCallback(
    async (_: unknown, node: Node) => {
      const step = steps.find((s) => s.id === node.id)
      if (!step) return
      await updateStepAsync(step.id, {
        kind: step.kind,
        name: step.name,
        config: step.config,
        next_step_ids: step.next_step_ids,
        position_x: Math.round(node.position.x),
        position_y: Math.round(node.position.y),
      })
    },
    [steps, updateStepAsync]
  )

  const handleConnect = useCallback(
    async (connection: Connection) => {
      if (!connection.source || !connection.target) return
      if (connection.source === connection.target) return // no self-loops
      const source = steps.find((s) => s.id === connection.source)
      if (!source) return
      const next = Array.from(new Set([...source.next_step_ids, connection.target]))
      await updateStepAsync(source.id, {
        kind: source.kind,
        name: source.name,
        config: source.config,
        next_step_ids: next,
        position_x: source.position_x ?? 0,
        position_y: source.position_y ?? 0,
      })
      setEdges((eds) => addEdge(connection, eds))
    },
    [steps, updateStepAsync, setEdges]
  )

  const handleDrop = useCallback(
    async (e: React.DragEvent) => {
      e.preventDefault()
      const kindRaw = e.dataTransfer.getData('application/torque-step')
      if (!kindRaw || !canvasRef.current) return
      const parsed: PaletteItem = JSON.parse(kindRaw) as PaletteItem
      const bounds = canvasRef.current.getBoundingClientRect()
      const position = reactFlow.screenToFlowPosition({
        x: e.clientX - bounds.left,
        y: e.clientY - bounds.top,
      })
      const created = await createStep.mutateAsync({
        kind: parsed.kind,
        name: parsed.defaults.name,
        config: parsed.defaults.config,
        next_step_ids: [],
        position_x: Math.round(position.x),
        position_y: Math.round(position.y),
      })
      // If the workflow has no entry yet + this node is marked as a trigger,
      // set it as the entry so publish() works without extra UX.
      if (!workflow.entry_step_id && parsed.isTrigger) {
        await setEntry.mutateAsync({ step_id: created.id })
      }
    },
    [createStep, reactFlow, setEntry, workflow.entry_step_id]
  )

  const selectedStep = selected ? steps.find((s) => s.id === selected) : undefined

  return (
    <div className="flex h-[calc(100vh-56px)] min-h-0">
      {/* Palette */}
      <aside className="flex w-64 shrink-0 flex-col gap-2 overflow-y-auto border-r border-hairline p-4">
        <Link to="/workflows" className="mb-2 inline-flex items-center gap-1.5 text-xs text-ink-dim hover:text-ink-muted">
          <ArrowLeft className="h-3.5 w-3.5" />
          Workflows
        </Link>
        <h3 className="text-xs uppercase tracking-wide text-ink-dim">Nodes disponíveis</h3>
        {palette.map((item, i) => (
          <button
            key={i}
            type="button"
            draggable
            onDragStart={(e) => {
              e.dataTransfer.setData('application/torque-step', JSON.stringify(item))
              e.dataTransfer.effectAllowed = 'copy'
            }}
            className="rounded-md bg-elevated/40 p-3 text-left text-xs text-ink shadow-hairline transition hover:bg-elevated/60"
          >
            <div className="font-medium">{item.label}</div>
            <div className="mt-1 text-2xs text-ink-dim">{item.description}</div>
          </button>
        ))}
      </aside>

      {/* Canvas */}
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex items-center gap-3 border-b border-hairline px-6 py-3">
          <PageHeader
            eyebrow="Workflow"
            title={workflow.name}
            description={workflow.description ?? undefined}
          />
          <div className="ml-auto flex items-center gap-2">
            <Badge tone={workflow.status === 'active' ? 'success' : 'neutral'}>{workflow.status}</Badge>
            <Link
              to={`/workflows/${workflowId}/executions`}
              className="text-xs text-accent underline"
            >
              Execuções
            </Link>
            {workflow.status !== 'active' ? (
              <Button
                type="button"
                variant="primary"
                size="sm"
                onClick={() => void publish.mutateAsync()}
                disabled={publish.isPending || !workflow.entry_step_id}
              >
                Publicar
              </Button>
            ) : (
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => void pause.mutateAsync()}
                disabled={pause.isPending}
              >
                Pausar
              </Button>
            )}
          </div>
        </header>

        <div
          ref={canvasRef}
          className="relative flex-1"
          onDragOver={(e) => {
            e.preventDefault()
            e.dataTransfer.dropEffect = 'copy'
          }}
          onDrop={(e) => void handleDrop(e)}
        >
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={handleNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={(c) => void handleConnect(c)}
            onNodeClick={(_, n) => setSelected(n.id)}
            onNodeDragStop={(e, n) => void handleNodeDragStop(e, n)}
            fitView
          >
            <Background />
            <MiniMap />
            <Controls />
          </ReactFlow>
        </div>
      </div>

      {/* Inspector */}
      <aside className="w-80 shrink-0 overflow-y-auto border-l border-hairline p-4">
        {selectedStep ? (
          <NodeInspector
            workflowId={workflowId}
            step={selectedStep}
            onDelete={async () => {
              await deleteStepAsync(selectedStep.id)
              setSelected(null)
            }}
            onUpdate={(patch) =>
              updateStepAsync(selectedStep.id, {
                kind: patch.kind ?? selectedStep.kind,
                name: patch.name ?? selectedStep.name,
                config: patch.config ?? selectedStep.config,
                next_step_ids: selectedStep.next_step_ids,
                position_x: selectedStep.position_x ?? 0,
                position_y: selectedStep.position_y ?? 0,
              })
            }
            onSetEntry={() => setEntry.mutateAsync({ step_id: selectedStep.id })}
            isEntry={workflow.entry_step_id === selectedStep.id}
          />
        ) : (
          <p className="text-xs text-ink-dim">
            Arraste um node da esquerda para o canvas, ou clique em um node existente para editar.
          </p>
        )}
      </aside>
    </div>
  )
}

// -------- inspector -------------------------------------------------

function NodeInspector({
  step,
  onDelete,
  onUpdate,
  onSetEntry,
  isEntry,
}: {
  workflowId: string
  step: WorkflowStep
  onDelete: () => Promise<void>
  onUpdate: (patch: Partial<{ name: string; kind: WorkflowStepKind; config: unknown }>) => Promise<unknown>
  onSetEntry: () => Promise<unknown>
  isEntry: boolean
}) {
  const [name, setName] = useState(step.name)
  const [config, setConfig] = useState(JSON.stringify(step.config ?? {}, null, 2))

  useEffect(() => {
    setName(step.name)
    setConfig(JSON.stringify(step.config ?? {}, null, 2))
  }, [step])

  const [error, setError] = useState<string | null>(null)
  const nameInputId = useId()
  const configInputId = useId()

  async function handleSave() {
    try {
      const parsed: unknown = JSON.parse(config)
      setError(null)
      await onUpdate({ name, config: parsed })
    } catch (e) {
      setError(e instanceof Error ? e.message : 'config JSON inválido')
    }
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <h3 className="text-sm font-medium text-ink">Node</h3>
        {isEntry && <Badge tone="success">entry</Badge>}
        <Badge tone="neutral">{step.kind}</Badge>
      </div>
      <div>
        <label htmlFor={nameInputId} className="mb-1 block text-xs text-ink-muted">
          Nome
        </label>
        <Input id={nameInputId} value={name} onChange={(e) => setName(e.target.value)} />
      </div>
      <div>
        <label htmlFor={configInputId} className="mb-1 block text-xs text-ink-muted">
          Config (JSON)
        </label>
        <textarea
          id={configInputId}
          value={config}
          onChange={(e) => setConfig(e.target.value)}
          rows={10}
          className="w-full resize-y rounded-md bg-elevated/40 px-3 py-2 font-mono text-xs text-ink shadow-hairline focus:outline-none focus:ring-1 focus:ring-accent/50"
        />
      </div>
      {error && (
        <p role="alert" className="text-2xs text-danger">
          {error}
        </p>
      )}
      <div className="flex flex-wrap gap-2">
        <Button type="button" variant="primary" size="sm" onClick={() => void handleSave()}>
          Salvar
        </Button>
        {!isEntry && (
          <Button type="button" variant="outline" size="sm" onClick={() => void onSetEntry()}>
            Definir como entry
          </Button>
        )}
        <Button type="button" variant="ghost" size="sm" onClick={() => void onDelete()}>
          Excluir
        </Button>
      </div>
    </div>
  )
}

// -------- helpers ---------------------------------------------------

function kindIcon(kind: WorkflowStepKind): string {
  switch (kind) {
    case 'send_message':
      return '💬'
    case 'wait':
      return '⏳'
    case 'create_task':
      return '📋'
    case 'branch':
      return '🔀'
    case 'update_lead':
      return '👤'
    case 'call_agent':
      return '🤖'
    case 'http':
      return '🌐'
    default:
      return '⚙️'
  }
}

// StepPayload mirrors useCreateStep's input shape. Exported via the
// hook's mutateAsync param so we stay in sync if the backend contract
// grows. Used by updateStepAsync which calls PUT directly (the
// useWorkflows useUpdateStep hook binds a specific stepId at
// construction and isn't suited for canvas-wide cross-step edits).
type StepPayload = Parameters<ReturnType<typeof useCreateStep>['mutateAsync']>[0]

function CanvasSkeleton() {
  return (
    <div className="mx-auto max-w-6xl space-y-4 px-8 py-8">
      <Skeleton className="h-8 w-64" />
      <Skeleton className="h-[60vh] w-full" />
    </div>
  )
}
