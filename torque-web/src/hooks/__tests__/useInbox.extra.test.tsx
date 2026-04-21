import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  useAssignConversation,
  useConversations,
  useMarkConversationRead,
  useMessages,
  useSendMessage,
  useSetConversationState,
} from '@/hooks/useInbox'

const fetchMock = vi.fn()
beforeEach(() => vi.stubGlobal('fetch', fetchMock))
afterEach(() => {
  vi.unstubAllGlobals()
  fetchMock.mockReset()
})

function wrap(client: QueryClient) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>
  }
}

function mockJSON(body: unknown, status = 200) {
  return {
    ok: true,
    status,
    json: async () => body,
    headers: new Headers(),
  } as Response
}

describe('useInbox — hooks', () => {
  it('useConversations builds query string from filters', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({ data: [] }))
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(
      () => useConversations({ state: 'open', assigned_to: 'me', channel_id: 'wa1' }),
      { wrapper: wrap(client) }
    )
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    const url = String(fetchMock.mock.calls[0]?.[0])
    expect(url).toContain('/api/v1/conversations')
    expect(url).toContain('state=open')
    expect(url).toContain('assigned_to=me')
    expect(url).toContain('channel_id=wa1')
  })

  it('useConversations without filters omits ?', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({ data: [] }))
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useConversations(), { wrapper: wrap(client) })
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    expect(String(fetchMock.mock.calls[0]?.[0])).toBe('/api/v1/conversations')
  })

  it('useMessages stays disabled without conversationId', () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useMessages(undefined), { wrapper: wrap(client) })
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('useMessages GETs scoped path', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({ data: [] }))
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useMessages('conv1'), { wrapper: wrap(client) })
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/conversations/conv1/messages')
  })

  it('mutations hit the right POST/PATCH paths', async () => {
    fetchMock
      .mockResolvedValueOnce(mockJSON(null, 204))
      .mockResolvedValueOnce(mockJSON(null, 204))
      .mockResolvedValueOnce(mockJSON(null, 204))
      .mockResolvedValueOnce(
        mockJSON({
          id: 'm1',
          conversation_id: 'c1',
          direction: 'outbound',
          kind: 'text',
          status: 'queued',
          occurred_at: '2026-04-20T00:00:00Z',
        })
      )
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })

    const read = renderHook(() => useMarkConversationRead('c1'), { wrapper: wrap(client) })
    await read.result.current.mutateAsync()
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/conversations/c1/read')

    const assign = renderHook(() => useAssignConversation('c1'), { wrapper: wrap(client) })
    await assign.result.current.mutateAsync({ assigned_to: 'u1' })
    expect(fetchMock.mock.calls[1]?.[0]).toBe('/api/v1/conversations/c1/assign')

    const state = renderHook(() => useSetConversationState('c1'), { wrapper: wrap(client) })
    await state.result.current.mutateAsync({ state: 'resolved' })
    expect(fetchMock.mock.calls[2]?.[0]).toBe('/api/v1/conversations/c1')
    expect((fetchMock.mock.calls[2]?.[1] as RequestInit).method).toBe('PATCH')

    const send = renderHook(() => useSendMessage('c1'), { wrapper: wrap(client) })
    await send.result.current.mutateAsync({ body: 'oi' })
    expect(fetchMock.mock.calls[3]?.[0]).toBe('/api/v1/conversations/c1/messages')
  })
})
