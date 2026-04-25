import {
  Phone,
  Mail,
  MessageSquare,
  Send,
  Calendar,
  MapPin,
  Tag as TagIcon,
  ClipboardCheck,
  Clock,
  Sparkles,
  ArrowRight,
} from 'lucide-react'
import { Avatar } from '@/ui/avatar'
import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { Input } from '@/ui/input'
import { ScoreMeter } from '@/ui/score-meter'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/ui/tabs'
import { formatRelative } from '@/lib/utils'
import type { Lead } from '@/lib/seed'
import { stageMeta } from '@/lib/seed'

export function LeadDrawer({ lead }: { lead: Lead }) {
  return (
    <div className="flex min-h-0 flex-1 flex-col">
      {/* Header */}
      <div className="shadow-hairline-b shrink-0 px-6 pt-5 pb-4">
        <div className="flex items-start justify-between">
          <div className="min-w-0 flex-1">
            <div className="text-2xs text-ink-dim mb-2 inline-flex items-center gap-2 font-medium tracking-[0.14em] uppercase">
              <span
                className="h-1.5 w-1.5 rounded-full"
                style={{ backgroundColor: stageMeta[lead.stage].color }}
              />
              {stageMeta[lead.stage].label}
              <span className="text-ink-dim">·</span>
              {lead.source}
            </div>
            <h2 className="font-display tracking-tightest text-ink text-[1.5rem] leading-tight">
              {lead.name}
            </h2>
            <div className="text-ink-muted mt-0.5 text-sm">{lead.company}</div>
          </div>
          <ScoreMeter value={lead.score} size={52} strokeWidth={3} />
        </div>

        {/* Stats row */}
        <div className="bg-hairline mt-5 grid grid-cols-4 gap-px overflow-hidden rounded-md">
          <Stat label="Valor" value={`R$ ${lead.value.toLocaleString('pt-BR')}`} />
          <Stat label="Tempo no stage" value="3d 7h" />
          <Stat label="Último toque" value={formatRelative(lead.lastTouch)} />
          <Stat label="Mensagens" value="28" />
        </div>

        {/* Quick actions */}
        <div className="mt-4 flex items-center gap-1.5">
          <Button variant="primary" size="sm" className="gap-1.5">
            <MessageSquare className="h-3.5 w-3.5" />
            Abrir conversa
          </Button>
          <Button variant="secondary" size="sm" className="gap-1.5">
            <ArrowRight className="h-3.5 w-3.5" />
            Avançar stage
          </Button>
          <Button variant="secondary" size="sm" className="gap-1.5">
            <Calendar className="h-3.5 w-3.5" />
            Agendar
          </Button>
          <Button variant="ghost" size="sm" className="gap-1.5">
            <Sparkles className="text-accent h-3.5 w-3.5" />
            Sugestão IA
          </Button>
        </div>
      </div>

      {/* Tabs */}
      <Tabs defaultValue="overview" className="flex min-h-0 flex-1 flex-col">
        <div className="shrink-0 px-6 pt-3">
          <TabsList>
            <TabsTrigger value="overview">Visão geral</TabsTrigger>
            <TabsTrigger value="conversation">Conversa</TabsTrigger>
            <TabsTrigger value="notes">Notas</TabsTrigger>
            <TabsTrigger value="history">Histórico</TabsTrigger>
          </TabsList>
        </div>

        <TabsContent
          value="overview"
          className="mt-0 min-h-0 flex-1 space-y-6 overflow-y-auto px-6 py-5"
        >
          {/* Copilot suggestion */}
          <section className="bg-accent/5 rounded-md p-4 shadow-[inset_0_0_0_1px_hsl(var(--accent)/0.2)]">
            <div className="mb-2 flex items-center gap-2">
              <Sparkles className="text-accent h-3.5 w-3.5" />
              <span className="text-2xs text-accent font-medium tracking-[0.14em] uppercase">
                Sugestão do Copilot · Mila
              </span>
              <span className="font-metric text-2xs text-ink-dim ml-auto">há 4min</span>
            </div>
            <p className="text-ink text-sm leading-relaxed">
              Lead respondeu positivamente e pediu prazo. Próxima ação recomendada:{' '}
              <span className="text-accent">enviar termo de proposta com validade de 48h</span> e
              criar task de follow-up para amanhã 9h.
            </p>
            <div className="mt-3 flex items-center gap-1.5">
              <Button size="xs" variant="primary">
                Aplicar
              </Button>
              <Button size="xs" variant="ghost">
                Descartar
              </Button>
            </div>
          </section>

          {/* Contact info */}
          <section>
            <SectionTitle>Contato</SectionTitle>
            <dl className="space-y-2 text-sm">
              <InfoRow icon={Phone} label="Celular" value="+55 11 98214-5502" />
              <InfoRow icon={Mail} label="Email" value="lucas@kaizen.ind.br" />
              <InfoRow icon={MapPin} label="Localização" value="São Bernardo do Campo · SP" />
              <InfoRow icon={ClipboardCheck} label="CNPJ" value="12.345.678/0001-90" />
            </dl>
          </section>

          {/* Ownership */}
          <section>
            <SectionTitle>Responsável</SectionTitle>
            <div className="bg-elevated/50 flex items-center gap-3 rounded-md p-3">
              <Avatar size="lg" fallback={lead.owner.initials} />
              <div className="flex-1">
                <div className="text-ink text-sm font-medium">{lead.owner.name}</div>
                <div className="text-ink-dim text-xs">SDR · online agora</div>
              </div>
              <Button variant="ghost" size="sm">
                Reatribuir
              </Button>
            </div>
          </section>

          {/* Tags */}
          <section>
            <SectionTitle>Tags</SectionTitle>
            <div className="flex flex-wrap gap-1.5">
              {lead.tags.map((t) => (
                <Badge key={t} tone={t === 'Hot' ? 'accent' : t === 'Decisor' ? 'info' : 'neutral'}>
                  {t}
                </Badge>
              ))}
              <button className="bg-elevated text-2xs text-ink-dim hover:text-ink inline-flex h-5 items-center gap-1 rounded-xs px-1.5 tracking-[0.08em] uppercase">
                <TagIcon className="h-2.5 w-2.5" />
                adicionar
              </button>
            </div>
          </section>

          {/* Pipeline progress */}
          <section>
            <SectionTitle>Progressão no funil</SectionTitle>
            <ol className="space-y-0">
              {['novo', 'abordado', 'qualificado', 'agendado', 'proposta', 'vendido'].map(
                (s, i) => {
                  const currentIdx = [
                    'novo',
                    'abordado',
                    'qualificado',
                    'agendado',
                    'proposta',
                    'vendido',
                  ].indexOf(lead.stage)
                  const done = i < currentIdx
                  const current = i === currentIdx
                  return (
                    <li key={s} className="relative flex items-start gap-3 py-2">
                      <div className="relative mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center">
                        {i < 5 && (
                          <span
                            className={`absolute top-5 left-1/2 h-[calc(100%+4px)] w-px -translate-x-1/2 ${done ? 'bg-accent' : 'bg-hairline'}`}
                          />
                        )}
                        <span
                          className={[
                            'relative z-10 h-2 w-2 rounded-full',
                            done
                              ? 'bg-accent'
                              : current
                                ? 'bg-accent ring-accent/20 ring-4'
                                : 'bg-hairline',
                          ].join(' ')}
                        />
                      </div>
                      <div className="flex-1 pb-2">
                        <div
                          className={[
                            'text-sm capitalize',
                            current
                              ? 'text-ink font-medium'
                              : done
                                ? 'text-ink-muted'
                                : 'text-ink-dim',
                          ].join(' ')}
                        >
                          {s}
                        </div>
                        {current && <div className="text-2xs text-ink-dim">atual · há 3d 7h</div>}
                      </div>
                    </li>
                  )
                }
              )}
            </ol>
          </section>
        </TabsContent>

        <TabsContent value="conversation" className="mt-0 flex min-h-0 flex-1 flex-col px-6 py-5">
          <div className="flex-1 space-y-4 overflow-y-auto pb-4">
            {sampleThread.map((m, i) => (
              <ThreadBubble key={i} {...m} />
            ))}
          </div>
          <div className="bg-elevated/60 shadow-hairline shrink-0 rounded-md p-3">
            <div className="flex items-center gap-2">
              <Input
                placeholder="Digite uma mensagem…"
                className="h-8 flex-1 bg-transparent shadow-none"
              />
              <Button size="sm" variant="primary" className="gap-1.5">
                <Send className="h-3.5 w-3.5" />
                Enviar
              </Button>
            </div>
            <div className="text-2xs text-ink-dim mt-2 flex items-center gap-2">
              <Sparkles className="text-accent h-3 w-3" />
              Copilot sugere:{' '}
              <span className="text-ink-muted italic">
                "Consigo liberar 10% com antecipação. Fechamos?"
              </span>
            </div>
          </div>
        </TabsContent>

        <TabsContent value="notes" className="mt-0 px-6 py-5">
          <div className="space-y-3">
            {sampleNotes.map((n, i) => (
              <div key={i} className="bg-elevated/40 shadow-hairline rounded-md p-3">
                <div className="text-2xs text-ink-dim flex items-center gap-2">
                  <Avatar size="xs" fallback={n.by} />
                  <span className="text-ink-muted">{n.author}</span>
                  <span>·</span>
                  <span className="font-metric">{n.at}</span>
                </div>
                <p className="text-ink mt-2 text-sm leading-relaxed">{n.text}</p>
              </div>
            ))}
          </div>
        </TabsContent>

        <TabsContent value="history" className="mt-0 space-y-3 px-6 py-5">
          {sampleHistory.map((h, i) => (
            <HistoryItem key={i} {...h} />
          ))}
        </TabsContent>
      </Tabs>
    </div>
  )
}

