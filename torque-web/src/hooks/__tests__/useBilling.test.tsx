import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useCancelSubscription, useStartCheckout, useSubscription } from '@/hooks/useBilling'

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

describe('useBilling', () => {
  it('useSubscription hits /api/v1/billing/subscription', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({
        id: 's1',
        plan_id: 'growth',
        status: 'active',
        provider: 'mock',
        amount_cents: 19900,
        currency: 'BRL',
        created_at: '2026-04-20T00:00:00Z',
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useSubscription(), { wrapper: wrap(client) })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/billing/subscription')
  })

  it('useStartCheckout POSTs /api/v1/billing/checkout', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 201,
      json: async () => ({
        id: 's1',
        plan_id: 'growth',
        status: 'pending',
        provider: 'mock',
        amount_cents: 19900,
        currency: 'BRL',
        created_at: '2026-04-20T00:00:00Z',
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useStartCheckout(), { wrapper: wrap(client) })
    await result.current.mutateAsync({ plan_id: 'growth', amount_cents: 19900 })

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/billing/checkout')
  })

  it('useCancelSubscription POSTs /api/v1/billing/subscription/cancel', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 204,
      json: async () => null,
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useCancelSubscription(), { wrapper: wrap(client) })
    await result.current.mutateAsync()

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/billing/subscription/cancel')
  })
})
