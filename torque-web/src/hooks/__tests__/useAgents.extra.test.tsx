import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  useActivateAgent,
  useAgent,
  useCreateCollection,
  useDeleteAgent,
  useDisableAgent,
  useEndSession,
  useEnqueueSource,
  useOpenSession,
  useSessionMessages,
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
    ok: true, status, json: async () => body, headers: new Headers(),
  } as Response
}

describe('useAgents extras', () => {
  it('useAgent GETs /api/v1/agents/:id', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({
      id: 'a1', name: 'A', model: 'gpt', temperature: 0.3,
      max_output_tokens: 1024, tools_allowlist: [], kill_switch: false, status: 'active',
    }))
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useAgent('a1'), { wrapper: wrap(client) })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/agents/a1')
  })

  it('activate/disable/delete hit the right sub-path', async () => {
    fetchMock
      .mockResolvedValueOnce(mockJSON(null, 204))
      .mockResolvedValueOnce(mockJSON(null, 204))
      .mockResolvedValueOnce(mockJSON(null, 204))

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result: act } = renderHook(() => useActivateAgent('a1'), { wrapper: wrap(client) })
    await act.current.mutateAsync()
    const { result: dis } = renderHook(() => useDisableAgent('a1'), { wrapper: wrap(client) })
    await dis.current.mutateAsync()
    const { result: del } = renderHook(() => useDeleteAgent('a1'), { wrapper: wrap(client) })
    await del.current.mutateAsync()

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/agents/a1/activate')
    expect(fetchMock.mock.calls[1]?.[0]).toBe('/api/v1/agents/a1/disable')
    expect(fetchMock.mock.calls[2]?.[0]).toBe('/api/v1/agents/a1')
    expect((fetchMock.mock.calls[2]?.[1] as RequestInit).method).toBe('DELETE')
  })

  it('session open/messages/end wire correctly', async () => {
    fetchMock
      .mockResolvedValueOnce(mockJSON({ id: 's1', agent_id: 'a1', status: 'open' }, 201))
      .mockResolvedValueOnce(mockJSON({ data: [] }))
      .mockResolvedValueOnce(mockJSON(null, 204))

    const client = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    })
    const { result: open } = renderHook(() => useOpenSession('a1'), { wrapper: wrap(client) })
    await open.current.mutateAsync({})
    const { result: msgs } = renderHook(() => useSessionMessages('s1'), { wrapper: wrap(client) })
    await waitFor(() => expect(msgs.current.isSuccess).toBe(true))
    const { result: end } = renderHook(() => useEndSession('s1'), { wrapper: wrap(client) })
    await end.current.mutateAsync()

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/agents/a1/sessions')
    expect(fetchMock.mock.calls[1]?.[0]).toContain('/api/v1/sessions/s1/messages')
    expect(fetchMock.mock.calls[2]?.[0]).toBe('/api/v1/sessions/s1/end')
  })

  it('knowledge collection + enqueue source', async () => {
    fetchMock
      .mockResolvedValueOnce(mockJSON({ id: 'k1', name: 'FAQ' }, 201))
      .mockResolvedValueOnce(mockJSON({ id: 'op1' }, 202))

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result: create } = renderHook(() => useCreateCollection(), { wrapper: wrap(client) })
    await create.current.mutateAsync({ name: 'FAQ' })
    const { result: enq } = renderHook(() => useEnqueueSource('k1'), { wrapper: wrap(client) })
    await enq.current.mutateAsync({ kind: 'url', title: 'Docs', uri: 'https://docs.x' })

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/knowledge/collections')
    expect(fetchMock.mock.calls[1]?.[0]).toBe('/api/v1/knowledge/collections/k1/sources')
  })
})
