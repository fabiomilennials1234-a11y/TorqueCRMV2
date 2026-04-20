import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  useMovePipeEntry,
  usePipeEntries,
  usePipeStages,
  usePipes,
} from '@/hooks/usePipes'

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
  return { ok: true, status, json: async () => body, headers: new Headers() } as Response
}

describe('usePipes read hooks', () => {
  it('usePipes GETs /api/v1/pipes', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({ data: [{ id: 'p1', kind: 'whatsapp', name: 'WhatsApp', is_default: true, is_archived: false, position: 0 }] }))
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => usePipes(), { wrapper: wrap(client) })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/pipes')
  })

  it('usePipeStages hits /api/v1/pipes/:id/stages when id present', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({ data: [] }))
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => usePipeStages('p1'), { wrapper: wrap(client) })
    await waitFor(() => expect(fetchMock).toHaveBeenCalledOnce())
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/pipes/p1/stages')
  })

  it('usePipeEntries disabled when id is undefined', () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => usePipeEntries(undefined), { wrapper: wrap(client) })
    expect(result.current.fetchStatus).toBe('idle')
    expect(fetchMock).not.toHaveBeenCalled()
  })
})

describe('useMovePipeEntry', () => {
  it('POSTs /api/v1/pipes/:id/entries/move', async () => {
    fetchMock.mockResolvedValueOnce(mockJSON({
      id: 'e1', stage_id: 's2', lead_id: 'l1',
      entered_stage_at: '2026-04-20T00:00:00Z',
    }))
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const { result } = renderHook(() => useMovePipeEntry('p1'), { wrapper: wrap(client) })
    await result.current.mutateAsync({ lead_id: 'l1', new_stage_id: 's2' })
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/pipes/p1/entries/move')
  })
})
