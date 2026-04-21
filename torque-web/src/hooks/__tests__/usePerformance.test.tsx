import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  useAwards,
  useCommissions,
  useCreateAward,
  useCreateGoal,
  useGoals,
  useRanking,
  useSetCommissionStatus,
} from '@/hooks/usePerformance'

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

describe('usePerformance — hooks', () => {
  it('useRanking disabled without window', () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useRanking(undefined), { wrapper: wrap(client) })
    expect(fetchMock).not.toHaveBeenCalled()
    expect(result.current.isFetched).toBe(false)
  })

  it('useRanking encodes since/until', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({ data: [], since: '2026-03-21T00:00:00Z', until: '2026-04-20T00:00:00Z' })
    )
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(
      () => useRanking({ since: '2026-03-21T00:00:00Z', until: '2026-04-20T00:00:00Z' }),
      { wrapper: wrap(client) }
    )
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    const url = String(fetchMock.mock.calls[0]?.[0])
    expect(url).toContain('/api/v1/performance/ranking')
    expect(url).toContain('since=2026-03-21T00%3A00%3A00Z')
  })

  it('useGoals GET + useCreateGoal POST', async () => {
    fetchMock
      .mockResolvedValueOnce(mockJSON({ data: [] }))
      .mockResolvedValueOnce(
        mockJSON({
          id: 'g1',
          metric: 'deals_won',
          target: 5,
          period_start: '2026-04-01T00:00:00Z',
          period_end: '2026-04-30T00:00:00Z',
          created_at: '2026-04-20T00:00:00Z',
        })
      )
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    })
    const list = renderHook(() => useGoals(), { wrapper: wrap(client) })
    await waitFor(() => expect(list.result.current.isSuccess).toBe(true))
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/performance/goals')

    const create = renderHook(() => useCreateGoal(), { wrapper: wrap(client) })
    await create.result.current.mutateAsync({
      metric: 'deals_won',
      target: 5,
      period_start: '2026-04-01T00:00:00Z',
      period_end: '2026-04-30T00:00:00Z',
    })
    expect((fetchMock.mock.calls[1]?.[1] as RequestInit).method).toBe('POST')
  })

  it('useCommissions unfiltered + filtered', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({ data: [] }))
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useCommissions(), { wrapper: wrap(client) })
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/performance/commissions')

    fetchMock.mockResolvedValueOnce(mockJSON({ data: [] }))
    const filtered = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useCommissions('m1'), { wrapper: wrap(filtered) })
    await waitFor(() => expect(fetchMock.mock.calls.length).toBeGreaterThanOrEqual(2))
    expect(String(fetchMock.mock.calls[1]?.[0])).toContain('member_id=m1')
  })

  it('useSetCommissionStatus POSTs /:id/status', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON(null, 204))
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useSetCommissionStatus('cm1'), { wrapper: wrap(client) })
    await result.current.mutateAsync({ status: 'approved' })
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/performance/commissions/cm1/status')
  })

  it('useAwards GET + useCreateAward POST', async () => {
    fetchMock
      .mockResolvedValueOnce(mockJSON({ data: [] }))
      .mockResolvedValueOnce(
        mockJSON({ id: 'a1', title: 'Top', criteria: {}, winners: [], created_at: '2026-04-20T00:00:00Z' })
      )
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    })
    const list = renderHook(() => useAwards(), { wrapper: wrap(client) })
    await waitFor(() => expect(list.result.current.isSuccess).toBe(true))
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/performance/awards')

    const create = renderHook(() => useCreateAward(), { wrapper: wrap(client) })
    await create.result.current.mutateAsync({ title: 'Top' })
    expect((fetchMock.mock.calls[1]?.[1] as RequestInit).method).toBe('POST')
  })
})
