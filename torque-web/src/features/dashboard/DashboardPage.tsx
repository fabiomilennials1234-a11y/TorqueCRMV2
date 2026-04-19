import {
  ArrowUpRight,
  ArrowDownRight,
  Sparkles,
  MessageSquare,
  Columns3,
  Trophy,
  Flame,
  Timer,
} from 'lucide-react'
import { PageHeader } from '@/ui/page-header'
import { Card, CardHeader, CardTitle, CardBody, CardFooter } from '@/ui/card'
import { Sparkline } from '@/ui/spark'
import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { Avatar } from '@/ui/avatar'
import { kpis, activity, leads, formatShort } from './helpers'
import { formatCurrency, formatCompact, formatRelative } from '@/lib/utils'

export function DashboardPage() {
  const hot = leads
    .filter((l) => l.score >= 80 && l.stage !== 'vendido' && l.stage !== 'perdido')
    .sort((a, b) => b.score - a.score)
    .slice(0, 5)

  return (
    <div className="relative">
      <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-accent/40 to-transparent" />

      <div className="mx-auto max-w-[1400px] px-8">
        <PageHeader
          eyebrow="Visão geral · Hoje"
          title={
            <>
              Boa tarde, Fábio.
              <br />
              <span className="text-ink-dim">Sua operação está saudável.</span>
            </>
          }
          description="Resumo em tempo real do funil, time e conversas. Pressione ⌘K para pular direto ao lead certo."
          actions={
            <>
              <Button variant="outline" size="md">
                Últimos 7 dias
              </Button>
              <Button variant="primary" size="md">
                Compartilhar
              </Button>
            </>
          }
        />

        {/* KPI strip — claymorphism */}
        <section className="mt-8 grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-4">
          {kpis.map((k) => (
            <Kpi key={k.label} {...k} />
          ))}
        </section>

        <div className="mt-8 grid gap-6 lg:grid-cols-3">
          {/* Pipeline snapshot — clay */}
          <Card className="tactile-surface !rounded-xl !shadow-clay-1 hover:!shadow-clay-2 lg:col-span-2">
            <CardHeader>
              <CardTitle>Funil · WhatsApp</CardTitle>
              <div className="flex items-center gap-2">
                <Badge tone="neutral">24 cards ativos</Badge>
                <Button variant="ghost" size="xs">
                  Ver kanban
                </Button>
              </div>
            </CardHeader>
            <CardBody>
              <StageBar />
              <div className="mt-6 space-y-1">
                {hot.map((l) => (
                  <div
                    key={l.id}
                    className="group -mx-2 flex cursor-pointer items-center gap-3 rounded-md px-2 py-2 transition-colors hover:bg-elevated/50"
                  >
                    <div className="flex h-2 w-2 shrink-0 rounded-full bg-accent shadow-[0_0_0_3px_hsl(var(--accent)/0.15)]" />
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <span className="truncate text-sm text-ink">{l.name}</span>
                        <span className="text-2xs uppercase tracking-[0.1em] text-ink-dim">
                          {l.company}
                        </span>
                      </div>
                      <div className="mt-0.5 flex items-center gap-2 text-2xs text-ink-dim">
                        <span className="font-metric tabular-nums">{formatCurrency(l.value)}</span>
                        <span>·</span>
                        <span>{formatShort(l.stage)}</span>
                        <span>·</span>
                        <span>último toque {formatRelative(l.lastTouch)}</span>
                      </div>
                    </div>
                    <div className="font-metric text-sm tabular-nums text-accent">{l.score}</div>
                    <Avatar size="sm" fallback={l.owner.initials} />
                  </div>
                ))}
              </div>
            </CardBody>
            <CardFooter className="flex items-center justify-between">
              <span className="inline-flex items-center gap-2">
                <Flame className="h-3 w-3 text-accent" />5 leads quentes na sua operação
              </span>
              <button type="button" className="flex items-center gap-1 text-ink hover:text-accent">
                abrir tudo <ArrowUpRight className="h-3 w-3" />
              </button>
            </CardFooter>
          </Card>

          {/* Live activity — clay */}
          <Card className="tactile-surface !rounded-xl !shadow-clay-1 hover:!shadow-clay-2">
            <CardHeader>
              <CardTitle>Atividade em tempo real</CardTitle>
              <div className="flex items-center gap-1.5 text-2xs text-ink-dim">
                <span className="relative flex h-2 w-2">
                  <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-success opacity-75" />
                  <span className="relative inline-flex h-2 w-2 rounded-full bg-success" />
                </span>
                ao vivo
              </div>
            </CardHeader>
            <CardBody className="pr-3">
              <ol className="relative space-y-4">
                <div className="absolute bottom-1 left-[7px] top-1 w-px bg-hairline" />
                {activity.map((a, i) => (
                  <ActivityRow key={i} {...a} />
                ))}
              </ol>
            </CardBody>
          </Card>
        </div>

        {/* Bottom strip */}
        <div className="mt-6 grid gap-6 md:grid-cols-3">
          <ShortcutCard
            icon={Columns3}
            title="Funil WhatsApp"
            subtitle="12 cards em Qualificado"
            to="/pipeline"
          />
          <ShortcutCard
            icon={MessageSquare}
            title="9 conversas não lidas"
            subtitle="3 aguardando > 10min"
            to="/inbox"
            tone="warning"
          />
          <ShortcutCard
            icon={Sparkles}
            title="3 agentes IA ativos"
            subtitle="Handoff rate 14% · estável"
            to="/copilot"
          />
        </div>

        <div className="mb-10 mt-10 flex items-center justify-between pt-6 text-2xs text-ink-dim shadow-[inset_0_1px_0_0_hsl(var(--hairline))]">
          <div className="flex items-center gap-3">
            <Trophy className="h-3 w-3 text-accent" />
            <span className="font-metric">
              Rafael Bento lidera o mês com R$ 340k em funil movimentado
            </span>
          </div>
          <div className="flex items-center gap-3">
            <Timer className="h-3 w-3" />
            <span>Dashboard atualizado há 4s</span>
          </div>
        </div>
      </div>
    </div>
  )
}

