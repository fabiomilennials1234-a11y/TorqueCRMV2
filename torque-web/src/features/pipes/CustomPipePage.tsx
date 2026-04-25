/**
 * F12 Pipes custom — CustomPipePage (S48).
 *
 * Rota `/pipe/:id` abre qualquer pipe (custom ou canonical) em um
 * board simples de colunas por stage + contagem de entries. Criar,
 * renomear, arquivar stages acontece em Settings (hooks admin em
 * S22 já existem); esta página é a visualização operacional que
 * um vendedor abre para ver seu pipe custom "Churn Rescue".
 *
 * KanbanPage da home ainda é seed-based; migrar ela para este
 * pattern é uma sprint própria (não cabe em S48).
 */

import { ArrowLeft } from 'lucide-react'
import { useMemo } from 'react'
import { Link, Navigate, useParams } from 'react-router-dom'

import { Badge } from '@/ui/badge'
import { EmptyState } from '@/ui/empty-state'
import { PageHeader } from '@/ui/page-header'
import { Skeleton } from '@/ui/skeleton'
import { usePipeEntries, usePipeStages, usePipes } from '@/hooks/usePipes'

export function CustomPipePage() {
  const { id } = useParams<{ id: string }>()
  if (!id) return <Navigate to="/pipeline" replace />
  return <CustomPipeInner pipeId={id} />
}

function CustomPipeInner({ pipeId }: { pipeId: string }) {
  const pipes = usePipes()
  const stages = usePipeStages(pipeId)
  const entries = usePipeEntries(pipeId)

  const pipe = pipes.data?.find((p) => p.id === pipeId)
  const entriesByStage = useMemo(() => {
    const m = new Map<string, number>()
    for (const e of entries.data ?? []) {
      m.set(e.stage_id, (m.get(e.stage_id) ?? 0) + 1)
    }
    return m
  }, [entries.data])

  if (pipes.isLoading || stages.isLoading) {
    return (
      <div className="mx-auto max-w-6xl px-8 py-8">
        <Skeleton className="h-40" />
      </div>
    )
  }
  if (!pipe) {
    return (
      <div className="mx-auto max-w-md p-12 text-center">
        <p className="text-danger text-sm">Pipe não encontrado ou fora do tenant.</p>
        <Link to="/pipeline" className="text-accent mt-4 inline-block text-sm underline">
          Voltar
        </Link>
      </div>
    )
  }

  const ordered = [...(stages.data ?? [])].sort((a, b) => a.position - b.position)

  return (
    <div className="mx-auto max-w-6xl px-8 py-8">
      <Link
        to="/pipeline"
        className="text-ink-dim hover:text-ink-muted mb-3 inline-flex items-center gap-1.5 text-xs"
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        Todos os pipes
      </Link>
      <PageHeader
        eyebrow={pipe.kind}
        title={pipe.name}
        description={pipe.is_archived ? 'Pipe arquivado' : 'Board de stages e contagem de entries.'}
      />

      {ordered.length === 0 ? (
        <EmptyState
          title="Nenhum stage configurado"
          description="Crie stages em Configurações antes de operar o pipe."
        />
      ) : (
        <div className="mt-6 flex gap-3 overflow-x-auto pb-3">
          {ordered.map((s) => (
            <div
              key={s.id}
              className="bg-surface shadow-elev-1 min-w-[220px] flex-1 rounded-lg p-3"
            >
              <div className="mb-2 flex items-center gap-2">
                <span className="text-ink truncate text-sm font-medium">{s.name}</span>
                <Badge tone="neutral" className="ml-auto">
                  {entriesByStage.get(s.id) ?? 0}
                </Badge>
                {s.is_final_positive && <Badge tone="success">ganho</Badge>}
                {s.is_final_negative && <Badge tone="danger">perda</Badge>}
              </div>
              <p className="text-2xs text-ink-dim">
                Leads nesta etapa: {entriesByStage.get(s.id) ?? 0}
              </p>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
