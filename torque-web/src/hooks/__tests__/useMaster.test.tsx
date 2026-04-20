import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  useImpersonate,
  useMasterOrganizations,
  useSystemHealth,
} from '@/hooks/useMaster'

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

describe('useMaster', () => {
  it('useSystemHealth hits /api/v1/master/health', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true, status: 200,
      json: async () => ({
        org_count: 1, active_org_count: 1, user_count: 1, lead_count: 0,
        active_subscriptions: 0, pending_subscriptions: 0,
        operations_running: 0, operations_failed_24h: 0,
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useSystemHealth(), { wrapper: wrap(client) })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/master/health')
  })

  it('useMasterOrganizations returns data[]', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true, status: 200,
      json: async () => ({
        data: [{
          id: 'o1', slug: 's', name: 'Org',
          payment_status: 'active', member_count: 1, lead_count: 10,
          created_at: '2026-04-20T00:00:00Z',
        }],
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useMasterOrganizations(), { wrapper: wrap(client) })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.[0]?.id).toBe('o1')
  })

  it('useImpersonate POSTs /api/v1/master/organizations/:id/impersonate', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true, status: 200,
      json: async () => ({
        organization_id: 'o1', team_member_id: 'm1', user_id: 'u1',
        expires_at: '2026-04-20T01:00:00Z',
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useImpersonate(), { wrapper: wrap(client) })
    await result.current.mutateAsync({ organization_id: 'o1' })

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/master/organizations/o1/impersonate')
  })
})
