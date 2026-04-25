import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useConfirm, useMarkNoShow, useUpsertConfirmation } from '@/hooks/useConfirmations'

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

describe('useConfirmations', () => {
  it('useUpsertConfirmation PUTs /api/v1/confirmations/:entryId', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({
        pipe_entry_id: 'e1',
        meeting_at: '2026-04-20T10:00:00Z',
        confirmed_at: null,
        no_show: false,
      }),
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useUpsertConfirmation('e1'), { wrapper: wrap(client) })
    await result.current.mutateAsync({
      lead_id: 'l1',
      meeting_at: '2026-04-20T10:00:00Z',
    })

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/confirmations/e1')
    expect((fetchMock.mock.calls[0]?.[1] as RequestInit | undefined)?.method).toBe('PUT')
  })

  it('useConfirm POSTs /api/v1/confirmations/:entryId/confirm', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 204,
      json: async () => null,
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useConfirm('e1'), { wrapper: wrap(client) })
    await result.current.mutateAsync()

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/confirmations/e1/confirm')
  })

  it('useMarkNoShow POSTs /api/v1/confirmations/:entryId/no-show', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 204,
      json: async () => null,
      headers: new Headers(),
    } as Response)

    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useMarkNoShow('e1'), { wrapper: wrap(client) })
    await result.current.mutateAsync({ reason: 'esqueceu' })

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/confirmations/e1/no-show')
  })
})
