/**
 * Inbox — ConversationList (Fase B / S33).
 *
 * Lista de conversas da sidebar do Inbox. Consome o hook `useConversations`
 * (WS-aware — invalida em `conversation.read|assigned|state_changed` +
 * `message.sent`) e renderiza rows com preview da última mensagem, badge
 * não-lida e canal.
 *
 * Design choices contra o v8 (`ChatWhatsApp.tsx` 2443 LOC):
 * - URL-driven filters (canal/estado/responsavel/busca) — o estado vive em
 *   searchParams, pode ser compartilhado por link e não se perde em reload.
 * - Busca client-side por nome/handle — 200 conversas por tenant cabem na
 *   mente. A busca server-side vira query param em S36 quando o volume
 *   justificar.
 * - Empty + loading + error centralizados em <QueryBoundary>; nada de
 *   "tela piscando" enquanto dados chegam.
 */

import { Search } from 'lucide-react'
import { useMemo } from 'react'

import { ChannelBadge } from '@/ui/channel-badge'
import { Input } from '@/ui/input'
import { Pill } from '@/ui/pill'
import { QueryBoundary } from '@/components/QueryBoundary'
import { cn, formatRelative } from '@/lib/utils'
import {
  useConversations,
  type Conversation,
  type ConversationState,
  type ConversationsFilters,
} from '@/hooks/useInbox'

type StateFilter = 'all' | ConversationState

export interface ConversationListProps {
  /** Currently selected conversation id — highlights the row. */
  activeId: string | null
  onSelect: (id: string) => void

  /** Filter values — controlled by the parent so the page can sync to URL. */
  search: string
  onSearchChange: (value: string) => void
  stateFilter: StateFilter
  onStateFilterChange: (value: StateFilter) => void
}

export function ConversationList(props: ConversationListProps) {
  const { activeId, onSelect, search, onSearchChange, stateFilter, onStateFilterChange } = props

  // Server-side filter only for `state`. Channel + assigned_to will be
  // added when S36 lands the full filter UI.
  const filters: ConversationsFilters = useMemo(() => {
    if (stateFilter === 'all') return {}
    return { state: stateFilter }
  }, [stateFilter])

  const query = useConversations(filters)

  // Client-side name/handle search. O backend suporta filtro por estado;
  // busca textual chega em S36 quando virar query param real.
  const filtered = useMemo<Conversation[]>(() => {
    const rows = query.data ?? []
    const term = search.trim().toLowerCase()
    if (!term) return rows
    return rows.filter((c) => {
      const name = c.contact_name?.toLowerCase() ?? ''
      const handle = c.contact_handle?.toLowerCase() ?? ''
      return name.includes(term) || handle.includes(term)
    })
  }, [query.data, search])

  return (
    <aside
      aria-label="Lista de conversas"
      className="flex w-[340px] shrink-0 flex-col shadow-[inset_-1px_0_0_0_hsl(var(--hairline))]"
    >
      <div className="shrink-0 px-4 pb-3 pt-4">
        <h2 className="mb-3 font-display text-xl tracking-tightest text-ink">Conversas</h2>
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-ink-dim" />
          <Input
            placeholder="Buscar por nome ou telefone…"
            className="h-8 pl-9"
            value={search}
            onChange={(e) => onSearchChange(e.target.value)}
          />
        </div>
        <div className="mt-3 flex items-center gap-1.5">
          {(
            [
              { k: 'all', label: 'Todas' },
              { k: 'open', label: 'Abertas' },
              { k: 'pending', label: 'Aguardando' },
              { k: 'resolved', label: 'Resolvidas' },
            ] as { k: StateFilter; label: string }[]
          ).map((t) => (
            <Pill key={t.k} active={stateFilter === t.k} onClick={() => onStateFilterChange(t.k)}>
              {t.label}
            </Pill>
          ))}
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <QueryBoundary
          query={query}
          isEmpty={() => filtered.length === 0}
          emptyFallback={
            <div className="px-6 py-12 text-center text-sm text-ink-muted">
              {search ? (
                <>
                  Nenhuma conversa bate com "<span className="text-ink">{search}</span>".
                </>
              ) : (
                'Sem conversas neste filtro ainda.'
              )}
            </div>
          }
        >
          {() => (
            <ul>
              {filtered.map((c, i) => (
                <li key={c.id}>
                  <ConversationRow
                    conversation={c}
                    active={c.id === activeId}
                    separator={i > 0}
                    onClick={() => onSelect(c.id)}
                  />
                </li>
              ))}
            </ul>
          )}
        </QueryBoundary>
      </div>
    </aside>
  )
}

interface ConversationRowProps {
  conversation: Conversation
  active: boolean
  separator: boolean
  onClick: () => void
}

function ConversationRow({ conversation, active, separator, onClick }: ConversationRowProps) {
  const c = conversation
  const lastAt = c.last_message_at ? formatRelative(new Date(c.last_message_at)) : null
  const name = c.contact_name ?? c.contact_handle ?? 'Sem nome'
  const preview = c.last_message_preview ?? 'Sem mensagens ainda.'

  return (
    <button
      type="button"
      onClick={onClick}
      aria-current={active}
      className={cn(
        'block w-full px-4 py-3 text-left transition-colors',
        separator && 'shadow-[inset_0_1px_0_0_hsl(var(--hairline))]',
        active ? 'bg-elevated/60' : 'hover:bg-elevated/30'
      )}
    >
      <div className="flex items-baseline justify-between gap-2">
        <div className="flex min-w-0 items-center gap-2">
          <ChannelBadge channel={channelOf(c.channel_kind)} variant="dot" size="sm" />
          <span className={cn('truncate text-sm', active ? 'text-ink' : 'text-ink-muted')}>
            {name}
          </span>
        </div>
        {lastAt && <span className="font-metric shrink-0 text-2xs text-ink-dim">{lastAt}</span>}
      </div>
      <div className="mt-1 flex items-center justify-between gap-2">
        <p className="truncate text-xs text-ink-dim">{preview}</p>
        {c.unread_count > 0 && (
          <span
            aria-label={`${c.unread_count} mensagens não lidas`}
            className="font-metric text-background shrink-0 rounded-full bg-accent px-1.5 text-2xs tabular-nums"
          >
            {c.unread_count > 99 ? '99+' : c.unread_count}
          </span>
        )}
      </div>
    </button>
  )
}

/**
 * Backend returns a free-form channel_kind string; the UI enum is fixed.
 * Map unknown kinds to `sz_chat` as a neutral fallback — the ChannelBadge
 * still renders a readable "canal externo" pill.
 */
function channelOf(kind: string): 'whatsapp' | 'messenger' | 'instagram' | 'sz_chat' {
  if (kind === 'whatsapp' || kind === 'messenger' || kind === 'instagram' || kind === 'sz_chat') {
    return kind
  }
  return 'sz_chat'
}