function Kpi({
  label,
  value,
  suffix,
  delta,
  spark,
  currency,
}: {
  label: string
  value: number
  suffix?: string
  delta: number
  spark: number[]
  currency?: boolean
}) {
  const up = delta > 0
  const display = currency
    ? formatCompact(value).replace('mil', 'k')
    : value.toLocaleString('pt-BR', { maximumFractionDigits: 1 })

  return (
    <div className="tactile-kpi ease-[var(--ease-out-soft)] rounded-xl p-6 shadow-clay-1 transition-shadow duration-200 hover:shadow-clay-2">
      <div className="text-2xs font-medium uppercase tracking-[0.14em] text-ink-dim">{label}</div>
      <div className="mt-3 flex items-end justify-between gap-4">
        <div className="flex items-baseline gap-1">
          {currency && <span className="font-metric text-sm text-ink-dim">R$</span>}
          <span className="font-display text-[1.9rem] tabular-nums leading-none tracking-tightest text-ink">
            {display}
          </span>
          {suffix && <span className="font-metric pb-1 text-sm text-ink-dim">{suffix}</span>}
        </div>
        <Sparkline data={spark} width={100} height={28} />
      </div>
      <div
        className={[
          'font-metric mt-2 inline-flex items-center gap-1 text-xs tabular-nums',
          up ? 'text-success' : 'text-danger',
        ].join(' ')}
      >
        {up ? <ArrowUpRight className="h-3 w-3" /> : <ArrowDownRight className="h-3 w-3" />}
        {(delta * 100).toFixed(1)}%<span className="ml-1 text-ink-dim">vs semana passada</span>
      </div>
    </div>
  )
}

