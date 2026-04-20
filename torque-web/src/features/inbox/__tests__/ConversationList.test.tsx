import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { ConversationList } from '@/features/inbox/ConversationList'
import type { Conversation } from '@/hooks/useInbox'

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

function makeConv(overrides: Partial<Conversation> = {}): Conversation {
  return {
    id: 'c1',
    channel_id: 'ch1',
    channel_kind: 'whatsapp',
    lead_id: null,
    contact_name: 'Alice',
    contact_handle: '+5511999999999',
    state: 'open',
    assigned_to: null,
    unread_count: 0,
    last_message_at: null,
    last_message_preview: null,
    ...overrides,
  }
}

describe('ConversationList', () => {
  it('renders conversations from the server and filters by search', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({
        data: [
          makeConv({ id: 'a', contact_name: 'Alice' }),
          makeConv({ id: 'b', contact_name: 'Bob' }),
        ],
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { rerender } = render(
      <ConversationList
        activeId={null}
        onSelect={() => {}}
        search=""
        onSearchChange={() => {}}
        stateFilter="all"
        onStateFilterChange={() => {}}
      />,
      { wrapper: wrap(client) },
    )

    await waitFor(() => expect(screen.getByText('Alice')).toBeDefined())
    expect(screen.queryByText('Bob')).not.toBeNull()

    // Apply client-side search filter — Bob should disappear.
    rerender(
      <ConversationList
        activeId={null}
        onSelect={() => {}}
        search="ali"
        onSearchChange={() => {}}
        stateFilter="all"
        onStateFilterChange={() => {}}
      />,
    )
    expect(screen.queryByText('Alice')).not.toBeNull()
    expect(screen.queryByText('Bob')).toBeNull()
  })

  it('propagates the state filter to the server as ?state=open', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ data: [] }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(
      <ConversationList
        activeId={null}
        onSelect={() => {}}
        search=""
        onSearchChange={() => {}}
        stateFilter="open"
        onStateFilterChange={() => {}}
      />,
      { wrapper: wrap(client) },
    )

    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    const url = fetchMock.mock.calls[0]?.[0] as string
    expect(url).toContain('state=open')
  })

  it('calls onSelect when a row is clicked', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ data: [makeConv({ id: 'row-1', contact_name: 'Click me' })] }),
      headers: new Headers(),
    } as Response)

    const onSelect = vi.fn()
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(
      <ConversationList
        activeId={null}
        onSelect={onSelect}
        search=""
        onSearchChange={() => {}}
        stateFilter="all"
        onStateFilterChange={() => {}}
      />,
      { wrapper: wrap(client) },
    )

    await waitFor(() => expect(screen.getByText('Click me')).toBeDefined())
    fireEvent.click(screen.getByText('Click me'))
    expect(onSelect).toHaveBeenCalledWith('row-1')
  })
})
