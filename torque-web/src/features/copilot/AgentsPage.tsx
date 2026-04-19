import { Plus, Play, Pause, Sparkles, MessageSquare, Settings2, Brain } from 'lucide-react'
import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { Card, CardHeader, CardTitle, CardBody } from '@/ui/card'
import { Sparkline } from '@/ui/spark'
import { PageHeader } from '@/ui/page-header'
import { agents } from '@/lib/seed'

export function AgentsPage() {
  return (
    <div className="mx-auto max-w-[1400px] px-8">
      <PageHeader
        eyebrow="Agentes IA · Copilot"
        title="Time de agentes"
        description="Vendedores sintéticos treinados no contexto do seu negócio. Nunca robóticos — sempre humanos em tom, com handoff gracioso quando preciso."
        actions={
          <>
            <Button variant="outline" size="md">
              Biblioteca de templates
            </Button>
            <Button variant="primary" size="md" className="gap-1.5">
              <Plus className="h-4 w-4" />
              Novo agente
            </Button>
          </>
        }
      />

      {/* Roster */}
      <div className="mt-8 grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-3">
        {agents.map((a) => (
          <AgentCard key={a.id} a={a} />
        ))}
        <NewAgentCard />
      </div>

      {/* Playground */}
      <section className="mt-12">
        <div className="mb-4 flex items-end justify-between">
          <div>
            <h2 className="font-display text-[1.375rem] tracking-tightest text-ink">Playground</h2>
            <p className="mt-1 text-sm text-ink-muted">
              Simule uma conversa com o agente antes de publicar. Sem custo, sem envio real.
            </p>
          </div>
          <Button variant="secondary" size="sm">
            Carregar cenário
          </Button>
        </div>

        <div className="grid gap-6 lg:grid-cols-[1fr_320px]">
          <Card className="flex min-h-[440px] flex-col">
            <CardHeader>
              <CardTitle>Mila · Qualificação inbound</CardTitle>
              <Badge tone="accent">modo teste</Badge>
            </CardHeader>
            <div className="relative flex-1 space-y-4 bg-dot-grid px-5 py-4 [background-size:18px_18px]">
              <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_at_top,transparent,hsl(var(--surface)))]" />
              <div className="relative space-y-3">
                <TestBubble side="lead" text="Oi, vi o anúncio. Quanto custa?" />
                <TestBubble
                  side="agent"
                  ai
                  text="Oi! Tudo bom? Pra te passar um valor justo eu preciso entender 2 coisas rápidas: que volume mensal vocês trabalham e se o uso é contínuo ou sazonal?"
                />
              </div>
            </div>
            <div className="shrink-0 p-4 shadow-[inset_0_1px_0_0_hsl(var(--hairline))]">
              <div className="flex gap-2">
                <input
                  className="h-9 flex-1 rounded-md bg-elevated/60 px-3 text-sm text-ink shadow-hairline focus:outline-none"
                  placeholder="Simule como lead…"
                />
                <Button variant="primary" size="md">
                  Enviar
                </Button>
              </div>
            </div>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Métricas do agente</CardTitle>
            </CardHeader>
            <CardBody className="space-y-5">
              <MetricRow label="Conversas 24h" value="142" />
              <MetricRow label="Tempo médio de resposta" value="3.4s" />
              <MetricRow label="Cache hit LLM" value="74%" tone="up" />
              <MetricRow label="Handoff rate" value="14%" />
              <MetricRow label="Satisfação (CSAT)" value="4.7 / 5" tone="up" />
              <div className="rounded-md bg-elevated/40 p-3 shadow-hairline">
                <div className="text-2xs uppercase tracking-[0.12em] text-ink-dim">
                  Custo acumulado · mês
                </div>
                <div className="font-metric mt-1 text-lg tabular-nums text-ink">R$ 184,20</div>
                <div className="text-2xs text-ink-dim">12% do teto definido no plano</div>
              </div>
            </CardBody>
          </Card>
        </div>
      </section>
    </div>
  )
}

