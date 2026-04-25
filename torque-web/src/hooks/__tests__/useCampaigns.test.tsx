import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useCampaigns, useCreateCampaign, useLaunchCampaign } from '@/hooks/useCampaigns'

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

describe('useCampaigns', () => {
  it('list hits /api/v1/campaigns without status', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ data: [] }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useCampaigns(), { wrapper: wrap(client) })
    await waitFor(() => expect(fetchMock).toHaveBeenCalledOnce())
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/campaigns')
  })

  it('list appends ?status=running when filter provided', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ data: [] }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useCampaigns('running'), { wrapper: wrap(client) })
    await waitFor(() => expect(fetchMock).toHaveBeenCalledOnce())
    expect(fetchMock.mock.calls[0]?.[0]).toContain('status=running')
  })

  it('create then launch hit the right endpoints in sequence', async () => {
    fetchMock
      .mockResolvedValueOnce({
        ok: true,
        status: 201,
        json: async () => ({
          id: 'c1',
          name: 'x',
          template_body: 't',
          status: 'draft',
          stats_queued: 0,
          stats_sent: 0,
          stats_failed: 0,
          stats_skipped: 0,
        }),
        headers: new Headers(),
      } as Response)
      .mockResolvedValueOnce({
        ok: true,
        status: 202,
        json: async () => ({ queued: 12 }),
        headers: new Headers(),
      } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result: create } = renderHook(() => useCreateCampaign(), { wrapper: wrap(client) })
    await create.current.mutateAsync({ name: 'x', template_body: 't' })

    const { result: launch } = renderHook(() => useLaunchCampaign('c1'), { wrapper: wrap(client) })
    await launch.current.mutateAsync({ lead_ids: ['a', 'b'] })

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/campaigns')
    expect(fetchMock.mock.calls[1]?.[0]).toBe('/api/v1/campaigns/c1/launch')
  })
})
