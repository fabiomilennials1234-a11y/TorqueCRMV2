/**
 * F04 Inbox — shell de 2 colunas (Fase B / S33).
 *
 * Esquerda: ConversationList com filtros URL-driven + paginação já
 *   servida por useConversations (WS-aware).
 * Direita: ConversationPane — por enquanto só ConversationHeader. O
 *   MessageList real chega em S34; o ComposerBar em S35.
 *
 * Os filtros ficam em `useSearchParams`: um link copiado restaura a
 * busca, a aba e a conversa selecionada.
 */

import { Inbox as InboxIcon } from 'lucide-react'
import { useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'

import { ConversationHeader } from '@/features/inbox/ConversationHeader'
import { ConversationList } from '@/features/inbox/ConversationList'
import { MessageComposer } from '@/features/inbox/MessageComposer'
import { MessageList } from '@/features/inbox/MessageList'
import { useConversations, type ConversationState } from '@/hooks/useInbox'
import { useSession } from '@/hooks/useSession'
import { EmptyState } from '@/ui/empty-state'

type StateFilter = 'all' | ConversationState

const VALID_STATES = new Set<StateFilter>(['all', 'open', 'pending', 'resolved', 'archived'])

export function InboxPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const session = useSession()

  const search = searchParams.get('q') ?? ''
  const stateRaw = searchParams.get('state') ?? 'all'
  const stateFilter: StateFilter = VALID_STATES.has(stateRaw as StateFilter)
    ? (stateRaw as StateFilter)
    : 'all'
  const activeId = searchParams.get('c')

  const setParam = (key: string, value: string | null) => {
    const next = new URLSearchParams(searchParams)
    if (value === null || value === '') next.delete(key)
    else next.set(key, value)
    setSearchParams(next, { replace: true })
  }

  // Shared cache: ConversationList already fetches this; we read it
  // from React Query cache to find the active row synchronously.
  const conversationsQuery = useConversations(
    stateFilter === 'all' ? {} : { state: stateFilter },
  )
  const activeConversation = useMemo(
    () => (activeId ? conversationsQuery.data?.find((c) => c.id === activeId) : undefined),
    [conversationsQuery.data, activeId],
  )

  return (
    <div className="flex h-[calc(100vh-56px)]">
      <ConversationList
        activeId={activeId}
        onSelect={(id) => setParam('c', id)}
        search={search}
        onSearchChange={(v) => setParam('q', v || null)}
        stateFilter={stateFilter}
        onStateFilterChange={(v) => setParam('state', v === 'all' ? null : v)}
      />

      <section className="flex flex-1 flex-col">
        {activeConversation ? (
          <>
            {/*
              currentMemberId ligado em S36 via SessionBundle.user.teamMemberId.
              ConversationHeader usa o id para (a) renderizar badge "Minha"
              quando assigned_to bate e (b) passar o id como novo
              assignee ao clicar Assumir.
            */}
            <ConversationHeader
              conversation={activeConversation}
              {...(session.user.teamMemberId
                ? { currentMemberId: session.user.teamMemberId }
                : {})}
            />
            <MessageList conversationId={activeConversation.id} />
            <MessageComposer
              conversationId={activeConversation.id}
              {...(activeConversation.contact_name !== undefined
                ? { contactName: activeConversation.contact_name }
                : {})}
              disabled={
                activeConversation.state === 'resolved' ||
                activeConversation.state === 'archived'
              }
            />
          </>
        ) : (
          <div className="flex flex-1 items-center justify-center">
            <EmptyState
              icon={InboxIcon}
              title="Selecione uma conversa"
              description="Escolha uma conversa na lista à esquerda."
            />
          </div>
        )}
      </section>
    </div>
  )
}