function AgentCard({ a }: { a: (typeof agents)[number] }) {
  const isActive = a.status === 'active'
  return (
    <Card className="transition-all hover:shadow-elev-2">
      <div className="relative overflow-hidden">
        <div
          className={[
            'absolute left-0 right-0 top-0 h-24 opacity-60',
            isActive
              ? 'bg-[radial-gradient(ellipse_at_top,hsl(var(--accent)/0.15),transparent_70%)]'
              : 'bg-[radial-gradient(ellipse_at_top,hsl(var(--ink-dim)/0.08),transparent_70%)]',
          ].join(' ')}
        />
        <div className="relative p-5">
          <div className="flex items-start gap-4">
            <div className="relative">
              <div className="flex h-12 w-12 items-center justify-center rounded-md bg-elevated font-display text-lg text-accent shadow-hairline">
                {a.name[0]}
              </div>
              {isActive && (
                <span className="absolute -bottom-0.5 -right-0.5 h-3 w-3 rounded-full bg-success ring-2 ring-surface" />
              )}
            </div>
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <h3 className="font-display text-[1.125rem] tracking-tightest text-ink">
                  {a.name}
                </h3>
                <Badge tone={isActive ? 'success' : 'neutral'}>
                  {isActive ? 'ativo' : 'pausado'}
                </Badge>
              </div>
              <div className="text-sm text-ink-muted">{a.role}</div>
              <div className="mt-1 flex items-center gap-1.5 text-2xs text-ink-dim">
                <Brain className="h-3 w-3" />
                <span className="font-metric">{a.model}</span>
                <span>·</span>
                <span>{a.tone}</span>
              </div>
            </div>
          </div>

          <div className="mt-5 grid grid-cols-2 gap-4">
            <div>
              <div className="text-2xs uppercase tracking-[0.12em] text-ink-dim">Conversas 24h</div>
              <div className="mt-0.5 flex items-baseline gap-2">
                <span className="font-metric text-lg tabular-nums text-ink">
                  {a.conversations24h}
                </span>
                <Sparkline
                  data={[3, 7, 5, 9, 11, 10, 14, 12, 15, 13, 16, 18]}
                  width={60}
                  height={20}
                />
              </div>
            </div>
            <div>
              <div className="text-2xs uppercase tracking-[0.12em] text-ink-dim">Handoff rate</div>
              <div className="font-metric mt-0.5 text-lg tabular-nums text-ink">
                {(a.handoffRate * 100).toFixed(0)}%
              </div>
            </div>
          </div>

          <div className="mt-5 flex items-center gap-1.5 pt-4 shadow-[inset_0_1px_0_0_hsl(var(--hairline))]">
            <Button variant="ghost" size="sm" className="gap-1.5">
              {isActive ? <Pause className="h-3.5 w-3.5" /> : <Play className="h-3.5 w-3.5" />}
              {isActive ? 'Pausar' : 'Ativar'}
            </Button>
            <Button variant="ghost" size="sm" className="gap-1.5">
              <Settings2 className="h-3.5 w-3.5" />
              Configurar
            </Button>
            <Button variant="ghost" size="sm" className="ml-auto gap-1.5">
              <MessageSquare className="h-3.5 w-3.5" />
              Conversas
            </Button>
          </div>
        </div>
      </div>
    </Card>
  )
}

function NewAgentCard() {
  return (
    <button className="group flex min-h-[260px] flex-col items-center justify-center rounded-lg border border-dashed border-transparent p-8 text-center shadow-[inset_0_0_0_1px_hsl(var(--hairline))] transition-shadow hover:shadow-[inset_0_0_0_1px_hsl(var(--ink-dim))]">
      <div className="mb-3 flex h-10 w-10 items-center justify-center rounded-md bg-accent/10 text-accent shadow-[inset_0_0_0_1px_hsl(var(--accent)/0.3)]">
        <Sparkles className="h-4 w-4" />
      </div>
      <div className="font-display text-lg tracking-tightest text-ink">Novo agente</div>
      <p className="mt-1 max-w-[220px] text-xs text-ink-muted">
        Defina papel, tom e objetivo. A IA cuida do resto.
      </p>
    </button>
  )
}

function TestBubble({ side, text, ai }: { side: 'lead' | 'agent'; text: string; ai?: boolean }) {
  const isAgent = side === 'agent'
  return (
    <div className={`flex ${isAgent ? 'justify-end' : 'justify-start'} gap-2`}>
      <div
        className={[
          'max-w-[80%] rounded-md px-3 py-2 text-sm leading-relaxed',
          isAgent
            ? ai
              ? 'bg-accent/10 text-ink shadow-[inset_0_0_0_1px_hsl(var(--accent)/0.2)]'
              : 'bg-ink text-bg'
            : 'bg-elevated text-ink shadow-hairline',
        ].join(' ')}
      >
        {text}
      </div>
    </div>
  )
}

function MetricRow({ label, value, tone }: { label: string; value: string; tone?: 'up' }) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-sm text-ink-muted">{label}</span>
      <span
        className={[
          'font-metric text-sm tabular-nums',
          tone === 'up' ? 'text-success' : 'text-ink',
        ].join(' ')}
      >
        {value}
      </span>
    </div>
  )
}
