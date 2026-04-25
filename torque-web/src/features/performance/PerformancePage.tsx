/**
 * F09 Performance — PerformancePage (S47).
 *
 * Consolida o Performance.tsx do v8 (1443 LOC) em 4 tabs orientadas
 * a ação: Ranking (leaderboard live do ledger de proposals won),
 * Metas (progresso contra targets persistidos), Comissões (ledger
 * com lifecycle pending→approved→paid), Premiações (awards
 * históricos + snapshot JSON de winners).
 */

import { useMemo, useState } from 'react'

import { Badge } from '@/ui/badge'
import { EmptyState } from '@/ui/empty-state'
import { PageHeader } from '@/ui/page-header'
import { Skeleton } from '@/ui/skeleton'
import {
  useAwards,
  useCommissions,
  useGoals,
  useRanking,
  type Award,
  type Commission,
  type CommissionStatus,
  type Goal,
  type RankingEntry,
} from '@/hooks/usePerformance'

type Tab = 'ranking' | 'goals' | 'commissions' | 'awards'

const tabLabel: Record<Tab, string> = {
  ranking: 'Ranking',
  goals: 'Metas',
  commissions: 'Comissões',
  awards: 'Premiações',
}

function windowRange(days: number): { since: string; until: string } {
  const until = new Date()
  const since = new Date(until)
  since.setDate(since.getDate() - days)
  return { since: since.toISOString(), until: until.toISOString() }
}

export function PerformancePage() {
  const [tab, setTab] = useState<Tab>('ranking')
  return (
    <div className="mx-auto max-w-5xl px-8 py-8">
      <PageHeader
        eyebrow="Time"
        title="Performance"
        description="Ranking, metas, comissões e premiações."
      />

      <div className="bg-elevated/40 shadow-hairline mt-4 inline-flex rounded-md p-1">
        {(Object.keys(tabLabel) as Tab[]).map((t) => (
          <button
            key={t}
            type="button"
            className={
              'rounded px-3 py-1 text-xs ' +
              (t === tab ? 'bg-surface text-ink shadow-elev-1' : 'text-ink-dim')
            }
            onClick={() => setTab(t)}
          >
            {tabLabel[t]}
          </button>
        ))}
      </div>

      <div className="mt-6">
        {tab === 'ranking' && <RankingTab />}
        {tab === 'goals' && <GoalsTab />}
        {tab === 'commissions' && <CommissionsTab />}
        {tab === 'awards' && <AwardsTab />}
      </div>
    </div>
  )
}

// -------- Ranking ----------------------------------------------------

function RankingTab() {
  const [days, setDays] = useState<7 | 30 | 90>(30)
  const range = useMemo(() => windowRange(days), [days])
  const query = useRanking(range)

  return (
    <div>
      <div className="bg-elevated/40 shadow-hairline mb-3 inline-flex rounded-md p-1">
        {([7, 30, 90] as const).map((n) => (
          <button
            key={n}
            type="button"
            className={
              'text-2xs rounded px-3 py-1 ' +
              (n === days ? 'bg-surface text-ink shadow-elev-1' : 'text-ink-dim')
            }
            onClick={() => setDays(n)}
          >
            {n}d
          </button>
        ))}
      </div>

      {query.isLoading ? (
        <Skeleton className="h-48" />
      ) : !query.data || query.data.data.length === 0 ? (
        <EmptyState title="Sem dados no período" description="Nenhum fechamento encontrado." />
      ) : (
        <RankingBars entries={query.data.data} />
      )}
    </div>
  )
}

function RankingBars({ entries }: { entries: RankingEntry[] }) {
  const max = Math.max(...entries.map((e) => e.revenue_cents), 1)
  return (
    <ol className="space-y-2">
      {entries.map((e, idx) => {
        const pct = max === 0 ? 0 : (e.revenue_cents / max) * 100
        return (
          <li key={e.member_id} className="bg-surface shadow-elev-1 rounded-lg p-3">
            <div className="flex items-center gap-2 text-xs">
              <span className="text-ink-dim w-6 font-mono">#{idx + 1}</span>
              <span className="text-ink flex-1 font-medium">{e.member_name}</span>
              <span className="text-2xs text-ink-dim">{e.deals_won} deals</span>
              <span className="text-ink font-mono">
                {(e.revenue_cents / 100).toLocaleString('pt-BR', {
                  style: 'currency',
                  currency: 'BRL',
                })}
              </span>
            </div>
            <div className="bg-elevated/40 mt-2 h-2 overflow-hidden rounded-full">
              <div className="bg-accent/60 h-full" style={{ width: `${pct}%` }} />
            </div>
          </li>
        )
      })}
    </ol>
  )
}

// -------- Metas ------------------------------------------------------

