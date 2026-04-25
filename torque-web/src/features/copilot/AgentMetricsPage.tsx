/**
 * F06 Copilot — AgentMetricsPage (S42).
 *
 * Route `/copilot/:id/metrics`. Admin-only per backend guard. Reads the
 * aggregation payload from GET /agents/:id/metrics and renders four KPI
 * cards + a stacked session-state bar. No Visx dependency yet — a single
 * accessible bar chart covers the S42 acceptance. Time-series + FunnelChart
 * land once the aggregation surfaces per-day buckets.
 */

import { ArrowLeft } from 'lucide-react'
import { useMemo, useState } from 'react'
import { Link, Navigate, useParams } from 'react-router-dom'

import { PageHeader } from '@/ui/page-header'
import { Skeleton } from '@/ui/skeleton'
import { friendlyMessage } from '@/api/errors'
import { useAgent, useAgentMetrics } from '@/hooks/useAgents'

type WindowKey = '7d' | '30d' | '90d'

const windowDays: Record<WindowKey, number> = { '7d': 7, '30d': 30, '90d': 90 }

function windowRange(key: WindowKey): { since: string; until: string } {
  const until = new Date()
  const since = new Date(until)
  since.setDate(since.getDate() - windowDays[key])
  return { since: since.toISOString(), until: until.toISOString() }
}

export function AgentMetricsPage() {
  const { id } = useParams<{ id: string }>()
  const [windowKey, setWindowKey] = useState<WindowKey>('30d')
  const range = useMemo(() => windowRange(windowKey), [windowKey])
  const agent = useAgent(id)
  const metrics = useAgentMetrics(id, range)

  if (!id) return <Navigate to="/copilot" replace />

  if (agent.isLoading) return <MetricsSkeleton />
  if (agent.isError || !agent.data) {
    return (
      <div className="mx-auto max-w-md p-12 text-center">
        <p className="text-danger text-sm">{friendlyMessage(agent.error)}</p>
        <Link to="/copilot" className="text-accent mt-4 inline-block text-sm underline">
          Voltar
        </Link>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-5xl px-8 py-8">
      <Link
        to={`/copilot/${id}`}
        className="text-ink-dim hover:text-ink-muted mb-3 inline-flex items-center gap-1.5 text-xs"
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        Voltar ao agente
      </Link>
      <PageHeader
        eyebrow="Copilot · Métricas"
        title={agent.data.name}
        description={`Performance do agente — janela ${windowKey}`}
      />

      <div className="bg-elevated/40 shadow-hairline mt-4 inline-flex rounded-md p-1">
        {(['7d', '30d', '90d'] as const).map((w) => (
          <button
            key={w}
            type="button"
            className={
              'rounded px-3 py-1 text-xs ' +
              (w === windowKey ? 'bg-surface text-ink shadow-elev-1' : 'text-ink-dim')
            }
            onClick={() => setWindowKey(w)}
          >
            {w}
          </button>
        ))}
      </div>

      {metrics.isLoading ? (
        <MetricsSkeleton noHeader />
      ) : metrics.isError || !metrics.data ? (
        <p className="text-danger mt-6 text-sm">{friendlyMessage(metrics.error)}</p>
      ) : (
        <MetricsContent data={metrics.data} />
      )}
    </div>
  )
}

function MetricsContent({
  data,
}: {
  data: NonNullable<ReturnType<typeof useAgentMetrics>['data']>
}) {
  const kpis = [
    { label: 'Sessões', value: data.total_sessions.toLocaleString('pt-BR') },
    { label: 'Mensagens', value: data.total_messages.toLocaleString('pt-BR') },
    {
      label: 'Tokens (in / out)',
      value: `${data.tokens_input.toLocaleString('pt-BR')} / ${data.tokens_output.toLocaleString('pt-BR')}`,
    },
    {
      label: 'Latência média',
      value:
        data.avg_latency_ms != null
          ? `${Math.round(data.avg_latency_ms).toLocaleString('pt-BR')} ms`
          : '—',
    },
  ]

  return (
    <div className="mt-6 space-y-8">
      <section className="grid grid-cols-1 gap-3 md:grid-cols-4">
        {kpis.map((k) => (
          <div key={k.label} className="bg-surface shadow-elev-1 rounded-lg p-4">
            <div className="text-2xs text-ink-dim tracking-wide uppercase">{k.label}</div>
            <div className="font-fraunces text-ink mt-2 text-2xl">{k.value}</div>
          </div>
        ))}
      </section>

      <section>
        <h2 className="text-ink mb-2 text-sm font-medium">Sessões por estado</h2>
        <SessionsStateBar buckets={data.sessions_by_state} />
      </section>
    </div>
  )
}

function SessionsStateBar({ buckets }: { buckets: Record<string, number> }) {
  const total = Object.values(buckets).reduce((acc, n) => acc + n, 0)
  if (total === 0) {
    return <p className="text-2xs text-ink-dim">Sem sessões na janela.</p>
  }
  const order: Array<{ key: string; label: string; className: string }> = [
    { key: 'open', label: 'Abertas', className: 'bg-accent/60' },
    { key: 'ended', label: 'Encerradas', className: 'bg-ink/30' },
    { key: 'escalated', label: 'Escaladas', className: 'bg-danger/50' },
  ]

  return (
    <div>
      <div
        className="bg-elevated/40 flex h-4 w-full overflow-hidden rounded-full"
        role="img"
        aria-label={`${total} sessões`}
      >
        {order.map((segment) => {
          const n = buckets[segment.key] ?? 0
          const pct = total === 0 ? 0 : (n / total) * 100
          if (pct <= 0) return null
          return (
            <div
              key={segment.key}
              className={segment.className}
              style={{ width: `${pct}%` }}
              title={`${segment.label}: ${n}`}
            />
          )
        })}
      </div>
      <ul className="text-2xs text-ink-dim mt-3 flex flex-wrap gap-4">
        {order.map((segment) => {
          const n = buckets[segment.key] ?? 0
          return (
            <li key={segment.key} className="flex items-center gap-1.5">
              <span className={`h-2 w-2 rounded-full ${segment.className}`} />
              {segment.label}
              <span className="text-ink">{n}</span>
            </li>
          )
        })}
      </ul>
    </div>
  )
}

function MetricsSkeleton({ noHeader }: { noHeader?: boolean }) {
  return (
    <div className="mx-auto max-w-5xl space-y-4 px-8 py-8">
      {!noHeader && <Skeleton className="h-8 w-64" />}
      <div className="grid grid-cols-4 gap-3">
        <Skeleton className="h-24" />
        <Skeleton className="h-24" />
        <Skeleton className="h-24" />
        <Skeleton className="h-24" />
      </div>
      <Skeleton className="h-16 w-full" />
    </div>
  )
}
