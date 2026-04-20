/**
 * Copilot — lista de agentes real (S38).
 *
 * Substitui o mockup `AgentsPage.tsx` que consumia `lib/seed`. Consome
 * `useAgents` WS-aware e liga:
 *   - card click → /copilot/:id (playground)
 *   - toggle kill-switch inline (admin gesture)
 *   - Activate / Disable conforme status
 *
 * Criar novo agente fica em TODO S39 (wizard ou modal); por enquanto o
 * botão está visível mas desabilitado para manter o shell estável.
 */

import { AlertTriangle, Brain, ChevronRight, Plus } from 'lucide-react'
import { Link } from 'react-router-dom'

import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { EmptyState } from '@/ui/empty-state'
import { PageHeader } from '@/ui/page-header'
import { Skeleton } from '@/ui/skeleton'
import { QueryBoundary } from '@/components/QueryBoundary'
import {
  useActivateAgent,
  useAgents,
  useDisableAgent,
  useSetKillSwitch,
  type Agent,
} from '@/hooks/useAgents'

export function AgentListPage() {
  const query = useAgents()

  return (
    <div className="mx-auto max-w-[1400px] px-8">
      <PageHeader
        eyebrow="Agentes IA · Copilot"
        title="Time de agentes"
        description="Vendedores sintéticos com tom humano e handoff gracioso. Clique em um agente para abrir o playground."
        actions={
          <Button variant="primary" size="md" className="gap-1.5" disabled title="Criação via wizard chega em S39">
            <Plus className="h-4 w-4" />
            Novo agente
          </Button>
        }
      />

      <div className="mt-8">
        <QueryBoundary
          query={query}
          isEmpty={(d) => d.length === 0}
          loadingFallback={
            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
              <Skeleton className="h-44 w-full" />
              <Skeleton className="h-44 w-full" />
              <Skeleton className="h-44 w-full" />
            </div>
          }
          emptyFallback={
            <EmptyState
              icon={Brain}
              title="Sem agentes ainda"
              description="Quando o wizard de criação chegar (S39) você verá os agentes listados aqui."
            />
          }
        >
          {(agents) => (
            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
              {agents.map((a) => (
                <AgentCard key={a.id} agent={a} />
              ))}
            </div>
          )}
        </QueryBoundary>
      </div>
    </div>
  )
}

function AgentCard({ agent }: { agent: Agent }) {
  const activate = useActivateAgent(agent.id)
  const disable = useDisableAgent(agent.id)
  const toggleKill = useSetKillSwitch(agent.id)

  return (
    <article className="group flex flex-col rounded-lg bg-surface p-5 shadow-elev-1 transition-shadow hover:shadow-elev-2">
      <header className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <Link
            to={`/copilot/${agent.id}`}
            className="inline-flex items-center gap-1 text-base text-ink hover:text-accent"
          >
            {agent.name}
            <ChevronRight className="h-4 w-4 text-ink-dim transition-transform group-hover:translate-x-0.5" />
          </Link>
          {agent.description && (
            <p className="mt-1 line-clamp-2 text-xs text-ink-muted">{agent.description}</p>
          )}
        </div>
        <StatusBadge status={agent.status} killSwitch={agent.kill_switch} />
      </header>

      <dl className="mt-4 grid grid-cols-3 gap-3 text-2xs text-ink-dim">
        <div>
          <dt className="uppercase tracking-[0.12em]">Modelo</dt>
          <dd className="font-metric mt-0.5 truncate text-ink-muted">{agent.model}</dd>
        </div>
        <div>
          <dt className="uppercase tracking-[0.12em]">Temp</dt>
          <dd className="font-metric mt-0.5 text-ink-muted">{agent.temperature.toFixed(1)}</dd>
        </div>
        <div>
          <dt className="uppercase tracking-[0.12em]">Tokens</dt>
          <dd className="font-metric mt-0.5 text-ink-muted">{agent.max_output_tokens}</dd>
        </div>
      </dl>

      <footer className="mt-5 flex items-center justify-between gap-2">
        <Button
          variant="ghost"
          size="xs"
          disabled={toggleKill.isPending}
          onClick={() =>
            void toggleKill.mutateAsync({ enabled: !agent.kill_switch })
          }
          title="Kill-switch bloqueia todas as mensagens em vôo"
        >
          <AlertTriangle className="mr-1 h-3 w-3" />
          {agent.kill_switch ? 'Desarmar' : 'Kill-switch'}
        </Button>
        {agent.status === 'active' ? (
          <Button
            variant="outline"
            size="xs"
            disabled={disable.isPending}
            onClick={() => void disable.mutateAsync()}
          >
            Desativar
          </Button>
        ) : (
          <Button
            variant="primary"
            size="xs"
            disabled={activate.isPending}
            onClick={() => void activate.mutateAsync()}
          >
            Ativar
          </Button>
        )}
      </footer>
    </article>
  )
}

function StatusBadge({ status, killSwitch }: { status: Agent['status']; killSwitch: boolean }) {
  if (killSwitch) return <Badge tone="danger">kill-switch</Badge>
  switch (status) {
    case 'active':
      return <Badge tone="success">ativo</Badge>
    case 'disabled':
      return <Badge tone="neutral">desativado</Badge>
    case 'draft':
    default:
      return <Badge tone="warning">rascunho</Badge>
  }
}
