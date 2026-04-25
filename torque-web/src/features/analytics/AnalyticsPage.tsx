import { ArrowUpRight, Download, Calendar } from 'lucide-react'
import { Button } from '@/ui/button'
import { Pill } from '@/ui/pill'
import { Card, CardHeader, CardTitle, CardBody } from '@/ui/card'
import { PageHeader } from '@/ui/page-header'
import { Sparkline } from '@/ui/spark'

export function AnalyticsPage() {
  return (
    <div className="mx-auto max-w-[1400px] px-8">
      <PageHeader
        eyebrow="Analytics · Comercial"
        title="Funil, time e atribuição"
        description="Métricas agregadas com granularidade diária. Todos os números são materializados — carregam em menos de 2s."
        actions={
          <>
            <Button variant="outline" size="md" className="gap-1.5">
              <Calendar className="h-4 w-4" />
              Últimos 30 dias
            </Button>
            <Button variant="secondary" size="md" className="gap-1.5">
              <Download className="h-4 w-4" />
              Exportar CSV
            </Button>
          </>
        }
      />

      <div className="mt-6 flex flex-wrap items-center gap-1.5">
        <span className="text-2xs text-ink-dim pr-1 tracking-[0.12em] uppercase">Segmento</span>
        {['Todos', 'Inbound', 'Outbound', 'Indicação'].map((s, i) => (
          <Pill key={s} active={i === 1}>
            {s}
          </Pill>
        ))}
        <div className="bg-hairline mx-2 h-4 w-px" />
        <span className="text-2xs text-ink-dim pr-1 tracking-[0.12em] uppercase">Funil</span>
        {['Todos', 'WhatsApp', 'Confirmação', 'Propostas'].map((s, i) => (
          <Pill key={s} active={i === 0}>
            {s}
          </Pill>
        ))}
      </div>

      {/* Headline metric */}
      <section className="mt-8 grid gap-6 lg:grid-cols-[1.6fr_1fr]">
        <Card>
          <CardHeader>
            <CardTitle>Receita fechada · Abril</CardTitle>
            <div className="font-metric text-success flex items-center gap-1.5 text-xs">
              <ArrowUpRight className="h-3 w-3" />
              +18.4% vs mês anterior
            </div>
          </CardHeader>
          <CardBody>
            <div className="flex items-baseline gap-3">
              <span className="font-metric text-ink-dim">R$</span>
              <span className="font-display tracking-tightest text-ink text-[3.25rem] leading-none tabular-nums">
                2,4M
              </span>
              <span className="text-ink-muted text-sm">vs R$ 2,1M · Mar</span>
            </div>

            {/* Full-width chart mock */}
            <div className="relative mt-8 h-48">
              <AreaChart />
            </div>

            <div className="bg-hairline mt-6 grid grid-cols-4 gap-px overflow-hidden rounded">
              {[
                { l: 'Meta', v: 'R$ 2,2M', t: 'up', d: '+9%' },
                { l: 'Realizado', v: 'R$ 2,4M', t: 'up', d: '109%' },
                { l: 'Funil aberto', v: 'R$ 1,3M', t: 'up', d: '+12%' },
                { l: 'Win rate', v: '28.4%', t: 'up', d: '+2.1pp' },
              ].map((k) => (
                <div key={k.l} className="bg-surface p-4">
                  <div className="text-2xs text-ink-dim tracking-[0.12em] uppercase">{k.l}</div>
                  <div className="font-metric text-ink mt-1 text-lg tabular-nums">{k.v}</div>
                  <div className="font-metric text-2xs text-success">{k.d}</div>
                </div>
              ))}
            </div>
          </CardBody>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Funil · Últimos 30d</CardTitle>
          </CardHeader>
          <CardBody>
            <Funnel />
          </CardBody>
        </Card>
      </section>

      {/* Team + Attribution */}
      <section className="mt-6 grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Ranking do time</CardTitle>
          </CardHeader>
          <CardBody className="p-0">
            <RankingTable />
          </CardBody>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Atribuição · UTM source</CardTitle>
          </CardHeader>
          <CardBody>
            <UTMList />
          </CardBody>
        </Card>
      </section>
    </div>
  )
}

