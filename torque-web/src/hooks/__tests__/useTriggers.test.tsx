import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  useAgentTriggers,
  useCreateTrigger,
  useDeleteTrigger,
  useUpdateTrigger,
} from '@/hooks/useAgents'

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

describe('S40 — trigger hooks', () => {
  it('useAgentTriggers stays disabled without agentId', () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useAgentTriggers(undefined), { wrapper: wrap(client) })
    expect(fetchMock).not.toHaveBeenCalled()
    expect(result.current.isFetched).toBe(false)
  })

  it('useAgentTriggers GETs scoped path', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({
        data: [
          {
            id: 't1',
            agent_id: 'a1',
            name: 'Meta Ads',
            priority: 10,
            filter: { all: [{ field: 'origin', op: 'eq', value: 'meta-ads' }] },
            is_active: true,
            created_at: '2026-04-20T00:00:00Z',
            updated_at: '2026-04-20T00:00:00Z',
          },
        ],
      })
    )
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useAgentTriggers('a1'), { wrapper: wrap(client) })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/agents/a1/triggers')
    expect(result.current.data?.[0]?.priority).toBe(10)
  })

  it('useCreateTrigger POSTs with filter body', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({
        id: 't1',
        agent_id: 'a1',
        name: 'VIP',
        priority: 5,
        filter: { all: [{ field: 'tags', op: 'contains', value: 'vip' }] },
        is_active: true,
        created_at: '2026-04-20T00:00:00Z',
        updated_at: '2026-04-20T00:00:00Z',
      })
    )
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useCreateTrigger('a1'), { wrapper: wrap(client) })
    await result.current.mutateAsync({
      name: 'VIP',
      priority: 5,
      filter: { all: [{ field: 'tags', op: 'contains', value: 'vip' }] },
    })
    const call = fetchMock.mock.calls[0]
    expect(call?.[0]).toBe('/api/v1/agents/a1/triggers')
    const init = call?.[1] as RequestInit
    expect(init.method).toBe('POST')
    const body = JSON.parse(String(init.body))
    expect(body.filter.all[0].op).toBe('contains')
  })

  it('useUpdateTrigger PATCHes /triggers/:tid with partial', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({
        id: 't1',
        agent_id: 'a1',
        name: 'VIP',
        priority: 5,
        filter: {},
        is_active: false,
        created_at: '2026-04-20T00:00:00Z',
        updated_at: '2026-04-20T00:00:00Z',
      })
    )
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useUpdateTrigger('a1', 't1'), { wrapper: wrap(client) })
    await result.current.mutateAsync({ is_active: false })
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/triggers/t1')
    expect((fetchMock.mock.calls[0]?.[1] as RequestInit).method).toBe('PATCH')
  })

  it('useDeleteTrigger DELETEs /triggers/:tid', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON(null, 204))
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useDeleteTrigger('a1', 't1'), { wrapper: wrap(client) })
    await result.current.mutateAsync()
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/triggers/t1')
    expect((fetchMock.mock.calls[0]?.[1] as RequestInit).method).toBe('DELETE')
  })
})
