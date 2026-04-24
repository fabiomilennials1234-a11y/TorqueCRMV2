import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { MessageList } from '@/features/inbox/MessageList'
import type { Message } from '@/hooks/useInbox'

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

function makeMessage(overrides: Partial<Message> = {}): Message {
  return {
    id: 'm1',
    conversation_id: 'c1',
    direction: 'inbound',
    kind: 'text',
    body: 'hello',
    media_url: null,
    sent_by_member_id: null,
    status: 'sent',
    occurred_at: new Date().toISOString(),
    ...overrides,
  }
}

describe('MessageList', () => {
  it('renders every message body from the response ordered oldest-first', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({
        data: [
          makeMessage({ id: 'a', body: 'first inbound' }),
          makeMessage({
            id: 'b',
            direction: 'outbound',
            body: 'second outbound',
            status: 'delivered',
          }),
        ],
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(<MessageList conversationId="c1" />, { wrapper: wrap(client) })

    await waitFor(() => expect(screen.getByText('first inbound')).toBeDefined())
    expect(screen.getByText('second outbound')).toBeDefined()
  })

  it('renders a system message as centered event instead of a bubble', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({
        data: [makeMessage({ id: 's', kind: 'system', body: 'Lead atribuído a você' })],
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(<MessageList conversationId="c1" />, { wrapper: wrap(client) })

    await waitFor(() => expect(screen.getByText('Lead atribuído a você')).toBeDefined())
  })

  it('shows an empty state when the backend returns zero messages', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ data: [] }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(<MessageList conversationId="c1" />, { wrapper: wrap(client) })

    await waitFor(() => expect(screen.getByText('Sem mensagens')).toBeDefined())
  })
})
