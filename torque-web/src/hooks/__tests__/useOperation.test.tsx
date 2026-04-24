import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useOperationCancel, useOperationStatus, useOperationSubmit } from '@/hooks/useOperation'

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

describe('useOperation', () => {
  it('useOperationSubmit POSTs /api/v1/operations', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 202,
      json: async () => ({
        id: 'op1',
        kind: 'x',
        status: 'pending',
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useOperationSubmit(), { wrapper: wrap(client) })
    await result.current.mutateAsync({ kind: 'leads.import' })

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/operations')
    expect((fetchMock.mock.calls[0]?.[1] as RequestInit | undefined)?.method).toBe('POST')
  })

  it('useOperationStatus fetches /api/v1/operations/:id', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({
        id: 'op1',
        kind: 'x',
        status: 'succeeded',
        result: null,
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useOperationStatus('op1'), { wrapper: wrap(client) })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/operations/op1')
  })

  it('useOperationCancel DELETEs /api/v1/operations/:id', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 204,
      json: async () => null,
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useOperationCancel('op1'), { wrapper: wrap(client) })
    await result.current.mutateAsync()
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/operations/op1')
    expect((fetchMock.mock.calls[0]?.[1] as RequestInit | undefined)?.method).toBe('DELETE')
  })
})
