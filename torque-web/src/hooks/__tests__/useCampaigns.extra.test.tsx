import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  useCampaign,
  useCampaigns,
  useCancelCampaign,
  usePauseCampaign,
  useRecipients,
  useResumeCampaign,
} from '@/hooks/useCampaigns'

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
    ok: true,
    status,
    json: async () => body,
    headers: new Headers(),
  } as Response
}

describe('useCampaigns — extras', () => {
  it('useCampaigns passes status filter as query string', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({ data: [] }))
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useCampaigns('paused'), { wrapper: wrap(client) })
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    expect(String(fetchMock.mock.calls[0]?.[0])).toContain('status=paused')
  })

  it('useCampaign stays disabled without id', () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useCampaign(undefined), { wrapper: wrap(client) })
    expect(fetchMock).not.toHaveBeenCalled()
    expect(result.current.isFetched).toBe(false)
  })

  it('useCampaign GETs scoped path', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({
        id: 'c1',
        name: 'Alvo',
        template_body: 't',
        status: 'running',
        stats_queued: 0,
        stats_sent: 0,
        stats_failed: 0,
        stats_skipped: 0,
      })
    )
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useCampaign('c1'), { wrapper: wrap(client) })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/campaigns/c1')
  })

  it('useRecipients scopes the GET to the campaign id', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({ data: [] }))
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useRecipients('c1'), { wrapper: wrap(client) })
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    expect(String(fetchMock.mock.calls[0]?.[0])).toBe('/api/v1/campaigns/c1/recipients')
  })

  it('pause/resume/cancel POST the correct sub-paths', async () => {
    fetchMock
      .mockResolvedValueOnce(mockJSON(null, 204))
      .mockResolvedValueOnce(mockJSON(null, 204))
      .mockResolvedValueOnce(mockJSON(null, 204))
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })

    const pause = renderHook(() => usePauseCampaign('c1'), { wrapper: wrap(client) })
    await pause.result.current.mutateAsync()
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/campaigns/c1/pause')

    const resume = renderHook(() => useResumeCampaign('c1'), { wrapper: wrap(client) })
    await resume.result.current.mutateAsync()
    expect(fetchMock.mock.calls[1]?.[0]).toBe('/api/v1/campaigns/c1/resume')

    const cancel = renderHook(() => useCancelCampaign('c1'), { wrapper: wrap(client) })
    await cancel.result.current.mutateAsync()
    expect(fetchMock.mock.calls[2]?.[0]).toBe('/api/v1/campaigns/c1/cancel')
  })
})
