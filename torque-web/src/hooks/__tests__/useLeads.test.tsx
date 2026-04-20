import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useLeads } from '@/hooks/useLeads'

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

describe('useLeads', () => {
  it('hits /api/v1/leads with the configured filters and page size', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ data: [], meta: { next_cursor: null } }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useLeads({ search: 'Ana', page_size: 10 }), { wrapper: wrap(client) })

    await waitFor(() => expect(fetchMock).toHaveBeenCalledOnce())
    const [url] = fetchMock.mock.calls[0] ?? []
    expect(url).toContain('/api/v1/leads?')
    expect(url).toContain('page_size=10')
    expect(url).toContain('search=Ana')
  })
})
