/**
 * TV Dashboard — TVDashboardPage (S48).
 *
 * Rota fullscreen `/tv` renderizada FORA do AppShell (ver routes.tsx)
 * para operar em totens da sala de vendas. Rotaciona 3 widgets a
 * cada 20 segundos reaproveitando hooks de Analytics + Performance.
 * Sem chrome interativo — apenas leitura, fundo dark, tipografia
 * cinematográfica.
 */

import { useEffect, useMemo, useState } from 'react'

import { useLeadsSummary, useProposalsSummary } from '@/hooks/useAnalytics'
import { useRanking } from '@/hooks/usePerformance'

type Widget = 'proposals' | 'ranking' | 'leads'

const widgetOrder: Widget[] = ['proposals', 'ranking', 'leads']
const ROTATION_MS = 20_000

function windowRange(days: number): { since: string; until: string } {
  const until = new Date()
  const since = new Date(until)
  since.setDate(since.getDate() - days)
  return { since: since.toISOString(), until: until.toISOString() }
}

export function TVDashboardPage() {
  const range = useMemo(() => windowRange(30), [])
  const [widgetIdx, setWidgetIdx] = useState(0)

  useEffect(() => {
    const handle = window.setInterval(() => {
      setWidgetIdx((i) => (i + 1) % widgetOrder.length)
    }, ROTATION_MS)
    return () => window.clearInterval(handle)
  }, [])

  const current = widgetOrder[widgetIdx]!

  return (
    <div className="bg-ink text-surface min-h-screen">
      <div className="flex h-screen flex-col items-center justify-center gap-6 px-12">
        <div className="text-2xs text-surface/50 tracking-[0.3em] uppercase">
          Torque · {new Date().toLocaleDateString('pt-BR')}
        </div>
        {current === 'proposals' && <ProposalsWidget range={range} />}
        {current === 'ranking' && <RankingWidget range={range} />}
        {current === 'leads' && <LeadsWidget range={range} />}
        <div className="mt-6 flex gap-2">
          {widgetOrder.map((w, i) => (
            <span
              key={w}
              className={
                'h-1.5 w-8 rounded-full ' + (i === widgetIdx ? 'bg-accent' : 'bg-surface/20')
              }
            />
          ))}
        </div>
      </div>
    </div>
  )
}

function ProposalsWidget({ range }: { range: { since: string; until: string } }) {
  const q = useProposalsSummary(range)
  const data = q.data
  return (
    <div className="flex flex-col items-center gap-4 text-center">
      <span className="text-surface/50 text-xs tracking-[0.3em] uppercase">Propostas (30d)</span>
      <span className="font-fraunces text-surface text-7xl">
        {data ? data.sent_in_window.toLocaleString('pt-BR') : '—'}
      </span>
      <span className="text-surface/70 text-sm">
        enviadas · {data?.accepted_in_window ?? 0} aceitas
      </span>
      {data && (
        <span className="text-accent mt-3 font-mono text-2xl">
          {(data.won_amount_cents / 100).toLocaleString('pt-BR', {
            style: 'currency',
            currency: 'BRL',
          })}
        </span>
      )}
    </div>
  )
}

function RankingWidget({ range }: { range: { since: string; until: string } }) {
  const q = useRanking(range)
  const top = (q.data?.data ?? []).slice(0, 5)
  return (
    <div className="w-full max-w-xl space-y-3">
      <div className="text-surface/50 text-center text-xs tracking-[0.3em] uppercase">
        Top 5 · Últimos 30 dias
      </div>
      {top.length === 0 ? (
        <p className="text-surface/60 text-center">Ainda sem fechamentos.</p>
      ) : (
        <ol className="space-y-2">
          {top.map((e, i) => (
            <li key={e.member_id} className="bg-surface/10 flex items-center gap-3 rounded-lg p-3">
              <span className="font-fraunces text-accent w-8 text-xl">#{i + 1}</span>
              <span className="text-surface flex-1">{e.member_name}</span>
              <span className="text-surface font-mono">
                {(e.revenue_cents / 100).toLocaleString('pt-BR', {
                  style: 'currency',
                  currency: 'BRL',
                })}
              </span>
            </li>
          ))}
        </ol>
      )}
    </div>
  )
}

function LeadsWidget({ range }: { range: { since: string; until: string } }) {
  const q = useLeadsSummary(range)
  const data = q.data
  return (
    <div className="flex flex-col items-center gap-4 text-center">
      <span className="text-surface/50 text-xs tracking-[0.3em] uppercase">Leads no período</span>
      <span className="font-fraunces text-surface text-7xl">
        {data ? data.in_window.toLocaleString('pt-BR') : '—'}
      </span>
      <div className="text-surface/70 mt-2 flex gap-6 text-sm">
        <span>{data?.assigned ?? 0} atribuídos</span>
        <span>{data?.unassigned ?? 0} sem dono</span>
      </div>
    </div>
  )
}
