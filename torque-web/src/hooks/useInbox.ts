/**
 * F04 Inbox hooks — conversations + messages.
 *
 * List updates live via WS:
 *   - `message.sent` / incoming webhook messages → refetch conversation list
 *   - `conversation.{read,assigned,state_changed}` → patch the row in place
 */

import { useQuery, useQueryClient } from '@tanstack/react-query'

import { get, patch, post } from '@/api/client'
import { useAppMutation } from '@/hooks/useAppMutation'
import { useWSSubscribe } from '@/hooks/useWSSubscribe'

export type ConversationState = 'open' | 'pending' | 'resolved' | 'archived'
export type MessageDirection = 'inbound' | 'outbound'
export type MessageKind = 'text' | 'image' | 'audio' | 'video' | 'document' | 'sticker' | 'system'
export type MessageStatus = 'queued' | 'sent' | 'delivered' | 'read' | 'failed'

export interface Conversation {
  id: string
  channel_id: string
  channel_kind: string
  lead_id?: string | null
  contact_name?: string | null
  contact_handle?: string | null
  state: ConversationState
  assigned_to?: string | null
  unread_count: number
  last_message_at?: string | null
  last_message_preview?: string | null
}

export interface Message {
  id: string
  conversation_id: string
  direction: MessageDirection
  kind: MessageKind
  body?: string | null
  media_url?: string | null
  sent_by_member_id?: string | null
  status: MessageStatus
  occurred_at: string
}

export interface ConversationsFilters {
  state?: ConversationState
  assigned_to?: string
  channel_id?: string
}

function inboxKeys() {
  return {
    list: (f: ConversationsFilters = {}) =>
      ['inbox', 'conversations', f] as const,
    detail: (id: string) => ['inbox', 'conversations', 'detail', id] as const,
    messages: (id: string) => ['inbox', 'conversations', id, 'messages'] as const,
  }
}

/** GET /api/v1/conversations */
export function useConversations(filters: ConversationsFilters = {}) {
  const client = useQueryClient()
  const key = inboxKeys().list(filters)

  // A conversation event (read/assigned/state) patches the row in place.
  useWSSubscribe<{ assigned_to?: string | null; state?: ConversationState }>(
    ['conversation.read', 'conversation.assigned', 'conversation.state_changed'],
    (evt) => {
      if (!evt.entity_id) return
      client.setQueriesData<{ data: Conversation[] }>({ queryKey: ['inbox', 'conversations'] }, (prev) => {
        if (!prev?.data) return prev
        return {
          ...prev,
          data: prev.data.map((c) => {
            if (c.id !== evt.entity_id) return c
            if (evt.type === 'conversation.read') return { ...c, unread_count: 0 }
            if (evt.type === 'conversation.assigned')
              return { ...c, assigned_to: evt.patch?.assigned_to ?? null }
            if (evt.type === 'conversation.state_changed' && evt.patch?.state)
              return { ...c, state: evt.patch.state }
            return c
          }),
        }
      })
    }
  )

  // A new message bumps the row to the top.
  useWSSubscribe<Message>('message.sent', () => {
    void client.invalidateQueries({ queryKey: ['inbox', 'conversations'] })
  })

  return useQuery<Conversation[]>({
    queryKey: key,
    queryFn: async () => {
      const qs = new URLSearchParams()
      if (filters.state) qs.set('state', filters.state)
      if (filters.assigned_to) qs.set('assigned_to', filters.assigned_to)
      if (filters.channel_id) qs.set('channel_id', filters.channel_id)
      const suffix = qs.toString() ? `?${qs.toString()}` : ''
      const res = await get<{ data: Conversation[] }>(`/api/v1/conversations${suffix}`)
      return res.data
    },
    staleTime: 30 * 1000,
  })
}

/** GET /api/v1/conversations/:id/messages */
export function useMessages(conversationId: string | undefined) {
  const client = useQueryClient()

  // A new message inbound/outbound → append to cache.
  useWSSubscribe<Message>('message.sent', (evt) => {
    if (!conversationId || evt.patch?.conversation_id !== conversationId) return
    client.setQueryData<Message[]>(inboxKeys().messages(conversationId), (prev) => {
      const patch = evt.patch
      if (!patch) return prev
      if (!prev) return [patch]
      if (prev.some((m) => m.id === patch.id)) return prev
      return [...prev, patch]
    })
  }, [conversationId])

  return useQuery<Message[]>({
    queryKey: conversationId ? inboxKeys().messages(conversationId) : ['inbox', 'messages', 'disabled'],
    enabled: Boolean(conversationId),
    queryFn: async () =>
      (await get<{ data: Message[] }>(`/api/v1/conversations/${conversationId}/messages`)).data,
  })
}

/** POST /api/v1/conversations/:id/read */
export function useMarkConversationRead(id: string) {
  return useAppMutation<void, void>(
    () => post<void>(`/api/v1/conversations/${id}/read`, {}),
    { invalidate: [['inbox', 'conversations']], errorContext: 'conversation.read' }
  )
}

/** POST /api/v1/conversations/:id/assign */
export function useAssignConversation(id: string) {
  return useAppMutation<void, { assigned_to: string | null }>(
    (body) => post<void>(`/api/v1/conversations/${id}/assign`, body),
    { invalidate: [['inbox', 'conversations']], errorContext: 'conversation.assign' }
  )
}

/** PATCH /api/v1/conversations/:id */
export function useSetConversationState(id: string) {
  return useAppMutation<void, { state: ConversationState }>(
    (body) => patch<void>(`/api/v1/conversations/${id}`, body),
    { invalidate: [['inbox', 'conversations']], errorContext: 'conversation.state' }
  )
}

/**
 * POST /api/v1/conversations/:id/messages
 *
 * Optimistic append would require TCache (Message[]) to differ from TData
 * (Message), which useAppMutation does not support today. The WS event
 * `message.sent` patches the messages cache within ~50 ms via
 * useMessages's subscription, and invalidate on settle is the fallback.
 */
export function useSendMessage(id: string) {
  return useAppMutation<Message, { body?: string; kind?: MessageKind; media_url?: string }>(
    (b) => post<Message>(`/api/v1/conversations/${id}/messages`, b),
    {
      invalidate: [inboxKeys().messages(id), ['inbox', 'conversations']],
      errorContext: 'inbox.send',
    }
  )
}

// Expose key factory so consumers can invalidate custom segments without
// importing queryKeys directly.
export { inboxKeys }
