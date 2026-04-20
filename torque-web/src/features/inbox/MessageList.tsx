/**
 * MessageList — histórico da conversa (S34).
 *
 * Consome useMessages (WS-aware: append idempotente em message.sent).
 * Auto-scroll para baixo em nova mensagem outbound; preserva a posição
 * quando a nova é inbound e o usuário já scrollou para cima (leitura
 * de história), evitando "puxar o usuário de volta" à força.
 *
 * Escopo parcial conhecido: a carga é única (cap 100-200 no repo).
 * Infinite scroll reverso para history antiga depende de cursor
 * backend que chega em S35 junto com o envio.
 */

import { useEffect, useLayoutEffect, useMemo, useRef } from 'react'

import { MessageBubble } from '@/features/inbox/MessageBubble'
import { QueryBoundary } from '@/components/QueryBoundary'
import { EmptyState } from '@/ui/empty-state'
import { useMessages } from '@/hooks/useInbox'
import { MessagesSquare } from 'lucide-react'

export interface MessageListProps {
  conversationId: string
}

export function MessageList({ conversationId }: MessageListProps) {
  const query = useMessages(conversationId)

  const scrollRef = useRef<HTMLDivElement>(null)
  const lastCountRef = useRef(0)
  const lastOutboundIdRef = useRef<string | null>(null)

  // useMemo keeps the empty-array identity stable across renders so the
  // useLayoutEffect dep list does not thrash when query.data is undefined.
  const items = useMemo(() => query.data ?? [], [query.data])

  // Initial render + new outbound auto-scroll to bottom. Inbound leaves the
  // scroll alone if the user had scrolled up (they're reading history).
  useLayoutEffect(() => {
    const el = scrollRef.current
    if (!el) return
    const currentCount = items.length
    const previousCount = lastCountRef.current
    lastCountRef.current = currentCount

    if (currentCount === 0) return

    // First load: anchor at bottom.
    if (previousCount === 0) {
      el.scrollTop = el.scrollHeight
      const last = items[currentCount - 1]
      if (last?.direction === 'outbound') lastOutboundIdRef.current = last.id
      return
    }

    // No new messages → nothing to do.
    if (currentCount <= previousCount) return

    const lastMessage = items[currentCount - 1]
    if (!lastMessage) return

    const isNewOutbound =
      lastMessage.direction === 'outbound' && lastMessage.id !== lastOutboundIdRef.current

    // Check if the user is near the bottom before the new append. If yes,
    // keep them glued. If they had scrolled up, only follow when it's our
    // own outbound.
    const distanceFromBottom = el.scrollHeight - (el.scrollTop + el.clientHeight)
    const wasAtBottom = distanceFromBottom < 120

    if (isNewOutbound || wasAtBottom) {
      el.scrollTop = el.scrollHeight
    }
    if (isNewOutbound) lastOutboundIdRef.current = lastMessage.id
  }, [items])

  // Reset scroll anchor refs when the conversation changes.
  useEffect(() => {
    lastCountRef.current = 0
    lastOutboundIdRef.current = null
  }, [conversationId])

  return (
    <div ref={scrollRef} className="flex-1 overflow-y-auto px-6 py-4">
      <QueryBoundary
        query={query}
        isEmpty={(d) => d.length === 0}
        emptyFallback={
          <div className="flex h-full items-center justify-center">
            <EmptyState
              icon={MessagesSquare}
              title="Sem mensagens"
              description="Envie a primeira mensagem quando o ComposerBar chegar em S35."
            />
          </div>
        }
      >
        {(data) => (
          <ul className="flex flex-col gap-2">
            {data.map((m) => (
              <li key={m.id}>
                <MessageBubble message={m} />
              </li>
            ))}
          </ul>
        )}
      </QueryBoundary>
    </div>
  )
}
