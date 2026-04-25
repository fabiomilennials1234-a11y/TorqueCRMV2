import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useArchivePipe, useCreatePipe, useCreateStage, useUpdateStage } from '@/hooks/usePipes'

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

/**
 * Admin methods on usePipes (S22). Every mutation sits under RequireRole
 * on the backend; non-admin callers hit 403 handled globally.
 */
describe('usePipes admin', () => {
  it('useCreatePipe POSTs /api/v1/pipes', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 201,
      json: async () => ({
        id: 'p1',
        kind: 'custom',
        name: 'Upsell',
        is_default: false,
        is_archived: false,
        position: 10,
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useCreatePipe(), { wrapper: wrap(client) })
    await result.current.mutateAsync({ kind: 'custom', name: 'Upsell' })

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/pipes')
    expect((fetchMock.mock.calls[0]?.[1] as RequestInit | undefined)?.method).toBe('POST')
  })

  it('useArchivePipe DELETEs /api/v1/pipes/:id', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 204,
      json: async () => null,
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useArchivePipe('p1'), { wrapper: wrap(client) })
    await result.current.mutateAsync()

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/pipes/p1')
    expect((fetchMock.mock.calls[0]?.[1] as RequestInit | undefined)?.method).toBe('DELETE')
  })

  it('useCreateStage POSTs /api/v1/pipes/:id/stages', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 201,
      json: async () => ({
        id: 's1',
        name: 'New',
        position: 0,
        is_final_positive: false,
        is_final_negative: false,
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useCreateStage('p1'), { wrapper: wrap(client) })
    await result.current.mutateAsync({ name: 'New', position: 0 })

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/pipes/p1/stages')
  })

  it('useUpdateStage PATCHes /api/v1/pipes/:pipeId/stages/:stageId', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({
        id: 's1',
        name: 'Updated',
        position: 1,
        is_final_positive: false,
        is_final_negative: false,
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useUpdateStage('p1', 's1'), { wrapper: wrap(client) })
    await result.current.mutateAsync({ name: 'Updated' })

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/pipes/p1/stages/s1')
    expect((fetchMock.mock.calls[0]?.[1] as RequestInit | undefined)?.method).toBe('PATCH')
  })
})