function AreaChart() {
  const data = [
    420, 440, 510, 480, 530, 560, 590, 620, 650, 680, 720, 760, 800, 830, 880, 920, 980, 1040, 1100,
    1180, 1240, 1320, 1400, 1480, 1560, 1640, 1760, 1880, 2040, 2240, 2400,
  ]
  const w = 720
  const h = 180
  const min = Math.min(...data)
  const max = Math.max(...data)
  const step = w / (data.length - 1)
  const points = data.map((v, i) => {
    const x = i * step
    const y = h - ((v - min) / (max - min)) * h
    return [x, y] as const
  })
  const d = points.map(([x, y], i) => (i === 0 ? `M ${x} ${y}` : `L ${x} ${y}`)).join(' ')

  return (
    <svg viewBox={`0 0 ${w} ${h}`} className="h-full w-full">
      <defs>
        <linearGradient id="area" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="hsl(var(--accent))" stopOpacity="0.22" />
          <stop offset="100%" stopColor="hsl(var(--accent))" stopOpacity="0" />
        </linearGradient>
      </defs>
      {/* y grid */}
      {[0, 0.25, 0.5, 0.75, 1].map((t) => (
        <line
          key={t}
          x1={0}
          x2={w}
          y1={h * t}
          y2={h * t}
          stroke="hsl(var(--hairline))"
          strokeDasharray="2 4"
          strokeWidth="1"
        />
      ))}
      <path d={`${d} L ${w} ${h} L 0 ${h} Z`} fill="url(#area)" />
      <path
        d={d}
        fill="none"
        stroke="hsl(var(--accent))"
        strokeWidth="1.5"
        strokeLinejoin="round"
      />
      <circle
        cx={points[points.length - 1]![0]}
        cy={points[points.length - 1]![1]}
        r={4}
        fill="hsl(var(--accent))"
      />
      <circle
        cx={points[points.length - 1]![0]}
        cy={points[points.length - 1]![1]}
        r={8}
        fill="hsl(var(--accent))"
        opacity="0.2"
      />
    </svg>
  )
}

function Funnel() {
  const steps = [
    { label: 'Leads novos', value: 1420, pct: 100 },
    { label: 'Abordados', value: 1180, pct: 83 },
    { label: 'Qualificados', value: 780, pct: 55 },
    { label: 'Agendados', value: 420, pct: 30 },
    { label: 'Propostas', value: 210, pct: 14.8 },
    { label: 'Vendidos', value: 118, pct: 8.3 },
  ]
  return (
    <div className="space-y-2">
      {steps.map((s, i) => (
        <div key={s.label}>
          <div className="mb-1 flex items-baseline justify-between">
            <span className="text-ink-muted text-sm">{s.label}</span>
            <span className="font-metric text-ink text-sm tabular-nums">
              {s.value.toLocaleString('pt-BR')}
            </span>
          </div>
          <div className="bg-elevated/50 relative h-8 rounded-xs">
            <div
              className="absolute inset-y-0 left-0 rounded-xs"
              style={{
                width: `${s.pct}%`,
                background: `linear-gradient(90deg, hsl(var(--stage-${i + 1})) 0%, hsl(var(--stage-${i + 1})/0.6) 100%)`,
              }}
            />
            <span className="font-metric text-2xs text-ink-dim absolute inset-y-0 right-2 flex items-center tabular-nums">
              {s.pct.toFixed(1)}%
            </span>
          </div>
        </div>
      ))}
    </div>
  )
}

