import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  useBindAgentCollection,
  useEnqueueSource,
  useKnowledgeCollections,
  useKnowledgeSources,
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

describe('S39 — knowledge hooks', () => {
  it('useKnowledgeCollections GETs /api/v1/knowledge/collections', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({
        data: [
          {
            id: 'c1',
            name: 'FAQ',
            description: null,
            source_count: 3,
            created_at: '2026-04-20T00:00:00Z',
            updated_at: '2026-04-20T00:00:00Z',
          },
        ],
      })
    )
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useKnowledgeCollections(), { wrapper: wrap(client) })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toHaveLength(1)
    expect(result.current.data?.[0]?.source_count).toBe(3)
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/knowledge/collections')
  })

  it('useKnowledgeSources stays disabled while collectionId is undefined', async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useKnowledgeSources(undefined), {
      wrapper: wrap(client),
    })
    // No fetch should fire while collection is absent.
    expect(fetchMock).not.toHaveBeenCalled()
    expect(result.current.isFetched).toBe(false)
  })

  it('useKnowledgeSources GETs scoped path with collectionId', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({
        data: [
          {
            id: 's1',
            collection_id: 'c1',
            kind: 'text',
            title: 'Doc',
            status: 'ready',
            created_at: '2026-04-20T00:00:00Z',
            updated_at: '2026-04-20T00:00:00Z',
          },
        ],
      })
    )
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useKnowledgeSources('c1'), { wrapper: wrap(client) })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/knowledge/collections/c1/sources')
    expect(result.current.data?.[0]?.status).toBe('ready')
  })

  it('useEnqueueSource POSTs with content for kind=text', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({ id: 'src1', status: 'queued' }, 202))
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useEnqueueSource('c1'), { wrapper: wrap(client) })
    await result.current.mutateAsync({ kind: 'text', title: 'T', content: 'body' })
    const call = fetchMock.mock.calls[0]
    expect(call?.[0]).toBe('/api/v1/knowledge/collections/c1/sources')
    const init = call?.[1] as RequestInit
    expect(init.method).toBe('POST')
    const body = JSON.parse(String(init.body))
    expect(body.kind).toBe('text')
    expect(body.content).toBe('body')
  })

  it('useBindAgentCollection PUTs with nullable collection_id', async () => {
    fetchMock
      .mockResolvedValueOnce(
        mockJSON({
          id: 'a1',
          name: 'A',
          system_prompt: 'p',
          model: 'm',
          temperature: 0.3,
          max_output_tokens: 1024,
          tools_allowlist: [],
          kill_switch: false,
          status: 'active',
          knowledge_collection_id: 'c1',
        })
      )
      .mockResolvedValueOnce(
        mockJSON({
          id: 'a1',
          name: 'A',
          system_prompt: 'p',
          model: 'm',
          temperature: 0.3,
          max_output_tokens: 1024,
          tools_allowlist: [],
          kill_switch: false,
          status: 'active',
          knowledge_collection_id: null,
        })
      )
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useBindAgentCollection('a1'), { wrapper: wrap(client) })
    await result.current.mutateAsync({ collection_id: 'c1' })
    expect((fetchMock.mock.calls[0]?.[1] as RequestInit).method).toBe('PUT')
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/agents/a1/knowledge-collection')
    await result.current.mutateAsync({ collection_id: null })
    const body = JSON.parse(String((fetchMock.mock.calls[1]?.[1] as RequestInit).body))
    expect(body.collection_id).toBeNull()
  })
})
