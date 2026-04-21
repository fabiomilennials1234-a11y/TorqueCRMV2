import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  useCreateMeeting,
  useDeleteMeeting,
  useMeetings,
  useSetMeetingStatus,
} from '@/hooks/useMeetings'

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

describe('useMeetings — hooks', () => {
  it('useMeetings stays disabled without window', () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useMeetings(undefined), { wrapper: wrap(client) })
    expect(fetchMock).not.toHaveBeenCalled()
    expect(result.current.isFetched).toBe(false)
  })

  it('useMeetings GETs with from/to URL-encoded', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({ data: [], from: '2026-04-13T00:00:00Z', to: '2026-05-20T00:00:00Z' })
    )
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(
      () =>
        useMeetings({
          from: '2026-04-13T00:00:00Z',
          to: '2026-05-20T00:00:00Z',
        }),
      { wrapper: wrap(client) }
    )
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    const url = String(fetchMock.mock.calls[0]?.[0])
    expect(url).toContain('/api/v1/meetings')
    expect(url).toContain('from=2026-04-13T00%3A00%3A00Z')
    expect(url).toContain('to=2026-05-20T00%3A00%3A00Z')
  })

  it('useCreateMeeting POSTs body with ISO timestamps', async () => {
    fetchMock.mockResolvedValueOnce(
      mockJSON({
        id: 'm1',
        title: 'Demo',
        starts_at: '2026-04-20T14:00:00Z',
        ends_at: '2026-04-20T15:00:00Z',
        status: 'scheduled',
        created_at: '2026-04-20T13:00:00Z',
        updated_at: '2026-04-20T13:00:00Z',
      })
    )
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useCreateMeeting(), { wrapper: wrap(client) })
    await result.current.mutateAsync({
      title: 'Demo',
      starts_at: '2026-04-20T14:00:00Z',
      ends_at: '2026-04-20T15:00:00Z',
    })
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit
    expect(init.method).toBe('POST')
    expect(JSON.parse(String(init.body)).title).toBe('Demo')
  })

  it('useSetMeetingStatus posts /:id/status', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON(null, 204))
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useSetMeetingStatus('m1'), { wrapper: wrap(client) })
    await result.current.mutateAsync({ status: 'completed' })
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/meetings/m1/status')
  })

  it('useDeleteMeeting DELETEs /:id', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON(null, 204))
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useDeleteMeeting('m1'), { wrapper: wrap(client) })
    await result.current.mutateAsync()
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/meetings/m1')
    expect((fetchMock.mock.calls[0]?.[1] as RequestInit).method).toBe('DELETE')
  })
})
