import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useCreateProduct, useProducts } from '@/hooks/useProducts'

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

describe('useProducts', () => {
  it('fetches /api/v1/products with page_size and active_only', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({
        data: [
          {
            id: 'p1',
            name: 'License',
            price_cents: 19900,
            currency: 'BRL',
            is_active: true,
            created_at: '2026-04-19T00:00:00Z',
            updated_at: '2026-04-19T00:00:00Z',
          },
        ],
        meta: { next_cursor: null },
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(
      () => useProducts({ active_only: true }),
      { wrapper: wrap(client) }
    )
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    const url = fetchMock.mock.calls[0]?.[0] as string
    expect(url).toContain('/api/v1/products')
    expect(url).toContain('page_size=50')
    expect(url).toContain('active_only=1')
    expect(result.current.items).toHaveLength(1)
    expect(result.current.items[0]?.name).toBe('License')
  })
})

describe('useCreateProduct', () => {
  it('POSTs /api/v1/products with the body', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 201,
      json: async () => ({
        id: 'new',
        name: 'Pro',
        price_cents: 9900,
        currency: 'BRL',
        is_active: true,
        created_at: '2026-04-19T00:00:00Z',
        updated_at: '2026-04-19T00:00:00Z',
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useCreateProduct(), { wrapper: wrap(client) })
    await result.current.mutateAsync({ name: 'Pro', price_cents: 9900 })

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/products')
    expect((fetchMock.mock.calls[0]?.[1] as RequestInit | undefined)?.method).toBe('POST')
  })
})