function GoalsTab() {
  const query = useGoals()
  if (query.isLoading) return <Skeleton className="h-40" />
  const goals = query.data ?? []
  if (goals.length === 0) {
    return (
      <EmptyState
        title="Nenhuma meta cadastrada"
        description="Admins podem criar metas via API POST /performance/goals."
      />
    )
  }
  return (
    <ul className="space-y-2">
      {goals.map((g) => (
        <GoalRow key={g.id} goal={g} />
      ))}
    </ul>
  )
}

function GoalRow({ goal }: { goal: Goal }) {
  // Progress seria um segundo endpoint (goals/:id/progress) — para S47
  // a lista mostra o target + período; a comparação contra a série real
  // chega quando o backend expor o endpoint agregado (follow-up).
  const periodLabel = useMemo(() => {
    const s = new Date(goal.period_start).toLocaleDateString('pt-BR')
    const e = new Date(goal.period_end).toLocaleDateString('pt-BR')
    return `${s} → ${e}`
  }, [goal])
  return (
    <li className="bg-surface shadow-elev-1 rounded-lg p-3">
      <div className="flex items-center gap-2 text-xs">
        <Badge tone="neutral">{goal.metric}</Badge>
        {goal.member_id ? (
          <span className="text-2xs text-ink-dim font-mono">
            member: {goal.member_id.slice(0, 8)}
          </span>
        ) : (
          <Badge tone="neutral">org</Badge>
        )}
        <span className="text-ink ml-auto font-mono">{goal.target.toLocaleString('pt-BR')}</span>
      </div>
      <div className="text-2xs text-ink-dim mt-1">{periodLabel}</div>
    </li>
  )
}

// -------- Comissões --------------------------------------------------

const commissionTone: Record<CommissionStatus, 'success' | 'danger' | 'neutral'> = {
  pending: 'neutral',
  approved: 'success',
  paid: 'success',
  cancelled: 'danger',
}

function CommissionsTab() {
  const query = useCommissions()
  if (query.isLoading) return <Skeleton className="h-40" />
  const rows = query.data ?? []
  if (rows.length === 0) {
    return <EmptyState title="Sem comissões ainda" description="Registradas ao fechar propostas." />
  }
  const totalPending = rows
    .filter((c) => c.status === 'pending')
    .reduce((acc, c) => acc + c.amount_cents, 0)
  const totalApproved = rows
    .filter((c) => c.status === 'approved' || c.status === 'paid')
    .reduce((acc, c) => acc + c.amount_cents, 0)

  return (
    <div>
      <div className="mb-3 grid grid-cols-2 gap-3">
        <AmountCard label="Pendentes" value={totalPending} />
        <AmountCard label="Aprovadas + pagas" value={totalApproved} />
      </div>
      <ul className="space-y-1.5">
        {rows.map((c) => (
          <CommissionRow key={c.id} commission={c} />
        ))}
      </ul>
    </div>
  )
}

function CommissionRow({ commission }: { commission: Commission }) {
  return (
    <li className="bg-elevated/30 flex items-center justify-between rounded-md px-3 py-2 text-xs">
      <span className="text-ink-dim font-mono">{commission.member_id.slice(0, 8)}</span>
      <div className="flex items-center gap-2">
        <span className="text-2xs text-ink-dim font-mono">{commission.percentage.toFixed(2)}%</span>
        <span className="text-ink font-mono">
          {(commission.amount_cents / 100).toLocaleString('pt-BR', {
            style: 'currency',
            currency: commission.currency,
          })}
        </span>
        <Badge tone={commissionTone[commission.status]}>{commission.status}</Badge>
      </div>
    </li>
  )
}

function AmountCard({ label, value }: { label: string; value: number }) {
  return (
    <div className="bg-surface shadow-elev-1 rounded-lg p-3">
      <div className="text-2xs text-ink-dim tracking-wide uppercase">{label}</div>
      <div className="font-fraunces text-ink mt-2 text-2xl">
        {(value / 100).toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}
      </div>
    </div>
  )
}

// -------- Premiações ------------------------------------------------

function AwardsTab() {
  const query = useAwards()
  if (query.isLoading) return <Skeleton className="h-40" />
  const rows = query.data ?? []
  if (rows.length === 0) {
    return (
      <EmptyState
        title="Sem premiações"
        description="Emita uma premiação via API POST /performance/awards."
      />
    )
  }
  return (
    <ul className="space-y-2">
      {rows.map((a) => (
        <AwardRow key={a.id} award={a} />
      ))}
    </ul>
  )
}

function AwardRow({ award }: { award: Award }) {
  return (
    <li className="bg-surface shadow-elev-1 rounded-lg p-3">
      <div className="flex items-center gap-2">
        <span className="text-ink font-medium">{award.title}</span>
        {award.awarded_at && (
          <Badge tone="success">{new Date(award.awarded_at).toLocaleDateString('pt-BR')}</Badge>
        )}
      </div>
      {award.description && <p className="text-ink-dim mt-1 text-xs">{award.description}</p>}
    </li>
  )
}