function SectionTitle({ children }: { children: React.ReactNode }) {
  return (
    <h3 className="text-2xs text-ink-dim mb-3 font-medium tracking-[0.14em] uppercase">
      {children}
    </h3>
  )
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-surface p-3">
      <div className="text-2xs text-ink-dim tracking-[0.1em] uppercase">{label}</div>
      <div className="font-metric text-ink mt-0.5 text-sm tabular-nums">{value}</div>
    </div>
  )
}

function InfoRow({
  icon: Icon,
  label,
  value,
}: {
  icon: React.ComponentType<{ className?: string | undefined }>
  label: string
  value: string
}) {
  return (
    <div className="group hover:bg-elevated/50 flex items-center gap-3 rounded-sm px-2 py-1.5 transition-colors">
      <Icon className="text-ink-dim h-3.5 w-3.5" />
      <span className="text-ink-dim w-20 text-xs">{label}</span>
      <span className="font-metric text-ink flex-1 text-[0.8125rem]">{value}</span>
    </div>
  )
}

const sampleThread = [
  {
    side: 'lead',
    text: 'Oi! Queria entender melhor a capacidade que vocês atendem por mês.',
    at: 'há 2h',
  },
  {
    side: 'agent',
    ai: true,
    text: 'Claro, Lucas. Conseguimos até 12 mil peças/mês no padrão atual. Qual seu volume estimado?',
    at: 'há 2h',
  },
  {
    side: 'lead',
    text: 'Estamos falando de 4k/mês inicial, podendo escalar para 8k no Q3.',
    at: 'há 1h',
  },
  {
    side: 'agent',
    text: 'Perfeito, dá pra fechar. Posso montar proposta com modulação de volume?',
    at: 'há 45min',
  },
]

