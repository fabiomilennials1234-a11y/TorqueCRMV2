import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useCompleteTask, useCreateTask, useTasks } from '@/hooks/useTasks'

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

describe('useTasks', () => {
  it('fetches /api/v1/tasks and returns data[]', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ data: [{ id: 't1', title: 'Task' }] }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useTasks(), { wrapper: wrap(client) })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.[0]?.id).toBe('t1')
  })

  it('POSTs /api/v1/tasks on create and PATCHes /complete', async () => {
    fetchMock
      .mockResolvedValueOnce({
        ok: true,
        status: 201,
        json: async () => ({ id: 'new' }),
        headers: new Headers(),
      } as Response)
      .mockResolvedValueOnce({
        ok: true,
        status: 204,
        json: async () => null,
        headers: new Headers(),
      } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result: create } = renderHook(() => useCreateTask(), { wrapper: wrap(client) })
    await create.current.mutateAsync({ title: 'x', assigned_to: 'me' })

    const { result: complete } = renderHook(() => useCompleteTask('new'), { wrapper: wrap(client) })
    await complete.current.mutateAsync({})

    const calls = fetchMock.mock.calls
    expect(calls[0]?.[0]).toBe('/api/v1/tasks')
    expect((calls[0]?.[1] as RequestInit | undefined)?.method).toBe('POST')
    expect(calls[1]?.[0]).toBe('/api/v1/tasks/new/complete')
  })
})