function StageBar() {
  const data = [
    { label: 'Novo', n: 8, color: 'hsl(var(--stage-1))' },
    { label: 'Abordado', n: 11, color: 'hsl(var(--stage-2))' },
    { label: 'Qualificado', n: 12, color: 'hsl(var(--stage-3))' },
    { label: 'Agendado', n: 6, color: 'hsl(var(--stage-4))' },
    { label: 'Proposta', n: 4, color: 'hsl(var(--stage-5))' },
    { label: 'Vendido', n: 9, color: 'hsl(var(--stage-6))' },
  ]
  const total = data.reduce((s, x) => s + x.n, 0)
  return (
    <div>
      <div className="flex h-2 overflow-hidden rounded-full">
        {data.map((d) => (
          <div
            key={d.label}
            className="transition-all hover:brightness-125"
            style={{
              width: `${(d.n / total) * 100}%`,
              backgroundColor: d.color,
              opacity: 0.8,
            }}
          />
        ))}
      </div>
      <div className="mt-3 grid grid-cols-6 gap-3">
        {data.map((d) => (
          <div key={d.label}>
            <div className="flex items-center gap-1.5">
              <span className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: d.color }} />
              <span className="text-2xs uppercase tracking-[0.1em] text-ink-dim">{d.label}</span>
            </div>
            <div className="font-metric mt-0.5 text-sm tabular-nums text-ink">{d.n}</div>
          </div>
        ))}
      </div>
    </div>
  )
}

function ActivityRow({
  at,
  type,
  who,
  text,
  meta,
}: {
  at: Date
  type: string
  who: string
  text: string
  meta?: string
}) {
  const dot =
    type === 'won'
      ? 'bg-success'
      : type === 'ai'
        ? 'bg-accent'
        : type === 'new'
          ? 'bg-info'
          : type === 'stage'
            ? 'bg-ink-muted'
            : 'bg-ink-dim'
  return (
    <li className="relative pl-6">
      <span className={`absolute left-1 top-1.5 h-1.5 w-1.5 rounded-full ${dot} ring-4 ring-bg`} />
      <div className="flex items-baseline justify-between gap-3">
        <div className="min-w-0 flex-1">
          <div className="text-sm text-ink">
            <span className="font-medium">{who}</span>{' '}
            <span className="text-ink-muted">{text}</span>
          </div>
          {meta && <div className="mt-0.5 text-2xs text-ink-dim">{meta}</div>}
        </div>
        <time className="font-metric text-2xs tabular-nums text-ink-dim">{formatRelative(at)}</time>
      </div>
    </li>
  )
}

function ShortcutCard({
  icon: Icon,
  title,
  subtitle,
  to,
  tone,
}: {
  icon: React.ComponentType<React.SVGProps<SVGSVGElement>>
  title: string
  subtitle: string
  to: string
  tone?: 'warning'
}) {
  return (
    <a
      href={to}
      className="tactile-surface ease-[var(--ease-out-soft)] group relative flex items-center gap-4 rounded-xl p-4 shadow-clay-1 transition-all duration-200 hover:-translate-y-px hover:shadow-clay-2 active:translate-y-0 active:shadow-clay-pressed"
    >
      <div
        className={[
          'flex h-10 w-10 items-center justify-center rounded-lg shadow-clay-pressed',
          tone === 'warning' ? 'bg-warning/10 text-warning' : 'bg-elevated/80 text-ink-muted',
        ].join(' ')}
      >
        <Icon className="h-4 w-4" strokeWidth={1.75} />
      </div>
      <div className="min-w-0 flex-1">
        <div className="text-sm font-medium text-ink">{title}</div>
        <div className="text-xs text-ink-dim">{subtitle}</div>
      </div>
      <ArrowUpRight className="h-4 w-4 text-ink-dim transition-transform duration-200 group-hover:-translate-y-0.5 group-hover:translate-x-0.5 group-hover:text-ink" />
    </a>
  )
}
