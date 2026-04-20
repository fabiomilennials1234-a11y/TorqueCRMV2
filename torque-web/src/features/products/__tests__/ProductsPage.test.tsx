import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { ProductsPage } from '@/features/products/ProductsPage'

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

describe('ProductsPage', () => {
  it('renders the catalog title and fetches /api/v1/products', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true, status: 200,
      json: async () => ({
        data: [{
          id: 'p1', name: 'Licença Pro', price_cents: 19900, currency: 'BRL',
          is_active: true,
          created_at: '2026-04-20T00:00:00Z', updated_at: '2026-04-20T00:00:00Z',
        }],
        meta: { next_cursor: null },
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(<ProductsPage />, { wrapper: wrap(client) })

    await waitFor(() => expect(screen.getByText('Produtos')).toBeDefined())
    await waitFor(() => expect(screen.getByText('Licença Pro')).toBeDefined())
    const firstCall = fetchMock.mock.calls[0]?.[0] as string
    expect(firstCall).toContain('/api/v1/products')
  })
})
