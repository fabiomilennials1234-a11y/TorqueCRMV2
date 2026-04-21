import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  useLeaderboard,
  useLeadsSummary,
  useMessagesSummary,
  useProposalsSummary,
  useStageVolume,
  useTasksSummary,
} from '@/hooks/useAnalytics'

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

function mockJSON(body: unknown) {
  return {
    ok: true,
    status: 200,
    json: async () => body,
    headers: new Headers(),
  } as Response
}

describe('useAnalytics — hooks', () => {
  const window = { since: '2026-03-21T00:00:00Z', until: '2026-04-20T00:00:00Z' }

  it('useLeadsSummary GETs with since & until', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({ total: 10, in_window: 2, assigned: 8, unassigned: 2 })
    )
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useLeadsSummary(window), { wrapper: wrap(client) })
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    const url = String(fetchMock.mock.calls[0]?.[0])
    expect(url).toContain('/api/v1/analytics/leads')
    expect(url).toContain('since=2026-03-21T00%3A00%3A00Z')
    expect(url).toContain('until=2026-04-20T00%3A00%3A00Z')
  })

  it('useMessagesSummary scopes its path', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({ inbound: 0, outbound: 0 }))
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useMessagesSummary(window), { wrapper: wrap(client) })
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    expect(String(fetchMock.mock.calls[0]?.[0])).toContain('/api/v1/analytics/messages')
  })

  it('useTasksSummary scopes its path', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({
        pending: 0,
        in_progress: 0,
        done_in_window: 0,
        missed_in_window: 0,
        overdue: 0,
      })
    )
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useTasksSummary(window), { wrapper: wrap(client) })
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    expect(String(fetchMock.mock.calls[0]?.[0])).toContain('/api/v1/analytics/tasks')
  })

  it('useProposalsSummary scopes its path', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({
        sent_in_window: 0,
        viewed_in_window: 0,
        accepted_in_window: 0,
        rejected_in_window: 0,
        won_amount_cents: 0,
      })
    )
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useProposalsSummary(window), { wrapper: wrap(client) })
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    expect(String(fetchMock.mock.calls[0]?.[0])).toContain('/api/v1/analytics/proposals')
  })

  it('useStageVolume stays disabled without pipe id', () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useStageVolume(undefined), { wrapper: wrap(client) })
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('useStageVolume scopes its path when enabled', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({ data: [] }))
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useStageVolume('p1'), { wrapper: wrap(client) })
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/analytics/pipes/p1/stages')
  })

  it('useLeaderboard scopes its path and unwraps data', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({
        data: [
          {
            member_id: 'm1',
            leads_assigned: 5,
            tasks_completed_in_window: 2,
            proposals_accepted_in_window: 1,
          },
        ],
      })
    )
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useLeaderboard(window), { wrapper: wrap(client) })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(String(fetchMock.mock.calls[0]?.[0])).toContain('/api/v1/analytics/leaderboard')
    expect(result.current.data?.[0]?.member_id).toBe('m1')
  })
})
