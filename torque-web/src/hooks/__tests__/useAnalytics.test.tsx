import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useLeadsSummary, useLeaderboard } from '@/hooks/useAnalytics'

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

describe('useAnalytics', () => {
  it('leads summary serializes since/until', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true, status: 200,
      json: async () => ({ total: 0, in_window: 0, assigned: 0, unassigned: 0 }),
      headers: new Headers(),
    } as Response)
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const since = '2026-04-01T00:00:00Z'
    const until = '2026-04-30T00:00:00Z'
    const { result } = renderHook(() => useLeadsSummary({ since, until }), { wrapper: wrap(client) })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const url = fetchMock.mock.calls[0]?.[0] as string
    expect(url.startsWith('/api/v1/analytics/leads?')).toBe(true)
    expect(url).toContain('since=')
    expect(url).toContain('until=')
  })

  it('leaderboard disabled when since is empty', async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useLeaderboard({ since: '' }), { wrapper: wrap(client) })
    // enabled:false → no fetch fired.
    expect(fetchMock).not.toHaveBeenCalled()
  })
})
