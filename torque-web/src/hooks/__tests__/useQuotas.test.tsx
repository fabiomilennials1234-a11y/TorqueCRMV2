import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useQuota, useQuotas } from '@/hooks/useQuotas'

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
    ok: status < 400,
    status,
    json: async () => body,
    headers: new Headers(),
  } as Response
}

describe('useQuotas — observability hooks', () => {
  it('useQuotas GETs /quotas and unwraps data', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({
        data: [
          {
            resource: 'leads',
            plan_base: 100,
            purchased_addons: 0,
            admin_adjustment: 0,
            current_usage: 42,
            effective_limit: 100,
            remaining: 58,
          },
        ],
      })
    )
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useQuotas(), { wrapper: wrap(client) })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.[0]?.resource).toBe('leads')
    expect(result.current.data?.[0]?.effective_limit).toBe(100)
    const url = String(fetchMock.mock.calls[0]?.[0])
    expect(url).toContain('/quotas')
  })

  it('useQuota GETs /quotas/:resource', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({
        resource: 'leads',
        plan_base: 100,
        purchased_addons: 0,
        admin_adjustment: 0,
        current_usage: 100,
        effective_limit: 100,
        remaining: 0,
      })
    )
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useQuota('leads'), { wrapper: wrap(client) })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.remaining).toBe(0)
    const url = String(fetchMock.mock.calls[0]?.[0])
    expect(url).toContain('/quotas/leads')
  })

  it('useQuota url-encodes the resource key', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({
        resource: 'special/slash',
        plan_base: 0,
        purchased_addons: 0,
        admin_adjustment: 0,
        current_usage: 0,
        effective_limit: 0,
        remaining: 0,
      })
    )
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useQuota('special/slash'), { wrapper: wrap(client) })
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    const url = String(fetchMock.mock.calls[0]?.[0])
    expect(url).toContain('/quotas/special%2Fslash')
  })
})
