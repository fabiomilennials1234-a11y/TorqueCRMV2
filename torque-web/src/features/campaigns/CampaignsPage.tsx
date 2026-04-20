/**
 * F08 Campanhas — CampaignsPage (S46).
 *
 * Substitui o placeholder anterior por duas abas ("Em andamento" / "Arquivadas")
 * populadas via useCampaigns, com stats cards + link para CampaignDetailPage.
 * CreateCampaignModal (wizard 3 steps) fica embutido via "Nova campanha".
 */

import { Plus } from 'lucide-react'
import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'

import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { EmptyState } from '@/ui/empty-state'
import { PageHeader } from '@/ui/page-header'
import { Skeleton } from '@/ui/skeleton'
import { QueryBoundary } from '@/components/QueryBoundary'
import {
  useCampaigns,
  type Campaign,
  type CampaignStatus,
} from '@/hooks/useCampaigns'
import { CreateCampaignModal } from '@/features/campaigns/CreateCampaignModal'

const statusTone: Record<CampaignStatus, 'success' | 'danger' | 'neutral'> = {
  draft: 'neutral',
  scheduled: 'neutral',
  running: 'success',
  paused: 'neutral',
  completed: 'success',
  cancelled: 'danger',
}

type Tab = 'active' | 'archived'

// Backend não expõe `archived` ainda; consideramos "archived" = cancelled|completed
// porque esse é o estado terminal que deixa a campanha fora da operação.
const archivedStatuses: CampaignStatus[] = ['completed', 'cancelled']

export function CampaignsPage() {
  const query = useCampaigns()
  const [tab, setTab] = useState<Tab>('active')
  const [creating, setCreating] = useState(false)

  const partitioned = useMemo(() => {
    const data = query.data ?? []
    const active = data.filter((c) => !archivedStatuses.includes(c.status))
    const archived = data.filter((c) => archivedStatuses.includes(c.status))
    return { active, archived }
  }, [query.data])

  const visible = tab === 'active' ? partitioned.active : partitioned.archived

  return (
    <div className="mx-auto max-w-5xl px-8 py-8">
      <PageHeader
        eyebrow="Disparo"
        title="Campanhas"
        description="Mensagens em massa para listas de leads filtradas."
        actions={
          <Button type="button" variant="primary" size="sm" onClick={() => setCreating(true)}>
            <Plus className="mr-1 h-3.5 w-3.5" />
            Nova campanha
          </Button>
        }
      />

      <div className="mt-4 inline-flex rounded-md bg-elevated/40 p-1 shadow-hairline">
        <button
          type="button"
          className={
            'rounded px-3 py-1 text-xs ' +
            (tab === 'active' ? 'bg-surface text-ink shadow-elev-1' : 'text-ink-dim')
          }
          onClick={() => setTab('active')}
        >
          Em andamento ({partitioned.active.length})
        </button>
        <button
          type="button"
          className={
            'rounded px-3 py-1 text-xs ' +
            (tab === 'archived' ? 'bg-surface text-ink shadow-elev-1' : 'text-ink-dim')
          }
          onClick={() => setTab('archived')}
        >
          Arquivadas ({partitioned.archived.length})
        </button>
      </div>

      <QueryBoundary
        query={query}
        loadingFallback={<Skeleton className="mt-6 h-32 w-full" />}
        isEmpty={() => visible.length === 0}
        emptyFallback={
          <EmptyState
            title={tab === 'active' ? 'Nenhuma campanha em andamento' : 'Nenhuma campanha arquivada'}
            description={
              tab === 'active'
                ? 'Crie uma nova campanha para começar a disparar.'
                : 'Campanhas finalizadas ou canceladas aparecem aqui.'
            }
          />
        }
      >
        {() => <CampaignGrid campaigns={visible} />}
      </QueryBoundary>

      {creating && <CreateCampaignModal onClose={() => setCreating(false)} />}
    </div>
  )
}

function CampaignGrid({ campaigns }: { campaigns: Campaign[] }) {
  return (
    <ul className="mt-6 grid grid-cols-1 gap-3 md:grid-cols-2">
      {campaigns.map((c) => (
        <li key={c.id}>
          <Link
            to={`/campaigns/${c.id}`}
            className="block rounded-lg bg-surface p-4 shadow-elev-1 transition hover:shadow-elev-2"
          >
            <div className="flex items-center gap-2">
              <span className="font-medium text-ink">{c.name}</span>
              <Badge tone={statusTone[c.status]}>{c.status}</Badge>
            </div>
            {c.description && (
              <p className="mt-1 truncate text-2xs text-ink-dim">{c.description}</p>
            )}
            <dl className="mt-3 grid grid-cols-4 gap-2 text-2xs">
              <Stat label="Enfileirados" value={c.stats_queued} />
              <Stat label="Enviados" value={c.stats_sent} />
              <Stat label="Falhas" value={c.stats_failed} tone="danger" />
              <Stat label="Pulados" value={c.stats_skipped} />
            </dl>
          </Link>
        </li>
      ))}
    </ul>
  )
}

function Stat({ label, value, tone }: { label: string; value: number; tone?: 'danger' }) {
  return (
    <div className="rounded-md bg-elevated/30 px-2 py-1.5">
      <dt className="text-ink-dim">{label}</dt>
      <dd
        className={
          'font-mono text-sm ' + (tone === 'danger' && value > 0 ? 'text-danger' : 'text-ink')
        }
      >
        {value.toLocaleString('pt-BR')}
      </dd>
    </div>
  )
}