function ThreadBubble({
  side,
  text,
  at,
  ai,
}: {
  side: string
  text: string
  at: string
  ai?: boolean
}) {
  const isAgent = side === 'agent'
  return (
    <div className={`flex ${isAgent ? 'justify-end' : 'justify-start'} gap-2`}>
      {!isAgent && <Avatar size="sm" fallback="LA" />}
      <div className={`max-w-[80%] ${isAgent ? 'items-end' : 'items-start'} flex flex-col`}>
        <div
          className={[
            'rounded-md px-3 py-2 text-sm leading-relaxed',
            isAgent
              ? 'bg-accent/10 text-ink shadow-[inset_0_0_0_1px_hsl(var(--accent)/0.2)]'
              : 'bg-elevated text-ink',
          ].join(' ')}
        >
          {text}
        </div>
        <div className="text-2xs text-ink-dim mt-1 flex items-center gap-1.5">
          {ai && (
            <span className="text-accent inline-flex items-center gap-1">
              <Sparkles className="h-2.5 w-2.5" />
              Mila
            </span>
          )}
          {at}
        </div>
      </div>
    </div>
  )
}

const sampleNotes = [
  {
    author: 'Rafael Bento',
    by: 'RB',
    at: 'há 2h',
    text: 'Decisor confirmado. Pagamento pode ser em 3x sem reajuste — alinhei com financeiro.',
  },
  {
    author: 'Maíra Duarte',
    by: 'MD',
    at: 'há 1d',
    text: 'Lead veio pelo Outbound Q2 — ICP alto, matriz SP. Empresa faturou R$ 18M em 2025.',
  },
]

const sampleHistory = [
  {
    icon: Sparkles,
    tone: 'accent' as const,
    text: 'Mila qualificou BANT em 3 mensagens',
    at: 'há 2h',
  },
  {
    icon: ArrowRight,
    tone: 'muted' as const,
    text: 'Movido de Abordado → Qualificado',
    at: 'há 3d',
  },
  { icon: MessageSquare, tone: 'muted' as const, text: 'Primeira resposta do lead', at: 'há 3d' },
  {
    icon: Clock,
    tone: 'muted' as const,
    text: "Criado via Meta Ads · Campanha 'ICP-Abril'",
    at: 'há 4d',
  },
]

function HistoryItem({
  icon: Icon,
  text,
  at,
  tone,
}: {
  icon: React.ComponentType<{ className?: string | undefined }>
  text: string
  at: string
  tone: 'accent' | 'muted'
}) {
  return (
    <div className="flex items-start gap-3 text-sm">
      <div
        className={[
          'mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-sm',
          tone === 'accent' ? 'bg-accent/10 text-accent' : 'bg-elevated text-ink-dim',
        ].join(' ')}
      >
        <Icon className="h-3 w-3" />
      </div>
      <div className="flex-1">
        <div className="text-ink">{text}</div>
        <div className="font-metric text-2xs text-ink-dim">{at}</div>
      </div>
    </div>
  )
}
