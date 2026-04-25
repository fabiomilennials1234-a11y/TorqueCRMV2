/**
 * F07 Workflow Builder — WorkflowListPage (S43).
 *
 * Replaces the seed-backed mockup with a live list that consumes
 * useWorkflows (WS-invalidated). Click a row → /workflows/:id to open
 * the xyflow canvas editor. "Novo workflow" creates a draft with a
 * default manual trigger and navigates straight to the editor.
 */

import { Plus } from 'lucide-react'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'

import { PageHeader } from '@/ui/page-header'
import { Button } from '@/ui/button'
import { Badge } from '@/ui/badge'
import { Input } from '@/ui/input'
import { Skeleton } from '@/ui/skeleton'
import { EmptyState } from '@/ui/empty-state'
import { QueryBoundary } from '@/components/QueryBoundary'
import {
  useCreateWorkflow,
  useWorkflows,
  type Workflow,
  type WorkflowTrigger,
} from '@/hooks/useWorkflows'

const triggerLabels: Record<WorkflowTrigger, string> = {
  manual: 'Manual',
  lead_created: 'Lead criado',
  lead_stage_changed: 'Estágio mudou',
  message_inbound: 'Mensagem recebida',
  schedule: 'Agendado',
}

export function WorkflowListPage() {
  const query = useWorkflows()

  return (
    <div className="mx-auto max-w-5xl px-8 py-8">
      <PageHeader
        eyebrow="Automação"
        title="Workflows"
        description="Editor visual de automações — trigger → ações encadeadas."
        actions={<NewWorkflowButton />}
      />
      <QueryBoundary
        query={query}
        loadingFallback={<Skeleton className="mt-6 h-32 w-full" />}
        isEmpty={(data) => data.length === 0}
        emptyFallback={
          <EmptyState
            title="Sem workflows ainda"
            description="Crie o primeiro para começar a automatizar o pipeline."
          />
        }
      >
        {(data) => <WorkflowTable workflows={data} />}
      </QueryBoundary>
    </div>
  )
}

function WorkflowTable({ workflows }: { workflows: Workflow[] }) {
  const navigate = useNavigate()
  return (
    <ul className="mt-6 space-y-2">
      {workflows.map((w) => (
        <li key={w.id}>
          <button
            type="button"
            onClick={() => navigate(`/workflows/${w.id}`)}
            className="bg-surface shadow-elev-1 hover:shadow-elev-2 block w-full cursor-pointer rounded-lg p-4 text-left transition"
          >
            <div className="flex items-center gap-3">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="text-ink font-medium">{w.name}</span>
                  <Badge tone={w.status === 'active' ? 'success' : 'neutral'}>{w.status}</Badge>
                  <Badge tone="neutral">{triggerLabels[w.trigger]}</Badge>
                </div>
                {w.description && (
                  <p className="text-ink-dim mt-1 truncate text-xs">{w.description}</p>
                )}
              </div>
            </div>
          </button>
        </li>
      ))}
    </ul>
  )
}

function NewWorkflowButton() {
  const create = useCreateWorkflow()
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const trimmed = name.trim()
    if (!trimmed) return
    const created = await create.mutateAsync({ name: trimmed, trigger: 'manual' })
    setName('')
    setOpen(false)
    void navigate(`/workflows/${created.id}`)
  }

  if (!open) {
    return (
      <Button type="button" variant="primary" size="sm" onClick={() => setOpen(true)}>
        <Plus className="mr-1 h-3.5 w-3.5" />
        Novo workflow
      </Button>
    )
  }
  return (
    <form className="flex items-center gap-2" onSubmit={handleSubmit}>
      <Input
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder="Nome do workflow"
        className="w-60"
      />
      <Button type="submit" variant="primary" size="sm" disabled={create.isPending}>
        Criar
      </Button>
      <Button type="button" variant="ghost" size="sm" onClick={() => setOpen(false)}>
        Cancelar
      </Button>
    </form>
  )
}
