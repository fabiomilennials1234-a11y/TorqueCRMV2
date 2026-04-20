import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useConversations, useSendMessage } from '@/hooks/useInbox'

const fetchMock = vi.fn()
beforeEach(() => vi.stubGlobal('fetch', fetchMock))
afterEach(() => {
  vi.unstubAllGlobals()
  fetchMock.mockReset()
})

function wrap(client: QueryClient) {
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
}

describe('useConversations', () => {
  it('serializes filters into the query string', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true, status: 200,
      json: async () => ({ data: [] }),
      headers: new Headers(),
    } as Response)
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useConversations({ state: 'open', assigned_to: 'u1' }), { wrapper: wrap(client) })
    await waitFor(() => expect(fetchMock).toHaveBeenCalledOnce())
    const [url] = fetchMock.mock.calls[0] ?? []
    expect(url).toBe('/api/v1/conversations?state=open&assigned_to=u1')
  })
})

describe('useSendMessage', () => {
  it('POSTs to /conversations/:id/messages', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true, status: 202,
      json: async () => ({ id: 'm1', conversation_id: 'c1', direction: 'outbound', kind: 'text', body: 'hi', status: 'queued', occurred_at: '2026-04-19T00:00:00Z' }),
      headers: new Headers(),
    } as Response)
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useSendMessage('c1'), { wrapper: wrap(client) })
    await result.current.mutateAsync({ body: 'hi' })
    const [url, init] = fetchMock.mock.calls[0] ?? []
    expect(url).toBe('/api/v1/conversations/c1/messages')
    expect((init as RequestInit).method).toBe('POST')
  })
})
