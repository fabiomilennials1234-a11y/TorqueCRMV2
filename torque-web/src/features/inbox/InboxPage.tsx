import { useState } from 'react'
import {
  Search,
  Paperclip,
  Smile,
  Send,
  Phone,
  Video,
  MoreVertical,
  Sparkles,
  CheckCircle2,
  ChevronLeft,
} from 'lucide-react'
import { Button } from '@/ui/button'
import { Input } from '@/ui/input'
import { Avatar } from '@/ui/avatar'
import { Badge } from '@/ui/badge'
import { Pill } from '@/ui/pill'
import { ScoreMeter } from '@/ui/score-meter'
import { conversations, messages, leads } from '@/lib/seed'
import { formatRelative, cn } from '@/lib/utils'

export function InboxPage() {
  const [active, setActive] = useState(conversations[0]!.id)
  const [tone, setTone] = useState<'all' | 'unread' | 'mine' | 'waiting'>('all')
  const activeConv = conversations.find((c) => c.id === active) ?? conversations[0]!
  const lead = leads.find((l) => l.id === activeConv.leadId)

  return (
    <div className="flex h-[calc(100vh-56px)]">
      {/* List column */}
      <aside className="flex w-[340px] shrink-0 flex-col shadow-[inset_-1px_0_0_0_hsl(var(--hairline))]">
        <div className="shrink-0 px-4 pb-3 pt-4">
          <h2 className="mb-3 font-display text-xl tracking-tightest text-ink">Conversas</h2>
          <div className="relative">
            <Search className="absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-ink-dim" />
            <Input placeholder="Buscar…" className="h-8 pl-9" />
          </div>
          <div className="mt-3 flex items-center gap-1.5">
            {[
              { k: 'all', label: 'Todas', n: 24 },
              { k: 'unread', label: 'Não lidas', n: 9 },
              { k: 'mine', label: 'Minhas', n: 7 },
              { k: 'waiting', label: 'Aguardando', n: 3 },
            ].map((t) => (
              <Pill key={t.k} active={tone === t.k} onClick={() => setTone(t.k as typeof tone)}>
                {t.label}
                <span className="font-metric ml-1 text-2xs tabular-nums text-ink-dim">{t.n}</span>
              </Pill>
            ))}
          </div>
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto">
          {conversations.map((c) => {
            const isActive = c.id === active
            return (
              <button
                key={c.id}
                onClick={() => setActive(c.id)}
                className={cn(
                  'flex w-full items-start gap-3 px-4 py-3 text-left',
                  'shadow-hairline-b transition-colors',
                  isActive ? 'bg-elevated' : 'hover:bg-elevated/40'
                )}
              >
                <div className="relative shrink-0">
                  <Avatar
                    size="md"
                    fallback={c.name
                      .split(' ')
                      .map((n) => n[0])
                      .join('')}
                  />
                  {c.online && (
                    <span className="absolute -bottom-0.5 -right-0.5 h-2.5 w-2.5 rounded-full bg-success ring-2 ring-bg" />
                  )}
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-baseline justify-between gap-2">
                    <span
                      className={cn(
                        'truncate text-sm',
                        c.unread ? 'font-medium text-ink' : 'text-ink-muted'
                      )}
                    >
                      {c.name}
                    </span>
                    <span className="font-metric shrink-0 text-2xs tabular-nums text-ink-dim">
                      {formatRelative(c.at)}
                    </span>
                  </div>
                  <div className="mt-0.5 flex items-center justify-between gap-2">
                    <span
                      className={cn(
                        'truncate text-xs',
                        c.unread ? 'text-ink-muted' : 'text-ink-dim'
                      )}
                    >
                      {c.preview}
                    </span>
                    {c.unread ? (
                      <span className="font-metric shrink-0 rounded-full bg-accent px-1.5 text-[0.625rem] tabular-nums text-bg">
                        {c.unread}
                      </span>
                    ) : null}
                  </div>
                  <div className="mt-1 text-2xs uppercase tracking-[0.1em] text-ink-dim">
                    {c.company}
                  </div>
                </div>
              </button>
            )
          })}
        </div>
      </aside>

      {/* Thread column */}
      <section className="flex min-w-0 flex-1 flex-col">
        <div className="flex shrink-0 items-center gap-3 px-6 py-3 shadow-hairline-b">
          <button className="md:hidden">
            <ChevronLeft className="h-4 w-4 text-ink-muted" />
          </button>
          <Avatar
            size="md"
            fallback={activeConv.name
              .split(' ')
              .map((n) => n[0])
              .join('')}
          />
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <span className="truncate text-sm font-medium text-ink">{activeConv.name}</span>
              <Badge tone="info">WhatsApp</Badge>
              {activeConv.online && (
                <span className="inline-flex items-center gap-1 text-2xs text-success">
                  <span className="h-1.5 w-1.5 rounded-full bg-success" />
                  online
                </span>
              )}
            </div>
            <div className="text-2xs uppercase tracking-[0.1em] text-ink-dim">
              {activeConv.company}
            </div>
          </div>
          <div className="flex items-center gap-0.5">
            <Button variant="ghost" size="icon">
              <Phone className="h-4 w-4" strokeWidth={1.75} />
            </Button>
            <Button variant="ghost" size="icon">
              <Video className="h-4 w-4" strokeWidth={1.75} />
            </Button>
            <Button variant="ghost" size="icon">
              <MoreVertical className="h-4 w-4" strokeWidth={1.75} />
            </Button>
          </div>
        </div>

        {/* AI Status strip */}
        <div className="flex shrink-0 items-center gap-3 bg-accent/5 px-6 py-2 shadow-hairline-b">
          <Sparkles className="h-3.5 w-3.5 text-accent" />
          <span className="text-xs font-medium text-accent">
            Mila está conduzindo esta conversa
          </span>
          <span className="text-2xs text-ink-dim">· qualificando BANT · última ação há 42s</span>
          <Button variant="ghost" size="xs" className="ml-auto">
            Assumir conversa
          </Button>
        </div>

        <div className="relative min-h-0 flex-1 overflow-y-auto">
          <div className="pointer-events-none absolute inset-0 bg-dot-grid opacity-30 [background-size:24px_24px]" />
          <div className="relative mx-auto max-w-[720px] space-y-5 px-6 py-6">
            <DayDivider label="Hoje" />
            {messages.map((m) => (
              <Message key={m.id} msg={m} />
            ))}

            {/* Typing indicator */}
            <div className="flex items-start gap-2">
              <Avatar size="sm" fallback="EV" />
              <div className="rounded-md bg-elevated px-3 py-2 text-sm shadow-hairline">
                <span className="inline-flex gap-1">
                  <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-ink-dim" />
                  <span
                    className="h-1.5 w-1.5 animate-pulse rounded-full bg-ink-dim"
                    style={{ animationDelay: '150ms' }}
                  />
                  <span
                    className="h-1.5 w-1.5 animate-pulse rounded-full bg-ink-dim"
                    style={{ animationDelay: '300ms' }}
                  />
                </span>
              </div>
            </div>
          </div>
        </div>

        {/* Composer */}
        <div className="shrink-0 bg-surface px-6 py-3 shadow-[inset_0_1px_0_0_hsl(var(--hairline))]">
          <div className="rounded-md bg-elevated/60 shadow-hairline">
            <textarea
              placeholder="Digite sua resposta…"
              rows={2}
              className="w-full resize-none bg-transparent px-3 py-2 text-sm text-ink placeholder:text-ink-dim focus:outline-none"
            />
            <div className="flex items-center gap-1 px-2 py-1.5 shadow-[inset_0_1px_0_0_hsl(var(--hairline))]">
              <Button variant="ghost" size="icon" className="h-7 w-7">
                <Paperclip className="h-3.5 w-3.5" />
              </Button>
              <Button variant="ghost" size="icon" className="h-7 w-7">
                <Smile className="h-3.5 w-3.5" />
              </Button>
              <Button variant="ghost" size="xs" className="ml-1 gap-1">
                <Sparkles className="h-3 w-3 text-accent" />
                Sugestão Copilot
              </Button>
              <div className="ml-auto flex items-center gap-1.5 text-2xs text-ink-dim">
                <span>Template:</span>
                <span className="font-medium text-ink-muted">Nenhum</span>
              </div>
              <Button variant="primary" size="sm" className="ml-2 gap-1.5">
                <Send className="h-3.5 w-3.5" />
                Enviar
              </Button>
            </div>
          </div>
        </div>
      </section>

      {/* Context panel */}
      <aside className="hidden w-[320px] shrink-0 flex-col overflow-y-auto shadow-[inset_1px_0_0_0_hsl(var(--hairline))] lg:flex">
        {lead && (
          <>
            <div className="px-5 py-5 shadow-hairline-b">
              <div className="flex items-start justify-between">
                <div>
                  <div className="text-2xs font-medium uppercase tracking-[0.14em] text-ink-dim">
                    Lead
                  </div>
                  <div className="mt-1 font-display text-lg tracking-tightest text-ink">
                    {lead.name}
                  </div>
                  <div className="text-xs text-ink-muted">{lead.company}</div>
                </div>
                <ScoreMeter value={lead.score} size={44} strokeWidth={3} />
              </div>
              <div className="mt-4 flex flex-wrap gap-1">
                {lead.tags.map((t) => (
                  <Badge key={t} tone={t === 'Hot' ? 'accent' : 'neutral'}>
                    {t}
                  </Badge>
                ))}
              </div>
              <Button variant="primary" size="sm" className="mt-4 w-full">
                Abrir card completo
              </Button>
            </div>

            <div className="px-5 py-5 shadow-hairline-b">
              <SidebarSection title="Funil">
                <div className="text-sm text-ink">{lead.stage}</div>
                <div className="mt-0.5 text-2xs text-ink-dim">há 3d 7h neste stage</div>
              </SidebarSection>
              <SidebarSection title="Valor">
                <div className="font-metric text-base tabular-nums text-ink">
                  R$ {lead.value.toLocaleString('pt-BR')}
                </div>
              </SidebarSection>
              <SidebarSection title="Origem">
                <div className="text-sm text-ink-muted">{lead.source}</div>
              </SidebarSection>
              <SidebarSection title="Responsável">
                <div className="flex items-center gap-2">
                  <Avatar size="sm" fallback={lead.owner.initials} />
                  <span className="text-sm text-ink">{lead.owner.name}</span>
                </div>
              </SidebarSection>
            </div>

            <div className="px-5 py-5">
              <div className="mb-3 text-2xs font-medium uppercase tracking-[0.14em] text-ink-dim">
                Copilot · sugestões
              </div>
              <div className="space-y-2">
                <SuggestionCard
                  title="Enviar termo de proposta"
                  desc="Lead sinalizou intenção. Modelo 'proposta-v2' sugerido."
                />
                <SuggestionCard
                  title="Agendar follow-up"
                  desc="Amanhã 9h · retomar discussão de escopo."
                />
              </div>
            </div>
          </>
        )}
      </aside>
    </div>
  )
}

