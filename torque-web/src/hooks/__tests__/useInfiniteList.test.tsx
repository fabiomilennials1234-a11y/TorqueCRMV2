import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useInfiniteList } from '@/hooks/useInfiniteList'

const fetchMock = vi.fn()

beforeEach(() => {
  vi.stubGlobal('fetch', fetchMock)
})

afterEach(() => {
  vi.unstubAllGlobals()
  fetchMock.mockReset()
})

type Row = { id: string; name: string }

function makePage(ids: string[], next: string | null) {
  return {
    ok: true,
    status: 200,
    json: async () => ({
      data: ids.map((id) => ({ id, name: `name-${id}` })),
      meta: { next_cursor: next },
    }),
    headers: new Headers(),
  } as Response
}

function Harness() {
  const q = useInfiniteList<Row>({
    path: '/api/v1/leads',
    queryKey: ['leads', 'list'],
    pageSize: 2,
  })
  return (
    <div>
      <ul>
        {q.items.map((r) => (
          <li key={r.id}>{r.name}</li>
        ))}
      </ul>
      <span data-testid="count">{q.items.length}</span>
      <button
        type="button"
        disabled={!q.hasNextPage || q.isFetchingNextPage}
        onClick={() => void q.fetchNextPage()}
      >
        Carregar mais
      </button>
    </div>
  )
}

function wrap(ui: React.ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={client}>{ui}</QueryClientProvider>
}

describe('useInfiniteList', () => {
  it('fetches the first page without a cursor', async () => {
    fetchMock.mockResolvedValueOnce(makePage(['a', 'b'], 'cur-1'))
    render(wrap(<Harness />))

    await waitFor(() => expect(screen.getByTestId('count').textContent).toBe('2'))
    const [url] = fetchMock.mock.calls[0] ?? []
    expect(url).toBe('/api/v1/leads?page_size=2')
  })

  it('fetches the next page using the server-provided cursor', async () => {
    fetchMock
      .mockResolvedValueOnce(makePage(['a', 'b'], 'cur-1'))
      .mockResolvedValueOnce(makePage(['c', 'd'], null))
    render(wrap(<Harness />))

    await waitFor(() => expect(screen.getByTestId('count').textContent).toBe('2'))
    await userEvent.click(screen.getByText('Carregar mais'))
    await waitFor(() => expect(screen.getByTestId('count').textContent).toBe('4'))

    const [, secondCall] = fetchMock.mock.calls
    expect(secondCall?.[0]).toBe('/api/v1/leads?page_size=2&cursor=cur-1')
  })

  it('stops when next_cursor is null', async () => {
    fetchMock.mockResolvedValueOnce(makePage(['a'], null))
    render(wrap(<Harness />))

    await waitFor(() => expect(screen.getByTestId('count').textContent).toBe('1'))
    expect(screen.getByRole('button', { name: 'Carregar mais' })).toBeDisabled()
  })
})
