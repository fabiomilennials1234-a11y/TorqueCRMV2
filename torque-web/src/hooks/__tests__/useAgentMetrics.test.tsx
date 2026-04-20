import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useAgentMetrics } from '@/hooks/useAgents'

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

describe('S42 — useAgentMetrics', () => {
  it('stays disabled without agentId or window', () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useAgentMetrics(undefined, undefined), {
      wrapper: wrap(client),
    })
    expect(fetchMock).not.toHaveBeenCalled()
    expect(result.current.isFetched).toBe(false)
  })

  it('GETs /agents/:id/metrics with since/until in query string', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({
        since: '2026-04-01T00:00:00Z',
        until: '2026-04-20T00:00:00Z',
        total_sessions: 12,
        total_messages: 48,
        tokens_input: 1200,
        tokens_output: 2400,
        avg_latency_ms: 850.5,
        sessions_by_state: { open: 2, ended: 10 },
      })
    )
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(
      () =>
        useAgentMetrics('a1', {
          since: '2026-04-01T00:00:00Z',
          until: '2026-04-20T00:00:00Z',
        }),
      { wrapper: wrap(client) }
    )
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const url = String(fetchMock.mock.calls[0]?.[0])
    expect(url).toContain('/api/v1/agents/a1/metrics')
    expect(url).toContain('since=2026-04-01T00%3A00%3A00Z')
    expect(url).toContain('until=2026-04-20T00%3A00%3A00Z')
    expect(result.current.data?.total_sessions).toBe(12)
    expect(result.current.data?.avg_latency_ms).toBe(850.5)
  })
})