function RankingTable() {
  const members = [
    {
      name: 'Rafael Bento',
      initials: 'RB',
      won: 14,
      revenue: 820_000,
      response: '2.1min',
      trend: [10, 12, 11, 14, 16, 15, 18],
    },
    {
      name: 'Maíra Duarte',
      initials: 'MD',
      won: 11,
      revenue: 640_000,
      response: '3.0min',
      trend: [8, 9, 11, 10, 12, 11, 13],
    },
    {
      name: 'Tatiana Alves',
      initials: 'TA',
      won: 8,
      revenue: 430_000,
      response: '4.2min',
      trend: [5, 7, 6, 8, 7, 9, 8],
    },
    {
      name: 'Pedro Lima',
      initials: 'PL',
      won: 6,
      revenue: 280_000,
      response: '5.6min',
      trend: [3, 5, 4, 6, 5, 7, 6],
    },
  ]
  return (
    <table className="w-full text-sm">
      <thead>
        <tr className="text-2xs text-ink-dim tracking-[0.12em] uppercase">
          <th className="px-5 py-2 text-left font-medium">#</th>
          <th className="px-5 py-2 text-left font-medium">Vendedor</th>
          <th className="px-3 py-2 text-right font-medium">Ganhos</th>
          <th className="px-3 py-2 text-right font-medium">Receita</th>
          <th className="px-3 py-2 text-right font-medium">1ª resposta</th>
          <th className="px-5 py-2 text-right font-medium">Tendência</th>
        </tr>
      </thead>
      <tbody>
        {members.map((m, i) => (
          <tr
            key={m.name}
            className="hover:bg-elevated/40 shadow-[inset_0_1px_0_0_hsl(var(--hairline))] transition-colors"
          >
            <td className="px-5 py-3">
              <span
                className={`font-metric text-2xs inline-flex h-5 min-w-5 items-center justify-center rounded-xs ${i === 0 ? 'bg-accent text-bg' : 'bg-elevated text-ink-muted shadow-hairline'}`}
              >
                {i + 1}
              </span>
            </td>
            <td className="px-5 py-3">
              <div className="flex items-center gap-2.5">
                <div className="font-metric bg-elevated text-2xs text-ink-muted flex h-7 w-7 items-center justify-center rounded-full">
                  {m.initials}
                </div>
                <span className="text-ink">{m.name}</span>
              </div>
            </td>
            <td className="font-metric text-ink px-3 py-3 text-right tabular-nums">{m.won}</td>
            <td className="font-metric text-ink px-3 py-3 text-right tabular-nums">
              R$ {m.revenue.toLocaleString('pt-BR')}
            </td>
            <td className="font-metric text-ink-muted px-3 py-3 text-right tabular-nums">
              {m.response}
            </td>
            <td className="px-5 py-3 text-right">
              <Sparkline data={m.trend} width={80} height={22} />
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

function UTMList() {
  const sources = [
    { name: 'meta-ads/ig', leads: 420, pct: 38, won: 12 },
    { name: 'meta-ads/fb', leads: 280, pct: 26, won: 8 },
    { name: 'indicação', leads: 190, pct: 17, won: 14 },
    { name: 'site / form', leads: 150, pct: 14, won: 5 },
    { name: 'eventos', leads: 60, pct: 5, won: 9 },
  ]
  return (
    <div className="space-y-3">
      {sources.map((s) => (
        <div key={s.name}>
          <div className="flex items-baseline justify-between">
            <span className="font-metric text-ink text-sm">{s.name}</span>
            <div className="font-metric text-ink-muted flex items-center gap-4 text-xs tabular-nums">
              <span>{s.leads} leads</span>
              <span className="text-success">{s.won} ganhos</span>
            </div>
          </div>
          <div className="bg-hairline mt-1 h-1 overflow-hidden rounded-full">
            <div className="bg-ink-muted h-full rounded-full" style={{ width: `${s.pct}%` }} />
          </div>
        </div>
      ))}
    </div>
  )
}