function DayDivider({ label }: { label: string }) {
  return (
    <div className="relative flex items-center justify-center py-2">
      <span className="absolute inset-x-0 top-1/2 h-px bg-hairline" />
      <span className="relative bg-bg px-3 text-2xs uppercase tracking-[0.14em] text-ink-dim">
        {label}
      </span>
    </div>
  )
}

function Message({ msg }: { msg: (typeof messages)[number] }) {
  const isAgent = msg.who === 'agent'
  return (
    <div className={`flex ${isAgent ? 'justify-end' : 'justify-start'} gap-2`}>
      {!isAgent && <Avatar size="sm" fallback={msg.name.slice(0, 2).toUpperCase()} />}
      <div className={`max-w-[70%] ${isAgent ? 'items-end' : 'items-start'} flex flex-col`}>
        <div
          className={[
            'rounded-md px-3.5 py-2 text-sm leading-relaxed',
            isAgent
              ? 'bg-accent/10 text-ink shadow-[inset_0_0_0_1px_hsl(var(--accent)/0.2)]'
              : 'bg-elevated text-ink shadow-hairline',
          ].join(' ')}
        >
          {msg.text}
        </div>
        <div className="mt-1 flex items-center gap-1.5 text-2xs text-ink-dim">
          {msg.ai && (
            <span className="inline-flex items-center gap-1 text-accent">
              <Sparkles className="h-2.5 w-2.5" />
              Mila
            </span>
          )}
          <span>{msg.name}</span>
          <span>·</span>
          <span className="font-metric">{formatRelative(msg.at)}</span>
          {isAgent && <CheckCircle2 className="h-2.5 w-2.5 text-info" />}
        </div>
      </div>
    </div>
  )
}

function SidebarSection({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="mb-4 last:mb-0">
      <div className="mb-1 text-2xs font-medium uppercase tracking-[0.14em] text-ink-dim">
        {title}
      </div>
      {children}
    </div>
  )
}

function SuggestionCard({ title, desc }: { title: string; desc: string }) {
  return (
    <div className="group cursor-pointer rounded-md bg-elevated/40 p-3 shadow-hairline transition-colors hover:bg-elevated">
      <div className="flex items-start gap-2">
        <Sparkles className="mt-0.5 h-3 w-3 shrink-0 text-accent" />
        <div className="min-w-0 flex-1">
          <div className="text-sm font-medium text-ink">{title}</div>
          <div className="mt-0.5 text-xs leading-snug text-ink-muted">{desc}</div>
        </div>
      </div>
    </div>
  )
}
